package cmd

import (
	"context"
	"io/ioutil"

	"github.com/operator-framework/operator-registry/pkg/image/containerdregistry"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/joelanford/kubectl-operator/internal/pkg/action"
	"github.com/joelanford/kubectl-operator/internal/pkg/log"
)

func newCatalogInstallCmd(cfg *action.Configuration) *cobra.Command {
	i := action.NewInstallCatalog(cfg)

	cmd := &cobra.Command{
		Use:   "install <index_image>",
		Short: "Install an operator catalog",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			regLogger := logrus.New()
			regLogger.SetOutput(ioutil.Discard)
			i.RegistryOptions = []containerdregistry.RegistryOption{
				containerdregistry.WithLog(logrus.NewEntry(regLogger)),
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(cmd.Context(), i.InstallTimeout)
			defer cancel()

			i.IndexImage = args[0]

			cs, err := i.Run(ctx)
			if err != nil {
				log.Fatalf("failed to install catalog: %v", err)
			}
			log.Printf("created catalogsource %q\n", cs.Name)
		},
	}
	i.BindFlags(cmd.Flags())

	return cmd
}
