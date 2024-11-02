package v1

import (
	"context"
	"fmt"
	"strings"
	"time"

	"errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

func deleteAndWait(ctx context.Context, cl client.Client, objs ...client.Object) error {
	var (
		wg   sync.WaitGroup
		errs = make([]error, len(objs))
	)
	for i := range objs {
		wg.Add(1)
		go func(objectIndex int) {
			defer wg.Done()
			obj := objs[objectIndex]
			gvk := obj.GetObjectKind().GroupVersionKind()
			if gvk.Empty() {
				gvks, unversioned, err := cl.Scheme().ObjectKinds(obj)
				if err == nil && !unversioned && len(gvks) > 0 {
					gvk = gvks[0]
				}
			}
			lowerKind := strings.ToLower(gvk.Kind)
			key := client.ObjectKeyFromObject(obj)

			err := cl.Delete(ctx, obj)
			if err != nil && !apierrors.IsNotFound(err) {
				errs[i] = fmt.Errorf("delete %s %q: %v", lowerKind, key.Name, err)
				return
			}

			if err := wait.PollUntilContextCancel(ctx, 250*time.Millisecond, true, func(conditionCtx context.Context) (bool, error) {
				if err := cl.Get(conditionCtx, key, obj); apierrors.IsNotFound(err) {
					return true, nil
				} else if err != nil {
					return false, err
				}
				return false, nil
			}); err != nil {
				errs[i] = fmt.Errorf("wait for %s %q deleted: %v", lowerKind, key.Name, err)
				return
			}
		}(i)
	}
	wg.Wait()
	return errors.Join(errs...)
}

func patchObject(ctx context.Context, cl client.Client, obj interface{}) error {
	var (
		u   unstructured.Unstructured
		err error
	)
	u.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return err
	}
	return cl.Patch(ctx, &u, client.Apply)
}
