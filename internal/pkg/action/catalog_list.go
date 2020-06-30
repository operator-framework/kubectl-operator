package action

import (
	"context"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ListCatalogs struct {
	config *Configuration
}

func NewListCatalogs(cfg *Configuration) *ListCatalogs {
	return &ListCatalogs{cfg}
}

func (l *ListCatalogs) Run(ctx context.Context) ([]v1alpha1.CatalogSource, error) {
	css := v1alpha1.CatalogSourceList{}
	if err := l.config.Client.List(ctx, &css, client.InNamespace(l.config.Namespace)); err != nil {
		return nil, err
	}
	return css.Items, nil
}
