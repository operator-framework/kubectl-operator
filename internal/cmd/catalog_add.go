package cmd

import (
	"context"
	"io/ioutil"

	"github.com/operator-framework/operator-registry/pkg/image/containerdregistry"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	"github.com/operator-framework/kubectl-operator/internal/pkg/action"
)

func newCatalogAddCmd(cfg *action.Configuration) *cobra.Command {
	a := action.NewCatalogAdd(cfg)
	a.Logf = log.Printf

	cmd := &cobra.Command{
		Use:   "add <name> <index_image>",
		Short: "Add an operator catalog",
		Args:  cobra.ExactArgs(2),
		PreRun: func(cmd *cobra.Command, args []string) {
			regLogger := logrus.New()
			regLogger.SetOutput(ioutil.Discard)
			a.RegistryOptions = []containerdregistry.RegistryOption{
				containerdregistry.WithLog(logrus.NewEntry(regLogger)),
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(cmd.Context(), a.AddTimeout)
			defer cancel()

			a.CatalogSourceName = args[0]
			a.IndexImage = args[1]

			cs, err := a.Run(ctx)
			if err != nil {
				log.Fatalf("failed to add catalog: %v", err)
			}
			log.Printf("created catalogsource %q\n", cs.Name)
		},
	}
	a.BindFlags(cmd.Flags())

	return cmd
}
