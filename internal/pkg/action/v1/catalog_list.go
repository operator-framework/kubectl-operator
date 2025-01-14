package v1

import (
	"context"

	catalogdv1 "github.com/operator-framework/catalogd/api/v1"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type CatalogList struct {
	config *action.Configuration
}

func NewCatalogList(cfg *action.Configuration) *CatalogList {
	return &CatalogList{cfg}
}

func (l *CatalogList) Run(ctx context.Context) ([]catalogdv1.ClusterCatalog, error) {
	clusterCatalogList := catalogdv1.ClusterCatalogList{}
	if err := l.config.Client.List(ctx, &clusterCatalogList); err != nil {
		return nil, err
	}
	return clusterCatalogList.Items, nil
}
