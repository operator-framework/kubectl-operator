package action

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type OperatorInstalledGet struct {
	config       *action.Configuration
	OperatorName string

	Logf func(string, ...interface{})
}

func NewOperatorInstalledGet(cfg *action.Configuration) *OperatorInstalledGet {
	return &OperatorInstalledGet{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (i *OperatorInstalledGet) Run(ctx context.Context) ([]olmv1.ClusterExtension, error) {
	// get
	if i.OperatorName != "" {
		var result olmv1.ClusterExtension
		opKey := types.NamespacedName{Name: i.OperatorName}
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
