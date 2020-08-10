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
	"sigs.k8s.io/yaml"
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

	// Since the install plan is owned by the subscription, we need to
	// read all of the resource references from the install plan before
	// deleting the subscription.
	var crds, csvs, others []controllerutil.Object
	if sub.Status.InstallPlanRef != nil {
		ipKey := types.NamespacedName{
			Namespace: sub.Status.InstallPlanRef.Namespace,
			Name:      sub.Status.InstallPlanRef.Name,
		}
		var err error
		crds, csvs, others, err = u.getInstallPlanResources(ctx, ipKey)
		if err != nil {
			return fmt.Errorf("get install plan resources: %v", err)
		}
	}

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

	// Delete CSVs and all other objects created by the install plan.
	objects := append(csvs, others...)
	if err := u.deleteObjects(ctx, objects...); err != nil {
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

func (u *OperatorUninstall) getInstallPlanResources(ctx context.Context, installPlanKey types.NamespacedName) (crds, csvs, others []controllerutil.Object, err error) {
	installPlan := &v1alpha1.InstallPlan{}
	if err := u.config.Client.Get(ctx, installPlanKey, installPlan); err != nil {
		return nil, nil, nil, fmt.Errorf("get install plan: %v", err)
	}

	for _, step := range installPlan.Status.Plan {
		lowerKind := strings.ToLower(step.Resource.Kind)
		obj := &unstructured.Unstructured{Object: map[string]interface{}{}}
		if err := yaml.Unmarshal([]byte(step.Resource.Manifest), &obj.Object); err != nil {
			return nil, nil, nil, fmt.Errorf("parse %s manifest %q: %v", lowerKind, step.Resource.Name, err)
		}
		obj.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   step.Resource.Group,
			Version: step.Resource.Version,
			Kind:    step.Resource.Kind,
		})

		// TODO(joelanford): This seems necessary for service accounts tied to
		//    cluster roles and cluster role bindings because the SA namespace
		//    is not set in the manifest in this case.
		//    See: https://github.com/operator-framework/operator-lifecycle-manager/blob/c9405d035bc50d9aa290220cb8d75b0402e72707/pkg/controller/registry/resolver/rbac.go#L133
		if step.Resource.Kind == "ServiceAccount" && obj.GetNamespace() == "" {
			obj.SetNamespace(installPlanKey.Namespace)
		}
		switch step.Resource.Kind {
		case crdKind:
			crds = append(crds, obj)
		case csvKind:
			csvs = append(csvs, obj)
		default:
			// Skip non-CRD/non-CSV resources in the install plan that were not created by the install plan.
			// This means we avoid deleting things like the default service account.
			if step.Status != v1alpha1.StepStatusCreated {
				continue
			}
			others = append(others, obj)
		}
	}
	return crds, csvs, others, nil
}

func contains(haystack []string, needle string) bool {
	for _, n := range haystack {
		if n == needle {
			return true
		}
	}
	return false
}
