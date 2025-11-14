package action

import (
	"context"
	"errors"
	"fmt"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CatalogDelete struct {
	config      *action.Configuration
	CatalogName string

	DeleteAll bool

	DryRun string
	Output string
	Logf   func(string, ...interface{})
}

func NewCatalogDelete(cfg *action.Configuration) *CatalogDelete {
	return &CatalogDelete{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (i *CatalogDelete) Run(ctx context.Context) ([]olmv1.ClusterCatalog, error) {
	// validate
	if i.DeleteAll && i.CatalogName != "" {
		return nil, ErrNameAndSelector
	}

	// delete single, specified catalog
	if !i.DeleteAll {
		obj, err := i.deleteCatalog(ctx, i.CatalogName)
		return []olmv1.ClusterCatalog{obj}, err
	}

	// delete all existing catalogs
	var catatalogList olmv1.ClusterCatalogList
	if err := i.config.Client.List(ctx, &catatalogList); err != nil {
		return nil, err
	}
	if len(catatalogList.Items) == 0 {
		return nil, ErrNoResourcesFound
	}

	errs := make([]error, 0, len(catatalogList.Items))
	result := []olmv1.ClusterCatalog{}
	for _, catalog := range catatalogList.Items {
		if obj, err := i.deleteCatalog(ctx, catalog.Name); err != nil {
			errs = append(errs, fmt.Errorf("failed deleting catalog %q: %w", catalog.Name, err))
		} else {
			result = append(result, obj)
		}
	}

	return result, errors.Join(errs...)
}

func (i *CatalogDelete) deleteCatalog(ctx context.Context, name string) (olmv1.ClusterCatalog, error) {
	op := &olmv1.ClusterCatalog{}
	op.SetName(name)

	if i.DryRun == DryRunAll {
		err := i.config.Client.Delete(ctx, op, client.DryRunAll)
		return *op, err
	}
	if err := i.config.Client.Delete(ctx, op); err != nil {
		return *op, err
	}

	return *op, waitForDeletion(ctx, i.config.Client, op)
}
