package action

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type CatalogCreate struct {
	config         *action.Configuration
	CatalogName    string
	ImageSourceRef string

	Priority            int32
	PollIntervalMinutes int
	Labels              map[string]string
	Available           bool
	CleanupTimeout      time.Duration

	Logf func(string, ...interface{})
}

func NewCatalogCreate(config *action.Configuration) *CatalogCreate {
	return &CatalogCreate{
		config: config,
		Logf:   func(string, ...interface{}) {},
	}
}

func (i *CatalogCreate) Run(ctx context.Context) error {
	catalog := i.buildCatalog()
	if err := i.config.Client.Create(ctx, &catalog); err != nil {
		return err
	}

	var err error
	if i.Available {
		err = waitUntilCatalogStatusCondition(ctx, i.config.Client, &catalog, olmv1.TypeServing, metav1.ConditionTrue)
	} else {
		err = waitUntilCatalogStatusCondition(ctx, i.config.Client, &catalog, olmv1.TypeServing, metav1.ConditionFalse)
	}

	if err != nil {
		if cleanupErr := deleteWithTimeout(i.config.Client, &catalog, i.CleanupTimeout); cleanupErr != nil {
			i.Logf("cleaning up failed catalog: %v", cleanupErr)
		}
		return err
	}

	return nil
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
					Ref:                 i.ImageSourceRef,
					PollIntervalMinutes: &i.PollIntervalMinutes,
				},
			},
			Priority:         i.Priority,
			AvailabilityMode: olmv1.AvailabilityModeAvailable,
		},
	}
	if !i.Available {
		catalog.Spec.AvailabilityMode = olmv1.AvailabilityModeUnavailable
	}

	return catalog
}
