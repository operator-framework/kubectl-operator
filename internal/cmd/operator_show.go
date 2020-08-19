package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	"github.com/operator-framework/kubectl-operator/internal/pkg/action"
)

func newOperatorShowCmd(cfg *action.Configuration) *cobra.Command {
	i := action.NewOperatorShow(cfg)
	i.Logf = log.Printf

	cmd := &cobra.Command{
		Use:   "show <operator>",
		Short: "Show details about an operator",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			i.Package = args[0]
			ctx, cancel := context.WithTimeout(cmd.Context(), i.ShowTimeout)
			defer cancel()
			out, err := i.Run(ctx)
			if err != nil {
				log.Fatalf("failed to find operator: %v", err)
			}
			for _, v := range out {
				fmt.Print(v)
			}
		},
	}
	i.BindFlags(cmd.Flags())
	return cmd
}
