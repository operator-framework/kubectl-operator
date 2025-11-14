package action

import (
	"context"
	"errors"
	"fmt"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"

	"github.com/operator-framework/kubectl-operator/pkg/action"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ExtensionDeletion deletes an extension or all extensions in the cluster
type ExtensionDeletion struct {
	config        *action.Configuration
	ExtensionName string

	DeleteAll bool

	DryRun string
	Output string
	Logf   func(string, ...interface{})
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

func (i *ExtensionDeletion) Run(ctx context.Context) ([]olmv1.ClusterExtension, error) {
	if i.DeleteAll && i.ExtensionName != "" {
		return nil, fmt.Errorf("cannot specify both --all and an extension name")
	}
	if !i.DeleteAll {
		ext, err := i.deleteExtension(ctx, i.ExtensionName)
		return []olmv1.ClusterExtension{ext}, err
	}

	// delete all existing extensions
	return i.deleteAllExtensions(ctx)
}

// deleteExtension deletes a single extension in the cluster
func (i *ExtensionDeletion) deleteExtension(ctx context.Context, extName string) (olmv1.ClusterExtension, error) {
	op := &olmv1.ClusterExtension{}
	op.SetName(extName)
	op.SetGroupVersionKind(olmv1.GroupVersion.WithKind("ClusterExtension"))

	if i.DryRun == DryRunAll {
		err := i.config.Client.Delete(ctx, op, client.DryRunAll)
		return *op, err
	}

	err := i.config.Client.Delete(ctx, op)
	if err != nil {
		return *op, err
	}
	// wait for deletion
	return *op, waitForDeletion(ctx, i.config.Client, op)
}

// deleteAllExtensions deletes all extensions in the cluster
func (i *ExtensionDeletion) deleteAllExtensions(ctx context.Context) ([]olmv1.ClusterExtension, error) {
	var extensionList olmv1.ClusterExtensionList
	if err := i.config.Client.List(ctx, &extensionList); err != nil {
		return nil, err
	}
	if len(extensionList.Items) == 0 {
		return nil, ErrNoResourcesFound
	}
	errs := make([]error, 0, len(extensionList.Items))
	result := []olmv1.ClusterExtension{}
	for _, extension := range extensionList.Items {
		if op, err := i.deleteExtension(ctx, extension.Name); err != nil {
			errs = append(errs, fmt.Errorf("failed deleting extension %q: %w", extension.Name, err))
		} else {
			result = append(result, op)
		}
	}
	return result, errors.Join(errs...)
}
