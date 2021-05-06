package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

func newOperatorUninstallCmd(cfg *action.Configuration) *cobra.Command {
	u := internalaction.NewOperatorUninstall(cfg)
	u.Logf = log.Printf

	cmd := &cobra.Command{
		Use:   "uninstall <operator>",
		Short: "Uninstall an operator and operands",
		Long: `Uninstall removes the subscription, operator and optionally operands managed by the operator as well as 
the relevant operatorgroup.

Warning: this command permanently deletes objects from the cluster. Running uninstall concurrently with other operations
could result in undefined behavior. 

The uninstall command first checks to find the subscription associated with the operator. It then 
lists all operands found throughout the cluster for the operator
specified if one is found. Since the scope of an operator is restricted by
its operator group, this search will include namespace-scoped operands from the
operator group's target namespaces and all cluster-scoped operands. 

The operand-deletion strategy is then considered if any operands are found on-cluster. One of cancel|ignore|delete. 
By default, the strategy is "cancel", which means that if any operands are found when deleting the operator abort the 
uninstall without deleting anything. 
The "ignore" strategy keeps the operands on cluster and deletes the subscription and the operator.
The "delete" strategy deletes the subscription, operands, and after they have finished finalizing, the operator itself.

To see which operands are on-cluster and would potentially be removed during uninstall, use the kubectl operator 
list-operands <operator> command.

Setting --delete-operator-groups to true will delete the operatorgroup in the provided namespace if no other active 
subscriptions are currently in that namespace, after removing the operator. The subscription and operatorgroup will be 
removed even if the operator is not found.`,
		Args: cobra.ExactArgs(1),
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

func bindOperatorUninstallFlags(fs *pflag.FlagSet, u *internalaction.OperatorUninstall) {
	fs.BoolVar(&u.DeleteOperatorGroups, "delete-operator-groups", false, "delete operator groups if no other operators remain")
	fs.StringSliceVar(&u.DeleteOperatorGroupNames, "delete-operator-group-names", nil, "specific operator group names to delete (only effective with --delete-operator-groups)")
	fs.VarP(&u.OperandStrategy, "operand-strategy", "s", "determines how to handle operands when deleting the operator, one of cancel|ignore|delete")
}
