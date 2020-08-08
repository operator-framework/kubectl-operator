package cmd

import (
	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	"github.com/operator-framework/kubectl-operator/internal/pkg/action"
)

func newOperatorUninstallCmd(cfg *action.Configuration) *cobra.Command {
	u := action.NewOperatorUninstall(cfg)
	u.Logf = log.Printf

	cmd := &cobra.Command{
		Use:   "uninstall <operator>",
		Short: "Uninstall an operator",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			u.Package = args[0]
			if err := u.Run(cmd.Context()); err != nil {
				log.Fatalf("uninstall operator: %v", err)
			}
			log.Printf("operator %q uninstalled", u.Package)
		},
	}
	u.BindFlags(cmd.Flags())
	return cmd
}
