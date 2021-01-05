package action

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type OperatorUninstall struct {
	config *Configuration

	Package                  string
	DeleteAll                bool
	DeleteCRDs               bool
	DeleteOperatorGroups     bool
	DeleteOperatorGroupNames []string

	Logf func(string, ...interface{})
}

func NewOperatorUninstall(cfg *Configuration) *OperatorUninstall {
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
	if u.DeleteAll {
		u.DeleteCRDs = true
		u.DeleteOperatorGroups = true
	}

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
			return fmt.Errorf("get subscription CSV: %v", err)
		}
		return fmt.Errorf("get subscription CSV %q: %v", csvName, err)
	}

	// Deletion order:
	//
	// 1. Subscription to prevent further installs or upgrades of the operator while cleaning up.
	// 2. CustomResourceDefinitions so the operator has a chance to handle CRs that have finalizers.
	// 3. ClusterServiceVersion. OLM puts an ownerref on every namespaced resource to the CSV,
	//    and an owner label on every cluster scoped resource so they get gc'd on deletion.

	// Subscriptions can be deleted asynchronously.
	if err := u.deleteObjects(ctx, sub); err != nil {
		return err
	}

	if csv != nil {
		// Ensure CustomResourceDefinitions are deleted next, so that the operator
		// has a chance to handle CRs that have finalizers.
		if u.DeleteCRDs {
			crds := getCRDs(csv)
			if err := u.deleteObjects(ctx, crds...); err != nil {
				return err
			}
		}

		// OLM puts an ownerref on every namespaced resource to the CSV,
		// and an owner label on every cluster scoped resource. When CSV is deleted
		// kube and olm gc will remove all the referenced resources.
		if err := u.deleteObjects(ctx, csv); err != nil {
			return err
		}
	}

	if u.DeleteOperatorGroups {
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
	}
	return nil
}

func (u *OperatorUninstall) deleteObjects(ctx context.Context, objs ...client.Object) error {
	for _, obj := range objs {
		obj := obj
		lowerKind := strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind)
		if err := u.config.Client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete %s %q: %v", lowerKind, obj.GetName(), err)
		} else if err == nil {
			u.Logf("%s %q deleted", lowerKind, obj.GetName())
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

func csvNameFromSubscription(subscription *v1alpha1.Subscription) string {
	if subscription.Status.InstalledCSV != "" {
		return subscription.Status.InstalledCSV
	}
	return subscription.Status.CurrentCSV
}

// getCRDs returns the list of CRDs required by a CSV.
func getCRDs(csv *v1alpha1.ClusterServiceVersion) (crds []client.Object) {
	for _, resource := range csv.Status.RequirementStatus {
		if resource.Kind == crdKind {
			obj := &unstructured.Unstructured{}
			obj.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   resource.Group,
				Version: resource.Version,
				Kind:    resource.Kind,
			})
			obj.SetName(resource.Name)
			crds = append(crds, obj)
		}
	}
	return
}

func contains(haystack []string, needle string) bool {
	for _, n := range haystack {
		if n == needle {
			return true
		}
	}
	return false
}
