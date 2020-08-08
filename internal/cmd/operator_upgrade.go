package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	"github.com/operator-framework/kubectl-operator/internal/pkg/action"
)

func newOperatorUpgradeCmd(cfg *action.Configuration) *cobra.Command {
	u := action.NewOperatorUpgrade(cfg)
	cmd := &cobra.Command{
		Use:   "upgrade <operator>",
		Short: "Upgrade an operator",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			u.Package = args[0]
			ctx, cancel := context.WithTimeout(cmd.Context(), u.UpgradeTimeout)
			defer cancel()
			csv, err := u.Run(ctx)
			if err != nil {
				log.Fatalf("failed to upgrade operator: %v", err)
			}
			log.Printf("operator %q upgraded; installed csv is %q", u.Package, csv.Name)
		},
	}
	u.BindFlags(cmd.Flags())
	return cmd
}
