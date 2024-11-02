package v1

import (
	"context"
	catalogdv1 "github.com/operator-framework/catalogd/api/v1"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type CatalogRemove struct {
	config *action.Configuration

	CatalogName string
}

func NewCatalogRemove(cfg *action.Configuration) *CatalogRemove {
	return &CatalogRemove{
		config: cfg,
	}
}

func (r *CatalogRemove) Run(ctx context.Context) error {
	clusterCatalog := catalogdv1.ClusterCatalog{}
	clusterCatalog.SetName(r.CatalogName)
	return deleteAndWait(ctx, r.config.Client, &clusterCatalog)
}
