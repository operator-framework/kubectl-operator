package action

import (
	"context"
	"errors"
	"fmt"
	"strings"

	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/operator-framework/kubectl-operator/internal/pkg/operand"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type OperatorUninstall struct {
	config *action.Configuration

	Package                  string
	OperandStrategy          operand.DeletionStrategy
	DeleteOperatorGroups     bool
	DeleteOperatorGroupNames []string
	Logf                     func(string, ...interface{})
}

func NewOperatorUninstall(cfg *action.Configuration) *OperatorUninstall {
	return &OperatorUninstall{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

type ErrPackageNotFound struct {
	PackageName string
}

func (e ErrPackageNotFound) Error() string {
	return fmt.Sprintf("package %q not found", e.PackageName)
}

func (u *OperatorUninstall) Run(ctx context.Context) error {
	subs := v1alpha1.SubscriptionList{}
	if err := u.config.Client.List(ctx, &subs, client.InNamespace(u.config.Namespace)); err != nil {
		return fmt.Errorf("list subscriptions: %v", err)
	}

	var sub *v1alpha1.Subscription
	for _, s := range subs.Items {
		s := s
		if u.Package == s.Spec.Package {
			sub = &s
			break
		}
	}
	if sub == nil {
		return &ErrPackageNotFound{u.Package}
	}

	csv, csvName, err := u.getSubscriptionCSV(ctx, sub)
	if err != nil && !apierrors.IsNotFound(err) {
		if csvName == "" {
			return fmt.Errorf("get subscription csv: %v", err)
		}
		return fmt.Errorf("get subscription csv %q: %v", csvName, err)
	}

	// find operands related to the operator on cluster
	lister := action.NewOperatorListOperands(u.config)
	operands, err := lister.Run(ctx, u.Package)
	if err != nil {
		listError := action.OperandListError{}
		if errors.As(err, &listError) {
			// operands are not part of the operator uninstall
			u.Logf("list operands for operator %q: %v", u.Package, err)
		} else {
			return fmt.Errorf("list operands for operator %q: %v", u.Package, err)
		}
	}
	// validate the provided deletion strategy before proceeding to deletion
	if err := u.validStrategy(operands); err != nil {
		return fmt.Errorf("could not proceed with deletion of %q: %s", u.Package, err)
	}

	/*
		Deletion order:
		1. Subscription to prevent further installs or upgrades of the operator while cleaning up.

		If the CSV exists:
		  2. Operands so the operator has a chance to handle CRs that have finalizers.
		  Note: the correct strategy must be chosen in order to process an opertor delete with operand on-cluster.
		  3. ClusterServiceVersion. OLM puts an ownerref on every namespaced resource to the CSV,
		   and an owner label on every cluster scoped resource so they get gc'd on deletion.

		4. OperatorGroup in the namespace if no other subscriptions are in that namespace and OperatorGroup deletion is specified
	*/

	// Subscriptions can be deleted asynchronously.
	if err := u.deleteObjects(ctx, sub); err != nil {
		return err
	}

	// If we could not find a csv associated with the subscription, that likely
	// means there is no CSV associated with it yet. Delete non-CSV related items only like the operatorgroup.
	if csv == nil {
		u.Logf("csv for package %q not found", u.Package)
	} else {
		if err := u.deleteCSVRelatedResources(ctx, csv, operands); err != nil {
			return err
		}
	}

	if u.DeleteOperatorGroups {
		if err := u.deleteOperatorGroup(ctx); err != nil {
			return fmt.Errorf("delete operatorgroup: %v", err)
		}
	}

	return nil
}

func (u *OperatorUninstall) deleteObjects(ctx context.Context, objs ...client.Object) error {
	for _, obj := range objs {
		obj := obj
		lowerKind := strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind)
		if err := u.config.Client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete %s %q: %v", lowerKind, prettyPrint(obj), err)
		} else if err == nil {
			u.Logf("%s %q deleted", lowerKind, prettyPrint(obj))
		}
	}
	return waitForDeletion(ctx, u.config.Client, objs...)
}

