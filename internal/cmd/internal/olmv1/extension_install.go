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

type extensionInstallOptions struct {
	dryRunOptions
	mutableExtensionOptions
}

// NewExtensionInstallCmd installs a new extension for a package, requiring a minimum
// of a name for the new extension and the name of the package to install
func NewExtensionInstallCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewExtensionInstall(cfg)
	i.Logf = log.Printf
	var opts extensionInstallOptions

	cmd := &cobra.Command{
		Use:     "extension <extension_name>",
		Aliases: []string{"extensions <extension_name>"},
		Short:   "Install an extension",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			i.ExtensionName = args[0]
			if err := opts.validate(); err != nil {
				log.Fatalf("failed to parse flags: %w", err)
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
				log.Fatalf("failed to install extension %q: %w", i.ExtensionName, err)
			}
			if len(i.DryRun) == 0 {
				log.Printf("extension %q created", i.ExtensionName)
				return
			}
			if len(i.Output) == 0 {
				log.Printf("extension %q created (dry run)", i.ExtensionName)
				return
			}

			extObj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: olmv1.GroupVersion.Group,
				Version: olmv1.GroupVersion.Version, Kind: olmv1.ClusterExtensionKind})
			printFormattedExtensions(i.Output, *extObj)
		},
	}
	bindMutableExtensionFlags(cmd.Flags(), &opts.mutableExtensionOptions)
	bindExtensionInstallFlags(cmd.Flags(), i)
	bindDryRunFlags(cmd.Flags(), &opts.dryRunOptions)

	return cmd
}

func bindExtensionInstallFlags(fs *pflag.FlagSet, i *v1action.ExtensionInstall) {
	fs.StringVarP(&i.Namespace.Name, "namespace", "n", "olmv1-system", "namespace to install the extension in.") //infer?
	fs.StringVarP(&i.PackageName, "package-name", "p", "", "package name of the extension to install. Required.")
	fs.StringVarP(&i.ServiceAccount, "service-account", "s", "default", "service account name to use for the extension installation.")
	fs.DurationVar(&i.CleanupTimeout, "cleanup-timeout", time.Minute, "the amount of time to wait before cancelling cleanup after a failed creation attempt.")

	if err := cobra.MarkFlagRequired(fs, "package-name"); err != nil {
		log.Fatalf("failed to process command flags: %w", err)
	}
}

func (o *extensionInstallOptions) validate() error {
	var errs []error
	if err := o.dryRunOptions.validate(); err != nil {
		errs = append(errs, err)
	}
	if err := o.mutableExtensionOptions.validate(); err != nil {
		errs = append(errs, err)
	}
	return errors.NewAggregate(errs)
}
