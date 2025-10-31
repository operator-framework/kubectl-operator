package olmv1

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewExtensionUpdateCmd allows updating a selected operator
func NewExtensionUpdateCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewExtensionUpdate(cfg)
	i.Logf = log.Printf

	cmd := &cobra.Command{
		Use:   "extension <extension name>",
		Short: "Update an extension",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			i.ExtensionName = args[0]
			if len(i.DryRun) > 0 && i.DryRun != v1action.DryRunAll {
				log.Fatalf("invalid value for `--dry-run` %s, must be one of (%s)\n", i.DryRun, v1action.DryRunAll)
			}
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

	return cmd
}

func bindExtensionUpdateFlags(fs *pflag.FlagSet, i *v1action.ExtensionUpdate) {
	fs.StringArrayVar(&i.Channels, "channels", []string{}, "desired channels for extension versions. AND operation with version. Empty list means all available channels will be taken into consideration")
	fs.StringVar(&i.Version, "version", "", "desired extension version (single or range) in semVer format. AND operation with channels")
	fs.StringVar(&i.Selector, "catalog-selector", "", "selector (label query) to filter catalogs to search for the package, "+
		"supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 "+
		"in (value3)). Matching objects must satisfy all of the specified label constraints.")
	fs.StringVar(&i.UpgradeConstraintPolicy, "upgrade-constraint-policy", "", "controls whether the upgrade path(s) defined in the catalog are enforced."+
		" One of CatalogProvided, SelfCertified), Default: CatalogProvided")
	fs.StringToStringVar(&i.Labels, "labels", map[string]string{}, "labels that will be set on the extension")
	fs.BoolVar(&i.IgnoreUnset, "ignore-unset", true, "when enabled, any unset flag value will not be changed. Disabling means that for each unset value a default will be used instead")
	fs.StringVar(&i.CRDUpgradeSafetyEnforcement, "crd-upgrade-safety-enforcement", "", "policy for preflight CRD Upgrade safety checks. One of: (Strict, None), default: Strict")
	fs.StringVar(&i.DryRun, "dry-run", "", "display the object that would be sent on a request without applying it. One of: (All)")
	fs.StringVarP(&i.Output, "output", "o", "", "output format for dry-run manifests. One of: (json, yaml)")
}