// getSubscriptionCSV looks up the installed CSV name from the provided subscription and fetches it.
func (u *OperatorUninstall) getSubscriptionCSV(ctx context.Context, subscription *v1alpha1.Subscription) (*v1alpha1.ClusterServiceVersion, string, error) {
	name := csvNameFromSubscription(subscription)

	// If we could not find a name in the subscription, that likely
	// means there is no CSV associated with it yet. This should
	// not be treated as an error, so return a nil CSV with a nil error.
	if name == "" {
		return nil, "", nil
	}

	key := types.NamespacedName{
		Name:      name,
		Namespace: subscription.GetNamespace(),
	}

	csv := &v1alpha1.ClusterServiceVersion{}
	if err := u.config.Client.Get(ctx, key, csv); err != nil {
		return nil, name, err
	}
	csv.SetGroupVersionKind(v1alpha1.SchemeGroupVersion.WithKind(csvKind))
	return csv, name, nil
}

func (u *OperatorUninstall) deleteOperatorGroup(ctx context.Context) error {
	subs := v1alpha1.SubscriptionList{}
	if err := u.config.Client.List(ctx, &subs, client.InNamespace(u.config.Namespace)); err != nil {
		return fmt.Errorf("list subscriptions: %v", err)
	}

	// If there are no subscriptions left, delete the operator group(s).
	if len(subs.Items) == 0 {
		ogs := v1.OperatorGroupList{}
		if err := u.config.Client.List(ctx, &ogs, client.InNamespace(u.config.Namespace)); err != nil {
			return fmt.Errorf("list operatorgroups: %v", err)
		}
		for _, og := range ogs.Items {
			og := og
			if len(u.DeleteOperatorGroupNames) == 0 || contains(u.DeleteOperatorGroupNames, og.GetName()) {
				if err := u.deleteObjects(ctx, &og); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// validStrategy validates the deletion strategy against the operands on-cluster
// TODO define and use an OperandStrategyError that the cmd can use errors.As() on to provide external callers a more generic error
func (u *OperatorUninstall) validStrategy(operands *unstructured.UnstructuredList) error {
	if operands == nil {
		return nil
	}
	if len(operands.Items) > 0 && u.OperandStrategy.Kind == operand.Cancel {
		return fmt.Errorf("%d operands exist and operand strategy %q is in use: "+
			"delete operands manually or re-run uninstall with a different operand deletion strategy."+
			"\n\nSee kubectl operator uninstall --help for more information on operand deletion strategies.", len(operands.Items), operand.Cancel)
	}
	return nil
}

func (u *OperatorUninstall) deleteCSVRelatedResources(ctx context.Context, csv *v1alpha1.ClusterServiceVersion, operands *unstructured.UnstructuredList) error {
	switch u.OperandStrategy.Kind {
	case operand.Cancel:
		// At this point no operands were found with the cancel strategy in place
		// Safe to proceed to CSV deletion
		break
	case operand.Ignore:
		if operands == nil {
			break
		}
		for _, op := range operands.Items {
			u.Logf("%s %q orphaned", strings.ToLower(op.GetKind()), prettyPrint(&op))
		}
	case operand.Delete:
		if operands == nil {
			break
		}
		for _, op := range operands.Items {
			op := op
			if err := u.deleteObjects(ctx, &op); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unknown operand deletion strategy %q", u.OperandStrategy)
	}

	// OLM puts an ownerref on every namespaced resource to the CSV,
	// and an owner label on every cluster scoped resource. When CSV is deleted
	// kube and olm gc will remove all the referenced resources.
	if err := u.deleteObjects(ctx, csv); err != nil {
		return err
	}

	return nil
}

func csvNameFromSubscription(subscription *v1alpha1.Subscription) string {
	if subscription.Status.InstalledCSV != "" {
		return subscription.Status.InstalledCSV
	}
	return subscription.Status.CurrentCSV
}

func contains(haystack []string, needle string) bool {
	for _, n := range haystack {
		if n == needle {
			return true
		}
	}
	return false
}

func prettyPrint(op client.Object) string {
	namespaced := op.GetNamespace() != ""
	if namespaced {
		return fmt.Sprint(op.GetName() + "/" + op.GetNamespace())
	}
	return op.GetName()
}
