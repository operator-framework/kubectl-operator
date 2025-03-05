package action

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/blang/semver/v4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type OperatorUpdate struct {
	config *action.Configuration

	Package string

	Version                 string
	Channels                []string
	UpgradeConstraintPolicy string
	OverrideUnset           bool

	CleanupTimeout time.Duration

	Logf func(string, ...interface{})
}

func NewOperatorUpdate(cfg *action.Configuration) *OperatorUpdate {
	return &OperatorUpdate{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (ou *OperatorUpdate) Run(ctx context.Context) (*olmv1.ClusterExtension, error) {
	var ext olmv1.ClusterExtension

	opKey := types.NamespacedName{Name: ou.Package}
	if err := ou.config.Client.Get(ctx, opKey, &ext); err != nil {
		return nil, err
	}

	if ext.Spec.Source.SourceType != olmv1.SourceTypeCatalog {
		return nil, fmt.Errorf("unrecognized source type: %q", ext.Spec.Source.SourceType)
	}

	ou.setDefaults(ext.Spec.Source.Catalog)
	constraintPolicy := olmv1.UpgradeConstraintPolicy(ou.UpgradeConstraintPolicy)
	if !ou.needsUpdate(ext.Spec.Source.Catalog, constraintPolicy) {
		return nil, errNoChange
	}

	if ou.Version != "" {
		if _, err := semver.ParseRange(ou.Version); err != nil {
			return nil, fmt.Errorf("failed parsing version: %w", err)
		}
	}

	ext.Spec.Source.Catalog.Version = ou.Version
	ext.Spec.Source.Catalog.Channels = ou.Channels
	ext.Spec.Source.Catalog.UpgradeConstraintPolicy = constraintPolicy
	if err := ou.config.Client.Update(ctx, &ext); err != nil {
		return nil, err
	}

	if err := waitUntilOperatorStatusCondition(ctx, ou.config.Client, &ext, olmv1.TypeInstalled, metav1.ConditionTrue); err != nil {
		return nil, fmt.Errorf("timed out waiting for operator: %w", err)
	}

	return &ext, nil
}

func (ou *OperatorUpdate) setDefaults(catalogSrc *olmv1.CatalogSource) {
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

		return
	}

	if ou.UpgradeConstraintPolicy == "" {
		ou.UpgradeConstraintPolicy = string(olmv1.UpgradeConstraintPolicyCatalogProvided)
	}
}

func (ou *OperatorUpdate) needsUpdate(catalogSrc *olmv1.CatalogSource, constraintPolicy olmv1.UpgradeConstraintPolicy) bool {
	if catalogSrc.Version == ou.Version &&
		slices.Equal(catalogSrc.Channels, ou.Channels) &&
		catalogSrc.UpgradeConstraintPolicy == constraintPolicy {
		return false
	}

	return true
}
