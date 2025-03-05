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
	fs.StringVar(&i.Version, "version", "", "desired operator version (single or range) in semver format. AND operation with channels")
	fs.StringArrayVar(&i.Channels, "channels", []string{}, "desired channels for operator versions. AND operation with version")
	fs.StringVar(&i.UpgradeConstraintPolicy, "upgrade-constraint-policy", "", "controls whether the upgrade path(s) defined in the catalog are enforced, one of CatalogProvided|SelfCertified), Default: CatalogProvided")
	fs.BoolVar(&i.OverrideUnset, "override-unset-with-current", false, "when enabled, any unset flag value will be overridden with value already set in current operator")
}
