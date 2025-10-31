package olmv1

import (
	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewCatalogUpdateCmd allows updating a selected clustercatalog
func NewCatalogUpdateCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewCatalogUpdate(cfg)
	i.Logf = log.Printf

	var priority int32
	var pollInterval int
	var labels map[string]string
	var available string

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
			if len(available) > 0 {
				switch available {
				case "true":
					i.AvailabilityMode = "Available"
				case "false":
					i.AvailabilityMode = "Unavailable"
				default:
					log.Fatalf("invalid value for `--available` %s; must be one of (true, false)\n", available)
				}
			}
			if len(i.DryRun) > 0 && i.DryRun != v1action.DryRunAll {
				log.Fatalf("invalid value for `--dry-run` %s, must be one of (%s)\n", i.DryRun, v1action.DryRunAll)
			}
			catalogObj, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to update catalog: %v", err)
			}

			if len(i.DryRun) == 0 {
				log.Printf("catalog %q updated\n", i.CatalogName)
				return
			}
			if len(i.Output) == 0 {
				log.Printf("catalog %q updated (dry run)\n", i.CatalogName)
				return
			}

			catalogObj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: olmv1.GroupVersion.Group,
				Version: olmv1.GroupVersion.Version, Kind: "ClusterCatalog"})
			printFormattedCatalogs(i.Output, *catalogObj)
		},
	}
	cmd.Flags().Int32Var(&priority, "priority", 0, "priority determines the likelihood of a catalog being selected in conflict scenarios")
	cmd.Flags().StringVar(&available, "available", "", "determines whether a catalog should be active and serving data. default: true, meaning new catalogs serve their contents by default.")
	cmd.Flags().IntVar(&pollInterval, "source-poll-interval-minutes", 5, "catalog source polling interval [in minutes]. Set to 0 or -1 to remove the polling interval.")
	cmd.Flags().StringToStringVar(&labels, "labels", map[string]string{}, "labels that will be added to the catalog")
	cmd.Flags().StringVar(&i.ImageRef, "image", "", "Image reference for the catalog source. Leave unset to retain the current image.")
	cmd.Flags().BoolVar(&i.IgnoreUnset, "ignore-unset", true, "when enabled, any unset flag value will not be changed. Disabling means that for each unset value a default will be used instead")
	cmd.Flags().StringVar(&i.DryRun, "dry-run", "", "display the object that would be sent on a request without applying it if non-empty. One of: (All)")
	cmd.Flags().StringVarP(&i.Output, "output", "o", "", "output format for dry-run manifests. One of: (json, yaml)")

	return cmd
}
