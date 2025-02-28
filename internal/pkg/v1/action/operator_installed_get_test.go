package action_test

import (
	"context"
	"fmt"
	"slices"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

var _ = Describe("OperatorInstalledGet", func() {
	setupEnv := func(operators ...client.Object) action.Configuration {
		var cfg action.Configuration

		sch, err := action.NewScheme()
		Expect(err).To(BeNil())

		cl := fake.NewClientBuilder().
			WithObjects(operators...).
			WithScheme(sch).
			Build()
		cfg.Scheme = sch
		cfg.Client = cl

		return cfg
	}

	It("lists all installed operators", func() {
		cfg := setupEnv(setupTestOperators(3)...)

		getter := internalaction.NewOperatorInstalledGet(&cfg)
		operators, err := getter.Run(context.TODO())
		Expect(err).To(BeNil())
		Expect(operators).NotTo(BeEmpty())
		Expect(operators).To(HaveLen(3))

		for _, testOperatorName := range []string{"ext1", "ext2", "ext3"} {
			Expect(slices.ContainsFunc(operators, func(op olmv1.ClusterExtension) bool {
				return op.Name == testOperatorName
			})).To(BeTrue())
		}
	})

	It("returns empty list in case no operators were found", func() {
		cfg := setupEnv()

		getter := internalaction.NewOperatorInstalledGet(&cfg)
		operators, err := getter.Run(context.TODO())
		Expect(err).To(BeNil())
		Expect(operators).To(BeEmpty())
	})

	It("gets an installed operator", func() {
		cfg := setupEnv(setupTestOperators(3)...)

		getter := internalaction.NewOperatorInstalledGet(&cfg)
		getter.OperatorName = "ext2"
		operators, err := getter.Run(context.TODO())
		Expect(err).To(BeNil())
		Expect(operators).NotTo(BeEmpty())
		Expect(operators).To(HaveLen(1))
		Expect(operators[0].Name).To(Equal("ext2"))
	})

	It("returns an empty list and an error when an installed operator was not found", func() {
		cfg := setupEnv()

		getter := internalaction.NewOperatorInstalledGet(&cfg)
		getter.OperatorName = "ext2"
		operators, err := getter.Run(context.TODO())
		Expect(err).NotTo(BeNil())
		Expect(operators).To(BeEmpty())
	})
})

func setupTestOperators(n int) []client.Object {
	var result []client.Object
	for i := 1; i <= n; i++ {
		result = append(result, newClusterExtension(fmt.Sprintf("ext%d", i), fmt.Sprintf("%d.0", n)))
	}

	return result
}

func newClusterExtension(name, version string) *olmv1.ClusterExtension {
	return &olmv1.ClusterExtension{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: olmv1.ClusterExtensionStatus{
			Install: &olmv1.ClusterExtensionInstallStatus{
				Bundle: olmv1.BundleMetadata{
					Name:    name,
					Version: version,
				},
			},
		},
	}
}
