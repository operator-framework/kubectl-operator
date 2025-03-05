package action

import (
	"context"
	"slices"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1catalogd "github.com/operator-framework/catalogd/api/v1"
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
	catalog *olmv1catalogd.ClusterCatalog,
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

func deleteWithTimeout(cl deleter, obj client.Object, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := cl.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}
