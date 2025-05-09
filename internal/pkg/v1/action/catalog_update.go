package action

import (
	"context"
	"fmt"
	"regexp"

	"k8s.io/apimachinery/pkg/types"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type CatalogUpdate struct {
	config      *action.Configuration
	CatalogName string

	Priority            *int32
	PollIntervalMinutes *int
	Labels              map[string]string
	AvailabilityMode    string
	ImageRef            string

	Logf func(string, ...interface{})
}

func NewCatalogUpdate(config *action.Configuration) *CatalogUpdate {
	return &CatalogUpdate{
		config:              config,
		Logf:                func(string, ...interface{}) {},
		Priority:            new(int32),
		PollIntervalMinutes: new(int),
		Labels:              make(map[string]string),
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

	if cu.ImageRef != "" && !isValidImageRef(cu.ImageRef) {
		return nil, fmt.Errorf("invalid image reference: %q, it must be a valid image reference format", cu.ImageRef)
	}

	cu.setDefaults(catalog)

	cu.setUpdatedCatalog(&catalog)
	if err := cu.config.Client.Update(ctx, &catalog); err != nil {
		return nil, err
	}

	return &catalog, nil
}

func (cu *CatalogUpdate) setUpdatedCatalog(catalog *olmv1.ClusterCatalog) {
	existingLabels := catalog.GetLabels()
	if existingLabels == nil {
		existingLabels = make(map[string]string)
	}

	if cu.Labels != nil {
		for k, v := range cu.Labels {
			if v == "" {
				delete(existingLabels, k) // remove keys with empty values
			} else {
				existingLabels[k] = v
			}
		}
		catalog.SetLabels(existingLabels)
	}

	if cu.Priority != nil {
		catalog.Spec.Priority = *cu.Priority
	}

	if catalog.Spec.Source.Image != nil {
		if cu.PollIntervalMinutes != nil {
			// Set PollIntervalMinutes to the value if it's not nil
			catalog.Spec.Source.Image.PollIntervalMinutes = cu.PollIntervalMinutes
		} else {
			// If it's nil, explicitly unset it
			catalog.Spec.Source.Image.PollIntervalMinutes = nil
		}

		if cu.ImageRef != "" {
			catalog.Spec.Source.Image.Ref = cu.ImageRef
		}
	}

	if cu.AvailabilityMode != "" {
		catalog.Spec.AvailabilityMode = olmv1.AvailabilityMode(cu.AvailabilityMode)
	}
}

func (cu *CatalogUpdate) setDefaults(catalog olmv1.ClusterCatalog) {
	catalogSrc := catalog.Spec.Source

	if cu.PollIntervalMinutes != nil && (*cu.PollIntervalMinutes == 0 || *cu.PollIntervalMinutes == -1) {
		cu.PollIntervalMinutes = nil
	} else if cu.PollIntervalMinutes == nil && catalogSrc.Image != nil && catalogSrc.Image.PollIntervalMinutes != nil {
		// Only default if user didn’t explicitly set anything
		cu.PollIntervalMinutes = catalogSrc.Image.PollIntervalMinutes
	}

	if cu.ImageRef == "" && catalogSrc.Image != nil {
		cu.ImageRef = catalogSrc.Image.Ref
	}
	if cu.AvailabilityMode == "" {
		cu.AvailabilityMode = string(catalog.Spec.AvailabilityMode)
	}
	if cu.Priority == nil {
		cu.Priority = &catalog.Spec.Priority
	}
	if len(cu.Labels) == 0 {
		cu.Labels = catalog.Labels
	}
}

func isValidImageRef(imageRef string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9-_\.]+(?:\/[a-zA-Z0-9-_\.]+)+(:[a-zA-Z0-9-_\.]+)?$`)
	return re.MatchString(imageRef)
}
