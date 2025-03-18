package action

import (
	"context"
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ocv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

type ExtensionInstall struct {
	config                         *action.Configuration
	ExtensionName                  string
	Namespace                      NamespaceConfig
	PackageName                    string
	Channels                       []string
	Version                        string
	ServiceAccount                 string
	CatalogSelector                metav1.LabelSelector
	UnsafeCreateClusterRoleBinding bool
	CleanupTimeout                 time.Duration
	Logf                           func(string, ...interface{})
}
type NamespaceConfig struct {
	Name        string
	Labels      map[string]string
	Annotations map[string]string
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
	// Add catalog selector to extension
	if len(i.CatalogSelector.MatchLabels) > 0 {
		extension.Spec.Source.Catalog.Selector = &i.CatalogSelector
	}
	// Add Channels to extension
	if len(i.Channels) > 0 {
		extension.Spec.Source.Catalog.Channels = i.Channels
	}
	//Add CatalogSelector to extension
	if len(i.CatalogSelector.MatchLabels) > 0 {
		extension.Spec.Source.Catalog.Selector = &i.CatalogSelector
	}
	// Create namespace
	/*
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: i.Namespace.Name,
				},
			}

		if err := i.config.Client.Create(ctx, namespace); err != nil {
			return nil, err
		}
	*/
	// Create the extension
	if err := i.config.Client.Create(ctx, &extension); err != nil {
		return nil, err
	}
	clusterExtension, err := i.waitForClusterExtensionInstalled(ctx)
	if err != nil {
		cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), i.CleanupTimeout)
		defer cancelCleanup()
		cleanupErr := i.cleanup(cleanupCtx)
		return nil, errors.Join(err, cleanupErr)
	}
	return clusterExtension, nil
}

func (i *ExtensionInstall) waitForClusterExtensionInstalled(ctx context.Context) (*ocv1.ClusterExtension, error) {
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
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("kubectl-operator-%s-cluster-admin", i.ServiceAccount),
		},
	}
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.Namespace.Name,
			Name:      i.ServiceAccount,
		},
	}
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: i.Namespace.Name,
		},
	}
	if err := waitForDeletion(ctx, i.config.Client, clusterExtension); err != nil {
		return fmt.Errorf("delete clusterextension %q: %v", i.ExtensionName, err)
	}
	return waitForDeletion(ctx, i.config.Client, clusterRoleBinding, serviceAccount, namespace)
}
