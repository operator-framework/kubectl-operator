package action

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"
)

const pollInterval = 250 * time.Millisecond

func objectKeyForObject(obj client.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}

func waitUntilCatalogStatusCondition(
	ctx context.Context,
	cl getter,
	catalog *olmv1.ClusterCatalog,
	conditionType string,
	conditionStatus metav1.ConditionStatus,
) error {
	opKey := objectKeyForObject(catalog)
	return wait.PollUntilContextCancel(ctx, pollInterval, true, func(conditionCtx context.Context) (bool, error) {
		if err := cl.Get(conditionCtx, opKey, catalog); err != nil {
			return false, err
		}

		if slices.ContainsFunc(catalog.Status.Conditions, func(cond metav1.Condition) bool {
			return cond.Type == conditionType && cond.Status == conditionStatus
		}) {
			return true, nil
		}
		return false, nil
	})
}

func waitUntilOperatorStatusCondition(
	ctx context.Context,
	cl getter,
	operator *olmv1.ClusterExtension,
	conditionType string,
	conditionStatus metav1.ConditionStatus,
) error {
	opKey := objectKeyForObject(operator)
	return wait.PollUntilContextCancel(ctx, pollInterval, true, func(conditionCtx context.Context) (bool, error) {
		if err := cl.Get(conditionCtx, opKey, operator); err != nil {
			return false, err
		}

		if slices.ContainsFunc(operator.Status.Conditions, func(cond metav1.Condition) bool {
			return cond.Type == conditionType && cond.Status == conditionStatus
		}) {
			return true, nil
		}
		return false, nil
	})
}

func deleteWithTimeout(cl deleter, obj client.Object, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := cl.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

func waitForDeletion(ctx context.Context, cl client.Client, objs ...client.Object) error {
	for _, obj := range objs {
		obj := obj
		lowerKind := strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind)
		key := objectKeyForObject(obj)
		if err := wait.PollUntilContextCancel(ctx, pollInterval, true, func(conditionCtx context.Context) (bool, error) {
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
