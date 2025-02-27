package olmv1

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

// NewCatalogCreateCmd allows creating a new catalog
func NewCatalogCreateCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewCatalogCreate(cfg.Client)
	i.Logf = log.Printf

	cmd := &cobra.Command{
		Use:     "catalog <catalog_name> <image_source_ref>",
		Aliases: []string{"catalogs <catalog_name> <image_source_ref>"},
		Args:    cobra.ExactArgs(2),
		Short:   "Create a new catalog",
		Run: func(cmd *cobra.Command, args []string) {
			i.CatalogName = args[0]
			i.ImageSourceRef = args[1]

			if err := i.Run(cmd.Context()); err != nil {
				log.Fatalf("failed to create catalog %q: %v", i.CatalogName, err)
			}
			log.Printf("catalog %q created", i.CatalogName)
		},
	}
	bindCatalogCreateFlags(cmd.Flags(), i)

	return cmd
}

func bindCatalogCreateFlags(fs *pflag.FlagSet, i *v1action.CatalogCreate) {
	fs.Int32Var(&i.Priority, "priority", 0, "priority determines the likelihood of a catalog being selected in conflict scenarios")
	fs.BoolVar(&i.Available, "available", true, "true means that the catalog should be active and serving data")
	fs.IntVar(&i.PollIntervalMinutes, "source-poll-interval-minutes", 10, "catalog source polling interval [in minutes]")
	fs.StringToStringVar(&i.Labels, "labels", map[string]string{}, "labels that will be added to the catalog")
	fs.DurationVar(&i.CleanupTimeout, "cleanup-timeout", time.Minute, "the amount of time to wait before cancelling cleanup after a failed creation attempt")
}
