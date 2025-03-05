package action_test

import (
	"context"
	"slices"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	olmv1catalogd "github.com/operator-framework/catalogd/api/v1"

	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

var _ = Describe("CatalogDelete", func() {
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

	It("fails because of both resource name and --all specifier being present", func() {
		cfg := setupEnv(setupTestCatalogs(2)...)

		deleter := internalaction.NewCatalogDelete(&cfg)
		deleter.CatalogName = "name"
		deleter.DeleteAll = true
		catNames, err := deleter.Run(context.TODO())
		Expect(err).NotTo(BeNil())
		Expect(catNames).To(BeEmpty())

		validateExistingCatalogs(cfg.Client, []string{"cat1", "cat2"})
	})

	It("fails deleting a non-existing catalog", func() {
		cfg := setupEnv(setupTestCatalogs(2)...)

		deleter := internalaction.NewCatalogDelete(&cfg)
		deleter.CatalogName = "does-not-exist"
		catNames, err := deleter.Run(context.TODO())
		Expect(err).NotTo(BeNil())
		Expect(catNames).To(BeEmpty())

		validateExistingCatalogs(cfg.Client, []string{"cat1", "cat2"})
	})

	It("successfully deletes an existing catalog", func() {
		cfg := setupEnv(setupTestCatalogs(3)...)

		deleter := internalaction.NewCatalogDelete(&cfg)
		deleter.CatalogName = "cat2"
		catNames, err := deleter.Run(context.TODO())
		Expect(err).To(BeNil())
		Expect(catNames).To(BeEmpty())

		validateExistingCatalogs(cfg.Client, []string{"cat1", "cat3"})
	})

	It("fails deleting catalogs because there are none", func() {
		cfg := setupEnv()

		deleter := internalaction.NewCatalogDelete(&cfg)
		deleter.DeleteAll = true
		catNames, err := deleter.Run(context.TODO())
		Expect(err).NotTo(BeNil())
		Expect(catNames).To(BeEmpty())

		validateExistingCatalogs(cfg.Client, []string{})
	})

	It("successfully deletes all catalogs", func() {
		cfg := setupEnv(setupTestCatalogs(3)...)

		deleter := internalaction.NewCatalogDelete(&cfg)
		deleter.DeleteAll = true
		catNames, err := deleter.Run(context.TODO())
		Expect(err).To(BeNil())
		Expect(catNames).To(ContainElements([]string{"cat1", "cat2", "cat3"}))

		validateExistingCatalogs(cfg.Client, []string{})
	})
})

func validateExistingCatalogs(c client.Client, wantedNames []string) {
	var catalogsList olmv1catalogd.ClusterCatalogList
	err := c.List(context.TODO(), &catalogsList)
	Expect(err).To(BeNil())

	catalogs := catalogsList.Items
	Expect(catalogs).To(HaveLen(len(wantedNames)))
	for _, wantedName := range wantedNames {
		Expect(slices.ContainsFunc(catalogs, func(cat olmv1catalogd.ClusterCatalog) bool {
			return cat.Name == wantedName
		})).To(BeTrue())
	}
}
