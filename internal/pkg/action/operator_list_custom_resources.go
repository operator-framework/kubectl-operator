package action

import (
	"context"

	lib "github.com/operator-framework/kubectl-operator/pkg/action"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

// OperatorListCustomResources knows how to find and list custom resources given a package name and namespace.
type OperatorListCustomResources struct {
	config      *Configuration
	PackageName string
}

func NewOperatorListCustomResources(cfg *Configuration) *OperatorListCustomResources {
	return &OperatorListCustomResources{
		config: cfg,
	}
}

func (o *OperatorListCustomResources) Run(ctx context.Context) (*unstructured.UnstructuredList, error) {
	opKey := types.NamespacedName{
		Name:      o.PackageName,
		Namespace: o.config.Namespace,
	}

	result, err := lib.ListAll(ctx, o.config.Client, opKey)
	if err != nil {
		return nil, err
	}
	return result, nil
}
