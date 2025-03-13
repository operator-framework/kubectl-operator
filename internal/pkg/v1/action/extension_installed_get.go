package action

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type ExtensionInstalledGet struct {
	config        *action.Configuration
	ExtensionName string

	Logf func(string, ...interface{})
}

func NewExtensionInstalledGet(cfg *action.Configuration) *ExtensionInstalledGet {
	return &ExtensionInstalledGet{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (i *ExtensionInstalledGet) Run(ctx context.Context) ([]olmv1.ClusterExtension, error) {
	// get
	if i.ExtensionName != "" {
		var result olmv1.ClusterExtension
		opKey := types.NamespacedName{Name: i.ExtensionName}
		err := i.config.Client.Get(ctx, opKey, &result)
		if err != nil {
			return nil, err
		}

		return []olmv1.ClusterExtension{result}, nil
	}

	// list
	var result olmv1.ClusterExtensionList
	err := i.config.Client.List(ctx, &result, &client.ListOptions{})

	return result.Items, err
}
