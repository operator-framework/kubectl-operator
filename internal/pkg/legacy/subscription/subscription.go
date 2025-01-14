package subscription

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
)

type Option func(*v1alpha1.Subscription)

func InstallPlanApproval(v v1alpha1.Approval) Option {
	return func(s *v1alpha1.Subscription) {
		s.Spec.InstallPlanApproval = v
	}
}

func StartingCSV(v string) Option {
	return func(s *v1alpha1.Subscription) {
		s.Spec.StartingCSV = v
	}
}

func Build(key types.NamespacedName, channel string, source types.NamespacedName, opts ...Option) *v1alpha1.Subscription {
	s := &v1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		Spec: &v1alpha1.SubscriptionSpec{
			Package:                key.Name,
			Channel:                channel,
			CatalogSource:          source.Name,
			CatalogSourceNamespace: source.Namespace,
		},
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

const defaultApproval = v1alpha1.ApprovalManual

type ApprovalValue struct {
	v1alpha1.Approval
}

func (a *ApprovalValue) Set(str string) error {
	switch v := v1alpha1.Approval(str); v {
	case v1alpha1.ApprovalAutomatic, v1alpha1.ApprovalManual:
		a.Approval = v
		return nil
	}
	return fmt.Errorf("invalid approval value %q", str)
}

func (a *ApprovalValue) String() string {
	if a.Approval == "" {
		a.Approval = defaultApproval
	}
	return string(a.Approval)
}

func (a ApprovalValue) Type() string {
	return "ApprovalValue"
}
