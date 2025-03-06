package action

import (
	"context"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

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
