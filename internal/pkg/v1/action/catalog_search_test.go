package action_test

// import (

// 	. "github.com/onsi/ginkgo"
// 	. "github.com/onsi/gomega"

// 	"fmt"
// 	"time"

// 	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
// 	"github.com/operator-framework/kubectl-operator/pkg/action"
// 	olmv1 "github.com/operator-framework/operator-controller/api/v1"
// 	corev1 "k8s.io/api/core/v1"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/util/intstr"
// 	"k8s.io/client-go/rest"
// 	"sigs.k8s.io/controller-runtime/pkg/client/fake"
// )

// var _ = Describe("", func() {
// 	catalogdNamespace := "test"
// 	var serviceName, serviceNamespace, podName, catalogName string
// 	var servicePort, podPort int32
// 	var serverHost, serverPort string

// 	catalog := olmv1.ClusterCatalog{
// 		Status: olmv1.ClusterCatalogStatus{
// 			Conditions: []metav1.Condition{{
// 				Type: olmv1.TypeServing,
// 				Status: metav1.ConditionTrue,
// 			}},
// 			URLs: &olmv1.ClusterCatalogURLs{
// 				Base: fmt.Sprintf("http://%s.%s:%d", serviceName, serviceNamespace, servicePort), //port optional if scheme present
// 			},
// 		},
// 	}
// 	secret := corev1.Secret{
// 		Data: map[string][]byte{
// 			"ca.crt": []byte{},//AppendCertsFromPEM
// 		},
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name: "catalogd-*",
// 			Namespace: catalogdNamespace,
// 		},
// 	}
// 	svc := corev1.Service{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name: serviceName,
// 			Namespace: serviceNamespace,
// 		},
// 		Spec: corev1.ServiceSpec{
// 			Ports: []corev1.ServicePort{{
// 				Port: int32(servicePort),
// 				TargetPort: intstr.IntOrString{IntVal: podPort},
// 			}},
// 		},
// 	}
// 	endpoints := corev1.Endpoints{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name: serviceName,
// 			Namespace: serviceNamespace,
// 		},
// 		Subsets: []corev1.EndpointSubset{{
// 			Addresses: []corev1.EndpointAddress{{
// 				TargetRef: &corev1.ObjectReference{
// 					Name: fmt.Sprintf("%s", podName),
// 				},
// 			}},
// 		}},
// 	}
// 	scheme, err := action.NewScheme()
// 	Expect(err).ShouldNot(HaveOccurred())
// 	fakeClientBuilder := fake.NewClientBuilder()
// 	fakeClientBuilder.WithScheme(scheme)
// 	fakeClientBuilder.WithObjects(&catalog, &secret, &svc, &endpoints)
// 	cfg := &action.Configuration {
// 		Config: &rest.Config{
// 			Host: "localhost",
// 			APIPath: "",
// 			Timeout: 1*time.Minute,
// 			TLSClientConfig: rest.TLSClientConfig{
// 				ServerName: "",
// 				NextProtos: []string{},
// 				CAData: []byte{}, //certData, keyData not here?
// 			},

// 		},
// 		Client: fakeClientBuilder.Build(),
// 		Scheme: scheme,
// 	}
// 	searchCmd := v1action.NewCatalogSearch(cfg)
// 	searchCmd.Timeout = "1m"
// 	searchCmd.CatalogName = catalogName
// 	BeforeEach(func(){

// 	})
// 	It("", func() {
// 		Expect("")
// 	})
// 	// server: localhost: fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s/portforward", c.cfg.Host, namespace, podName);
// 	//  ports:     []string{fmt.Sprintf("0:%d", podPort)},
// //	protocol := resp.Header.Get(httpstream.HeaderProtocolVersion) //must equal PortForwardProtocolV1Name

// })

// func newServer(host string, port int) {

// }
