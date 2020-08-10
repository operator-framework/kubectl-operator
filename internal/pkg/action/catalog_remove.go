package action

import (
	"context"
	"fmt"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
)

type CatalogRemove struct {
	config *Configuration

	CatalogName string
}

func NewCatalogRemove(cfg *Configuration) *CatalogRemove {
	return &CatalogRemove{
		config: cfg,
	}
}

func (r *CatalogRemove) Run(ctx context.Context) error {
	cs := v1alpha1.CatalogSource{}
	cs.SetNamespace(r.config.Namespace)
	cs.SetName(r.CatalogName)
	if err := r.config.Client.Delete(ctx, &cs); err != nil {
		return fmt.Errorf("delete catalogsource %q: %v", cs.Name, err)
	}
	return waitForDeletion(ctx, r.config.Client, &cs)
}
