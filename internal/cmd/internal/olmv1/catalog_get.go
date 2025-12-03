package olmv1

import (
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime/schema"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

// NewCatalogInstalledGetCmd handles get commands in the form of:
// catalog(s) [catalog_name] - this will either list all the installed catalogs
// if no catalog_name has been provided or display the details of the specific
// one otherwise
func NewCatalogInstalledGetCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewCatalogInstalledGet(cfg)
	i.Logf = log.Printf
	var opts getOptions

	cmd := &cobra.Command{
		Use:     "catalog [catalog_name]",
		Aliases: []string{"catalogs [catalog_name]"},
		Args:    cobra.RangeArgs(0, 1),
		Short:   "Display one or many installed catalogs",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 1 {
				i.CatalogName = args[0]
			}
			if err := opts.validate(); err != nil {
				log.Fatalf("failed to parse flags: %v", err)
			}
			i.Selector = opts.ParsedSelector
			installedCatalogs, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed getting installed catalog(s): %v", err)
			}

			for i := range installedCatalogs {
				installedCatalogs[i].GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: olmv1.GroupVersion.Group,
					Version: olmv1.GroupVersion.Version, Kind: "ClusterCatalog"})
			}
			printFormattedCatalogs(opts.Output, installedCatalogs...)
		},
	}
	bindGetFlags(cmd.Flags(), &opts)

	return cmd
}
