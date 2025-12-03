package olmv1

import (
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/errors"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type catalogUpdateOptions struct {
	dryRunOptions
	mutableCatalogOptions
	updateDefaultFieldOptions
}

// NewCatalogUpdateCmd updates one or more mutable fields
// of a catalog specified by name
func NewCatalogUpdateCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewCatalogUpdate(cfg)
	i.Logf = log.Printf
	var opts catalogUpdateOptions

	cmd := &cobra.Command{
		Use:     "catalog <catalog_name>",
		Aliases: []string{"catalogs <catalog_name>"},
		Short:   "Update a catalog",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			i.CatalogName = args[0]
			if err := opts.validate(); err != nil {
				log.Fatalf("failed to parse flags: %v", err)
			}
			if cmd.Flags().Changed("priority") {
				i.Priority = &opts.Priority
			}
			if cmd.Flags().Changed("source-poll-interval-minutes") {
				i.PollIntervalMinutes = &opts.PollIntervalMinutes
			}
			if cmd.Flags().Changed("labels") {
				i.Labels = opts.Labels
			}
			i.ImageRef = opts.Image
			i.AvailabilityMode = opts.AvailabilityMode
			i.IgnoreUnset = opts.IgnoreUnset
			i.DryRun = opts.DryRun
			i.Output = opts.Output
			catalogObj, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to update catalog: %v", err)
			}

			if len(i.DryRun) == 0 {
				log.Printf("catalog %q updated", i.CatalogName)
				return
			}
			if len(i.Output) == 0 {
				log.Printf("catalog %q updated (dry run)", i.CatalogName)
				return
			}

			catalogObj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: olmv1.GroupVersion.Group,
				Version: olmv1.GroupVersion.Version, Kind: "ClusterCatalog"})
			printFormattedCatalogs(i.Output, *catalogObj)
		},
	}
	bindMutableCatalogFlags(cmd.Flags(), &opts.mutableCatalogOptions)
	bindUpdateFieldOptions(cmd.Flags(), &opts.updateDefaultFieldOptions, "clustercatalog")
	bindDryRunFlags(cmd.Flags(), &opts.dryRunOptions)

	return cmd
}

func (o *catalogUpdateOptions) validate() error {
	var errs []error
	if err := o.dryRunOptions.validate(); err != nil {
		errs = append(errs, err)
	}
	if err := o.mutableCatalogOptions.validate(); err != nil {
		errs = append(errs, err)
	}
	return errors.NewAggregate(errs)
}
