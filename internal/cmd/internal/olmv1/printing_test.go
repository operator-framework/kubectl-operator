package olmv1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"
)

var _ = Describe("SortCatalogs", func() {
	It("sorts catalogs in correct order", func() {
		catalogs := []olmv1.ClusterCatalog{
			newClusterCatalog("cat-unavailable-0", olmv1.AvailabilityModeUnavailable, 0),
			newClusterCatalog("cat-unavailable-1", olmv1.AvailabilityModeUnavailable, 1),
			newClusterCatalog("cat-available-0", olmv1.AvailabilityModeAvailable, 0),
			newClusterCatalog("cat-available-1", olmv1.AvailabilityModeAvailable, 1),
		}
		sortCatalogs(catalogs)

		Expect(catalogs[0].Name).To(Equal("cat-available-1"))
		Expect(catalogs[1].Name).To(Equal("cat-available-0"))
		Expect(catalogs[2].Name).To(Equal("cat-unavailable-1"))
		Expect(catalogs[3].Name).To(Equal("cat-unavailable-0"))
	})
})

var _ = Describe("SortExtensions", func() {
	It("sorts extensions in correct order", func() {
		extensions := []olmv1.ClusterExtension{
			newClusterExtension("op-1", "1.0.0"),
			newClusterExtension("op-1", "1.0.1"),
			newClusterExtension("op-1", "1.0.1-rc4"),
			newClusterExtension("op-1", "1.0.1-rc2"),
			newClusterExtension("op-2", "2.0.0"),
		}
		sortExtensions(extensions)

		Expect(extensions[0].Status.Install.Bundle.Version).To(Equal("1.0.1"))
		Expect(extensions[1].Status.Install.Bundle.Version).To(Equal("1.0.1-rc4"))
		Expect(extensions[2].Status.Install.Bundle.Version).To(Equal("1.0.1-rc2"))
		Expect(extensions[3].Status.Install.Bundle.Version).To(Equal("1.0.0"))
		Expect(extensions[4].Status.Install.Bundle.Version).To(Equal("2.0.0"))
	})
})

func newClusterCatalog(name string, availabilityMode olmv1.AvailabilityMode, priority int32) olmv1.ClusterCatalog {
	return olmv1.ClusterCatalog{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       olmv1.ClusterCatalogSpec{AvailabilityMode: availabilityMode, Priority: priority},
	}
}

func newClusterExtension(name, version string) olmv1.ClusterExtension {
	return olmv1.ClusterExtension{
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
