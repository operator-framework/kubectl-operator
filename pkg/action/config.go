package action

import (
	catalogdv1 "github.com/operator-framework/catalogd/api/v1"
	"github.com/spf13/pflag"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ofapiv1 "github.com/operator-framework/api/pkg/operators/v1"
	ofapiv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	ocv1 "github.com/operator-framework/operator-controller/api/v1"
	packageserveroperatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
)

func NewScheme() (*runtime.Scheme, error) {
	sch := scheme.Scheme
	for _, f := range []func(*runtime.Scheme) error{
		ofapiv1alpha1.AddToScheme,
		packageserveroperatorsv1.AddToScheme,
		ofapiv1.AddToScheme,
		apiextensionsv1.AddToScheme,
		ocv1.AddToScheme,
		catalogdv1.AddToScheme,
	} {
		if err := f(sch); err != nil {
			return nil, err
		}
	}
	return sch, nil
}

type Configuration struct {
	Config    *rest.Config
	Client    client.Client
	Namespace string
	Scheme    *runtime.Scheme

	overrides *clientcmd.ConfigOverrides
}

func (c *Configuration) BindFlags(fs *pflag.FlagSet) {
	if c.overrides == nil {
		c.overrides = &clientcmd.ConfigOverrides{}
	}
	clientcmd.BindOverrideFlags(c.overrides, fs, clientcmd.ConfigOverrideFlags{
		ContextOverrideFlags: clientcmd.ContextOverrideFlags{
			Namespace: clientcmd.FlagInfo{
				LongName:    "namespace",
				ShortName:   "n",
				Default:     "",
				Description: "If present, namespace scope for this CLI request",
			},
		},
	})
}

func (c *Configuration) Load() error {
	if c.overrides == nil {
		c.overrides = &clientcmd.ConfigOverrides{}
	}
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	mergedConfig, err := loadingRules.Load()
	if err != nil {
		return err
	}
	cfg := clientcmd.NewDefaultClientConfig(*mergedConfig, c.overrides)
	cc, err := cfg.ClientConfig()
	if err != nil {
		return err
	}

	ns, _, err := cfg.Namespace()
	if err != nil {
		return err
	}

	sch, err := NewScheme()
	if err != nil {
		return err
	}
	cl, err := client.New(cc, client.Options{
		Scheme: sch,
	})
	if err != nil {
		return err
	}

	c.Config = cc
	c.Scheme = sch
	c.Client = client.WithFieldOwner(cl, "kubectl-operator")
	c.Namespace = ns

	return nil
}
