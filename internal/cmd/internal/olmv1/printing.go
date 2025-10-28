package olmv1

import (
	"cmp"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/blang/semver/v4"
	"github.com/operator-framework/operator-registry/alpha/declcfg"
	"github.com/operator-framework/operator-registry/alpha/property"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/apimachinery/pkg/util/json"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"
	"github.com/operator-framework/operator-registry/alpha/declcfg"
	"github.com/operator-framework/operator-registry/alpha/property"
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

func printFormattedDeclCfg(w io.Writer, catalogDcfg map[string]*declcfg.DeclarativeConfig, listVersions bool) {
	var printedHeaders bool
	tw := tabwriter.NewWriter(w, 3, 4, 2, ' ', 0)
	sortedCatalogs := []string{}
	for catalogName := range catalogDcfg {
		sortedCatalogs = append(sortedCatalogs, catalogName)
	}
	sort.Strings(sortedCatalogs)
	for _, catalogName := range sortedCatalogs {
		dcfg := catalogDcfg[catalogName]
		type dcfgPrintMeta struct {
			provider string
			channels []string
			versions []semver.Version
		}
		pkgProviders := map[string]*dcfgPrintMeta{}
		sort.SliceStable(dcfg.Packages, func(i, j int) bool {
			return dcfg.Packages[i].Name < dcfg.Packages[j].Name
		})

		if listVersions {
			for _, b := range dcfg.Bundles {
				if pkgProviders[b.Package] == nil {
					pkgProviders[b.Package] = &dcfgPrintMeta{
						versions: []semver.Version{},
						provider: getCSVProvider(&b),
					}
				}
				bundleVersion, err := getBundleVersion(&b)
				if err == nil {
					pkgProviders[b.Package].versions = append(pkgProviders[b.Package].versions, bundleVersion)
				}
			}
		} else {
			for _, c := range dcfg.Channels {
				if pkgProviders[c.Package] == nil {
					pkgProviders[c.Package] = &dcfgPrintMeta{channels: []string{}}
				}
				pkgProviders[c.Package].channels = append(pkgProviders[c.Package].channels, c.Name)
			}
		}

		for _, p := range dcfg.Packages {
			if listVersions {
				sort.SliceStable(pkgProviders[p.Name].versions, func(i, j int) bool {
					return pkgProviders[p.Name].versions[i].GT(pkgProviders[p.Name].versions[j])
				})
				for _, v := range pkgProviders[p.Name].versions {
					if !printedHeaders {
						_, _ = fmt.Fprint(tw, "PACKAGE\tCATALOG\tPROVIDER\tVERSION\n")
						printedHeaders = true
					}
					_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
						p.Name,
						catalogName,
						pkgProviders[p.Name].provider,
						v)
				}
			} else {
				sort.Strings(pkgProviders[p.Name].channels)
				if !printedHeaders {
					_, _ = fmt.Fprint(tw, "PACKAGE\tCATALOG\tPROVIDER\tCHANNELS\n")
					printedHeaders = true
				}
				_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
					p.Name,
					catalogName,
					pkgProviders[p.Name].provider,
					strings.Join(pkgProviders[p.Name].channels, ","))
			}
		}
	}
	if !printedHeaders {
		_, _ = fmt.Fprint(tw, "No resources found.\n")
	}
	_ = tw.Flush()
}

func getBundleVersion(bundle *declcfg.Bundle) (semver.Version, error) {
	for _, p := range bundle.Properties {
		if p.Type == property.TypePackage {
			var pkgProp property.Package
			if err := json.Unmarshal(p.Value, &pkgProp); err == nil && len(pkgProp.Version) > 0 {
				parsedVersion, err := semver.Parse(pkgProp.Version)
				if err != nil {
					return semver.Version{}, err
				}
				return parsedVersion, nil
			}
		}
	}
	return semver.Version{}, fmt.Errorf("no version property")
}

func getCSVProvider(bundle *declcfg.Bundle) string {
	for _, csvProp := range bundle.Properties {
		if csvProp.Type == property.TypeCSVMetadata {
			var pkgProp property.CSVMetadata
			if err := json.Unmarshal(csvProp.Value, &pkgProp); err == nil && len(pkgProp.Provider.Name) > 0 {
				return pkgProp.Provider.Name
			}
		}
	}
	return ""
}

func printDeclCfgJSON(w io.Writer, catalogDcfg map[string]*declcfg.DeclarativeConfig) {
	for _, dcfg := range catalogDcfg {
		_ = declcfg.WriteJSON(*dcfg, w)
	}
}

func printDeclCfgYAML(w io.Writer, catalogDcfg map[string]*declcfg.DeclarativeConfig) {
	for _, dcfg := range catalogDcfg {
		_ = declcfg.WriteYAML(*dcfg, w)
		_, _ = w.Write([]byte("---\n"))
	}
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
