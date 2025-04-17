package action_test

import (
	"context"
	"maps"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
	olmv1 "github.com/operator-framework/operator-controller/api/v1"
)

var _ = Describe("CatalogUpdate", func() {
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

	It("fails finding existing catalog", func() {
		cfg := setupEnv()

		updater := internalaction.NewCatalogUpdate(&cfg)
		updater.CatalogName = "does-not-exist"
		cat, err := updater.Run(context.TODO())

		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("not found"))
		Expect(cat).To(BeNil())
	})

	It("fails to handle catalog with unknown source type", func() {
		cfg := setupEnv(buildCatalog("test", withCatalogSourceType("invalid-type")))

		updater := internalaction.NewCatalogUpdate(&cfg)
		updater.CatalogName = "test"
		_, err := updater.Run(context.TODO())

		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("unrecognized source type"))
	})

	It("successfully updates catalog", func() {
		testCatalog := buildCatalog(
			"testCatalog",
			withCatalogSourceType(olmv1.SourceTypeImage),
			withCatalogPollInterval(5, "testCatalog"),
			withCatalogSourcePriority(1),
		)
		cfg := setupEnv(testCatalog)

		updater := internalaction.NewCatalogUpdate(&cfg)
		updater.CatalogName = "testCatalog"
		updater.Priority = int32(1)
		updater.Labels = map[string]string{"c": "d"}
		updater.AvailabilityMode = string(olmv1.AvailabilityModeAvailable)
		updater.PollIntervalMinutes = int(5)
		catalog, err := updater.Run(context.TODO())

		Expect(err).To(BeNil())
		Expect(testCatalog).NotTo(BeNil())
		Expect(maps.Equal(catalog.Labels, updater.Labels)).To(BeTrue())
		Expect(catalog.Spec.Priority).To(Equal(updater.Priority))
		Expect(catalog.Spec.Source.Image.PollIntervalMinutes).ToNot(BeNil())
		Expect(*catalog.Spec.Source.Image.PollIntervalMinutes).To(Equal(int(5)))
		Expect(catalog.Spec.AvailabilityMode).
			To(Equal(olmv1.AvailabilityMode(updater.AvailabilityMode)))

	})
})

func TestCatalogUpdate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Catalog Update Suite")
}
