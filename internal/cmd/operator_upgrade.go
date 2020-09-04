package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	"github.com/operator-framework/kubectl-operator/internal/pkg/action"
)

func newOperatorUpgradeCmd(cfg *action.Configuration) *cobra.Command {
	var timeout time.Duration
	u := action.NewOperatorUpgrade(cfg)
	cmd := &cobra.Command{
		Use:   "upgrade <operator>",
		Short: "Upgrade an operator",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			u.Package = args[0]
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()
			csv, err := u.Run(ctx)
			if err != nil {
				log.Fatalf("failed to upgrade operator: %v", err)
			}
			log.Printf("operator %q upgraded; installed csv is %q", u.Package, csv.Name)
		},
	}
	bindOperatorUpgradeFlags(cmd.Flags(), u)
	cmd.Flags().DurationVarP(&timeout, "timeout", "t", time.Minute, "the amount of time to wait before cancelling the upgrade")
	return cmd
}

func bindOperatorUpgradeFlags(fs *pflag.FlagSet, u *action.OperatorUpgrade) {
	fs.StringVarP(&u.Channel, "channel", "c", "", "subscription channel")
}
