package cmd

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/action"
	"github.com/operator-framework/kubectl-operator/internal/pkg/operand"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

func newOperatorUninstallCmd(cfg *action.Configuration) *cobra.Command {
	u := internalaction.NewOperatorUninstall(cfg)
	u.Logf = log.Printf

	cmd := &cobra.Command{
		Use:   "uninstall <operator>",
		Short: "Uninstall an operator and operands",
		Long: `Uninstall removes the subscription, operator and optionally operands managed
by the operator as well as the relevant operatorgroup.

Warning: this command permanently deletes objects from the cluster. Running
uninstall concurrently with other operations could result in undefined behavior.

The uninstall command first checks to find the subscription associated with the
operator. It then lists all operands found throughout the cluster for the
operator specified if one is found. Since the scope of an operator is restricted
by its operator group, this search will include namespace-scoped operands from
the operator group's target namespaces and all cluster-scoped operands.

The operand-deletion strategy is then considered if any operands are found
on-cluster. One of cancel|ignore|delete. By default, the strategy is "cancel",
which means that if any operands are found when deleting the operator abort the
uninstall without deleting anything. The "ignore" strategy keeps the operands on
cluster and deletes the subscription and the operator. The "delete" strategy
deletes the subscription, operands, and after they have finished finalizing, the
operator itself.

Setting --delete-operator-groups to true will delete the operatorgroup in the
provided namespace if no other active subscriptions are currently in that
namespace, after removing the operator. The subscription and operatorgroup will
be removed even if the operator is not found.

There are other deletion flags for removing additional objects, such as custom
resource definitions, operator objects, and any other objects deployed alongside
the operator (e.g. RBAC objects for the operator). These flags are:

  --delete-operator

      Deletes all objects associated with the operator by looking up the
      operator object for the operator and deleting every referenced object
      and then deleting the operator object itself. This implies the flag
      '--operand-strategy=delete' because it is impossible to delete CRDs
      without also deleting instances of those CRDs.

  -X, --delete-all

      This is a convenience flag that is effectively equivalent to the flags
      '--delete-operator=true --delete-operator-groups=true'.

NOTE: This command does not recursively uninstall unused dependencies. To return
a cluster to its state prior to a 'kubectl operator install' call, each
dependency of the operator that was installed automatically by OLM must be
individually uninstalled.
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			u.Package = args[0]
			if err := u.Run(cmd.Context()); err != nil {
				if errors.Is(err, operand.ErrCancelStrategy) {
					log.Fatalf("uninstall operator: %v"+"\n\n%s", err,
						"See kubectl operator uninstall --help for more information on operand deletion strategies.")
				}
				log.Fatalf("uninstall operator: %v", err)
			}
		},
	}
	bindOperatorUninstallFlags(cmd.Flags(), u)
	return cmd
}

func bindOperatorUninstallFlags(fs *pflag.FlagSet, u *internalaction.OperatorUninstall) {
	fs.BoolVarP(&u.DeleteAll, "delete-all", "X", false, "delete all objects associated with the operator, implies --delete-operator, --operand-strategy=delete, --delete-operator-groups")
	fs.BoolVar(&u.DeleteOperator, "delete-operator", false, "delete operator object associated with the operator, --operand-strategy=delete")
	fs.BoolVar(&u.DeleteOperatorGroups, "delete-operator-groups", false, "delete operator groups if no other operators remain")
	fs.StringSliceVar(&u.DeleteOperatorGroupNames, "delete-operator-group-names", nil, "specific operator group names to delete (only effective with --delete-operator-groups)")
	fs.VarP(&u.OperandStrategy, "operand-strategy", "s", "determines how to handle operands when deleting the operator, one of cancel|ignore|delete (default: cancel)")
}
