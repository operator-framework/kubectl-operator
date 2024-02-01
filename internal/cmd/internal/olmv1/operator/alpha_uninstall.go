package operator

import (
	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	experimentalaction "github.com/operator-framework/kubectl-operator/internal/pkg/experimental/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

func NewOperatorUninstallCmd(cfg *action.Configuration) *cobra.Command {
	u := experimentalaction.NewOperatorUninstall(cfg)
	u.Logf = log.Printf

	cmd := &cobra.Command{
		Use:   "uninstall <operator>",
		Short: "Uninstall an operator",
		Long: `Uninstall deletes the named Operator object.

Warning: this command permanently deletes objects from the cluster. If the
uninstalled Operator bundle contains CRDs, the CRDs will be deleted, which
cascades to the deletion of all operands.
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			u.Package = args[0]
			if err := u.Run(cmd.Context()); err != nil {
				log.Fatalf("uninstall operator: %v", err)
			}
			log.Printf("deleted operator %q", u.Package)
		},
	}
	return cmd
}
