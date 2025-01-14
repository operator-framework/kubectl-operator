package v0

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type CatalogList struct {
	config *action.Configuration
}

func NewCatalogList(cfg *action.Configuration) *CatalogList {
	return &CatalogList{cfg}
}

func (l *CatalogList) Run(ctx context.Context) ([]v1alpha1.CatalogSource, error) {
	css := v1alpha1.CatalogSourceList{}
	if err := l.config.Client.List(ctx, &css, client.InNamespace(l.config.Namespace)); err != nil {
		return nil, err
	}
	return css.Items, nil
}
