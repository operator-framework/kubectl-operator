package action

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/api/pkg/operators/v2alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FindOperator finds an operator object on-cluster provided a package and namespace.
func FindOperator(ctx context.Context, client client.Client, key types.NamespacedName) (*v2alpha1.Operator, error) {
	operator := v2alpha1.Operator{}

	err := client.Get(ctx, key, &operator)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, fmt.Errorf("package %s not found", key.Name)
		}
		return nil, err
	}
	return &operator, nil
}

// Unzip finds the CSV referenced by the provided operator and then inspects the spec.customresourcedefinitions.owned
// section of the CSV to return a list of APIs that are owned by the CSV.
func Unzip(ctx context.Context, client client.Client, operator *v2alpha1.Operator) ([]v1alpha1.CRDDescription, error) {
	csv := v1alpha1.ClusterServiceVersion{}
	csvKey := types.NamespacedName{}

	if operator.Status.Components == nil {
		return nil, fmt.Errorf("could not find underlying components for operator %s", operator.Name)
	}
	for _, resource := range operator.Status.Components.Refs {
		if resource.Kind == v1alpha1.ClusterServiceVersionKind {
			csvKey.Name = resource.Name
			csvKey.Namespace = resource.Namespace
			break
		}
	}

	if csvKey.Name == "" && csvKey.Namespace == "" {
		return nil, fmt.Errorf("could not find underlying CSV for operator %s", operator.Name)
	}

	err := client.Get(ctx, csvKey, &csv)
	if err != nil {
		return nil, fmt.Errorf("could not get %s CSV on cluster: %s", csvKey.String(), err)
	}

	// check if owned CRDs are defined on the csv
	if len(csv.Spec.CustomResourceDefinitions.Owned) == 0 {
		return nil, fmt.Errorf("no owned CustomResourceDefinitions specified on CSV %s, no custom resources to display", csvKey.String())
	}

	return csv.Spec.CustomResourceDefinitions.Owned, nil
}

// List takes in a CRD description and finds the associated CRs on-cluster.
// List can return a potentially unbounded list that callers may need to paginate.
func List(ctx context.Context, crClient client.Client, crdDesc v1alpha1.CRDDescription, namespace string) (*unstructured.UnstructuredList, error) {
	result := &unstructured.UnstructuredList{}

	// find CRD on-cluster to determine CRD scope (not included in description)
	crd := apiextensionsv1.CustomResourceDefinition{}
	crdKey := types.NamespacedName{
		Name: crdDesc.Name,
	}
	err := crClient.Get(ctx, crdKey, &crd)
	if err != nil {
		return nil, nil
	}
	scope := crd.Spec.Scope

	if scope == apiextensionsv1.ClusterScoped {
		// get all CRs across the cluster for the given CRD since namespace is not relevant for cluster-scoped CRs
		result := unstructured.UnstructuredList{}
		gvk := schema.GroupVersionKind{
			Group:   crdDesc.Name,
			Version: crdDesc.Version,
			Kind:    crdDesc.Kind,
		}
		result.SetGroupVersionKind(gvk)
		if err := crClient.List(ctx, &result); err != nil {
			return nil, err
		}
		return &result, nil

	} else if scope == apiextensionsv1.NamespaceScoped {
		// get CRs in the cluster for the given namespace only
		result := unstructured.UnstructuredList{}
		gvk := schema.GroupVersionKind{
			Group:   crdDesc.Name,
			Version: crdDesc.Version,
			Kind:    crdDesc.Kind,
		}
		result.SetGroupVersionKind(gvk)

		options := client.ListOptions{Namespace: namespace}
		if err := crClient.List(ctx, &result, &options); err != nil {
			return nil, err
		}
		return &result, nil
	}

	return result, nil
}

// ListAll wraps the above functions to provide a convenient command to go from package/namespace to custom resources.
func ListAll(ctx context.Context, client client.Client, opKey types.NamespacedName) (*unstructured.UnstructuredList, error) {
	operator, err := FindOperator(ctx, client, opKey)
	if err != nil {
		return nil, err
	}

	crdDescs, err := Unzip(ctx, client, operator)
	if err != nil {
		return nil, err
	}

	var result unstructured.UnstructuredList
	for _, crd := range crdDescs {
		list, err := List(ctx, client, crd, opKey.Namespace)
		if err != nil {
			return nil, err
		}
		for _, cr := range list.Items {
			result.Items = append(result.Items, cr)
		}
	}
	return &result, nil
}
