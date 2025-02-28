package olmv1

import (
	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

// NewOperatorInstalledGetCmd handles get commands in the form of:
// operator(s) [operator_name] - this will either list all the installed operators
// if no operator_name has been provided or display the details of the specific
// one otherwise
func NewOperatorInstalledGetCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewOperatorInstalledGet(cfg)
	i.Logf = log.Printf

	cmd := &cobra.Command{
		Use:     "operator [operator_name]",
		Aliases: []string{"operators"},
		Args:    cobra.RangeArgs(0, 1),
		Short:   "Display one or many installed operators",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 1 {
				i.OperatorName = args[0]
			}
			installedExtensions, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed getting installed operator(s): %v", err)
			}

			printFormattedOperators(installedExtensions...)
		},
	}

	return cmd
}
