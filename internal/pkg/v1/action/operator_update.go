package action

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/blang/semver/v4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"
)

type updateClient interface {
	updater
	getter
}

type OperatorUpdate struct {
	client updateClient

	Package string

	Version                 string
	Channels                []string
	UpgradeConstraintPolicy string
	Labels                  map[string]string
	OverrideUnset           bool

	CleanupTimeout time.Duration

	Logf func(string, ...interface{})
}

func NewOperatorUpdate(client updateClient) *OperatorUpdate {
	return &OperatorUpdate{
		client: client,
		Logf:   func(string, ...interface{}) {},
	}
}

func (ou *OperatorUpdate) Run(ctx context.Context) (*olmv1.ClusterExtension, error) {
	var ext olmv1.ClusterExtension

	opKey := types.NamespacedName{Name: ou.Package}
	if err := ou.client.Get(ctx, opKey, &ext); err != nil {
		return nil, err
	}

	if ext.Spec.Source.SourceType != olmv1.SourceTypeCatalog {
		return nil, fmt.Errorf("unrecognized source type: %q", ext.Spec.Source.SourceType)
	}

	ou.setDefaults(ext)
	constraintPolicy := olmv1.UpgradeConstraintPolicy(ou.UpgradeConstraintPolicy)
	if !ou.needsUpdate(ext, constraintPolicy) {
		return nil, ErrNoChange
	}

	if ou.Version != "" {
		if _, err := semver.ParseRange(ou.Version); err != nil {
			return nil, fmt.Errorf("failed parsing version: %w", err)
		}
	}

	ext.SetLabels(ou.Labels)
	ext.Spec.Source.Catalog.Version = ou.Version
	ext.Spec.Source.Catalog.Channels = ou.Channels
	ext.Spec.Source.Catalog.UpgradeConstraintPolicy = constraintPolicy
	if err := ou.client.Update(ctx, &ext); err != nil {
		return nil, err
	}

	if err := waitUntilOperatorStatusCondition(ctx, ou.client, &ext, olmv1.TypeInstalled, metav1.ConditionTrue); err != nil {
		return nil, fmt.Errorf("timed out waiting for operator: %w", err)
	}

	return &ext, nil
}

func (ou *OperatorUpdate) setDefaults(ext olmv1.ClusterExtension) {
	catalogSrc := ext.Spec.Source.Catalog
	if ou.OverrideUnset {
		if ou.Version == "" {
			ou.Version = catalogSrc.Version
		}
		if len(ou.Channels) == 0 {
			ou.Channels = catalogSrc.Channels
		}
		if ou.UpgradeConstraintPolicy == "" {
			ou.UpgradeConstraintPolicy = string(catalogSrc.UpgradeConstraintPolicy)
		}
		if len(ou.Labels) == 0 {
			ou.Labels = ext.Labels
		}

		return
	}

	if ou.UpgradeConstraintPolicy == "" {
		ou.UpgradeConstraintPolicy = string(olmv1.UpgradeConstraintPolicyCatalogProvided)
	}
}

func (ou *OperatorUpdate) needsUpdate(ext olmv1.ClusterExtension, constraintPolicy olmv1.UpgradeConstraintPolicy) bool {
	catalogSrc := ext.Spec.Source.Catalog

	if catalogSrc.Version == ou.Version &&
		slices.Equal(catalogSrc.Channels, ou.Channels) &&
		catalogSrc.UpgradeConstraintPolicy == constraintPolicy &&
		maps.Equal(ext.Labels, ou.Labels) {
		return false
	}

	return true
}
