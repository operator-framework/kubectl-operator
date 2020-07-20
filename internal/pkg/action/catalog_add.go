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
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/joelanford/kubectl-operator/internal/pkg/catalog"
	"github.com/joelanford/kubectl-operator/internal/pkg/log"
)

type AddCatalog struct {
	config *Configuration

	CatalogSourceName string
	IndexImage        string
	DisplayName       string
	Publisher         string
	AddTimeout        time.Duration
	CleanupTimeout    time.Duration

	RegistryOptions []containerdregistry.RegistryOption

	registry *containerdregistry.Registry
}

func NewAddCatalog(cfg *Configuration) *AddCatalog {
	return &AddCatalog{
		config: cfg,
	}
}

func (a *AddCatalog) BindFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&a.DisplayName, "display-name", "d", "", "display name of the index")
	fs.StringVarP(&a.Publisher, "publisher", "p", "", "publisher of the index")
	fs.DurationVarP(&a.AddTimeout, "timeout", "t", time.Minute, "the amount of time to wait before cancelling the catalog addition")
	fs.DurationVar(&a.CleanupTimeout, "cleanup-timeout", time.Minute, "the amount to time to wait before cancelling cleanup")
}

func (a *AddCatalog) Run(ctx context.Context) (*v1alpha1.CatalogSource, error) {
	var err error
	a.registry, err = containerdregistry.NewRegistry(a.RegistryOptions...)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := a.registry.Destroy(); err != nil {
			log.Printf("registry cleanup: %v", err)
		}
	}()

	csKey := types.NamespacedName{
		Namespace: a.config.Namespace,
		Name:      a.CatalogSourceName,
	}

	labels, err := a.labelsFor(ctx, a.IndexImage)
	if err != nil {
		return nil, err
	}

	a.setDefaults(labels)

	opts := []catalog.Option{
		catalog.Image(a.IndexImage),
		catalog.DisplayName(a.DisplayName),
		catalog.Publisher(a.Publisher),
	}
	cs := catalog.Build(csKey, opts...)
	if err := a.add(ctx, cs); err != nil {
		defer a.cleanup(cs)
		return nil, err
	}
	return cs, nil
}

func (a *AddCatalog) labelsFor(ctx context.Context, indexImage string) (map[string]string, error) {
	simpleRef := image.SimpleReference(indexImage)
	if err := a.registry.Pull(ctx, simpleRef); err != nil {
		return nil, fmt.Errorf("pull image: %v", err)
	}
	labels, err := a.registry.Labels(ctx, simpleRef)
	if err != nil {
		return nil, fmt.Errorf("get image labels: %v", err)
	}
	return labels, nil
}

func (a *AddCatalog) setDefaults(labels map[string]string) {
	if a.DisplayName == "" {
		if v, ok := labels["operators.operatorframework.io.index.display-name"]; ok {
			a.DisplayName = v
		}
	}
	if a.Publisher == "" {
		if v, ok := labels["operators.operatorframework.io.index.publisher"]; ok {
			a.Publisher = v
		}
	}
}

func (a *AddCatalog) add(ctx context.Context, cs *v1alpha1.CatalogSource) error {
	if err := a.config.Client.Create(ctx, cs); err != nil {
		return fmt.Errorf("create catalogsource: %v", err)
	}

	csKey, err := client.ObjectKeyFromObject(cs)
	if err != nil {
		return fmt.Errorf("get catalogsource key: %v", err)
	}
	if err := wait.PollImmediateUntil(time.Millisecond*250, func() (bool, error) {
		if err := a.config.Client.Get(ctx, csKey, cs); err != nil {
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

func (a *AddCatalog) cleanup(cs *v1alpha1.CatalogSource) {
	ctx, cancel := context.WithTimeout(context.Background(), a.CleanupTimeout)
	defer cancel()
	if err := a.config.Client.Delete(ctx, cs); err != nil {
		log.Printf("delete catalogsource %q: %v", cs.Name, err)
	}
}
