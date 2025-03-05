package action

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	olmv1catalogd "github.com/operator-framework/catalogd/api/v1"
)

type createClient interface {
	creator
	deleter
	getter
}

type CatalogCreate struct {
	client         createClient
	CatalogName    string
	ImageSourceRef string

	Priority            int32
	PollIntervalMinutes int
	Labels              map[string]string
	Available           bool
	CleanupTimeout      time.Duration

	Logf func(string, ...interface{})
}

func NewCatalogCreate(client createClient) *CatalogCreate {
	return &CatalogCreate{
		client: client,
		Logf:   func(string, ...interface{}) {},
	}
}

func (i *CatalogCreate) Run(ctx context.Context) error {
	catalog := i.buildCatalog()
	if err := i.client.Create(ctx, &catalog); err != nil {
		return err
	}

	var err error
	if i.Available {
		err = waitUntilCatalogStatusCondition(ctx, i.client, &catalog, olmv1catalogd.TypeServing, metav1.ConditionTrue)
	} else {
		err = waitUntilCatalogStatusCondition(ctx, i.client, &catalog, olmv1catalogd.TypeServing, metav1.ConditionFalse)
	}

	if err != nil {
		if cleanupErr := deleteWithTimeout(i.client, &catalog, i.CleanupTimeout); cleanupErr != nil {
			i.Logf("cleaning up failed catalog: %v", cleanupErr)
		}
		return err
	}

	return nil
}

func (i *CatalogCreate) buildCatalog() olmv1catalogd.ClusterCatalog {
	catalog := olmv1catalogd.ClusterCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:   i.CatalogName,
			Labels: i.Labels,
		},
		Spec: olmv1catalogd.ClusterCatalogSpec{
			Source: olmv1catalogd.CatalogSource{
				Type: olmv1catalogd.SourceTypeImage,
				Image: &olmv1catalogd.ImageSource{
					Ref:                 i.ImageSourceRef,
					PollIntervalMinutes: &i.PollIntervalMinutes,
				},
			},
			Priority:         i.Priority,
			AvailabilityMode: olmv1catalogd.AvailabilityModeAvailable,
		},
	}
	if !i.Available {
		catalog.Spec.AvailabilityMode = olmv1catalogd.AvailabilityModeUnavailable
	}

	return catalog
}
