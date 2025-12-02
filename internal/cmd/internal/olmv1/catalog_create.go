package olmv1

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/errors"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type catalogCreateOptions struct {
	dryRunOptions
	mutableCatalogOptions
}

// NewCatalogCreateCmd creates a new catalog, requiring a minimum
// of a name for the new catalog and a source image reference
func NewCatalogCreateCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewCatalogCreate(cfg)
	i.Logf = log.Printf
	var opts catalogCreateOptions

	cmd := &cobra.Command{
		Use:     "catalog <catalog_name> <image_source_ref>",
		Aliases: []string{"catalogs <catalog_name> <image_source_ref>"},
		Args:    cobra.ExactArgs(2),
		Short:   "Create a new catalog",
		Run: func(cmd *cobra.Command, args []string) {
			i.CatalogName = args[0]
			i.ImageSourceRef = args[1]
			opts.Image = i.ImageSourceRef
			if err := opts.validate(); err != nil {
				log.Fatalf("failed to parse flags: %w", err)
			}
			i.DryRun = opts.DryRun
			i.Output = opts.Output
			i.AvailabilityMode = opts.AvailabilityMode
			i.Priority = opts.Priority
			i.Labels = opts.Labels
			i.PollIntervalMinutes = opts.PollIntervalMinutes
			catalogObj, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to create catalog %q: %w", i.CatalogName, err)
			}
			if len(i.DryRun) == 0 {
				log.Printf("catalog %q created", i.CatalogName)
				return
			}
			if len(i.Output) == 0 {
				log.Printf("catalog %q created (dry run)", i.CatalogName)
				return
			}

			catalogObj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: olmv1.GroupVersion.Group,
				Version: olmv1.GroupVersion.Version, Kind: "ClusterCatalog"})
			printFormattedCatalogs(i.Output, *catalogObj)
		},
	}
	bindMutableCatalogFlags(cmd.Flags(), &opts.mutableCatalogOptions)
	bindCatalogCreateFlags(cmd.Flags(), i)
	bindDryRunFlags(cmd.Flags(), &opts.dryRunOptions)

	return cmd
}

func bindCatalogCreateFlags(fs *pflag.FlagSet, i *v1action.CatalogCreate) {
	fs.DurationVar(&i.CleanupTimeout, "cleanup-timeout", time.Minute, "the amount of time to wait before cancelling cleanup after a failed creation attempt.")
}

func (o *catalogCreateOptions) validate() error {
	var errs []error
	if err := o.dryRunOptions.validate(); err != nil {
		errs = append(errs, err)
	}
	if err := o.mutableCatalogOptions.validate(); err != nil {
		errs = append(errs, err)
	}
	return errors.NewAggregate(errs)
}
