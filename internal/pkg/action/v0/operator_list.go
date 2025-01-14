package v0

import (
	"context"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type OperatorList struct {
	config *action.Configuration
}

func NewOperatorList(cfg *action.Configuration) *OperatorList {
	return &OperatorList{cfg}
}

func (l *OperatorList) Run(ctx context.Context) ([]v1alpha1.Subscription, error) {
	subs := v1alpha1.SubscriptionList{}
	if err := l.config.Client.List(ctx, &subs); err != nil {
		return nil, err
	}
	return subs.Items, nil
}
