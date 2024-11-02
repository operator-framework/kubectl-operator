package v0

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"

	"github.com/operator-framework/kubectl-operator/internal/pkg/legacy/operator"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type OperatorListAvailable struct {
	config *action.Configuration

	Catalog NamespacedName
	Package string
}

func NewOperatorListAvailable(cfg *action.Configuration) *OperatorListAvailable {
	return &OperatorListAvailable{
		config: cfg,
	}
}

func (l *OperatorListAvailable) Run(ctx context.Context) ([]operator.PackageManifest, error) {
	labelSelector := client.MatchingLabels{}
	if l.Catalog.Name != "" {
		labelSelector["catalog"] = l.Catalog.Name
	}
	if l.Catalog.Namespace != "" {
		labelSelector["catalog-namespace"] = l.Catalog.Namespace
	}

	if l.Package != "" {
		pm := v1.PackageManifest{}
		if err := l.config.Client.Get(ctx, types.NamespacedName{Name: l.Package, Namespace: l.config.Namespace}, &pm); err != nil {
			return nil, err
		}
		return []operator.PackageManifest{{PackageManifest: pm}}, nil
	}

	pms := v1.PackageManifestList{}
	if err := l.config.Client.List(ctx, &pms, labelSelector, client.InNamespace(l.config.Namespace)); err != nil {
		return nil, err
	}
	pkgs := make([]operator.PackageManifest, 0, len(pms.Items))
	for _, pm := range pms.Items {
		pkgs = append(pkgs, operator.PackageManifest{PackageManifest: pm})
	}
	return pkgs, nil
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
