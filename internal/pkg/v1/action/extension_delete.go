package action

import (
	"context"
	"errors"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
)

// ExtensionDeletion deletes an extension or all extensions in the cluster
type ExtensionDeletion struct {
	config        *action.Configuration
	ExtensionName string
	DeleteAll     bool
	Logf          func(string, ...interface{})
}

// NewExtensionDelete creates a new ExtensionDeletion action
// with the given configuration
// and a logger function that can be used to log messages
func NewExtensionDelete(cfg *action.Configuration) *ExtensionDeletion {
	return &ExtensionDeletion{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (u *ExtensionDeletion) Run(ctx context.Context) ([]string, error) {
	if u.DeleteAll && u.ExtensionName != "" {
		return nil, fmt.Errorf("cannot specify both --all and an extension name")
	}
	if !u.DeleteAll {
		return u.deleteExtension(ctx, u.ExtensionName)
	}

	// delete all existing extensions
	return u.deleteAllExtensions(ctx)
}

// deleteExtension deletes a single extension in the cluster
func (u *ExtensionDeletion) deleteExtension(ctx context.Context, extName string) ([]string, error) {
	op := &olmv1.ClusterExtension{}
	op.SetName(extName)
	op.SetGroupVersionKind(olmv1.GroupVersion.WithKind("ClusterExtension"))
	lowerKind := strings.ToLower(op.GetObjectKind().GroupVersionKind().Kind)
	err := u.config.Client.Delete(ctx, op)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return []string{u.ExtensionName}, fmt.Errorf("delete %s %q: %v", lowerKind, op.GetName(), err)
		}
		return nil, err
	}
	// wait for deletion
	return []string{u.ExtensionName}, waitForDeletion(ctx, u.config.Client, op)
}

// deleteAllExtensions deletes all extensions in the cluster
func (u *ExtensionDeletion) deleteAllExtensions(ctx context.Context) ([]string, error) {
	var extensionList olmv1.ClusterExtensionList
	if err := u.config.Client.List(ctx, &extensionList); err != nil {
		return nil, err
	}
	if len(extensionList.Items) == 0 {
		return nil, ErrNoResourcesFound
	}
	errs := make([]error, 0, len(extensionList.Items))
	names := make([]string, 0, len(extensionList.Items))
	for _, extension := range extensionList.Items {
		names = append(names, extension.Name)
		if _, err := u.deleteExtension(ctx, extension.Name); err != nil {
			errs = append(errs, fmt.Errorf("failed deleting extension %q: %w", extension.Name, err))
		}
	}
	return names, errors.Join(errs...)
}
