package v0

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type OperatorUpgrade struct {
	config *action.Configuration

	Package string
	Channel string
}

func NewOperatorUpgrade(cfg *action.Configuration) *OperatorUpgrade {
	return &OperatorUpgrade{
		config: cfg,
	}
}

func (u *OperatorUpgrade) Run(ctx context.Context) (*v1alpha1.ClusterServiceVersion, error) {
	sub, err := u.findSubscriptionForPackage(ctx)
	if err != nil {
		return nil, err
	}

	ip, err := u.getInstallPlan(ctx, sub)
	if err != nil {
		return nil, err
	}

	if err := approveInstallPlan(ctx, u.config.Client, ip); err != nil {
		return nil, fmt.Errorf("approve install plan: %v", err)
	}

	csv, err := getCSV(ctx, u.config.Client, ip)
	if err != nil {
		return nil, fmt.Errorf("get clusterserviceversion: %v", err)
	}
	return csv, nil
}

func (u *OperatorUpgrade) findSubscriptionForPackage(ctx context.Context) (*v1alpha1.Subscription, error) {
	subs := v1alpha1.SubscriptionList{}
	if err := u.config.Client.List(ctx, &subs, client.InNamespace(u.config.Namespace)); err != nil {
		return nil, fmt.Errorf("list subscriptions: %v", err)
	}

	for _, s := range subs.Items {
		s := s
		if u.Package == s.Spec.Package {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("subscription for package %q not found", u.Package)
}

func (u *OperatorUpgrade) getInstallPlan(ctx context.Context, sub *v1alpha1.Subscription) (*v1alpha1.InstallPlan, error) {
	if sub.Status.InstallPlanRef == nil {
		return nil, fmt.Errorf("subscription does not reference an install plan")
	}
	if sub.Status.InstalledCSV == sub.Status.CurrentCSV {
		return nil, fmt.Errorf("operator is already at latest version")
	}

	ip := v1alpha1.InstallPlan{}
	ipKey := types.NamespacedName{
		Namespace: sub.Status.InstallPlanRef.Namespace,
		Name:      sub.Status.InstallPlanRef.Name,
	}
	if err := u.config.Client.Get(ctx, ipKey, &ip); err != nil {
		return nil, fmt.Errorf("get install plan: %v", err)
	}
	return &ip, nil
}
