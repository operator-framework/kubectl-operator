package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/action/v1"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

func newCatalogAddCmd(cfg *action.Configuration) *cobra.Command {
	a := internalaction.NewCatalogAdd(cfg)
	a.Logf = log.Printf

	cmd := &cobra.Command{
		Use:   "add <name> <catalog_image>",
		Short: "Add an cluster catalog",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			a.CatalogName = args[0]
			a.CatalogImage = args[1]

			cs, err := a.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to add clustercatalog: %v", err)
			}
			log.Printf("created clustercatalog %q\n", cs.Name)
		},
	}
	bindCatalogAddFlags(cmd.Flags(), a)

	return cmd
}

func bindCatalogAddFlags(fs *pflag.FlagSet, a *internalaction.CatalogAdd) {
	fs.Int32Var(&a.Priority, "priority", 0, "the priority of the catalog")
	fs.DurationVar(&a.PollInterval, "poll-interval", 10*time.Minute, "the poll interval to configure for the catalog, set to 0 to disable")
	fs.DurationVar(&a.CleanupTimeout, "cleanup-timeout", time.Minute, "the amount of time to wait before cancelling cleanup")
}
