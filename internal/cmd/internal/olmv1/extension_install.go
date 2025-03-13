package olmv1

import (
	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

func NewExtensionInstallCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewExtensionInstall(cfg)
	i.Logf = log.Printf

	cmd := &cobra.Command{
		Use:   "install <extension>",
		Short: "Install an extension",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			i.Package = args[0]
			_, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to install extension: %v", err)
			}
			log.Printf("extension %q created", i.Package)
		},
	}

	return cmd
}
