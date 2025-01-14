package v1

import (
	"context"
	catalogdv1 "github.com/operator-framework/catalogd/api/v1"
	"github.com/operator-framework/kubectl-operator/internal/pkg/catalog"
	"github.com/operator-framework/kubectl-operator/pkg/action"
	"github.com/operator-framework/operator-registry/alpha/declcfg"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CatalogContent struct {
	config *action.Configuration

	CatalogName string
	WalkMetas   declcfg.WalkMetasReaderFunc
}

func NewCatalogContent(cfg *action.Configuration) *CatalogContent {
	return &CatalogContent{
		config: cfg,
	}
}

func (r *CatalogContent) Run(ctx context.Context) error {
	clusterCatalog := catalogdv1.ClusterCatalog{}
	if err := r.config.Client.Get(ctx, client.ObjectKey{Name: r.CatalogName}, &clusterCatalog); err != nil {
		return err
	}

	catalogClient := catalog.NewK8sClient(r.config.Config, r.config.Client, &clusterCatalog)
	catalogContent, err := catalogClient.V1().All(ctx)
	if err != nil {
		return err
	}
	defer catalogContent.Close()
	return declcfg.WalkMetasReader(catalogContent, r.WalkMetas)
}
