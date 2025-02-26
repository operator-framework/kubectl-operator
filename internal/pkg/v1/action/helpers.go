package action

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const pollInterval = 250 * time.Millisecond

func objectKeyForObject(obj client.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}

func waitForDeletion(ctx context.Context, cl client.Client, obj client.Object) error {
	key := objectKeyForObject(obj)
	if err := wait.PollUntilContextCancel(ctx, pollInterval, true, func(conditionCtx context.Context) (bool, error) {
		if err := cl.Get(conditionCtx, key, obj); apierrors.IsNotFound(err) {
			return true, nil
		} else if err != nil {
			return false, err
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("waiting for deletion: %w", err)
	}

	return nil
}
