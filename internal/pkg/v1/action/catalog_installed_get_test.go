package action_test

import (
	"context"
	"slices"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

var _ = Describe("CatalogInstalledGet", func() {
	setupEnv := func(catalogs ...client.Object) action.Configuration {
		var cfg action.Configuration

		sch, err := action.NewScheme()
		Expect(err).To(BeNil())

		cl := fake.NewClientBuilder().
			WithObjects(catalogs...).
			WithScheme(sch).
			Build()
		cfg.Scheme = sch
		cfg.Client = cl

		return cfg
	}

	It("lists all installed catalogs", func() {
		cfg := setupEnv(setupTestCatalogs(3)...)

		getter := internalaction.NewCatalogInstalledGet(&cfg)
		catalogs, err := getter.Run(context.TODO())
		Expect(err).To(BeNil())
		Expect(catalogs).NotTo(BeEmpty())
		Expect(catalogs).To(HaveLen(3))

		for _, testCatalogName := range []string{"cat1", "cat2", "cat3"} {
			Expect(slices.ContainsFunc(catalogs, func(cat olmv1.ClusterCatalog) bool {
				return cat.Name == testCatalogName
			})).To(BeTrue())
		}
	})

	It("returns empty list in case no catalogs were found", func() {
		cfg := setupEnv()

		getter := internalaction.NewCatalogInstalledGet(&cfg)
		catalogs, err := getter.Run(context.TODO())
		Expect(err).To(BeNil())
		Expect(catalogs).To(BeEmpty())
	})

	It("gets an installed catalog", func() {
		cfg := setupEnv(setupTestCatalogs(3)...)

		getter := internalaction.NewCatalogInstalledGet(&cfg)
		getter.CatalogName = "cat2"
		catalogs, err := getter.Run(context.TODO())
		Expect(err).To(BeNil())
		Expect(catalogs).NotTo(BeEmpty())
		Expect(catalogs).To(HaveLen(1))
		Expect(catalogs[0].Name).To(Equal("cat2"))
	})

	It("returns an empty list when an installed catalog was not found", func() {
		cfg := setupEnv()

		getter := internalaction.NewCatalogInstalledGet(&cfg)
		getter.CatalogName = "cat2"
		catalogs, err := getter.Run(context.TODO())
		Expect(err).NotTo(BeNil())
		Expect(catalogs).To(BeEmpty())
	})
})
