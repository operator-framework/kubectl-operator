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
	var opts getOptions

	cmd := &cobra.Command{
		Use:     "catalog",
		Aliases: []string{"catalogs"},
		Short:   "Search catalogs for installable packages matching parameters",
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.validate(); err != nil {
				log.Fatalf("failed to parse flags: %v", err)
			}
			i.Selector = opts.ParsedSelector
			catalogContents, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed querying catalog(s): %v", err)
			}
			switch opts.Output {
			case "":
				printFormattedDeclCfg(os.Stdout, catalogContents, i.ListVersions)
			case "json":
				printDeclCfgJSON(os.Stdout, catalogContents)
			case "yaml":
				printDeclCfgYAML(os.Stdout, catalogContents)
			default:
				log.Fatalf("unsupported output format %q: allowed formats are (json|yaml)", opts.Output)
			}
		},
	}
	bindCatalogSearchFlags(cmd.Flags(), i)
	bindGetFlags(cmd.Flags(), &opts)

	return cmd
}

func bindCatalogSearchFlags(fs *pflag.FlagSet, i *v1action.CatalogSearch) {
	fs.StringVar(&i.CatalogName, "catalog", "", "name of the catalog to search. If not provided, all available catalogs are searched.")
	fs.BoolVar(&i.ListVersions, "list-versions", false, "list all versions available for each package.")
	fs.StringVar(&i.Package, "package", "", "search for package by name. If empty, all available packages will be listed.")
	fs.StringVar(&i.CatalogdNamespace, "catalogd-namespace", "olmv1-system", "namespace for the catalogd controller.")
	fs.StringVar(&i.Timeout, "timeout", "5m", "timeout for fetching catalog contents.")
}
