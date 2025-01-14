package v1

import (
	"context"

	"github.com/operator-framework/kubectl-operator/pkg/action"
	ocv1 "github.com/operator-framework/operator-controller/api/v1"
)

type OperatorList struct {
	config *action.Configuration
}

func NewOperatorList(cfg *action.Configuration) *OperatorList {
	return &OperatorList{cfg}
}

func (l *OperatorList) Run(ctx context.Context) ([]ocv1.ClusterExtension, error) {
	clusterExtensions := ocv1.ClusterExtensionList{}
	if err := l.config.Client.List(ctx, &clusterExtensions); err != nil {
		return nil, err
	}
	return clusterExtensions.Items, nil
}
