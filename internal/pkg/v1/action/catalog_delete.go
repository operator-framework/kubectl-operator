package action

import (
	"context"

	olmv1catalogd "github.com/operator-framework/catalogd/api/v1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type CatalogDelete struct {
	config      *action.Configuration
	CatalogName string

	Logf func(string, ...interface{})
}

func NewCatalogDelete(cfg *action.Configuration) *CatalogDelete {
	return &CatalogDelete{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (i *CatalogDelete) Run(ctx context.Context) error {
	op := &olmv1catalogd.ClusterCatalog{}
	op.SetName(i.CatalogName)

	if err := i.config.Client.Delete(ctx, op); err != nil {
		return err
	}

	return waitForDeletion(ctx, i.config.Client, op)
}
