package olmv1

import (
	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	experimentalaction "github.com/operator-framework/kubectl-operator/internal/pkg/experimental/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

func NewOperatorInstallCmd(cfg *action.Configuration) *cobra.Command {
	i := experimentalaction.NewOperatorInstall(cfg)
	i.Logf = log.Printf

	cmd := &cobra.Command{
		Use:   "install <operator>",
		Short: "Install an operator",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			i.Package = args[0]
			_, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to install operator: %v", err)
			}
			log.Printf("operator %q created", i.Package)
		},
	}

	return cmd
}
