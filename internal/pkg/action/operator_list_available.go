package action

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/operator-framework/kubectl-operator/internal/pkg/operator"
)

type OperatorListAvailable struct {
	config *Configuration

	Catalog NamespacedName
	Package string
}

func NewOperatorListAvailable(cfg *Configuration) *OperatorListAvailable {
	return &OperatorListAvailable{
		config: cfg,
	}
}

func (l *OperatorListAvailable) Run(ctx context.Context) ([]operator.PackageManifest, error) {
	if l.Package != "" {
		pm, err := l.getPackageManifestByName(ctx, l.Package)
		if err != nil {
			return nil, err
		}
		return []operator.PackageManifest{*pm}, nil
	}
	return l.getAllPackageManifests(ctx)
}

func (l *OperatorListAvailable) getPackageManifestByName(ctx context.Context, packageName string) (*operator.PackageManifest, error) {
	pm := v1.PackageManifest{}
	pmKey := types.NamespacedName{
		Namespace: l.config.Namespace,
		Name:      packageName,
	}
	if err := l.config.Client.Get(ctx, pmKey, &pm); err != nil {
		return nil, err
	}
	return &operator.PackageManifest{PackageManifest: pm}, nil
}

func (l *OperatorListAvailable) getAllPackageManifests(ctx context.Context) ([]operator.PackageManifest, error) {
	labelSelector := client.MatchingLabels{}
	if l.Catalog.Name != "" {
		labelSelector["catalog"] = l.Catalog.Name
	}
	if l.Catalog.Namespace != "" {
		labelSelector["catalog-namespace"] = l.Catalog.Namespace
	}

	pms := v1.PackageManifestList{}
	if err := l.config.Client.List(ctx, &pms, labelSelector); err != nil {
		return nil, err
	}
	pkgs := make([]operator.PackageManifest, len(pms.Items))
	for i := range pms.Items {
		pkgs[i] = operator.PackageManifest{PackageManifest: pms.Items[i]}
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
