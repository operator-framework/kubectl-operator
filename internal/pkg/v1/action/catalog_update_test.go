package action_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
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
			withCatalogPollInterval(pointerToInt(5)),
			withCatalogSourcePriority(pointerToInt32(1)),
			withCatalogImageRef("quay.io/myrepo/myimage"),
			withCatalogAvailabilityMode(olmv1.AvailabilityModeAvailable),
			withCatalogLabels(map[string]string{"foo": "bar"}),
		)
		cfg := setupEnv(testCatalog)

		updater := internalaction.NewCatalogUpdate(&cfg)
		updater.CatalogName = "testCatalog"
		updater.Priority = pointerToInt32(1)
		updater.Labels = map[string]string{"abc": "xyz"}
		updater.AvailabilityMode = string(olmv1.AvailabilityModeAvailable)
		updater.PollIntervalMinutes = pointerToInt(5)
		catalog, err := updater.Run(context.TODO())

		Expect(err).To(BeNil())
		Expect(testCatalog).NotTo(BeNil())
		Expect(catalog.Labels).To(HaveKeyWithValue("foo", "bar")) //existing
		Expect(catalog.Labels).To(HaveKeyWithValue("abc", "xyz")) //newly added
		Expect(catalog.Spec.Priority).To(Equal(*updater.Priority))
		Expect(catalog.Spec.Source.Image.PollIntervalMinutes).ToNot(BeNil())
		Expect(*catalog.Spec.Source.Image.PollIntervalMinutes).To(Equal(*updater.PollIntervalMinutes))
		Expect(catalog.Spec.AvailabilityMode).To(Equal(olmv1.AvailabilityMode(updater.AvailabilityMode)))
	})

	It("unsets the poll interval field when set to 0", func() {
		testCatalog := buildCatalog(
			"test",
			withCatalogSourceType(olmv1.SourceTypeImage),
			withCatalogPollInterval(pointerToInt(7)),
			withCatalogImageRef("quay.io/myrepo/myimage"),
		)
		cfg := setupEnv(testCatalog)

		updater := internalaction.NewCatalogUpdate(&cfg)
		updater.CatalogName = "test"
		updater.PollIntervalMinutes = pointerToInt(-1)
		catalog, err := updater.Run(context.TODO())

		Expect(err).NotTo(HaveOccurred())
		Expect(catalog.Spec.Source.Image.PollIntervalMinutes).To(BeNil())
	})

	It("unsets the poll interval field when set to 0", func() {
		testCatalog := buildCatalog(
			"test",
			withCatalogSourceType(olmv1.SourceTypeImage),
			withCatalogPollInterval(pointerToInt(10)),
			withCatalogImageRef("quay.io/myrepo/myimage"),
		)
		cfg := setupEnv(testCatalog)

		updater := internalaction.NewCatalogUpdate(&cfg)
		updater.CatalogName = "test"
		updater.PollIntervalMinutes = pointerToInt(0)

		catalog, err := updater.Run(context.TODO())

		Expect(err).NotTo(HaveOccurred())
		Expect(catalog.Spec.Source.Image.PollIntervalMinutes).To(BeNil())
	})

	It("succeessfully updates catalog with a valid image reference", func() {
		testCatalog := buildCatalog(
			"test",
			withCatalogSourceType(olmv1.SourceTypeImage),
			withCatalogImageRef("quay.io/myrepo/myimage"),
			withCatalogPollInterval(pointerToInt(10)),
			withCatalogSourcePriority(pointerToInt32(5)),
			withCatalogAvailabilityMode(olmv1.AvailabilityModeAvailable),
			withCatalogLabels(map[string]string{"foo": "bar"}),
		)
		cfg := setupEnv(testCatalog)

		updater := internalaction.NewCatalogUpdate(&cfg)
		updater.CatalogName = "test"
		updater.ImageRef = "quay.io/myrepo/mynewimage"
		catalog, err := updater.Run(context.TODO())

		Expect(err).NotTo(HaveOccurred())
		Expect(catalog.Spec.Source.Image.Ref).To(Equal(updater.ImageRef))
	})

	It("fails catalog update with an invalid image reference", func() {
		testCatalog := buildCatalog(
			"test",
			withCatalogSourceType(olmv1.SourceTypeImage),
			withCatalogImageRef("quay.io/valid/image"),
		)
		cfg := setupEnv(testCatalog)

		updater := internalaction.NewCatalogUpdate(&cfg)
		updater.CatalogName = "test"
		updater.ImageRef = "invalid//image!!"

		_, err := updater.Run(context.TODO())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid image reference"))
	})

	It("preserves existing catalog values if Priority and PollIntervalMinutes are nil", func() {
		testCatalog := buildCatalog(
			"test",
			withCatalogSourceType(olmv1.SourceTypeImage),
			withCatalogPollInterval(pointerToInt(15)),
			withCatalogSourcePriority(pointerToInt32(10)),
			withCatalogImageRef("quay.io/myrepo/image"),
		)

		cfg := setupEnv(testCatalog)

		updater := internalaction.NewCatalogUpdate(&cfg)
		updater.CatalogName = "test"
		updater.Priority = nil
		updater.PollIntervalMinutes = nil

		catalog, err := updater.Run(context.TODO())

		Expect(err).NotTo(HaveOccurred())
		Expect(catalog.Spec.Priority).To(Equal(int32(10)))
		Expect(*catalog.Spec.Source.Image.PollIntervalMinutes).To(Equal(15))
	})

	It("removes labels with empty values and merges the rest", func() {
		initial := map[string]string{"foo": "bar", "remove": "yes"}
		testCatalog := buildCatalog(
			"test",
			withCatalogSourceType(olmv1.SourceTypeImage),
			withCatalogLabels(initial),
		)
		cfg := setupEnv(testCatalog)

		updater := internalaction.NewCatalogUpdate(&cfg)
		updater.CatalogName = "test"
		updater.Labels = map[string]string{
			"remove": "",
			"new":    "label",
		}
		catalog, err := updater.Run(context.TODO())
		Expect(err).NotTo(HaveOccurred())

		Expect(catalog.Labels).To(Equal(map[string]string{
			"foo": "bar",
			"new": "label",
		}))
	})

	It("preserves labels when Labels field is nil", func() {
		testCatalog := buildCatalog(
			"test",
			withCatalogSourceType(olmv1.SourceTypeImage),
			withCatalogLabels(map[string]string{"retain": "this"}),
		)
		cfg := setupEnv(testCatalog)

		updater := internalaction.NewCatalogUpdate(&cfg)
		updater.CatalogName = "test"
		updater.Labels = nil

		catalog, err := updater.Run(context.TODO())
		Expect(err).NotTo(HaveOccurred())
		Expect(catalog.Labels).To(Equal(map[string]string{"retain": "this"}))
	})

})

func pointerToInt32(i int32) *int32 {
	return &i
}

func pointerToInt(i int) *int {
	return &i
}
