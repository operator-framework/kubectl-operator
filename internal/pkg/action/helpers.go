package action

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func objectKeyForObject(obj controllerutil.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}

func waitForDeletion(ctx context.Context, cl client.Client, objs ...controllerutil.Object) error {
	for _, obj := range objs {
		obj := obj
		lowerKind := strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind)
		key := objectKeyForObject(obj)
		if err := wait.PollImmediateUntil(250*time.Millisecond, func() (bool, error) {
			if err := cl.Get(ctx, key, obj); apierrors.IsNotFound(err) {
				return true, nil
			} else if err != nil {
				return false, err
			}
			return false, nil
		}, ctx.Done()); err != nil {
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
