package olmv1

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

// NewOperatorUpdateCmd allows updating a selected operator
func NewOperatorUpdateCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewOperatorUpdate(cfg)
	i.Logf = log.Printf

	cmd := &cobra.Command{
		Use:   "operator <operator>",
		Short: "Update an operator",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			i.Package = args[0]
			_, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to update operator: %v", err)
			}
			log.Printf("operator %q updated", i.Package)
		},
	}
	bindOperatorUpdateFlags(cmd.Flags(), i)

	return cmd
}

func bindOperatorUpdateFlags(fs *pflag.FlagSet, i *v1action.OperatorUpdate) {
	fs.StringVar(&i.Version, "version", "", "desired operator version (single or range) in semVer format. AND operation with channels")
	fs.StringVar(&i.Selector, "selector", "", "filters the set of catalogs used in the bundle selection process. Empty means that all catalogs will be used in the bundle selection process")
	fs.StringArrayVar(&i.Channels, "channels", []string{}, "desired channels for operator versions. AND operation with version. Empty list means all available channels will be taken into consideration")
	fs.StringVar(&i.UpgradeConstraintPolicy, "upgrade-constraint-policy", "", "controls whether the upgrade path(s) defined in the catalog are enforced. One of CatalogProvided|SelfCertified), Default: CatalogProvided")
	fs.StringToStringVar(&i.Labels, "labels", map[string]string{}, "labels that will be set on the operator")
	fs.BoolVar(&i.IgnoreUnset, "ignore-unset", true, "when enabled, any unset flag value will not be changed. Disabling means that for each unset value a default will be used instead")
}
