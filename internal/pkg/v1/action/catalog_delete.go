package action

import (
	"context"
	"errors"
	"fmt"

	olmv1catalogd "github.com/operator-framework/catalogd/api/v1"

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
	var catsList olmv1catalogd.ClusterCatalogList
	if err := cd.config.Client.List(ctx, &catsList); err != nil {
		return nil, err
	}
	if len(catsList.Items) == 0 {
		return nil, errNoResourcesFound
	}

	errs := make([]error, 0, len(catsList.Items))
	names := make([]string, 0, len(catsList.Items))
	for _, cat := range catsList.Items {
		names = append(names, cat.Name)
		if err := cd.deleteCatalog(ctx, cat.Name); err != nil {
			errs = append(errs, fmt.Errorf("failed deleting catalog %q: %w", cat.Name, err))
		}
	}

	return names, errors.Join(errs...)
}

func (cd *CatalogDelete) deleteCatalog(ctx context.Context, name string) error {
	op := &olmv1catalogd.ClusterCatalog{}
	op.SetName(name)

	if err := cd.config.Client.Delete(ctx, op); err != nil {
		return err
	}

	return waitForDeletion(ctx, cd.config.Client, op)
}
