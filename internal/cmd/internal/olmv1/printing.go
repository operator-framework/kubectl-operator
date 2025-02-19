package olmv1

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"text/tabwriter"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"

	catalogdv1 "github.com/operator-framework/catalogd/api/v1"
	olmv1 "github.com/operator-framework/operator-controller/api/v1"
)

func printFormattedOperators(extensions ...olmv1.ClusterExtension) {
	tw := tabwriter.NewWriter(os.Stdout, 3, 4, 2, ' ', 0)
	_, _ = fmt.Fprint(tw, "NAME\tINSTALLED BUNDLE\tVERSION\tSOURCE TYPE\tINSTALLED\tPROGRESSING\tAGE\n")

	// sort by name
	slices.SortFunc(extensions, func(a, b olmv1.ClusterExtension) int {
		return cmp.Compare(a.Name, b.Name)
	})

	for _, ext := range extensions {
		age := time.Since(ext.CreationTimestamp.Time)
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			ext.Name,
			ext.Status.Install.Bundle.Name,
			ext.Status.Install.Bundle.Version,
			ext.Spec.Source.SourceType,
			status(ext.Status.Conditions, olmv1.TypeInstalled),
			status(ext.Status.Conditions, olmv1.TypeProgressing),
			duration.HumanDuration(age),
		)
	}
	_ = tw.Flush()
}

func printFormattedCatalogs(catalogs ...catalogdv1.ClusterCatalog) {
	tw := tabwriter.NewWriter(os.Stdout, 3, 4, 2, ' ', 0)
	_, _ = fmt.Fprint(tw, "NAME\tAVAILABILITY\tPRIORITY\tLASTUNPACKED\tSERVING\tAGE\n")

	// sort by availability first, then by priority and name
	slices.SortFunc(catalogs, func(a, b catalogdv1.ClusterCatalog) int {
		return cmp.Or(
			cmp.Compare(a.Spec.AvailabilityMode, a.Spec.AvailabilityMode),
			cmp.Compare(a.Spec.Priority, b.Spec.Priority),
			cmp.Compare(a.Name, b.Name),
		)
	})

	for _, cat := range catalogs {
		age := time.Since(cat.CreationTimestamp.Time)
		lastUnpacked := time.Since(cat.Status.LastUnpacked.Time)
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%s\t%s\n",
			cat.Name,
			string(cat.Spec.AvailabilityMode),
			cat.Spec.Priority,
			duration.HumanDuration(lastUnpacked),
			status(cat.Status.Conditions, catalogdv1.TypeServing),
			duration.HumanDuration(age),
		)
	}
	_ = tw.Flush()
}

func status(conditions []metav1.Condition, typ string) string {
	for _, condition := range conditions {
		if condition.Type == typ {
			return string(condition.Status)
		}
	}

	return "Unknown"
}
