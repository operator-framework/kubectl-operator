package action

import (
	"context"
	"fmt"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
)

func objectKeyForObject(obj client.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}

func waitForDeletion(ctx context.Context, cl client.Client, objs ...client.Object) error {
	for _, obj := range objs {
		obj := obj
		lowerKind := strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind)
		key := objectKeyForObject(obj)
		if err := wait.PollUntilContextCancel(ctx, 250*time.Millisecond, true, func(conditionCtx context.Context) (bool, error) {
			if err := cl.Get(conditionCtx, key, obj); apierrors.IsNotFound(err) {
				return true, nil
			} else if err != nil {
				return false, err
			}
			return false, nil
		}); err != nil {
			return fmt.Errorf("wait for %s %q deleted: %v", lowerKind, key.Name, err)
		}
	}
	return nil
}

func approveInstallPlan(ctx context.Context, cl client.Client, ip *v1alpha1.InstallPlan) error {
	ipKey := objectKeyForObject(ip)
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := cl.Get(ctx, ipKey, ip); err != nil {
			return err
		}
		ip.Spec.Approved = true
		return cl.Update(ctx, ip)
	})
}

func getCSV(ctx context.Context, cl client.Client, ip *v1alpha1.InstallPlan) (*v1alpha1.ClusterServiceVersion, error) {
	ipKey := objectKeyForObject(ip)
	if err := wait.PollUntilContextCancel(ctx, time.Millisecond*250, true, func(conditionCtx context.Context) (bool, error) {
		if err := cl.Get(conditionCtx, ipKey, ip); err != nil {
			return false, err
		}
		if ip.Status.Phase == v1alpha1.InstallPlanPhaseComplete {
			return true, nil
		}
		return false, nil
	}); err != nil {
		return nil, fmt.Errorf("waiting for operator installation to complete: %v", err)
	}

	csvKey := types.NamespacedName{
		Namespace: ipKey.Namespace,
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
	if err := cl.Get(ctx, csvKey, csv); err != nil {
		return nil, fmt.Errorf("get clusterserviceversion: %v", err)
	}
	return csv, nil
}
