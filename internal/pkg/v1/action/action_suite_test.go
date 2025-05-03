package action_test

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

const (
	verbCreate = "create"
)

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Internal v1 action Suite")
}

type fakeClient struct {
	// Expected errors for create/delete/get.
	createErr error
	deleteErr error
	getErr    error

	// counters for number of create/delete/get calls seen.
	createCalled int
	deleteCalled int
	getCalled    int

	// transformer functions for applying changes to an object
	// matching the objectKey prior to an operation of the
	// type `verb` (get/create/delete), where the operation is
	// not set to error fail with a corresponding error (getErr/createErr/deleteErr).
	transformers []objectTransformer
	client.Client
}

type objectTransformer struct {
	verb          string
	objectKey     client.ObjectKey
	transformFunc func(obj *client.Object)
}

func (c *fakeClient) Initialize() error {
	scheme, err := action.NewScheme()
	if err != nil {
		return err
	}
	clientBuilder := fake.NewClientBuilder().WithInterceptorFuncs(interceptor.Funcs{
		Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			c.createCalled++
			if c.createErr != nil {
				return c.createErr
			}
			objKey := types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}
			for _, t := range c.transformers {
				if t.verb == verbCreate && objKey == t.objectKey && t.transformFunc != nil {
					t.transformFunc(&obj)
				}
			}
			// make sure to plumb request through to underlying client
			return client.Create(ctx, obj, opts...)
		},
		Delete: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
			c.deleteCalled++
			if c.deleteErr != nil {
				return c.deleteErr
			}
			return client.Delete(ctx, obj, opts...)
		},
		Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			c.getCalled++
			if c.getErr != nil {
				return c.getErr
			}
			return client.Get(ctx, key, obj, opts...)
		},
	}).WithScheme(scheme)
	c.Client = clientBuilder.Build()
	return nil
}

func setupTestCatalogs(n int) []client.Object {
	var result []client.Object
	for i := 1; i <= n; i++ {
		result = append(result, newClusterCatalog(fmt.Sprintf("cat%d", i)))
	}

	return result
}

func newClusterCatalog(name string) *olmv1.ClusterCatalog {
	return &olmv1.ClusterCatalog{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
}

type extensionOpt func(*olmv1.ClusterExtension)

type catalogOpt func(*olmv1.ClusterCatalog)

func withVersion(version string) extensionOpt {
	return func(ext *olmv1.ClusterExtension) {
		ext.Spec.Source.Catalog.Version = version
	}
}

func withSourceType(sourceType string) extensionOpt {
	return func(ext *olmv1.ClusterExtension) {
		ext.Spec.Source.SourceType = sourceType
	}
}

// nolint: unparam
func withConstraintPolicy(policy string) extensionOpt {
	return func(ext *olmv1.ClusterExtension) {
		ext.Spec.Source.Catalog.UpgradeConstraintPolicy = olmv1.UpgradeConstraintPolicy(policy)
	}
}

func withChannels(channels ...string) extensionOpt {
	return func(ext *olmv1.ClusterExtension) {
		ext.Spec.Source.Catalog.Channels = channels
	}
}

func withLabels(labels map[string]string) extensionOpt {
	return func(ext *olmv1.ClusterExtension) {
		ext.SetLabels(labels)
	}
}

func withCatalogSourceType(sourceType olmv1.SourceType) catalogOpt {
	return func(catalog *olmv1.ClusterCatalog) {
		catalog.Spec.Source.Type = sourceType
	}
}

func withCatalogSourcePriority(priority int32) catalogOpt {
	return func(catalog *olmv1.ClusterCatalog) {
		catalog.Spec.Priority = priority
	}
}

func withCatalogPollInterval(pollInterval int, ref string) catalogOpt {
	return func(catalog *olmv1.ClusterCatalog) {
		if catalog.Spec.Source.Image == nil {
			catalog.Spec.Source.Image = &olmv1.ImageSource{}
		}
		catalog.Spec.Source.Image.Ref = ref
		catalog.Spec.Source.Image.PollIntervalMinutes = &pollInterval
	}
}

func withCatalogImageRef(ref string) catalogOpt {
	return func(catalog *olmv1.ClusterCatalog) {
		catalog.Spec.Source.Image.Ref = ref
	}
}

func buildExtension(packageName string, opts ...extensionOpt) *olmv1.ClusterExtension {
	ext := &olmv1.ClusterExtension{
		Spec: olmv1.ClusterExtensionSpec{
			Source: olmv1.SourceConfig{
				Catalog: &olmv1.CatalogFilter{PackageName: packageName},
			},
		},
	}
	ext.SetName(packageName)
	for _, opt := range opts {
		opt(ext)
	}

	return ext
}

func updateExtensionConditionStatus(name string, cl client.Client, typ string, status metav1.ConditionStatus) error {
	var ext olmv1.ClusterExtension
	key := types.NamespacedName{Name: name}

	if err := cl.Get(context.TODO(), key, &ext); err != nil {
		return err
	}

	apimeta.SetStatusCondition(&ext.Status.Conditions, metav1.Condition{
		Type:               typ,
		Status:             status,
		ObservedGeneration: ext.GetGeneration(),
	})

	return cl.Update(context.TODO(), &ext)
}

func buildCatalog(catalogName string, opts ...catalogOpt) *olmv1.ClusterCatalog {
	catalog := &olmv1.ClusterCatalog{
		Spec: olmv1.ClusterCatalogSpec{
			Source: olmv1.CatalogSource{
				Type: olmv1.SourceTypeImage,
			},
		},
	}
	catalog.SetName(catalogName)
	for _, opt := range opts {
		opt(catalog)
	}

	return catalog
}
