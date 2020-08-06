package action

import (
	"context"
	"fmt"

	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/spf13/pflag"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/joelanford/kubectl-operator/internal/pkg/log"
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

	if err := u.config.Client.Delete(ctx, sub); err != nil {
		return fmt.Errorf("delete subscription %q: %v", sub.Name, err)
	}
	log.Printf("subscription %q deleted", sub.Name)

	if sub.Status.CurrentCSV != "" && sub.Status.CurrentCSV != sub.Status.InstalledCSV {
		if err := u.deleteCSVandCRDs(ctx, sub.Status.CurrentCSV, true); err != nil {
			return err
		}
	}
	if sub.Status.InstalledCSV != "" {
		if err := u.deleteCSVandCRDs(ctx, sub.Status.InstalledCSV, false); err != nil {
			return err
		}
	}

	if u.DeleteOperatorGroup {
		csvs := v1alpha1.ClusterServiceVersionList{}
		if err := u.config.Client.List(ctx, &csvs, client.InNamespace(u.config.Namespace)); err != nil {
			return fmt.Errorf("list clusterserviceversions: %v", err)
		}
		if len(csvs.Items) == 0 {
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

func (u *OperatorUninstall) deleteCSVandCRDs(ctx context.Context, csvName string, ignoreNotFound bool) error {
	csvKey := types.NamespacedName{
		Name:      csvName,
		Namespace: u.config.Namespace,
	}
	csv := v1alpha1.ClusterServiceVersion{}
	if err := u.config.Client.Get(ctx, csvKey, &csv); err != nil {
		if !apierrors.IsNotFound(err) || !ignoreNotFound {
			return fmt.Errorf("get clusterserviceversion %q: %v", csvName, err)
		}
	}

	if u.DeleteCRDs {
		ownedCRDs := csv.Spec.CustomResourceDefinitions.Owned
		for _, ownedCRD := range ownedCRDs {
			crd := apiextv1.CustomResourceDefinition{}
			crd.SetName(ownedCRD.Name)
			if err := u.config.Client.Delete(ctx, &crd); err != nil {
				return fmt.Errorf("delete crd %q: %v", ownedCRD.Name, err)
			}
			log.Printf("crd %q deleted", ownedCRD.Name)
		}
	}

	if err := u.config.Client.Delete(ctx, &csv); err != nil {
		return fmt.Errorf("delete csv %q: %v", csvName, err)
	}
	log.Printf("csv %q deleted", csvName)

	return nil
}
