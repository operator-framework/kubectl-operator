package action_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/client"
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
