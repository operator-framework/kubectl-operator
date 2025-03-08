package action_test

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/kubectl-operator/pkg/action"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"
)

const (
	verbCreate = "create"
)

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Internal v1 action Suite")
}

type fakeClient struct {
	createErr    error
	deleteErr    error
	getErr       error
	createCalled int
	deleteCalled int
	getCalled    int
	transformers []objectTransformer
	client.Client
}

// meant to apply a function to anything matching the objectKey
// when the fakeClient an action corresponding to verb.
// transformer will not run if an error is already specified
// for the verb in the fakeClient.
// only implemented in Create for now
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
