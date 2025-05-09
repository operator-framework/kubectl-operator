package action

import (
	"context"

	"github.com/spf13/pflag"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	olmv1 "github.com/operator-framework/operator-controller/api/v1"
	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
)

func NewScheme() (*runtime.Scheme, error) {
	sch := runtime.NewScheme()
	for _, f := range []func(*runtime.Scheme) error{
		v1alpha1.AddToScheme,
		operatorsv1.AddToScheme,
		v1.AddToScheme,
		apiextensionsv1.AddToScheme,
		olmv1.AddToScheme,
	} {
		if err := f(sch); err != nil {
			return nil, err
		}
	}
	return sch, nil
}

type Configuration struct {
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

	c.Scheme = sch
	c.Client = &operatorClient{cl}
	c.Namespace = ns

	return nil
}

type operatorClient struct {
	client.Client
}

func (c *operatorClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	opts = append(opts, client.FieldOwner("kubectl-operator"))
	return c.Client.Create(ctx, obj, opts...)
}
