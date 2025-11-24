package olmv1

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

// NewCatalogSearchCmd handles get commands in the form of:
// catalog(s) - this will either list all packages
// from available catalogs if no catalog has been provided.
// The results are restricted to only the contents of specific
// catalogs if either specified by name or label selector.
// results may also be restricted to the contents of a single
// package by name across one or more catalogs.
func NewCatalogSearchCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewCatalogSearch(cfg)
	i.Logf = log.Printf

	cmd := &cobra.Command{
		Use:     "catalog",
		Aliases: []string{"catalogs"},
		Args:    cobra.RangeArgs(0, 1),
		Short:   "Search catalogs for installable operators matching parameters",
		Run: func(cmd *cobra.Command, args []string) {
			catalogContents, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed querying catalog(s): %v", err)
			}
			switch i.OutputFormat {
			case "", "table":
				printFormattedDeclCfg(os.Stdout, catalogContents, i.ListVersions)
			case "json":
				printDeclCfgJSON(os.Stdout, catalogContents)
			case "yaml":
				printDeclCfgYAML(os.Stdout, catalogContents)
			default:
				log.Fatalf("unsupported output format %s: allwed formats are (json|yaml|table)", i.OutputFormat)
			}
		},
	}
	bindCatalogSearchFlags(cmd.Flags(), i)

	return cmd
}

func bindCatalogSearchFlags(fs *pflag.FlagSet, i *v1action.CatalogSearch) {
	fs.StringVar(&i.CatalogName, "catalog", "", "Catalog to search on. If not provided, all available catalogs are searched.")
	fs.StringVarP(&i.Selector, "selector", "l", "", "Selector (label query) to filter catalogs on, supports '=', '==', and '!='")
	fs.StringVarP(&i.OutputFormat, "output", "o", "", "output format. One of: (yaml|json)")
	fs.BoolVar(&i.ListVersions, "list-versions", false, "List all versions available for each package")
	fs.StringVar(&i.Package, "package", "", "Search for package by name. If empty, all available packages will be listed")
	fs.StringVar(&i.CatalogdNamespace, "catalogd-namespace", "olmv1-system", "Namespace for the catalogd controller")
	fs.StringVar(&i.Timeout, "timeout", "5m", "Timeout for fetching catalog contents")
	//	installable vs uninstallable, all versions, channels
	//	fs.StringVar(&i.showAll, "image", "", "Image reference for the catalog source. Leave unset to retain the current image.")
}
