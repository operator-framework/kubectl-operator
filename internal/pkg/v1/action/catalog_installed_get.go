package action

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1catalogd "github.com/operator-framework/catalogd/api/v1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type CatalogInstalledGet struct {
	config      *action.Configuration
	CatalogName string

	Logf func(string, ...interface{})
}

func NewCatalogInstalledGet(cfg *action.Configuration) *CatalogInstalledGet {
	return &CatalogInstalledGet{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (i *CatalogInstalledGet) Run(ctx context.Context) ([]olmv1catalogd.ClusterCatalog, error) {
	// get
	if i.CatalogName != "" {
		var result olmv1catalogd.ClusterCatalog

		opKey := types.NamespacedName{Name: i.CatalogName}
		err := i.config.Client.Get(ctx, opKey, &result)
		if err != nil {
			return nil, err
		}

		return []olmv1catalogd.ClusterCatalog{result}, nil
	}

	// list
	var result olmv1catalogd.ClusterCatalogList
	err := i.config.Client.List(ctx, &result, &client.ListOptions{})

	return result.Items, err
}
