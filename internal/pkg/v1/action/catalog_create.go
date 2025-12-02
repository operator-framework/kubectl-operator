package action

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type CatalogCreate struct {
	config      *action.Configuration
	CatalogName string

	ImageSourceRef      string
	Priority            int32
	PollIntervalMinutes int
	Labels              map[string]string
	AvailabilityMode    string
	CleanupTimeout      time.Duration

	DryRun string
	Output string
	Logf   func(string, ...interface{})
}

func NewCatalogCreate(config *action.Configuration) *CatalogCreate {
	return &CatalogCreate{
		config: config,
		Logf:   func(string, ...interface{}) {},
	}
}

func (i *CatalogCreate) Run(ctx context.Context) (*olmv1.ClusterCatalog, error) {
	catalog := i.buildCatalog()
	if i.DryRun == DryRunAll {
		if err := i.config.Client.Create(ctx, &catalog, client.DryRunAll); err != nil {
			return nil, err
		}
		return &catalog, nil
	}
	if err := i.config.Client.Create(ctx, &catalog); err != nil {
		return nil, err
	}

	var err error
	if i.AvailabilityMode == string(olmv1.AvailabilityModeAvailable) {
		err = waitUntilCatalogStatusCondition(ctx, i.config.Client, &catalog, olmv1.TypeServing, metav1.ConditionTrue)
	} else {
		err = waitUntilCatalogStatusCondition(ctx, i.config.Client, &catalog, olmv1.TypeServing, metav1.ConditionFalse)
	}

	if err != nil {
		if cleanupErr := deleteWithTimeout(i.config.Client, &catalog, i.CleanupTimeout); cleanupErr != nil {
			i.Logf("cleaning up failed catalog: %v", cleanupErr)
		}
		return nil, err
	}

	return &catalog, nil
}

func (i *CatalogCreate) buildCatalog() olmv1.ClusterCatalog {
	catalog := olmv1.ClusterCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:   i.CatalogName,
			Labels: i.Labels,
		},
		Spec: olmv1.ClusterCatalogSpec{
			Source: olmv1.CatalogSource{
				Type: olmv1.SourceTypeImage,
				Image: &olmv1.ImageSource{
					Ref: i.ImageSourceRef,
				},
			},
			Priority:         i.Priority,
			AvailabilityMode: olmv1.AvailabilityModeAvailable,
		},
	}
	if len(i.AvailabilityMode) != 0 {
		catalog.Spec.AvailabilityMode = olmv1.AvailabilityMode(i.AvailabilityMode)
	}
	if i.PollIntervalMinutes > 0 {
		catalog.Spec.Source.Image.PollIntervalMinutes = &i.PollIntervalMinutes
	}

	return catalog
}
