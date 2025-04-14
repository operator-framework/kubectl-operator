package action_test

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ocv1 "github.com/operator-framework/operator-controller/api/v1"

	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

var _ = Describe("InstallExtension", func() {
	extensionName := "testExtension"
	packageName := "testPackage"
	packageVersion := "1.0.0"
	serviceAccount := "testServiceAccount"
	namespace := "testNamespace"

	expectedExtension := ocv1.ClusterExtension{
		ObjectMeta: metav1.ObjectMeta{
			Name: extensionName,
		},
		Spec: ocv1.ClusterExtensionSpec{
			Source: ocv1.SourceConfig{
				SourceType: ocv1.SourceTypeCatalog,
				Catalog: &ocv1.CatalogFilter{
					PackageName: packageName,
					Version:     packageVersion,
				},
			},
			Namespace: namespace,
			ServiceAccount: ocv1.ServiceAccountReference{
				Name: serviceAccount,
			},
		},
	}
	It("Cluster extension install fails", func() {
		expectedErr := errors.New("extension install failed")
		testClient := fakeClient{createErr: expectedErr}
		Expect(testClient.Initialize()).To(Succeed())

		installer := internalaction.NewExtensionInstall(&action.Configuration{Client: testClient})
		installer.ExtensionName = expectedExtension.Name
		installer.PackageName = expectedExtension.Spec.Source.Catalog.PackageName
		installer.Channels = expectedExtension.Spec.Source.Catalog.Channels
		installer.Version = expectedExtension.Spec.Source.Catalog.Version
		installer.ServiceAccount = expectedExtension.Spec.ServiceAccount.Name
		installer.CleanupTimeout = 1 * time.Minute
		installer.Namespace.Name = expectedExtension.Spec.Namespace
		_, err := installer.Run(context.TODO())

		Expect(err).NotTo(BeNil())
		Expect(err).To(MatchError(expectedErr))
		Expect(testClient.createCalled).To(Equal(1))
	})
})
