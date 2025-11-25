package action

import (
	"context"
	"fmt"
	"regexp"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CatalogUpdate struct {
	config      *action.Configuration
	CatalogName string

	Priority            *int32
	PollIntervalMinutes *int
	Labels              map[string]string
	AvailabilityMode    string
	ImageRef            string
	IgnoreUnset         bool

	DryRun string
	Output string
	Logf   func(string, ...interface{})
}

func NewCatalogUpdate(config *action.Configuration) *CatalogUpdate {
	return &CatalogUpdate{
		config:              config,
		Logf:                func(string, ...interface{}) {},
		PollIntervalMinutes: ptr.To(0),
		Priority:            ptr.To(int32(0)),
	}
}

func (i *CatalogUpdate) Run(ctx context.Context) (*olmv1.ClusterCatalog, error) {
	var catalog olmv1.ClusterCatalog
	var err error

	cuKey := types.NamespacedName{
		Name:      i.CatalogName,
		Namespace: i.config.Namespace,
	}
	if err = i.config.Client.Get(ctx, cuKey, &catalog); err != nil {
		return nil, err
	}

	if catalog.Spec.Source.Type != olmv1.SourceTypeImage {
		return nil, fmt.Errorf("unrecognized source type: %q", catalog.Spec.Source.Type)
	}

	if i.ImageRef != "" && !isValidImageRef(i.ImageRef) {
		return nil, fmt.Errorf("invalid image reference: %q, it must be a valid image reference format", i.ImageRef)
	}

	i.setDefaults(&catalog)

	i.setUpdatedCatalog(&catalog)
	if i.DryRun == DryRunAll {
		if err := i.config.Client.Update(ctx, &catalog, client.DryRunAll); err != nil {
			return nil, err
		}
		return &catalog, nil
	}

	if err := i.config.Client.Update(ctx, &catalog); err != nil {
		return nil, err
	}

	i.Logf("Updating catalog %q in namespace %q", i.CatalogName, i.config.Namespace)

	return &catalog, nil
}

func (i *CatalogUpdate) setUpdatedCatalog(catalog *olmv1.ClusterCatalog) {
	existingLabels := catalog.GetLabels()
	if existingLabels == nil {
		existingLabels = make(map[string]string)
	}
	if i.Labels != nil {
		for k, v := range i.Labels {
			if v == "" {
				delete(existingLabels, k)
			} else {
				existingLabels[k] = v
			}
		}
		catalog.SetLabels(existingLabels)
	}

	if i.Priority != nil {
		catalog.Spec.Priority = *i.Priority
	}

	if catalog.Spec.Source.Image == nil {
		catalog.Spec.Source.Image = &olmv1.ImageSource{}
	}

	if i.PollIntervalMinutes != nil {
		if *i.PollIntervalMinutes == 0 || *i.PollIntervalMinutes == -1 {
			catalog.Spec.Source.Image.PollIntervalMinutes = nil
		} else {
			catalog.Spec.Source.Image.PollIntervalMinutes = i.PollIntervalMinutes
		}
	}

	if i.ImageRef != "" {
		catalog.Spec.Source.Image.Ref = i.ImageRef
	}

	catalog.Spec.AvailabilityMode = olmv1.AvailabilityMode(i.AvailabilityMode)
}

func (i *CatalogUpdate) setDefaults(catalog *olmv1.ClusterCatalog) {
	if !i.IgnoreUnset {
		return
	}

	catalogSrc := catalog.Spec.Source

	if i.Priority == nil {
		i.Priority = &catalog.Spec.Priority
	}

	if i.PollIntervalMinutes == nil && catalogSrc.Image != nil && catalogSrc.Image.PollIntervalMinutes != nil {
		i.PollIntervalMinutes = catalogSrc.Image.PollIntervalMinutes
	}

	if i.ImageRef == "" && catalogSrc.Image != nil {
		i.ImageRef = catalogSrc.Image.Ref
	}
	if i.AvailabilityMode == "" {
		i.AvailabilityMode = string(catalog.Spec.AvailabilityMode)
	}
	if len(i.Labels) == 0 {
		i.Labels = catalog.Labels
	}
}

func isValidImageRef(imageRef string) bool {
	var imageRefRegex = regexp.MustCompile(`^([a-z0-9]+(\.[a-z0-9]+)*(:[0-9]+)?/)?[a-z0-9-_]+(/[a-z0-9-_]+)*(:[a-zA-Z0-9_\.-]+)?(@sha256:[a-fA-F0-9]{64})?$`)

	return imageRefRegex.MatchString(imageRef)
}
