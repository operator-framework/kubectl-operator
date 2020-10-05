package action

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func (l *OperatorListAvailable) Run(ctx context.Context) ([]v1.PackageManifest, error) {
	if l.Package != "" {
		pm, err := l.getPackageManifestByName(ctx, l.Package)
		if err != nil {
			return nil, err
		}
		return []v1.PackageManifest{*pm}, nil
	}
	return l.getAllPackageManifests(ctx)
}

func (l *OperatorListAvailable) getPackageManifestByName(ctx context.Context, packageName string) (*v1.PackageManifest, error) {
	pm := v1.PackageManifest{}
	pmKey := types.NamespacedName{
		Namespace: l.config.Namespace,
		Name:      packageName,
	}
	if err := l.config.Client.Get(ctx, pmKey, &pm); err != nil {
		return nil, err
	}
	return &pm, nil
}

func (l *OperatorListAvailable) getAllPackageManifests(ctx context.Context) ([]v1.PackageManifest, error) {
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
	return pms.Items, nil
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
