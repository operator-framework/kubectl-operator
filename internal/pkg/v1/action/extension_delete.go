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

type ExtensionDeletion struct {
	config        *action.Configuration
	ExtensionName string
	DeleteAll     bool
	Logf          func(string, ...interface{})
}

func NewExtensionDelete(cfg *action.Configuration) *ExtensionDeletion {
	return &ExtensionDeletion{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (u *ExtensionDeletion) Run(ctx context.Context) error {
	opKey := types.NamespacedName{Name: u.ExtensionName}
	op := &olmv1.ClusterExtension{}
	op.SetName(opKey.Name)
	op.SetGroupVersionKind(olmv1.GroupVersion.WithKind("ClusterExtension"))
	lowerKind := strings.ToLower(op.GetObjectKind().GroupVersionKind().Kind)
	//Lala:Fixme: return error if extension does not exist
	if err := u.config.Client.Delete(ctx, op); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete %s %q: %v", lowerKind, op.GetName(), err)
	}
	return waitForDeletion(ctx, u.config.Client, op)
}
