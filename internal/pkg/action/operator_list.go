package action

import (
	"context"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
)

type OperatorList struct {
	config *Configuration
}

func NewOperatorList(cfg *Configuration) *OperatorList {
	return &OperatorList{cfg}
}

func (l *OperatorList) Run(ctx context.Context) ([]v1alpha1.Subscription, error) {
	subs := v1alpha1.SubscriptionList{}
	if err := l.config.Client.List(ctx, &subs); err != nil {
		return nil, err
	}
	return subs.Items, nil
}
