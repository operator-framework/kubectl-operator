package action

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ListAvailableOperators struct {
	config *Configuration

	Catalog NamespacedName
}

func NewListAvailableOperators(cfg *Configuration) *ListAvailableOperators {
	return &ListAvailableOperators{
		config: cfg,
	}
}

func (l *ListAvailableOperators) Run(ctx context.Context) ([]v1.PackageManifest, error) {
	pms := v1.PackageManifestList{}

	labelSelector := client.MatchingLabels{}
	if l.Catalog.Name != "" {
		labelSelector["catalog"] = l.Catalog.Name
	}
	if l.Catalog.Namespace != "" {
		labelSelector["catalog-namespace"] = l.Catalog.Namespace
	}
	if err := l.config.Client.List(ctx, &pms, labelSelector); err != nil {
		return nil, err
	}
	return pms.Items, nil
}

func (l *ListAvailableOperators) BindFlags(fs *pflag.FlagSet) {
	fs.VarP(&l.Catalog, "catalog", "c", "catalog to query (default: search all cluster catalogs)")
}

type NamespacedName struct {
	types.NamespacedName
}

func (f *NamespacedName) Set(str string) error {
	split := strings.Split(str, "/")
	switch len(split) {
	case 0:
	case 1:
		f.Name = split[0]
	case 2:
		f.Namespace = split[0]
		f.Name = split[1]
	default:
		return fmt.Errorf("invalid namespaced name value %q", str)
	}
	return nil
}

func (f NamespacedName) String() string {
	return f.NamespacedName.String()
}

func (f NamespacedName) Type() string {
	return "NamespacedNameValue"
}
