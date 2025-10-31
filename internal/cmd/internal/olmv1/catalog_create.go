package olmv1

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewCatalogCreateCmd allows creating a new catalog
func NewCatalogCreateCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewCatalogCreate(cfg)
	i.Logf = log.Printf

	cmd := &cobra.Command{
		Use:     "catalog <catalog_name> <image_source_ref>",
		Aliases: []string{"catalogs <catalog_name> <image_source_ref>"},
		Args:    cobra.ExactArgs(2),
		Short:   "Create a new catalog",
		Run: func(cmd *cobra.Command, args []string) {
			i.CatalogName = args[0]
			i.ImageSourceRef = args[1]
			if len(i.DryRun) > 0 && i.DryRun != v1action.DryRunAll {
				log.Fatalf("invalid value for `--dry-run` %s, must be one of (%s)\n", i.DryRun, v1action.DryRunAll)
			}

			catalogObj, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to create catalog %q: %v\n", i.CatalogName, err)
			}
			if len(i.DryRun) == 0 {
				log.Printf("catalog %q created\n", i.CatalogName)
				return
			}
			if len(i.Output) == 0 {
				log.Printf("catalog %q created (dry run)\n", i.CatalogName)
				return
			}

			catalogObj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: olmv1.GroupVersion.Group,
				Version: olmv1.GroupVersion.Version, Kind: "ClusterCatalog"})
			printFormattedCatalogs(i.Output, *catalogObj)
		},
	}
	bindCatalogCreateFlags(cmd.Flags(), i)

	return cmd
}

func bindCatalogCreateFlags(fs *pflag.FlagSet, i *v1action.CatalogCreate) {
	fs.Int32Var(&i.Priority, "priority", 0, "priority determines the likelihood of a catalog being selected in conflict scenarios")
	fs.BoolVar(&i.Available, "available", true, "determines whether a catalog should be active and serving data. default: true, meaning new catalogs serve their contents by default.")
	fs.IntVar(&i.PollIntervalMinutes, "source-poll-interval-minutes", 10, "catalog source polling interval [in minutes]")
	fs.StringToStringVar(&i.Labels, "labels", map[string]string{}, "labels to add to the new catalog")
	fs.DurationVar(&i.CleanupTimeout, "cleanup-timeout", time.Minute, "the amount of time to wait before cancelling cleanup after a failed creation attempt")
	// sigs.k8s.io/controller-runtime/pkg/client supported dry-run values only.
	fs.StringVar(&i.DryRun, "dry-run", "", "Display the object that would be sent on a request without applying it if non-empty. One of: (All)")
	fs.StringVarP(&i.Output, "output", "o", "", "Output format for dry-run manifests. One of: (json, yaml)")
}
