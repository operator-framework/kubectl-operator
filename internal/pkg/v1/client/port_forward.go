package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"
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

func NewK8sClient(cfg *rest.Config, cl client.Client, caNamespace string) Client {
	c := &portForwardClient{
		cfg: cfg,
		cl:  cl,
	}
	c.httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				RootCAs:    c.loadKnownCAs(caNamespace),
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
		return nil, fmt.Errorf("failed to parse ClusterCatalog URL %q: %w", cc.Status.URLs.Base, err)
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
	servicePort, err := strconv.ParseInt(servicePortStr, 10, 32)
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
	podName, podPort, err := c.getPodAndPortForService(ctx, namespace, serviceName, servicePort)
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
func (c *portForwardClient) getPodAndPortForService(ctx context.Context, namespace, serviceName string, servicePort int64) (string, int, error) {
	svc := corev1.Service{}
	svcKey := client.ObjectKey{Name: serviceName, Namespace: namespace}
	if err := c.cl.Get(ctx, svcKey, &svc); err != nil {
		return "", -1, err
	}

	podPort := -1
	for _, port := range svc.Spec.Ports {
		if int64(port.Port) == servicePort {
			podPort = port.TargetPort.IntValue()
			break
		}
	}
	if podPort == -1 {
		return "", -1, fmt.Errorf("service %q has no port %q", serviceName, servicePort)
	}

	ep := discoveryv1.EndpointSliceList{}
	err := c.cl.List(ctx, &ep, client.MatchingLabels{discoveryv1.LabelServiceName: serviceName}, client.InNamespace(namespace))
	if err != nil {
		return "", -1, err
	}

	var pods []string
	for _, e := range ep.Items {
		for _, a := range e.Endpoints {
			pods = append(pods, a.TargetRef.Name)
		}
	}
	if len(pods) == 0 {
		return "", -1, fmt.Errorf("no pods ready for service %q", svcKey)
	}

	// Select the first pod (or you could add load balancing logic here)
	return pods[0], podPort, nil
}

// Port forwarding logic to connect to a pod
func (c *portForwardClient) getPortForwarder(namespace, podName string, podPort int) (*portforward.PortForwarder, error) {
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

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport, Timeout: c.cfg.Timeout}, "POST", portForwardURL)

	ports := []string{fmt.Sprintf("0:%d", podPort)}
	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{}, 1)

	pf, err := portforward.New(dialer, ports, stopChan, readyChan, io.Discard, os.Stderr)
	if err != nil {
		return nil, err
	}

	return pf, nil
}

func (c *portForwardClient) loadKnownCAs(caNamespace string) *x509.CertPool {
	// for openshift, reference annotation service.beta.openshift.io/serving-cert-secret-name
	// on the openshift-catalogd/catalogd-service service
	secretPrefix := "catalogd"
	knownCAsSecrets := []struct {
		Namespace string
		Key       string
	}{
		{caNamespace, "ca.crt"},
	}
	rootCAs := x509.NewCertPool()
	for _, secretInfo := range knownCAsSecrets {
		secret := corev1.SecretList{}
		if err := c.cl.List(context.TODO(), &secret, &client.ListOptions{Namespace: caNamespace}); err != nil {
			continue
		}
		if len(secret.Items) == 0 {
			continue
		}
		for _, caSecret := range secret.Items {
			if strings.HasPrefix(caSecret.Name, secretPrefix) && len(caSecret.Data[secretInfo.Key]) > 0 {
				rootCAs.AppendCertsFromPEM(caSecret.Data[secretInfo.Key])
				continue
			}
		}
	}
	return rootCAs
}
