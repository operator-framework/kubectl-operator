package action

import (
	"context"
	"fmt"
	"time"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/image"
	"github.com/operator-framework/operator-registry/pkg/image/containerdregistry"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/cluster-api/util/container"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/joelanford/kubectl-operator/internal/pkg/catalog"
	"github.com/joelanford/kubectl-operator/internal/pkg/log"
)

type InstallCatalog struct {
	config *Configuration

	IndexImage     string
	DisplayName    string
	Publisher      string
	InstallTimeout time.Duration
	CleanupTimeout time.Duration

	RegistryOptions []containerdregistry.RegistryOption

	registry *containerdregistry.Registry
}

func NewInstallCatalog(cfg *Configuration) *InstallCatalog {
	return &InstallCatalog{
		config: cfg,
	}
}

func (i *InstallCatalog) BindFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&i.DisplayName, "display-name", "d", "", "display name of the index")
	fs.StringVarP(&i.Publisher, "publisher", "p", "", "publisher of the index")
	fs.DurationVarP(&i.InstallTimeout, "timeout", "t", time.Minute, "the amount of time to wait before cancelling the install")
	fs.DurationVar(&i.CleanupTimeout, "cleanup-timeout", time.Minute, "the amount to time to wait before cancelling cleanup")
}

func (i *InstallCatalog) Run(ctx context.Context) (*v1alpha1.CatalogSource, error) {
	var err error
	i.registry, err = containerdregistry.NewRegistry(i.RegistryOptions...)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := i.registry.Destroy(); err != nil {
			log.Printf("registry cleanup: %v", err)
		}
	}()

	imageRef, err := container.ImageFromString(i.IndexImage)
	if err != nil {
		return nil, err
	}
	csKey := types.NamespacedName{
		Namespace: i.config.Namespace,
		Name:      imageRef.Name,
	}

	labels, err := i.labelsFor(ctx, i.IndexImage)
	if err != nil {
		return nil, err
	}

	i.setDefaults(labels)

	opts := []catalog.Option{
		catalog.Image(i.IndexImage),
		catalog.DisplayName(i.DisplayName),
		catalog.Publisher(i.Publisher),
	}
	cs := catalog.Build(csKey, opts...)
	if err := i.install(ctx, cs); err != nil {
		defer i.cleanup(cs)
		return nil, err
	}
	return cs, nil
}

func (i *InstallCatalog) labelsFor(ctx context.Context, indexImage string) (map[string]string, error) {
	simpleRef := image.SimpleReference(indexImage)
	if err := i.registry.Pull(ctx, simpleRef); err != nil {
		return nil, fmt.Errorf("pull image: %v", err)
	}
	labels, err := i.registry.Labels(ctx, simpleRef)
	if err != nil {
		return nil, fmt.Errorf("get image labels: %v", err)
	}
	return labels, nil
}

func (i *InstallCatalog) setDefaults(labels map[string]string) {
	if i.DisplayName == "" {
		if v, ok := labels["operators.operatorframework.io.index.display-name"]; ok {
			i.DisplayName = v
		}
	}
	if i.Publisher == "" {
		if v, ok := labels["operators.operatorframework.io.index.publisher"]; ok {
			i.Publisher = v
		}
	}
}

func (i *InstallCatalog) install(ctx context.Context, cs *v1alpha1.CatalogSource) error {
	if err := i.config.Client.Create(ctx, cs); err != nil {
		return fmt.Errorf("create catalogsource: %v", err)
	}

	csKey, err := client.ObjectKeyFromObject(cs)
	if err != nil {
		return fmt.Errorf("get catalogsource key: %v", err)
	}
	if err := wait.PollImmediateUntil(time.Millisecond*250, func() (bool, error) {
		if err := i.config.Client.Get(ctx, csKey, cs); err != nil {
			return false, err
		}
		if cs.Status.GRPCConnectionState != nil {
			if cs.Status.GRPCConnectionState.LastObservedState == "READY" {
				return true, nil
			}
		}
		return false, nil
	}, ctx.Done()); err != nil {
		return fmt.Errorf("catalogsource connection not ready: %v", err)
	}
	return nil
}

func (i *InstallCatalog) cleanup(cs *v1alpha1.CatalogSource) {
	ctx, cancel := context.WithTimeout(context.Background(), i.CleanupTimeout)
	defer cancel()
	if err := i.config.Client.Delete(ctx, cs); err != nil {
		log.Printf("delete catalogsource %q: %v", cs.Name, err)
	}
}
