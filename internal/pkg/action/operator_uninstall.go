package action

import (
	"context"
	"fmt"
	"strings"
	"time"

	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/spf13/pflag"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/kubectl-operator/internal/pkg/log"
)

type OperatorUninstall struct {
	config *Configuration

	Package             string
	DeleteOperatorGroup bool
	DeleteCRDs          bool
	DeleteAll           bool
}

func NewOperatorUninstall(cfg *Configuration) *OperatorUninstall {
	return &OperatorUninstall{
		config: cfg,
	}
}

func (u *OperatorUninstall) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&u.DeleteOperatorGroup, "delete-operator-group", false, "delete operator group if no other operators remain")
	fs.BoolVar(&u.DeleteCRDs, "delete-crds", false, "delete all owned CRDs and all CRs")
	fs.BoolVarP(&u.DeleteAll, "delete-add", "X", false, "enable all delete flags")
}

func (u *OperatorUninstall) Run(ctx context.Context) error {
	if u.DeleteAll {
		u.DeleteCRDs = true
		u.DeleteOperatorGroup = true
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

	if err := u.config.Client.Delete(ctx, sub); err != nil {
		return fmt.Errorf("delete subscription %q: %v", sub.Name, err)
	}
	log.Printf("subscription %q deleted", sub.Name)

	if u.DeleteCRDs {
		if err := u.deleteCRDs(ctx, crds); err != nil {
			return err
		}
	}

	if err := u.deleteObjects(ctx, false, csvs); err != nil {
		return err
	}

	if err := u.deleteObjects(ctx, false, others); err != nil {
		return err
	}

	if u.DeleteOperatorGroup {
		subs := v1alpha1.SubscriptionList{}
		if err := u.config.Client.List(ctx, &subs, client.InNamespace(u.config.Namespace)); err != nil {
			return fmt.Errorf("list clusterserviceversions: %v", err)
		}
		if len(subs.Items) == 0 {
			ogs := v1.OperatorGroupList{}
			if err := u.config.Client.List(ctx, &ogs, client.InNamespace(u.config.Namespace)); err != nil {
				return fmt.Errorf("list operatorgroups: %v", err)
			}
			for _, og := range ogs.Items {
				og := og
				if err := u.config.Client.Delete(ctx, &og); err != nil {
					return fmt.Errorf("delete operatorgroup %q: %v", og.Name, err)
				}
				log.Printf("operatorgroup %q deleted", og.Name)
			}
		}
	}

	return nil
}

func (u *OperatorUninstall) deleteCRDs(ctx context.Context, crds []controllerutil.Object) error {
	if err := u.deleteObjects(ctx, true, crds); err != nil {
		return err
	}
	return nil
}

func (u *OperatorUninstall) deleteObjects(ctx context.Context, waitForDelete bool, objs []controllerutil.Object) error {
	for _, obj := range objs {
		obj := obj
		lowerKind := strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind)
		if err := u.config.Client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete %s %q: %v", lowerKind, obj.GetName(), err)
		} else if err == nil {
			log.Printf("%s %q deleted", lowerKind, obj.GetName())
		}
		if waitForDelete {
			key, err := client.ObjectKeyFromObject(obj)
			if err != nil {
				return fmt.Errorf("get %s key: %v", lowerKind, err)
			}
			if err := wait.PollImmediateUntil(250*time.Millisecond, func() (bool, error) {
				if err := u.config.Client.Get(ctx, key, obj); apierrors.IsNotFound(err) {
					return true, nil
				} else if err != nil {
					return false, err
				}
				return false, nil
			}, ctx.Done()); err != nil {
				return fmt.Errorf("wait for %s deleted: %v", lowerKind, err)
			}
		}
	}
	return nil
}

func (u *OperatorUninstall) getInstallPlanResources(ctx context.Context, installPlanKey types.NamespacedName) (crds, csvs, others []controllerutil.Object, err error) {
	installPlan := &v1alpha1.InstallPlan{}
	if err := u.config.Client.Get(ctx, installPlanKey, installPlan); err != nil {
		return nil, nil, nil, fmt.Errorf("get install plan: %v", err)
	}

	for _, step := range installPlan.Status.Plan {
		if step.Status != v1alpha1.StepStatusCreated {
			continue
		}
		obj := &unstructured.Unstructured{Object: map[string]interface{}{}}
		lowerKind := strings.ToLower(step.Resource.Kind)
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
			others = append(others, obj)
		}
	}
	return crds, csvs, others, nil
}
