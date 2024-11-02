package v1

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"errors"
	catalogdv1 "github.com/operator-framework/catalogd/api/v1"
	"github.com/operator-framework/kubectl-operator/pkg/action"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CatalogAdd struct {
	config *action.Configuration

	CatalogName  string
	Labels       map[string]string
	CatalogImage string
	Priority     int32
	PollInterval time.Duration

	CleanupTimeout time.Duration
	Logf           func(string, ...interface{})
}

func NewCatalogAdd(cfg *action.Configuration) *CatalogAdd {
	return &CatalogAdd{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (a *CatalogAdd) applyClusterCatalog(ctx context.Context) error {
	catalogMetadata := map[string]interface{}{
		"name": a.CatalogName,
	}
	if a.Labels != nil {
		catalogMetadata["labels"] = a.Labels
	}

	catalogImageSource := map[string]interface{}{
		"ref": a.CatalogImage,
	}
	if a.PollInterval != 0 {
		catalogImageSource["pollInterval"] = metav1.Duration{Duration: a.PollInterval}
	}
	u := unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": catalogdv1.GroupVersion.String(),
		"kind":       "ClusterCatalog",
		"metadata":   catalogMetadata,
		"spec": map[string]interface{}{
			"priority": a.Priority,
			"source": map[string]interface{}{
				"type":  catalogdv1.SourceTypeImage,
				"image": catalogImageSource,
			},
		},
	}}
	return patchObject(ctx, a.config.Client, &u)
}

func (a *CatalogAdd) Run(ctx context.Context) (*catalogdv1.ClusterCatalog, error) {
	if err := a.applyClusterCatalog(ctx); err != nil {
		return nil, fmt.Errorf("apply clustercatalog: %v", err)
	}

	clusterCatalog, err := a.waitForCatalogServing(ctx)
	if err != nil {
		cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), a.CleanupTimeout)
		defer cancelCleanup()
		cleanupErr := a.cleanup(cleanupCtx)
		return nil, errors.Join(err, cleanupErr)
	}

	return clusterCatalog, nil
}

func (a *CatalogAdd) waitForCatalogServing(ctx context.Context) (*catalogdv1.ClusterCatalog, error) {
	clusterCatalog := &catalogdv1.ClusterCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: a.CatalogName,
		},
	}

	errMsg := ""
	csKey := client.ObjectKeyFromObject(clusterCatalog)
	if err := wait.PollUntilContextCancel(ctx, time.Millisecond*250, true, func(conditionCtx context.Context) (bool, error) {
		if err := a.config.Client.Get(conditionCtx, csKey, clusterCatalog); err != nil {
			return false, err
		}
		progressingCondition := meta.FindStatusCondition(clusterCatalog.Status.Conditions, catalogdv1.TypeProgressing)
		if progressingCondition != nil && progressingCondition.Reason != catalogdv1.ReasonSucceeded {
			errMsg = progressingCondition.Message
			return false, nil
		}
		if !meta.IsStatusConditionPresentAndEqual(clusterCatalog.Status.Conditions, catalogdv1.TypeServing, metav1.ConditionTrue) {
			return false, nil
		}
		return true, nil
	}); err != nil {
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, fmt.Errorf("clustercatalog %q did not start serving: %s", clusterCatalog.Name, errMsg)
	}
	return clusterCatalog, nil
}

func (a *CatalogAdd) cleanup(ctx context.Context) error {
	clusterCatalog := &catalogdv1.ClusterCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: a.CatalogName,
		},
	}
	return deleteAndWait(ctx, a.config.Client, clusterCatalog)
}
