package olmv1

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

// NewCatalogDeleteCmd allows deleting either a single or all
// existing catalogs
func NewCatalogDeleteCmd(cfg *action.Configuration) *cobra.Command {
	d := v1action.NewCatalogDelete(cfg)
	d.Logf = log.Printf

	cmd := &cobra.Command{
		Use:     "catalog [catalog_name]",
		Aliases: []string{"catalogs [catalog_name]"},
		Args:    cobra.RangeArgs(0, 1),
		Short:   "Delete either a single or all of the existing catalogs",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				catalogs, err := d.Run(cmd.Context())
				if err != nil {
					log.Fatalf("failed deleting catalogs: %v", err)
				}
				for _, catalog := range catalogs {
					log.Printf("catalog %q deleted", catalog)
				}

				return
			}

			d.CatalogName = args[0]
			if _, err := d.Run(cmd.Context()); err != nil {
				log.Fatalf("failed to delete catalog %q: %v", d.CatalogName, err)
			}
			log.Printf("catalog %q deleted", d.CatalogName)
		},
	}
	bindCatalogDeleteFlags(cmd.Flags(), d)

	return cmd
}

func bindCatalogDeleteFlags(fs *pflag.FlagSet, d *v1action.CatalogDelete) {
	fs.BoolVar(&d.DeleteAll, "all", false, "delete all catalogs")
	fs.StringVar(&d.DryRun, "dry-run", "", "Display the object that would be sent on a request without applying it. One of: (All)")
	fs.StringVarP(&d.Output, "output", "o", "", "Output format for dry-run manifests. One of: (json, yaml)")
}
