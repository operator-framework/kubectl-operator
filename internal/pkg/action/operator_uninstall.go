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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
		return fmt.Errorf("operator package %q not found", u.Package)
	}

	var subObj, csvObj controllerutil.Object
	var crds []controllerutil.Object
	if sub != nil {
		subObj = sub
		// CSV name may either be the installed or current name in a subscription's status,
		// depending on installation state.
		csvKey := types.NamespacedName{
			Name:      sub.Status.InstalledCSV,
			Namespace: u.config.Namespace,
		}
		if csvKey.Name == "" {
			csvKey.Name = sub.Status.CurrentCSV
		}

		// This value can be empty which will cause errors.
		if csvKey.Name != "" {
			csv := &v1alpha1.ClusterServiceVersion{}
			if err := u.config.Client.Get(ctx, csvKey, csv); err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("error getting installed CSV %q: %v", csvKey.Name, err)
			} else if err == nil {
				crds = getCRDs(csv)
			}
			csvObj = csv
		}
	}

	// Deletion order:
	//
	// 1. Subscription to prevent further installs or upgrades of the operator while cleaning up.
	// 2. CustomResourceDefinitions so the operator has a chance to handle CRs that have finalizers.
	// 3. ClusterServiceVersion. OLM puts an ownerref on every namespaced resource to the CSV,
	//    and an owner label on every cluster scoped resource so they get gc'd on deletion.

	// Subscriptions can be deleted asynchronously.
	if err := u.deleteObjects(ctx, subObj); err != nil {
		return err
	}

	if u.DeleteCRDs {
		// Ensure CustomResourceDefinitions are deleted next, so that the operator
		// has a chance to handle CRs that have finalizers.
		if err := u.deleteObjects(ctx, crds...); err != nil {
			return err
		}
	}

	// OLM puts an ownerref on every namespaced resource to the CSV,
	// and an owner label on every cluster scoped resource. When CSV is deleted
	// kube and olm gc will remove all the referenced resources.
	if err := u.deleteObjects(ctx, csvObj); err != nil {
		return err
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

	// If no objects were cleaned up, it means the package was not found
	if subObj == nil && csvObj == nil && len(crds) == 0 {
		return &ErrPackageNotFound{u.Package}
	}

	return nil
}

func (u *OperatorUninstall) deleteObjects(ctx context.Context, objs ...controllerutil.Object) error {
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

// getCRDs returns the list of CRDs required by a CSV.
func getCRDs(csv *v1alpha1.ClusterServiceVersion) (crds []controllerutil.Object) {
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
