package action

import (
	"context"
	"errors"
	"fmt"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type CatalogDelete struct {
	config      *action.Configuration
	CatalogName string
	DeleteAll   bool

	Logf func(string, ...interface{})
}

func NewCatalogDelete(cfg *action.Configuration) *CatalogDelete {
	return &CatalogDelete{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (cd *CatalogDelete) Run(ctx context.Context) ([]string, error) {
	// validate
	if cd.DeleteAll && cd.CatalogName != "" {
		return nil, errNameAndSelector
	}

	// delete single, specified catalog
	if !cd.DeleteAll {
		return nil, cd.deleteCatalog(ctx, cd.CatalogName)
	}

	// delete all existing catalogs
	var catatalogList olmv1.ClusterCatalogList
	if err := cd.config.Client.List(ctx, &catatalogList); err != nil {
		return nil, err
	}
	if len(catatalogList.Items) == 0 {
		return nil, errNoResourcesFound
	}

	errs := make([]error, 0, len(catatalogList.Items))
	names := make([]string, 0, len(catatalogList.Items))
	for _, catalog := range catatalogList.Items {
		names = append(names, catalog.Name)
		if err := cd.deleteCatalog(ctx, catalog.Name); err != nil {
			errs = append(errs, fmt.Errorf("failed deleting catalog %q: %w", catalog.Name, err))
		}
	}

	return names, errors.Join(errs...)
}

func (cd *CatalogDelete) deleteCatalog(ctx context.Context, name string) error {
	op := &olmv1.ClusterCatalog{}
	op.SetName(name)

	if err := cd.config.Client.Delete(ctx, op); err != nil {
		return err
	}

	return waitForDeletion(ctx, cd.config.Client, op)
}
