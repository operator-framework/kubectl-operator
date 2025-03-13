package olmv1

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

// NewExtensionUpdateCmd allows updating a selected operator
func NewExtensionUpdateCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewExtensionUpdate(cfg)
	i.Logf = log.Printf

	cmd := &cobra.Command{
		Use:   "extension <extension>",
		Short: "Update an extension",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			i.Package = args[0]
			_, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to update extension: %v", err)
			}
			log.Printf("extension %q updated", i.Package)
		},
	}
	bindExtensionUpdateFlags(cmd.Flags(), i)

	return cmd
}

func bindExtensionUpdateFlags(fs *pflag.FlagSet, i *v1action.ExtensionUpdate) {
	fs.StringVar(&i.Version, "version", "", "desired extension version (single or range) in semVer format. AND operation with channels")
	fs.StringVar(&i.Selector, "selector", "", "filters the set of catalogs used in the bundle selection process. Empty means that all catalogs will be used in the bundle selection process")
	fs.StringArrayVar(&i.Channels, "channels", []string{}, "desired channels for extension versions. AND operation with version. Empty list means all available channels will be taken into consideration")
	fs.StringVar(&i.UpgradeConstraintPolicy, "upgrade-constraint-policy", "", "controls whether the upgrade path(s) defined in the catalog are enforced. One of CatalogProvided|SelfCertified), Default: CatalogProvided")
	fs.StringToStringVar(&i.Labels, "labels", map[string]string{}, "labels that will be set on the extension")
	fs.BoolVar(&i.IgnoreUnset, "ignore-unset", true, "when enabled, any unset flag value will not be changed. Disabling means that for each unset value a default will be used instead")
}
