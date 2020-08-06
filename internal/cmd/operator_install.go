package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/pkg/action"
	"github.com/operator-framework/kubectl-operator/internal/pkg/log"
)

func newOperatorInstallCmd(cfg *action.Configuration) *cobra.Command {
	i := action.NewOperatorInstall(cfg)
	cmd := &cobra.Command{
		Use:   "install <operator>",
		Short: "Install an operator",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			i.Package = args[0]
			ctx, cancel := context.WithTimeout(cmd.Context(), i.InstallTimeout)
			defer cancel()
			csv, err := i.Run(ctx)
			if err != nil {
				log.Fatalf("failed to install operator: %v", err)
			}
			log.Printf("operator %q installed; installed csv is %q", i.Package, csv.Name)
		},
	}
	i.BindFlags(cmd.Flags())
	return cmd
}
