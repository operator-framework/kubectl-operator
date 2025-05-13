package olmv1

import (
	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

// NewCatalogUpdateCmd allows updating a selected clustercatalog
func NewCatalogUpdateCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewCatalogUpdate(cfg)
	i.Logf = log.Printf

	var priority int32
	var pollInterval int
	var labels map[string]string

	cmd := &cobra.Command{
		Use:   "catalog <catalog>",
		Short: "Update a catalog",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			i.CatalogName = args[0]
			if cmd.Flags().Changed("priority") {
				i.Priority = &priority
			}
			if cmd.Flags().Changed("source-poll-interval-minutes") {
				i.PollIntervalMinutes = &pollInterval
			}
			if cmd.Flags().Changed("labels") {
				i.Labels = labels
			}
			_, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to update catalog: %v", err)
			}
			log.Printf("catalog %q updated", i.CatalogName)
		},
	}
	cmd.Flags().Int32Var(&priority, "priority", 0, "priority determines the likelihood of a catalog being selected in conflict scenarios")
	cmd.Flags().IntVar(&pollInterval, "source-poll-interval-minutes", 5, "catalog source polling interval [in minutes]. Set to 0 or -1 to remove the polling interval.")
	cmd.Flags().StringToStringVar(&labels, "labels", map[string]string{}, "labels that will be added to the catalog")
	cmd.Flags().StringVar(&i.AvailabilityMode, "availability-mode", "", "available means that the catalog should be active and serving data")
	cmd.Flags().StringVar(&i.ImageRef, "image", "", "Image reference for the catalog source. Leave unset to retain the current image.")
	cmd.Flags().BoolVar(&i.IgnoreUnset, "ignore-unset", true, "when enabled, any unset flag value will not be changed. Disabling means that for each unset value a default will be used instead")

	return cmd
}
