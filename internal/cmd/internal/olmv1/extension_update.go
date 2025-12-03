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

type extensionUpdateOptions struct {
	dryRunOptions
	mutableExtensionOptions
	updateDefaultFieldOptions
}

// NewExtensionUpdateCmd updates one or more mutable fields
// of an extension specified by name
func NewExtensionUpdateCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewExtensionUpdate(cfg)
	i.Logf = log.Printf
	var opts extensionUpdateOptions

	cmd := &cobra.Command{
		Use:     "extension <extension_name>",
		Aliases: []string{"extensions <extension_name>"},
		Short:   "Update an extension",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			i.ExtensionName = args[0]
			if err := opts.validate(); err != nil {
				log.Fatalf("failed to parse flags: %v", err)
			}
			i.Version = opts.Version
			i.Channels = opts.Channels
			i.Labels = opts.Labels
			i.UpgradeConstraintPolicy = opts.UpgradeConstraintPolicy
			i.CRDUpgradeSafetyEnforcement = opts.CRDUpgradeSafetyEnforcement
			i.CatalogSelector = opts.ParsedSelector
			i.IgnoreUnset = opts.IgnoreUnset
			i.DryRun = opts.DryRun
			i.Output = opts.Output
			extObj, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to update extension: %v", err)
			}
			if len(i.DryRun) == 0 {
				log.Printf("extension %q updated", i.ExtensionName)
				return
			}
			if len(i.Output) == 0 {
				log.Printf("extension %q updated (dry run)", i.ExtensionName)
				return
			}

			extObj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: olmv1.GroupVersion.Group,
				Version: olmv1.GroupVersion.Version, Kind: olmv1.ClusterExtensionKind})
			printFormattedExtensions(i.Output, *extObj)
		},
	}
	bindMutableExtensionFlags(cmd.Flags(), &opts.mutableExtensionOptions)
	bindUpdateFieldOptions(cmd.Flags(), &opts.updateDefaultFieldOptions, "clusterextension")
	bindDryRunFlags(cmd.Flags(), &opts.dryRunOptions)

	return cmd
}

func (o *extensionUpdateOptions) validate() error {
	var errs []error
	if err := o.dryRunOptions.validate(); err != nil {
		errs = append(errs, err)
	}
	if err := o.mutableExtensionOptions.validate(); err != nil {
		errs = append(errs, err)
	}
	return errors.NewAggregate(errs)
}
