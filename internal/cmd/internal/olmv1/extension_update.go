package olmv1

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/errors"
)

type extensionUpdateOptions struct {
	dryRunOptions
	mutableExtensionOptions
}

// NewExtensionUpdateCmd allows updating a selected operator
func NewExtensionUpdateCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewExtensionUpdate(cfg)
	i.Logf = log.Printf
	var opts extensionUpdateOptions

	cmd := &cobra.Command{
		Use:   "extension <extension name>",
		Short: "Update an extension",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			i.ExtensionName = args[0]
			if err := opts.validate(); err != nil {
				log.Fatalf("failed to parse flags: %s", err.Error())
			}
			i.Version = opts.Version
			i.Channels = opts.Channels
			i.Labels = opts.Labels
			i.UpgradeConstraintPolicy = opts.UpgradeConstraintPolicy
			i.CRDUpgradeSafetyEnforcement = opts.CRDUpgradeSafetyEnforcement
			i.CatalogSelector = opts.ParsedSelector
			i.DryRun = opts.DryRun
			i.Output = opts.Output
			extObj, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to update extension: %v", err)
			}
			if len(i.DryRun) == 0 {
				log.Printf("extension %q updated\n", i.ExtensionName)
				return
			}
			if len(i.Output) == 0 {
				log.Printf("extension %q updated (dry run)\n", i.ExtensionName)
				return
			}

			extObj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: olmv1.GroupVersion.Group,
				Version: olmv1.GroupVersion.Version, Kind: olmv1.ClusterExtensionKind})
			printFormattedExtensions(i.Output, *extObj)
		},
	}
	bindExtensionUpdateFlags(cmd.Flags(), i)
	bindMutableExtensionFlags(cmd.Flags(), &opts.mutableExtensionOptions)
	bindDryRunFlags(cmd.Flags(), &opts.dryRunOptions)

	return cmd
}

func bindExtensionUpdateFlags(fs *pflag.FlagSet, i *v1action.ExtensionUpdate) {
	fs.BoolVar(&i.IgnoreUnset, "ignore-unset", true, "set to false to revert all values not specifically set with flags in the command to their default as defined by the clusterextension customresoucedefinition.")
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
