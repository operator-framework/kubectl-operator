package action

import (
	"context"
	"fmt"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/api/pkg/operators/v2alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

type CustomResourceLister interface {
	List(ctx context.Context, ownedCRDS []v1alpha1.CRDDescription) (metav1.PartialObjectMetadataList, error)
}

type CustomResourceDefinitionFinder interface {
	FindOperator(ctx context.Context, packageName, namespace string) (*v2alpha1.Operator, error)
	Unzip(ctx context.Context, operator *v2alpha1.Operator) ([]v1alpha1.CRDDescription, error)
}

// OperatorListCustomResources knows how to find and list custom resources given a package name and namespace.
type OperatorListCustomResources struct {
	config  *Configuration
	PackageName string
	AllNamespaces bool
}

func NewOperatorListCustomResources(cfg *Configuration) *OperatorListCustomResources {
	return &OperatorListCustomResources{
		config: cfg,
	}
}

// FindOperator finds an operator object on-cluster provided a package and namespace. Returns a copy of the operator.
func (o *OperatorListCustomResources) FindOperator(ctx context.Context) (*v2alpha1.Operator, error) {
	opKey := types.NamespacedName{
		Name:      o.PackageName,
		Namespace: o.config.Namespace,
	}
	operator := v2alpha1.Operator{}
	err := o.config.Client.Get(ctx, opKey, &operator)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, ErrPackageNotFound{o.PackageName}
		}
		return nil, err
	}
	return &operator, nil
}

// Unzip finds the CSV referenced by the provided operator and then inspects the spec.customresourcedefinitions.owned
// section of the CSV to return a list of APIs that are owned by the CSV.
func (o *OperatorListCustomResources) Unzip(ctx context.Context, operator *v2alpha1.Operator) ([]v1alpha1.CRDDescription, error) {
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
		continue
	}

	if csvKey.Name == "" && csvKey.Namespace == "" {
		return nil, fmt.Errorf("could not find underlying CSV for operator %s", operator.Name)
	}

	err := o.config.Client.Get(ctx, csvKey, &csv)
	if err != nil {
		return nil, fmt.Errorf("could not get %s CSV on cluster: %s", csvKey.String(), err)
	}

	// check if owned CRDs are defined on the csv
	if len(csv.Spec.CustomResourceDefinitions.Owned) == 0 {
		return nil, fmt.Errorf("no owned CustomResourceDefinitions specified on CSV %s, no custom resources to display", csvKey.String())
	}

	return csv.Spec.CustomResourceDefinitions.Owned, nil
}

// List takes in a list of CRDs and finds the associated CRs metadata on-cluster using the client-go meta client.
// Note: List() can return a potentially unbounded list that callers may need to paginate.
func (o *OperatorListCustomResources) List(ctx context.Context, crds []v1alpha1.CRDDescription) (*metav1.PartialObjectMetadataList, error) {
	result := &metav1.PartialObjectMetadataList{}

	for _, c := range crds {
		// find CRD to determine scope
		// set up GVR for CRs based on CRD metadata
		crd := apiextensionsv1.CustomResourceDefinition{}
		crdKey := types.NamespacedName{
			Name: c.Name,
		}
		err := o.config.Client.Get(ctx, crdKey, &crd)
		if err != nil {
			return nil, nil
		}
	}

	for _, c := range crds {
		// setup GVR
		gvr := schema.GroupVersionResource{
			Group:    c.Name,
			Version:  c.Version,
			Resource: strings.ToLower(c.Kind) + "s", // is this always a valid assumption?
		}

		list := metav1.PartialObjectMetadataList{}
		err := o.config.Client.List(ctx, &list )
		if err != nil && !k8serrors.IsNotFound(err) {
			return nil, fmt.Errorf("finding crs for gvr %s: %s", gvr.String(), err)
		}
		for _, item := range r.Items {
			result.Items = append(result.Items, item)
		}
	}

	return result, nil
}
