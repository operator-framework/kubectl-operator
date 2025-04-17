package action

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type CatalogUpdate struct {
	config      *action.Configuration
	CatalogName string

	Priority            int32
	PollIntervalMinutes int
	Labels              map[string]string
	AvailabilityMode    string

	Logf func(string, ...interface{})
}

func NewCatalogUpdate(config *action.Configuration) *CatalogUpdate {
	return &CatalogUpdate{
		config: config,
		Logf:   func(string, ...interface{}) {},
	}
}

func (cu *CatalogUpdate) Run(ctx context.Context) (*olmv1.ClusterCatalog, error) {
	var catalog olmv1.ClusterCatalog
	var err error

	cuKey := types.NamespacedName{
		Name:      cu.CatalogName,
		Namespace: cu.config.Namespace,
	}
	if err = cu.config.Client.Get(ctx, cuKey, &catalog); err != nil {
		return nil, err
	}

	if catalog.Spec.Source.Type != olmv1.SourceTypeImage {
		return nil, fmt.Errorf("unrecognized source type: %q", catalog.Spec.Source.Type)
	}

	cu.setDefaults(catalog)

	cu.setUpdatedCatalog(&catalog)
	if err := cu.config.Client.Update(ctx, &catalog); err != nil {
		return nil, err
	}

	return &catalog, nil
}

func (cu *CatalogUpdate) setUpdatedCatalog(catalog *olmv1.ClusterCatalog) {
	catalog.SetLabels(cu.Labels)
	catalog.Spec.Priority = cu.Priority
	if catalog.Spec.Source.Image != nil && catalog.Spec.Source.Image.PollIntervalMinutes != nil {
		catalog.Spec.Source.Image.PollIntervalMinutes = &cu.PollIntervalMinutes
	}
	catalog.Spec.AvailabilityMode = olmv1.AvailabilityMode(cu.AvailabilityMode)
}

func (cu *CatalogUpdate) setDefaults(catalog olmv1.ClusterCatalog) {
	catalogSrc := catalog.Spec.Source
	if catalogSrc.Image != nil && catalogSrc.Image.PollIntervalMinutes != nil {
		if cu.PollIntervalMinutes == 0 {
			cu.PollIntervalMinutes = *catalogSrc.Image.PollIntervalMinutes
		}
	}
	if cu.AvailabilityMode == "" {
		cu.AvailabilityMode = string(catalog.Spec.AvailabilityMode)
	}
	if cu.Priority == 0 {
		cu.Priority = catalog.Spec.Priority
	}
	if len(cu.Labels) == 0 {
		cu.Labels = catalog.Labels
	}
}
