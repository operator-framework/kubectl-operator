package action

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/spf13/pflag"
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

func (u *OperatorUninstall) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&u.DeleteAll, "delete-all", "X", false, "enable all delete flags")
	fs.BoolVar(&u.DeleteCRDs, "delete-crds", false, "delete all owned CRDs and all CRs")
	fs.BoolVar(&u.DeleteOperatorGroups, "delete-operator-groups", false, "delete operator group if no other operators remain")
	fs.StringSliceVar(&u.DeleteOperatorGroupNames, "delete-operator-group-names", nil, "delete operator group if no other operators remain")
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

	csv, err := u.getInstalledCSV(ctx, sub)
	if err != nil {
		return fmt.Errorf("get installed CSV %q: %v", sub.Status.InstalledCSV, err)
	}

	crds := getCRDs(csv)

	// Delete the subscription first, so that no further installs or upgrades
	// of the operator occur while we're cleaning up.
	if err := u.deleteObjects(ctx, sub); err != nil {
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
	if err := u.deleteObjects(ctx, csv); err != nil {
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

func (u *OperatorUninstall) getInstalledCSV(ctx context.Context, subscription *v1alpha1.Subscription) (*v1alpha1.ClusterServiceVersion, error) {
	key := types.NamespacedName{
		Name:      subscription.Status.InstalledCSV,
		Namespace: subscription.GetNamespace(),
	}

	installedCSV := &v1alpha1.ClusterServiceVersion{}
	if err := u.config.Client.Get(ctx, key, installedCSV); err != nil {
		return nil, err
	}

	installedCSV.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    csvKind,
		Version: installedCSV.GroupVersionKind().Version,
		Group:   installedCSV.GroupVersionKind().Group,
	})
	return installedCSV, nil
}

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
			obj.SetNamespace(csv.GetNamespace())

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
