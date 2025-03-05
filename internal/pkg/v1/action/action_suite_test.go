package action_test

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"
)

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Internal v1 action Suite")
}

type mockCreator struct {
	createErr    error
	createCalled int
}

func (mc *mockCreator) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	mc.createCalled++
	return mc.createErr
}

type mockDeleter struct {
	deleteErr    error
	deleteCalled int
}

func (md *mockDeleter) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	md.deleteCalled++
	return md.deleteErr
}

type mockGetter struct {
	getErr    error
	getCalled int
}

func (mg *mockGetter) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	mg.getCalled++
	return mg.getErr
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
