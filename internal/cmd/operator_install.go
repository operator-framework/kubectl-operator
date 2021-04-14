package cmd

import (
	"fmt"
	"time"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

func newOperatorInstallCmd(cfg *action.Configuration) *cobra.Command {
	i := internalaction.NewOperatorInstall(cfg)
	i.Logf = log.Printf

	cmd := &cobra.Command{
		Use:   "install <operator>",
		Short: "Install an operator",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			i.Package = args[0]
			csv, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to install operator: %v", err)
			}
			log.Printf("operator %q installed; installed csv is %q", i.Package, csv.Name)
		},
	}
	bindOperatorInstallFlags(cmd.Flags(), i)

	return cmd
}

func bindOperatorInstallFlags(fs *pflag.FlagSet, i *internalaction.OperatorInstall) {
	fs.StringVarP(&i.Channel, "channel", "c", "", "subscription channel")
	fs.VarP(&i.Approval, "approval", "a", fmt.Sprintf("approval (%s or %s)", v1alpha1.ApprovalManual, v1alpha1.ApprovalAutomatic))
	fs.StringVarP(&i.Version, "version", "v", "", "install specific version for operator (default latest)")
	fs.StringSliceVarP(&i.WatchNamespaces, "watch", "w", []string{}, "namespaces to watch")
	fs.DurationVar(&i.CleanupTimeout, "cleanup-timeout", time.Minute, "the amount of time to wait before cancelling cleanup")
	fs.BoolVarP(&i.CreateOperatorGroup, "create-operator-group", "C", false, "create operator group if necessary")

	fs.VarP(&i.InstallMode, "install-mode", "i", "install mode")
	err := fs.MarkHidden("install-mode")
	if err != nil {
		panic(`requested flag "install-mode" missing`)
	}
}
