package v0

import (
	"cmp"
	"context"
	"fmt"
	"github.com/operator-framework/kubectl-operator/pkg/action"
	ocv1 "github.com/operator-framework/operator-controller/api/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"slices"
)

// OperatorListOperands knows how to find and listOperandsForCRD custom resources given a package name and namespace.
type OperatorListOperands struct {
	config *action.Configuration
}

func NewOperatorListOperands(cfg *action.Configuration) *OperatorListOperands {
	return &OperatorListOperands{
		config: cfg,
	}
}

func (o *OperatorListOperands) Run(ctx context.Context, clusterExtensionName string) (*unstructured.UnstructuredList, error) {
	crds, err := o.getCRDs(ctx, clusterExtensionName)
	if err != nil {
		return nil, err
	}

	var result unstructured.UnstructuredList
	result.SetGroupVersionKind(schema.GroupVersionKind{
		Version: "v1",
		Kind:    "List",
	})
	for _, crd := range crds {
		operands, err := o.listOperandsForCRD(ctx, crd)
		if err != nil {
			return nil, err
		}
		result.Items = append(result.Items, operands...)
	}

	// sort results
	slices.SortFunc(result.Items, func(a, b unstructured.Unstructured) int {
		if x := cmp.Compare(
			a.GroupVersionKind().GroupKind().String(),
			b.GroupVersionKind().GroupKind().String(),
		); x != 0 {
			return x
		}
		if x := cmp.Compare(a.GetNamespace(), b.GetNamespace()); x != 0 {
			return x
		}
		return cmp.Compare(a.GetName(), b.GetName())
	})

	return &result, nil
}

func (o *OperatorListOperands) getCRDs(ctx context.Context, clusterExtensionName string) ([]apiextensionsv1.CustomResourceDefinition, error) {
	ce := ocv1.ClusterExtension{}
	ceKey := types.NamespacedName{
		Name: clusterExtensionName,
	}
	if err := o.config.Client.Get(ctx, ceKey, &ce); err != nil {
		return nil, fmt.Errorf("get cluster extension %q: %v", clusterExtensionName, err)
	}

	progCond := meta.FindStatusCondition(ce.Status.Conditions, ocv1.TypeProgressing)
	ready := meta.IsStatusConditionTrue(ce.Status.Conditions, ocv1.TypeInstalled) &&
		progCond != nil && progCond.Status == metav1.ConditionFalse &&
		progCond.Reason == ocv1.ReasonSucceeded

	if !ready {
		return nil, fmt.Errorf("cluster extension %q is not at steady state: operand listOperandsForCRD may be inaccurate", clusterExtensionName)
	}

	crds := apiextensionsv1.CustomResourceDefinitionList{}
	labelSelector := client.MatchingLabels{
		"olm.operatorframework.io/owner-kind": "ClusterExtension",
		"olm.operatorframework.io/owner-name": clusterExtensionName,
	}
	if err := o.config.Client.List(ctx, &crds, labelSelector); err != nil {
		return nil, fmt.Errorf("list crds: %v", err)
	}
	return crds.Items, nil
}

func (o *OperatorListOperands) listOperandsForCRD(ctx context.Context, crd apiextensionsv1.CustomResourceDefinition) ([]unstructured.Unstructured, error) {
	servedVersion := ""
	for _, v := range crd.Spec.Versions {
		if v.Served {
			servedVersion = v.Name
			break
		}
	}

	list := unstructured.UnstructuredList{}
	gvk := schema.GroupVersionKind{
		Group:   crd.Spec.Group,
		Version: servedVersion,
		Kind:    crd.Spec.Names.ListKind,
	}
	list.SetGroupVersionKind(gvk)
	if err := o.config.Client.List(ctx, &list); err != nil {
		return nil, err
	}

	return list.Items, nil
}
