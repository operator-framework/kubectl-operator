package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

func newOperatorUpgradeCmd(cfg *action.Configuration) *cobra.Command {
	u := internalaction.NewOperatorUpgrade(cfg)
	cmd := &cobra.Command{
		Use:   "upgrade <operator>",
		Short: "Upgrade an operator",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			u.Package = args[0]
			csv, err := u.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to upgrade operator: %v", err)
			}
			log.Printf("operator %q upgraded; installed csv is %q", u.Package, csv.Name)
		},
	}
	bindOperatorUpgradeFlags(cmd.Flags(), u)
	return cmd
}

func bindOperatorUpgradeFlags(fs *pflag.FlagSet, u *internalaction.OperatorUpgrade) {
	fs.StringVarP(&u.Channel, "channel", "c", "", "subscription channel")
}
