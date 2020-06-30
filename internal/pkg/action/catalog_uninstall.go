package action

import (
	"context"
	"fmt"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
)

type UninstallCatalog struct {
	config *Configuration

	CatalogName string
}

func NewUninstallCatalog(cfg *Configuration) *UninstallCatalog {
	return &UninstallCatalog{
		config: cfg,
	}
}

func (u *UninstallCatalog) Run(ctx context.Context) error {
	cs := v1alpha1.CatalogSource{}
	cs.SetNamespace(u.config.Namespace)
	cs.SetName(u.CatalogName)
	if err := u.config.Client.Delete(ctx, &cs); err != nil {
		return fmt.Errorf("delete catalogsource %q: %v", cs.Name, err)
	}
	return nil
}
