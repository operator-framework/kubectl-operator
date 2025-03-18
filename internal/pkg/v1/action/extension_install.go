package action

import (
	"context"
	"errors"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ocv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type ExtensionInstall struct {
	config         *action.Configuration
	ExtensionName  string
	Namespace      NamespaceConfig
	PackageName    string
	Channels       []string
	Version        string
	ServiceAccount string
	CleanupTimeout time.Duration
	Logf           func(string, ...interface{})
}
type NamespaceConfig struct {
	Name string
}

func NewExtensionInstall(cfg *action.Configuration) *ExtensionInstall {
	return &ExtensionInstall{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (i *ExtensionInstall) buildClusterExtension() ocv1.ClusterExtension {
	extension := ocv1.ClusterExtension{
		ObjectMeta: metav1.ObjectMeta{
			Name: i.ExtensionName,
		},
		Spec: ocv1.ClusterExtensionSpec{
			Source: ocv1.SourceConfig{
				SourceType: ocv1.SourceTypeCatalog,
				Catalog: &ocv1.CatalogFilter{
					PackageName: i.PackageName,
					Version:     i.Version,
				},
			},
			Namespace: i.Namespace.Name,
			ServiceAccount: ocv1.ServiceAccountReference{
				Name: i.ServiceAccount,
			},
		},
	}

	return extension
}

func (i *ExtensionInstall) Run(ctx context.Context) (*ocv1.ClusterExtension, error) {
	extension := i.buildClusterExtension()

	// Add Channels to extension
	if len(i.Channels) > 0 {
		extension.Spec.Source.Catalog.Channels = i.Channels
	}

	// TODO: Add CatalogSelector to extension

	// Create the extension
	if err := i.config.Client.Create(ctx, &extension); err != nil {
		return nil, err
	}
	clusterExtension, err := i.waitForExtensionInstall(ctx)
	if err != nil {
		cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), i.CleanupTimeout)
		defer cancelCleanup()
		cleanupErr := i.cleanup(cleanupCtx)
		return nil, errors.Join(err, cleanupErr)
	}
	return clusterExtension, nil
}

// waitForClusterExtensionInstalled waits for the ClusterExtension to be installed
// and returns the ClusterExtension object
func (i *ExtensionInstall) waitForExtensionInstall(ctx context.Context) (*ocv1.ClusterExtension, error) {
	clusterExtension := &ocv1.ClusterExtension{
		ObjectMeta: metav1.ObjectMeta{
			Name: i.ExtensionName,
		},
	}
	errMsg := ""
	key := client.ObjectKeyFromObject(clusterExtension)
	if err := wait.PollUntilContextCancel(ctx, time.Millisecond*250, true, func(conditionCtx context.Context) (bool, error) {
		if err := i.config.Client.Get(conditionCtx, key, clusterExtension); err != nil {
			return false, err
		}
		progressingCondition := meta.FindStatusCondition(clusterExtension.Status.Conditions, ocv1.TypeProgressing)
		if progressingCondition != nil && progressingCondition.Reason != ocv1.ReasonSucceeded {
			errMsg = progressingCondition.Message
			return false, nil
		}
		if !meta.IsStatusConditionPresentAndEqual(clusterExtension.Status.Conditions, ocv1.TypeInstalled, metav1.ConditionTrue) {
			return false, nil
		}
		return true, nil
	}); err != nil {
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, fmt.Errorf("cluster extension %q did not finish installing: %s", clusterExtension.Name, errMsg)
	}
	return clusterExtension, nil
}

func (i *ExtensionInstall) cleanup(ctx context.Context) error {
	clusterExtension := &ocv1.ClusterExtension{
		ObjectMeta: metav1.ObjectMeta{
			Name: i.ExtensionName,
		},
	}
	if err := waitForDeletion(ctx, i.config.Client, clusterExtension); err != nil {
		return fmt.Errorf("delete clusterextension %q: %v", i.ExtensionName, err)
	}
	return nil
}
