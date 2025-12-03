package olmv1

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/schema"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type catalogDeleteOptions struct {
	dryRunOptions
}

// NewCatalogDeleteCmd deletes either a specific catalog by name
// or all catalogs on cluster.
func NewCatalogDeleteCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewCatalogDelete(cfg)
	i.Logf = log.Printf
	var opts catalogDeleteOptions

	cmd := &cobra.Command{
		Use:     "catalog [catalog_name]",
		Aliases: []string{"catalogs [catalog_name]"},
		Args:    cobra.RangeArgs(0, 1),
		Short:   "Delete either a single or all of the existing catalogs",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				if i.DeleteAll {
					log.Fatalf("failed to delete catalog: cannot specify both --all and a catalog name")
				}
				i.CatalogName = args[0]
			}
			if err := opts.validate(); err != nil {
				log.Fatalf("failed to parse flags: %v", err)
			}
			i.DryRun = opts.DryRun
			i.Output = opts.Output
			catalogs, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to delete catalog(s): %v", err)
			}
			if len(i.DryRun) == 0 {
				for _, c := range catalogs {
					log.Printf("catalog %q deleted", c.Name)
				}
				return
			}
			if len(i.Output) == 0 {
				for _, c := range catalogs {
					log.Printf("catalog %q deleted (dry run)", c.Name)
				}
				return
			}

			for _, c := range catalogs {
				c.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: olmv1.GroupVersion.Group,
					Version: olmv1.GroupVersion.Version, Kind: "ClusterCatalog"})
			}
			printFormattedCatalogs(i.Output, catalogs...)
		},
	}
	bindCatalogDeleteFlags(cmd.Flags(), i)
	bindDryRunFlags(cmd.Flags(), &opts.dryRunOptions)

	return cmd
}

func bindCatalogDeleteFlags(fs *pflag.FlagSet, d *v1action.CatalogDelete) {
	fs.BoolVarP(&d.DeleteAll, "all", "a", false, "delete all catalogs")
}
