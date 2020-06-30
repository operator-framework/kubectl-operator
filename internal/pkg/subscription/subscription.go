package subscription

import (
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type Option func(*v1alpha1.Subscription)

func InstallPlanApproval(v v1alpha1.Approval) Option {
	return func(cs *v1alpha1.Subscription) {
		cs.Spec.InstallPlanApproval = v
	}
}

func Build(key types.NamespacedName, channel string, source types.NamespacedName, opts ...Option) *v1alpha1.Subscription {
	s := &v1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
			Labels: map[string]string{
				"createdBy": "kubectl-operator",
			},
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
