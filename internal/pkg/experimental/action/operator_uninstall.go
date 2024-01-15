package action

import (
	"context"
	"fmt"
	"strings"

	olmv1 "github.com/operator-framework/operator-controller/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type OperatorUninstall struct {
	config *action.Configuration

	Package string

	Logf func(string, ...interface{})
}

func NewOperatorUninstall(cfg *action.Configuration) *OperatorUninstall {
	return &OperatorUninstall{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (u *OperatorUninstall) Run(ctx context.Context) error {
	opKey := types.NamespacedName{Name: u.Package}
	op := &olmv1.ClusterExtension{}
	op.SetName(opKey.Name)
	op.SetGroupVersionKind(olmv1.GroupVersion.WithKind("Operator"))

	lowerKind := strings.ToLower(op.GetObjectKind().GroupVersionKind().Kind)
	if err := u.config.Client.Delete(ctx, op); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete %s %q: %v", lowerKind, op.GetName(), err)
	}
	return waitForDeletion(ctx, u.config.Client, op)
}

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
		if err := wait.PollUntilContextCancel(ctx, pollTimeout, true, func(conditionCtx context.Context) (bool, error) {
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
