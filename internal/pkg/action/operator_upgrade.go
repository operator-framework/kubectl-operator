package action

import (
	"context"
	"fmt"
	"time"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type OperatorUpgrade struct {
	config *Configuration

	Package string
	Channel string
}

func NewOperatorUpgrade(cfg *Configuration) *OperatorUpgrade {
	return &OperatorUpgrade{
		config: cfg,
	}
}

func (u *OperatorUpgrade) Run(ctx context.Context) (*v1alpha1.ClusterServiceVersion, error) {
	subs := v1alpha1.SubscriptionList{}
	if err := u.config.Client.List(ctx, &subs, client.InNamespace(u.config.Namespace)); err != nil {
		return nil, fmt.Errorf("list subscriptions: %v", err)
	}

	var sub *v1alpha1.Subscription
	for _, s := range subs.Items {
		s := s
		if u.Package == s.Spec.Package {
			sub = &s
			break
		}
	}
	if sub == nil {
		return nil, fmt.Errorf("operator package %q not found", u.Package)
	}

	ip, err := u.getInstallPlan(ctx, sub)
	if err != nil {
		return nil, err
	}

	if err := approveInstallPlan(ctx, u.config.Client, ip); err != nil {
		return nil, fmt.Errorf("approve install plan: %v", err)
	}

	csv, err := u.getCSV(ctx, ip)
	if err != nil {
		return nil, fmt.Errorf("get clusterserviceversion: %v", err)
	}
	return csv, nil
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

func (u *OperatorUpgrade) getCSV(ctx context.Context, ip *v1alpha1.InstallPlan) (*v1alpha1.ClusterServiceVersion, error) {
	ipKey := objectKeyForObject(ip)
	if err := wait.PollImmediateUntil(time.Millisecond*250, func() (bool, error) {
		if err := u.config.Client.Get(ctx, ipKey, ip); err != nil {
			return false, err
		}
		if ip.Status.Phase == v1alpha1.InstallPlanPhaseComplete {
			return true, nil
		}
		return false, nil
	}, ctx.Done()); err != nil {
		return nil, fmt.Errorf("waiting for operator installation to complete: %v", err)
	}

	csvKey := types.NamespacedName{
		Namespace: u.config.Namespace,
	}
	for _, s := range ip.Status.Plan {
		if s.Resource.Kind == csvKind {
			csvKey.Name = s.Resource.Name
		}
	}
	if csvKey.Name == "" {
		return nil, fmt.Errorf("could not find installed CSV in install plan")
	}
	csv := &v1alpha1.ClusterServiceVersion{}
	if err := u.config.Client.Get(ctx, csvKey, csv); err != nil {
		return nil, fmt.Errorf("get clusterserviceversion: %v", err)
	}
	return csv, nil
}
