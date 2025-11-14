package action

import (
	"context"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type CatalogInstalledGet struct {
	config      *action.Configuration
	CatalogName string

	Selector string

	Logf func(string, ...interface{})
}

func NewCatalogInstalledGet(cfg *action.Configuration) *CatalogInstalledGet {
	return &CatalogInstalledGet{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (i *CatalogInstalledGet) Run(ctx context.Context) ([]olmv1.ClusterCatalog, error) {
	// get
	if i.CatalogName != "" {
		var result olmv1.ClusterCatalog

		opKey := types.NamespacedName{Name: i.CatalogName}
		err := i.config.Client.Get(ctx, opKey, &result)
		if err != nil {
			return nil, err
		}

		return []olmv1.ClusterCatalog{result}, nil
	}

	// list
	var result olmv1.ClusterCatalogList
	listOpts := &client.ListOptions{}
	if len(i.Selector) > 0 {
		selector, err := labels.Parse(i.Selector)
		if err != nil {
			return nil, err
		}
		listOpts.LabelSelector = selector
	}
	err := i.config.Client.List(ctx, &result, listOpts)

	return result.Items, err
}
