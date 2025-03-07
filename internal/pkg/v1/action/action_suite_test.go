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

type extensionOpt func(*olmv1.ClusterExtension)

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

func updateOperatorConditionStatus(name string, cl client.Client, typ string, status metav1.ConditionStatus) error {
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
