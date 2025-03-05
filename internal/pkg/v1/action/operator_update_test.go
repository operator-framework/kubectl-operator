package action_test

import (
	"context"
	"maps"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

var _ = Describe("OperatorUpdate", func() {
	setupEnv := func(extensions ...client.Object) action.Configuration {
		var cfg action.Configuration

		sch, err := action.NewScheme()
		Expect(err).To(BeNil())

		cl := fake.NewClientBuilder().
			WithObjects(extensions...).
			WithScheme(sch).
			Build()
		cfg.Scheme = sch
		cfg.Client = cl

		return cfg
	}

	It("fails finding existing operator", func() {
		cfg := setupEnv()

		updater := internalaction.NewOperatorUpdate(&cfg)
		updater.Package = "does-not-exist"
		ext, err := updater.Run(context.TODO())

		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("not found"))
		Expect(ext).To(BeNil())
	})

	It("fails to handle operator with non-catalog source type", func() {
		cfg := setupEnv(buildExtension("test", withSourceType("unknown")))

		updater := internalaction.NewOperatorUpdate(&cfg)
		updater.Package = "test"
		ext, err := updater.Run(context.TODO())

		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("unrecognized source type"))
		Expect(ext).To(BeNil())
	})

	It("fails because desired operator state matches current", func() {
		cfg := setupEnv(buildExtension(
			"test",
			withSourceType(olmv1.SourceTypeCatalog),
			withConstraintPolicy(string(olmv1.UpgradeConstraintPolicyCatalogProvided))),
		)

		updater := internalaction.NewOperatorUpdate(&cfg)
		updater.Package = "test"
		ext, err := updater.Run(context.TODO())

		Expect(err).NotTo(BeNil())
		Expect(err).To(MatchError(internalaction.ErrNoChange))
		Expect(ext).To(BeNil())
	})

	It("fails because desired operator state matches current with IgnoreUnset enabled", func() {
		cfg := setupEnv(buildExtension(
			"test",
			withSourceType(olmv1.SourceTypeCatalog),
			withConstraintPolicy(string(olmv1.UpgradeConstraintPolicyCatalogProvided)),
			withChannels("a", "b"),
			withLabels(map[string]string{"c": "d"}),
			withVersion("10.0.4"),
		))

		updater := internalaction.NewOperatorUpdate(&cfg)
		updater.Package = "test"
		updater.IgnoreUnset = true
		ext, err := updater.Run(context.TODO())

		Expect(err).NotTo(BeNil())
		Expect(err).To(MatchError(internalaction.ErrNoChange))
		Expect(ext).To(BeNil())
	})

	It("fails validating operator version", func() {
		cfg := setupEnv(buildExtension(
			"test",
			withSourceType(olmv1.SourceTypeCatalog),
			withConstraintPolicy(string(olmv1.UpgradeConstraintPolicyCatalogProvided))),
		)

		updater := internalaction.NewOperatorUpdate(&cfg)
		updater.Package = "test"
		updater.Version = "10-4"
		ext, err := updater.Run(context.TODO())

		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("parsing version"))
		Expect(ext).To(BeNil())
	})

	It("fails updating operator", func() {
		testExt := buildExtension(
			"test",
			withSourceType(olmv1.SourceTypeCatalog),
			withConstraintPolicy(string(olmv1.UpgradeConstraintPolicyCatalogProvided)),
		)
		cfg := setupEnv(testExt)

		ctx, cancel := context.WithCancel(context.TODO())
		cancel()

		updater := internalaction.NewOperatorUpdate(&cfg)
		updater.Package = "test"
		updater.Version = "10.0.4"
		updater.Channels = []string{"a", "b"}
		updater.Labels = map[string]string{"c": "d"}
		updater.UpgradeConstraintPolicy = string(olmv1.UpgradeConstraintPolicySelfCertified)
		ext, err := updater.Run(ctx)

		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("timed out"))
		Expect(ext).To(BeNil())
	})

	It("successfully updates operator", func() {
		testExt := buildExtension(
			"test",
			withSourceType(olmv1.SourceTypeCatalog),
			withConstraintPolicy(string(olmv1.UpgradeConstraintPolicyCatalogProvided)),
		)
		cfg := setupEnv(testExt, buildExtension("test2"), buildExtension("test3"))

		go func() {
			Eventually(updateOperatorConditionStatus("test", cfg.Client, olmv1.TypeInstalled, metav1.ConditionTrue)).
				WithTimeout(5 * time.Second).WithPolling(200 * time.Second).
				Should(Succeed())
		}()

		updater := internalaction.NewOperatorUpdate(&cfg)
		updater.Package = "test"
		updater.Version = "10.0.4"
		updater.Channels = []string{"a", "b"}
		updater.Labels = map[string]string{"c": "d"}
		updater.UpgradeConstraintPolicy = string(olmv1.UpgradeConstraintPolicySelfCertified)
		ext, err := updater.Run(context.TODO())

		Expect(err).To(BeNil())
		Expect(ext).NotTo(BeNil())
		Expect(ext.Spec.Source.Catalog.Version).To(Equal(updater.Version))
		Expect(maps.Equal(ext.Labels, updater.Labels)).To(BeTrue())
		Expect(ext.Spec.Source.Catalog.Channels).To(ContainElements(updater.Channels))
		Expect(ext.Spec.Source.Catalog.UpgradeConstraintPolicy).
			To(Equal(olmv1.UpgradeConstraintPolicy(updater.UpgradeConstraintPolicy)))

		// also verify that other objects were not updated
		validateNonUpdatedExtensions(cfg.Client, "test")
	})
})

func validateNonUpdatedExtensions(c client.Client, exceptName string) {
	var extList olmv1.ClusterExtensionList
	err := c.List(context.TODO(), &extList)
	Expect(err).To(BeNil())

	for _, ext := range extList.Items {
		if ext.Name == exceptName {
			continue
		}

		Expect(ext.Spec.Source.Catalog.Version).To(BeEmpty())
		Expect(ext.Labels).To(BeEmpty())
		Expect(ext.Spec.Source.Catalog.Channels).To(BeEmpty())
		Expect(ext.Spec.Source.Catalog.UpgradeConstraintPolicy).To(BeEmpty())
	}
}
