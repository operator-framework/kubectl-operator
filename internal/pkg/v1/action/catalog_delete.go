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
	DeleteAll   bool

	Logf func(string, ...interface{})

	DryRun string
	Output string
}

func NewCatalogDelete(cfg *action.Configuration) *CatalogDelete {
	return &CatalogDelete{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (cd *CatalogDelete) Run(ctx context.Context) ([]olmv1.ClusterCatalog, error) {
	// validate
	if cd.DeleteAll && cd.CatalogName != "" {
		return nil, ErrNameAndSelector
	}

	// delete single, specified catalog
	if !cd.DeleteAll {
		obj, err := cd.deleteCatalog(ctx, cd.CatalogName)
		return []olmv1.ClusterCatalog{obj}, err
	}

	// delete all existing catalogs
	var catatalogList olmv1.ClusterCatalogList
	if err := cd.config.Client.List(ctx, &catatalogList); err != nil {
		return nil, err
	}
	if len(catatalogList.Items) == 0 {
		return nil, ErrNoResourcesFound
	}

	errs := make([]error, 0, len(catatalogList.Items))
	result := []olmv1.ClusterCatalog{}
	for _, catalog := range catatalogList.Items {
		if obj, err := cd.deleteCatalog(ctx, catalog.Name); err != nil {
			errs = append(errs, fmt.Errorf("failed deleting catalog %q: %w", catalog.Name, err))
		} else {
			result = append(result, obj)
		}
	}

	return result, errors.Join(errs...)
}

func (cd *CatalogDelete) deleteCatalog(ctx context.Context, name string) (olmv1.ClusterCatalog, error) {
	op := &olmv1.ClusterCatalog{}
	op.SetName(name)

	if cd.DryRun == DryRunAll {
		err := cd.config.Client.Delete(ctx, op, client.DryRunAll)
		return *op, err
	}
	if err := cd.config.Client.Delete(ctx, op); err != nil {
		return *op, err
	}

	return *op, waitForDeletion(ctx, cd.config.Client, op)
}
