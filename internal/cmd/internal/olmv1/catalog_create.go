package olmv1

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	v1action "github.com/operator-framework/kubectl-operator/internal/pkg/v1/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/errors"
)

type catalogCreateOptions struct {
	mutableCatalogOptions
	dryRunOptions
}

// NewCatalogCreateCmd allows creating a new catalog
func NewCatalogCreateCmd(cfg *action.Configuration) *cobra.Command {
	i := v1action.NewCatalogCreate(cfg)
	i.Logf = log.Printf
	var opts catalogCreateOptions

	cmd := &cobra.Command{
		Use:     "catalog <catalog_name> <image_source_ref>",
		Aliases: []string{"catalogs <catalog_name> <image_source_ref>"},
		Args:    cobra.ExactArgs(2),
		Short:   "Create a new catalog",
		Run: func(cmd *cobra.Command, args []string) {
			i.CatalogName = args[0]
			i.ImageSourceRef = args[1]
			opts.Image = i.ImageSourceRef
			if err := opts.validate(); err != nil {
				log.Fatalf("failed to parse flags: %s", err.Error())
			}
			i.DryRun = opts.DryRun
			i.Output = opts.Output
			catalogObj, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to create catalog %q: %v\n", i.CatalogName, err)
			}
			if len(i.DryRun) == 0 {
				log.Printf("catalog %q created\n", i.CatalogName)
				return
			}
			if len(i.Output) == 0 {
				log.Printf("catalog %q created (dry run)\n", i.CatalogName)
				return
			}

			catalogObj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: olmv1.GroupVersion.Group,
				Version: olmv1.GroupVersion.Version, Kind: "ClusterCatalog"})
			printFormattedCatalogs(i.Output, *catalogObj)
		},
	}
	bindCatalogCreateFlags(cmd.Flags(), i)
	bindMutableCatalogFlags(cmd.Flags(), &opts.mutableCatalogOptions)
	bindDryRunFlags(cmd.Flags(), &opts.dryRunOptions)

	return cmd
}

func bindCatalogCreateFlags(fs *pflag.FlagSet, i *v1action.CatalogCreate) {
	fs.DurationVar(&i.CleanupTimeout, "cleanup-timeout", time.Minute, "the amount of time to wait before cancelling cleanup after a failed creation attempt.")
}

func (o *catalogCreateOptions) validate() error {
	var errs []error
	if err := o.dryRunOptions.validate(); err != nil {
		errs = append(errs, err)
	}
	if err := o.mutableCatalogOptions.validate(); err != nil {
		errs = append(errs, err)
	}
	return errors.NewAggregate(errs)
}
