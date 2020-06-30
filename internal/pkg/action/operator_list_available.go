package action

import (
	"context"

	v1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
)

type ListAvailableOperators struct {
	config *Configuration
}

func NewListAvailableOperators(cfg *Configuration) *ListAvailableOperators {
	return &ListAvailableOperators{cfg}
}

func (l *ListAvailableOperators) Run(ctx context.Context) ([]v1.PackageManifest, error) {
	pms := v1.PackageManifestList{}
	if err := l.config.Client.List(ctx, &pms); err != nil {
		return nil, err
	}
	return pms.Items, nil
}
