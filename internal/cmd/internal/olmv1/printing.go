package olmv1

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"text/tabwriter"
	"time"

	"github.com/blang/semver/v4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"
)

func printFormattedExtensions(extensions ...olmv1.ClusterExtension) {
	tw := tabwriter.NewWriter(os.Stdout, 3, 4, 2, ' ', 0)
	_, _ = fmt.Fprint(tw, "NAME\tINSTALLED BUNDLE\tVERSION\tSOURCE TYPE\tINSTALLED\tPROGRESSING\tAGE\n")

	sortExtensions(extensions)
	for _, ext := range extensions {
		var bundleName, bundleVersion string
		if ext.Status.Install != nil {
			bundleName = ext.Status.Install.Bundle.Name
			bundleVersion = ext.Status.Install.Bundle.Version
		}
		age := time.Since(ext.CreationTimestamp.Time)
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			ext.Name,
			bundleName,
			bundleVersion,
			ext.Spec.Source.SourceType,
			status(ext.Status.Conditions, olmv1.TypeInstalled),
			status(ext.Status.Conditions, olmv1.TypeProgressing),
			duration.HumanDuration(age),
		)
	}
	_ = tw.Flush()
}

func printFormattedCatalogs(catalogs ...olmv1.ClusterCatalog) {
	tw := tabwriter.NewWriter(os.Stdout, 3, 4, 2, ' ', 0)
	_, _ = fmt.Fprint(tw, "NAME\tAVAILABILITY\tPRIORITY\tLASTUNPACKED\tSERVING\tAGE\n")

	sortCatalogs(catalogs)
	for _, cat := range catalogs {
		var lastUnpacked string
		if cat.Status.LastUnpacked != nil {
			duration.HumanDuration(time.Since(cat.Status.LastUnpacked.Time))
		}
		age := time.Since(cat.CreationTimestamp.Time)
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%s\t%s\n",
			cat.Name,
			string(cat.Spec.AvailabilityMode),
			cat.Spec.Priority,
			lastUnpacked,
			status(cat.Status.Conditions, olmv1.TypeServing),
			duration.HumanDuration(age),
		)
	}
	_ = tw.Flush()
}

// sortExtensions sorts extensions in place and uses the following sorting order:
// name (asc), version (desc)
func sortExtensions(extensions []olmv1.ClusterExtension) {
	slices.SortFunc(extensions, func(a, b olmv1.ClusterExtension) int {
		if a.Status.Install == nil || b.Status.Install == nil {
			return cmp.Compare(a.Name, b.Name)
		}
		return cmp.Or(
			cmp.Compare(a.Name, b.Name),
			-semver.MustParse(a.Status.Install.Bundle.Version).Compare(semver.MustParse(b.Status.Install.Bundle.Version)),
		)
	})
}

// sortCatalogs sorts catalogs in place and uses the following sorting order:
// availability (asc), priority (desc), name (asc)
func sortCatalogs(catalogs []olmv1.ClusterCatalog) {
	slices.SortFunc(catalogs, func(a, b olmv1.ClusterCatalog) int {
		return cmp.Or(
			cmp.Compare(a.Spec.AvailabilityMode, b.Spec.AvailabilityMode),
			-cmp.Compare(a.Spec.Priority, b.Spec.Priority),
			cmp.Compare(a.Name, b.Name),
		)
	})
}

func status(conditions []metav1.Condition, typ string) string {
	for _, condition := range conditions {
		if condition.Type == typ {
			return string(condition.Status)
		}
	}

	return "Unknown"
}
