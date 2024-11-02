package cmd

import (
	"cmp"
	"encoding/json"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/action/v1"
	"github.com/operator-framework/kubectl-operator/pkg/action"
	"github.com/operator-framework/operator-registry/alpha/declcfg"
	"github.com/operator-framework/operator-registry/alpha/property"
	"github.com/spf13/cobra"
	"iter"
	"k8s.io/apimachinery/pkg/util/sets"
	"maps"
	"os"
	"slices"
	"strings"
	"sync"
	"text/tabwriter"
)

func newCatalogContentCmd(cfg *action.Configuration) *cobra.Command {
	cc := internalaction.NewCatalogContent(cfg)

	var (
		pkgName      string
		versionRange string
		channels     []string
	)

	cmd := &cobra.Command{
		Use:   "content <catalog_name>",
		Short: "View cluster catalog content",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var constraints *semver.Constraints
			if versionRange != "" {
				var err error
				constraints, err = semver.NewConstraint(versionRange)
				if err != nil {
					log.Fatalf("invalid version range %q: %v", versionRange, err)
				}
			}

			cc.CatalogName = args[0]
			var (
				bundles = map[string]map[string]*bundleMetadata{}
				mu      sync.Mutex
			)
			cc.WalkMetas = func(meta *declcfg.Meta, err error) error {
				if err != nil {
					return err
				}
				if pkgName != "" && pkgName != meta.Package {
					return nil
				}
				if meta.Schema == declcfg.SchemaChannel {
					var ch declcfg.Channel
					if err := json.Unmarshal(meta.Blob, &ch); err != nil {
						return err
					}
					for _, entry := range ch.Entries {
						mu.Lock()
						pkg, ok := bundles[ch.Package]
						if !ok {
							pkg = map[string]*bundleMetadata{}
						}
						b, ok := pkg[entry.Name]
						if !ok {
							b = &bundleMetadata{
								Name:     entry.Name,
								Channels: sets.New[string](),
							}
							pkg[entry.Name] = b
						}
						b.Channels.Insert(meta.Name)
						bundles[ch.Package] = pkg
						mu.Unlock()
					}
					return nil
				}
				if meta.Schema == declcfg.SchemaBundle {
					bundle := declcfg.Bundle{}
					if err := json.Unmarshal(meta.Blob, &bundle); err != nil {
						return err
					}
					bundleVersion, err := getBundleVersion(bundle)
					if err != nil {
						return err
					}
					mu.Lock()
					pkg, ok := bundles[bundle.Package]
					if !ok {
						pkg = map[string]*bundleMetadata{}
					}
					b, ok := pkg[bundle.Name]
					if !ok {
						b = &bundleMetadata{
							Name:     meta.Name,
							Channels: sets.New[string](),
						}
						pkg[bundle.Name] = b
					}
					b.Version = bundleVersion
					bundles[bundle.Package] = pkg
					mu.Unlock()
				}
				return nil
			}

			if err := cc.Run(cmd.Context()); err != nil {
				log.Fatalf("failed to get content for catalog %q: %v", cc.CatalogName, err)
			}

			if len(bundles) == 0 {
				log.Print("No resources found")
				return
			}

			tw := tabwriter.NewWriter(os.Stdout, 3, 4, 2, ' ', 0)
			_, _ = fmt.Fprintf(tw, "PACKAGE\tVERSION\tCHANNELS\t\n")

			pkgNames := collect(maps.Keys(bundles))
			slices.SortFunc(pkgNames, func(a, b string) int {
				return cmp.Compare(a, b)
			})
			for _, pkg := range pkgNames {
				bundleNames := collect(maps.Keys(bundles[pkg]))
				slices.SortFunc(bundleNames, func(a, b string) int {
					return -bundles[pkg][a].Version.Compare(bundles[pkg][b].Version)
				})
				for _, bundleName := range bundleNames {
					bundle := bundles[pkg][bundleName]
					if constraints != nil && !constraints.Check(bundle.Version) {
						continue
					}
					if len(channels) > 0 && !bundle.Channels.HasAny(channels...) {
						continue
					}
					_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\n", pkg, bundle.Version, strings.Join(sets.List(bundle.Channels), ","))
				}
			}
			_ = tw.Flush()
		},
	}
	cmd.Flags().StringVarP(&pkgName, "package", "p", "", "package name to filter")
	cmd.Flags().StringVarP(&versionRange, "version", "v", "", "version range to filter")
	cmd.Flags().StringSliceVarP(&channels, "channels", "c", []string{}, "channels to filter")
	return cmd
}

func collect[V any](i iter.Seq[V]) []V {
	var out []V
	for v := range i {
		out = append(out, v)
	}
	return out
}

type bundleMetadata struct {
	Name     string
	Version  *semver.Version
	Channels sets.Set[string]
}

func getBundleMetadata(b declcfg.Bundle) (string, string, error) {
	version, err := getBundleVersion(b)
	if err != nil {
		return "", "", err
	}
	return b.Package, version.String(), nil
}

func getBundleVersion(b declcfg.Bundle) (*semver.Version, error) {
	packageValue := json.RawMessage{}
	for _, p := range b.Properties {
		if p.Type == property.TypePackage {
			packageValue = p.Value
			break
		}
	}
	if len(packageValue) == 0 {
		return nil, fmt.Errorf("no package property found")
	}
	packageProp := property.Package{}
	if err := json.Unmarshal(packageValue, &packageProp); err != nil {
		return nil, err
	}
	return semver.NewVersion(packageProp.Version)
}
