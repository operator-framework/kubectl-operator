package action

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	olmv1 "github.com/operator-framework/operator-controller/api/v1alpha1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type OperatorInstall struct {
	config *action.Configuration

	Package string

	Logf func(string, ...interface{})
}

func NewOperatorInstall(cfg *action.Configuration) *OperatorInstall {
	return &OperatorInstall{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (i *OperatorInstall) Run(ctx context.Context) (*olmv1.ClusterExtension, error) {
	// TODO(developer): Lookup package information when the OLMv1 equivalent of the
	//     packagemanifests API is available. That way, we can check to see if the
	//     package is actually available to the cluster before creating the Operator
	//     object.

	opKey := types.NamespacedName{Name: i.Package}
	op := &olmv1.ClusterExtension{
		ObjectMeta: metav1.ObjectMeta{Name: opKey.Name},
		Spec: olmv1.ClusterExtensionSpec{
			Source: olmv1.SourceConfig{
				SourceType: "Catalog",
				Catalog: &olmv1.CatalogSource{
					PackageName: i.Package,
				},
			},
		},
	}
	if err := i.config.Client.Create(ctx, op); err != nil {
		return nil, err
	}

	// TODO(developer): Improve the logic in this poll wait once the Operator reconciler
	//     and conditions types and reasons are improved. For now, this will stop waiting as
	//     soon as a Ready condition is found, but we should probably wait until the Operator
	//     stops progressing.
	// All Types will exist, so Ready may have a false Status. So, wait until
	// Type=Ready,Status=True happens

	if err := wait.PollUntilContextCancel(ctx, pollTimeout, true, func(conditionCtx context.Context) (bool, error) {
		if err := i.config.Client.Get(conditionCtx, opKey, op); err != nil {
			return false, err
		}
		installedCondition := meta.FindStatusCondition(op.Status.Conditions, olmv1.TypeInstalled)
		if installedCondition != nil && installedCondition.Status == metav1.ConditionTrue {
			return true, nil
		}
		return false, nil
	}); err != nil {
		return nil, fmt.Errorf("waiting for operator to become ready: %v", err)
	}

	return op, nil
}
