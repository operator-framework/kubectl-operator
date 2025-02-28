package olmv1

import (
	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

// NewCatalogDeleteCmd allows deleting a specified, existing catalog
func NewCatalogDeleteCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewCatalogDelete(cfg)
	i.Logf = log.Printf

	cmd := &cobra.Command{
		Use:     "catalog <catalog_name>",
		Aliases: []string{"catalogs <catalog_name>"},
		Args:    cobra.ExactArgs(1),
		Short:   "Delete an existing catalog",
		Run: func(cmd *cobra.Command, args []string) {
			i.CatalogName = args[0]

			if err := i.Run(cmd.Context()); err != nil {
				log.Fatalf("failed to delete catalog %q: %v", i.CatalogName, err)
			}
			log.Printf("catalog %q deleted", i.CatalogName)
		},
	}

	return cmd
}
