package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client interface {
	V1() V1Client
}

type V1Client interface {
	All(ctx context.Context, cc *olmv1.ClusterCatalog) (io.ReadCloser, error)
}

type LiveClient struct {
	HTTPClient *http.Client
	BaseURL    *url.URL
}

func (c *LiveClient) V1() V1Client {
	return &LiveClientV1{c}
}

type LiveClientV1 struct {
	*LiveClient
}

func (c *LiveClientV1) All(ctx context.Context, _ *olmv1.ClusterCatalog) (io.ReadCloser, error) {
	allURL := c.LiveClient.BaseURL.JoinPath("api", "v1", "all").String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, allURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.LiveClient.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func NewK8sClient(cfg *rest.Config, cl client.Client) Client {
	c := &portForwardClient{
		cfg: cfg,
		cl:  cl,
	}
	c.httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: c.loadKnownCAs(),
			},
		},
	}
	return c
}

type portForwardClient struct {
	cfg        *rest.Config
	cl         client.Client
	httpClient *http.Client
}

func (c *portForwardClient) V1() V1Client {
	return &portForwardClientV1{c}
}

type portForwardClientV1 struct {
	*portForwardClient
}

func (c *portForwardClientV1) All(ctx context.Context, cc *olmv1.ClusterCatalog) (io.ReadCloser, error) {
	if !meta.IsStatusConditionTrue(cc.Status.Conditions, olmv1.TypeServing) {
		return nil, fmt.Errorf("cluster catalog %q is not serving", cc.Name)
	}
	if cc.Status.URLs == nil {
		return nil, fmt.Errorf("cluster catalog %q has no URLs", cc.Name)
	}
	baseURL, err := url.Parse(cc.Status.URLs.Base)
	if err != nil {
		return nil, err
	}
	serviceHostname := baseURL.Hostname()
	servicePortStr := baseURL.Port()
	if servicePortStr == "" {
		switch baseURL.Scheme {
		case "http":
			servicePortStr = "80"
		case "https":
			servicePortStr = "443"
		}
	}
	servicePort, err := strconv.Atoi(servicePortStr)
	if err != nil {
		return nil, err
	}

	labels := strings.Split(serviceHostname, ".")
	if len(labels) < 2 {
		return nil, fmt.Errorf("invalid base URL %q", cc.Status.URLs.Base)
	}
	serviceName := labels[0]
	namespace := labels[1]

	// Find a pod and pod port for the given service
	podName, podPort, err := c.getPodAndPortForService(ctx, namespace, serviceName, int32(servicePort))
	if err != nil {
		return nil, err
	}

	pf, err := c.getPortForwarder(namespace, podName, podPort)
	if err != nil {
		return nil, err
	}

	fwdErr := make(chan error, 1)
	go func() {
		fwdErr <- pf.ForwardPorts()
	}()

	defer pf.Close()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-fwdErr:
		return nil, err
	case <-pf.Ready:
	}
	forwardedPorts, err := pf.GetPorts()
	if err != nil {
		return nil, err
	}
	if len(forwardedPorts) != 1 {
		return nil, fmt.Errorf("expected 1 forwarded port, got %d", len(forwardedPorts))
	}
	localPort := forwardedPorts[0].Local

	localURL := url.URL{
		Scheme: baseURL.Scheme,
		Host:   fmt.Sprintf("localhost:%d", localPort),
		Path:   baseURL.Path,
	}
	liveClient := &LiveClient{
		HTTPClient: c.httpClient,
		BaseURL:    &localURL,
	}
	return liveClient.V1().All(ctx, cc)
}

// Get a pod for a given service
func (c *portForwardClient) getPodAndPortForService(ctx context.Context, namespace, serviceName string, servicePort int32) (string, int32, error) {
	svc := corev1.Service{}
	if err := c.cl.Get(ctx, client.ObjectKey{Name: serviceName, Namespace: namespace}, &svc); err != nil {
		return "", -1, err
	}

	podPort := -1
	for _, port := range svc.Spec.Ports {
		if port.Port == servicePort {
			podPort = port.TargetPort.IntValue()
			break
		}
	}
	if podPort == -1 {
		return "", -1, fmt.Errorf("service %q has no port %q", serviceName, servicePort)
	}

	endpoints := corev1.Endpoints{}
	if err := c.cl.Get(ctx, client.ObjectKey{Name: serviceName, Namespace: namespace}, &endpoints); err != nil {
		return "", -1, err
	}

	readyAddresses := []corev1.EndpointAddress{}
	for _, subset := range endpoints.Subsets {
		readyAddresses = append(readyAddresses, subset.Addresses...)
	}

	randAddress := rand.Int31n(int32(len(readyAddresses)))
	address := readyAddresses[randAddress]
	podName := address.TargetRef.Name

	// Select the first pod (or you could add load balancing logic here)
	return podName, int32(podPort), nil
}

// Port forwarding logic to connect to a pod
func (c *portForwardClient) getPortForwarder(namespace, podName string, podPort int32) (*portforward.PortForwarder, error) {
	apiserverURL, err := url.Parse(c.cfg.Host)
	if err != nil {
		return nil, err
	}

	portForwardURL := apiserverURL.JoinPath(
		"api", "v1",
		"namespaces", namespace,
		"pods", podName, "portforward",
	)

	transport, upgrader, err := spdy.RoundTripperFor(c.cfg)
	if err != nil {
		return nil, err
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", portForwardURL)

	ports := []string{fmt.Sprintf("0:%d", podPort)}
	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{}, 1)

	pf, err := portforward.New(dialer, ports, stopChan, readyChan, io.Discard, os.Stderr)
	if err != nil {
		return nil, err
	}

	return pf, nil
}

func (c *portForwardClient) loadKnownCAs() *x509.CertPool {
	knownCAsSecrets := []struct {
		Namespace string
		Name      string
		Key       string
	}{
		{"olmv1-system", "olmv1-cert", "ca.crt"},
	}
	rootCAs := x509.NewCertPool()
	for _, secretInfo := range knownCAsSecrets {
		secret := corev1.Secret{}
		if err := c.cl.Get(context.TODO(), client.ObjectKey{Name: secretInfo.Name, Namespace: secretInfo.Namespace}, &secret); err != nil {
			continue
		}
		caCert, ok := secret.Data[secretInfo.Key]
		if !ok {
			continue
		}
		rootCAs.AppendCertsFromPEM(caCert)
	}
	return rootCAs
}
