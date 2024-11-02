package cmd

import (
	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/action/v1"
	"github.com/operator-framework/kubectl-operator/pkg/action"
	"github.com/spf13/cobra"
)

func newExtensionUninstallCmd(cfg *action.Configuration) *cobra.Command {
	u := internalaction.NewOperatorUninstall(cfg)
	u.Logf = log.Printf

	cmd := &cobra.Command{
		Use:   "uninstall <clusterExtensionName>",
		Short: "Uninstall a cluster extension",
		Long: `Uninstall removes the cluster extension from the cluster.

If the cluster extension includes CRDs, the CRDs will be deleted, and therefore
all custom resources of those types will be deleted as well.

If the cluster extension includes a namespace, the namespace will be deleted,
and therefore all resources in that namespace will be deleted as well.

Warning: this command permanently deletes objects from the cluster. Running
uninstall concurrently with other operations could result in undefined behavior.
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			u.Package = args[0]
			if err := u.Run(cmd.Context()); err != nil {
				log.Fatalf("uninstall operator: %v", err)
			}
		},
	}
	return cmd
}
