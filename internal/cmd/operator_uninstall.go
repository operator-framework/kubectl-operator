package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

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
	bindOperatorUninstallFlags(cmd.Flags(), u)
	return cmd
}

func bindOperatorUninstallFlags(fs *pflag.FlagSet, u *action.OperatorUninstall) {
	fs.BoolVarP(&u.DeleteAll, "delete-all", "X", false, "enable all delete flags")
	fs.BoolVar(&u.DeleteCRDs, "delete-crds", false, "delete all owned CRDs and all CRs")
	fs.BoolVar(&u.DeleteOperatorGroups, "delete-operator-groups", false, "delete operator group if no other operators remain")
	fs.StringSliceVar(&u.DeleteOperatorGroupNames, "delete-operator-group-names", nil, "delete operator group if no other operators remain")
}
