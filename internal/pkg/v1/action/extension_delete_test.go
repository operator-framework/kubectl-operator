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

var _ = Describe("ExtensionDelete", func() {
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

	It("fails because of both extension name and --all specifier being present", func() {
		cfg := setupEnv(setupTestExtensions(2)...)

		deleter := internalaction.NewExtensionDelete(&cfg)
		deleter.ExtensionName = "foo"
		deleter.DeleteAll = true
		extensions, err := deleter.Run(context.TODO())
		Expect(err).NotTo(BeNil())
		Expect(extensions).To(BeEmpty())

		validateExistingExtensions(cfg.Client, []string{"ext1", "ext2"})
	})

	It("fails deleting a non-existent extensions", func() {
		cfg := setupEnv(setupTestExtensions(2)...)

		deleter := internalaction.NewExtensionDelete(&cfg)
		deleter.ExtensionName = "does-not-exist"
		extensions, err := deleter.Run(context.TODO())
		Expect(err).NotTo(BeNil())
		Expect(extensions).To(HaveLen(1))
		validateExtensionList(extensions, []string{deleter.ExtensionName})

		validateExistingExtensions(cfg.Client, []string{"ext1", "ext2"})
	})

	It("successfully deletes an existing extension", func() {
		cfg := setupEnv(setupTestExtensions(3)...)

		deleter := internalaction.NewExtensionDelete(&cfg)
		deleter.ExtensionName = "ext2"
		_, err := deleter.Run(context.TODO())
		Expect(err).To(BeNil())

		validateExistingExtensions(cfg.Client, []string{"ext1", "ext3"})
	})

	It("fails deleting all extensions because there are none", func() {
		cfg := setupEnv()

		deleter := internalaction.NewExtensionDelete(&cfg)
		deleter.DeleteAll = true
		extensions, err := deleter.Run(context.TODO())
		Expect(err).NotTo(BeNil())
		Expect(extensions).To(BeEmpty())

		validateExistingExtensions(cfg.Client, []string{})
	})

	It("successfully deletes all extensions", func() {
		cfg := setupEnv(setupTestExtensions(3)...)

		deleter := internalaction.NewExtensionDelete(&cfg)
		deleter.DeleteAll = true
		extensions, err := deleter.Run(context.TODO())
		Expect(err).To(BeNil())
		validateExtensionList(extensions, []string{"ext1", "ext2", "ext3"})

		validateExistingExtensions(cfg.Client, []string{})
	})
})

// validateExistingExtensions compares the names of the existing extensions with the wanted names
// and ensures that all wanted names are present in the existing extensions
func validateExistingExtensions(c client.Client, wantedNames []string) {
	var extensionList olmv1.ClusterExtensionList
	err := c.List(context.TODO(), &extensionList)
	Expect(err).To(BeNil())

	extensions := extensionList.Items
	Expect(extensions).To(HaveLen(len(wantedNames)))
	validateExtensionList(extensions, wantedNames)
}

func validateExtensionList(extensions []olmv1.ClusterExtension, wantedNames []string) {
	for _, wantedName := range wantedNames {
		Expect(slices.ContainsFunc(extensions, func(ext olmv1.ClusterExtension) bool {
			return ext.Name == wantedName
		})).To(BeTrue())
	}
}
