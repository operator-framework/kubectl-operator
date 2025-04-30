package olmv1

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

// NewCatalogUpdateCmd allows updating a selected clustercatalog
func NewCatalogUpdateCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewCatalogUpdate(cfg)
	i.Logf = log.Printf

	cmd := &cobra.Command{
		Use:   "catalog <catalog>",
		Short: "Update a catalog",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			i.CatalogName = args[0]
			_, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to update catalog: %v", err)
			}
			log.Printf("catalog %q updated", i.CatalogName)
		},
	}
	bindCatalogUpdateFlags(cmd.Flags(), i)

	return cmd
}

func bindCatalogUpdateFlags(fs *pflag.FlagSet, i *v1action.CatalogUpdate) {
	fs.Int32Var(&i.Priority, "priority", 1, "priority determines the likelihood of a catalog being selected in conflict scenarios")
	fs.IntVar(&i.PollIntervalMinutes, "source-poll-interval-minutes", 5, "catalog source polling interval [in minutes]")
	fs.StringToStringVar(&i.Labels, "labels", map[string]string{}, "labels that will be added to the catalog")
	fs.StringVar(&i.AvailabilityMode, "availability-mode", "", "available means that the catalog should be active and serving data")
}
