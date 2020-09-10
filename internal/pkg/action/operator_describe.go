package action

import (
	"context"
	"fmt"
	"time"

	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
	"k8s.io/apimachinery/pkg/types"
)

type OperatorDescribe struct {
	config *Configuration

	Package         string
	Channel         string
	LongDescription bool
	ShowTimeout     time.Duration

	Logf func(string, ...interface{})
}

func NewOperatorDescribe(cfg *Configuration) *OperatorDescribe {
	return &OperatorDescribe{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (d *OperatorDescribe) Run(ctx context.Context) (*operatorsv1.PackageManifest, *operatorsv1.PackageChannel, error) {
	// get packagemanifest for provided package name
	pm, err := d.getPackageManifest(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get package manifest: %v", err)
	}

	// determine channel from flag or default
	pc, err := d.getPackageChannel(pm)
	if err != nil {
		return nil, nil, fmt.Errorf("get package channel: %v", err)
	}

	return pm, pc, nil
}

// getPackageManifest returns the packagemanifest that matches the namespace and package arguments
// from the API server.
// TODO(): This is pretty much identical to OperatorInstall's getPackageChannel. Might be worth consolidating.
func (d *OperatorDescribe) getPackageManifest(ctx context.Context) (*operatorsv1.PackageManifest, error) {
	pm := &operatorsv1.PackageManifest{}
	key := types.NamespacedName{
		Namespace: d.config.Namespace,
		Name:      d.Package,
	}
	if err := d.config.Client.Get(ctx, key, pm); err != nil {
		return nil, err
	}
	return pm, nil
}

// getPackageChannel returns the package channel specified, or the default if none was specified.
// TODO(): This is pretty much identical to OperatorInstall's getPackageChannel. Might be worth consolidating.
func (d *OperatorDescribe) getPackageChannel(pm *operatorsv1.PackageManifest) (*operatorsv1.PackageChannel, error) {
	if d.Channel == "" {
		d.Channel = pm.Status.DefaultChannel
	}
	var packageChannel *operatorsv1.PackageChannel
	for _, ch := range pm.Status.Channels {
		ch := ch
		if ch.Name == d.Channel {
			packageChannel = &ch
		}
	}
	if packageChannel == nil {
		return nil, fmt.Errorf("channel %q does not exist for package %q", d.Channel, d.Package)
	}
	return packageChannel, nil
}

// GetAvailableChannels lists all available package channels for the operator.
func (d *OperatorDescribe) GetAvailableChannels(pm *operatorsv1.PackageManifest) []string {
	channels := make([]string, len(pm.Status.Channels))
	for i, channel := range pm.Status.Channels {
		n := channel.Name
		if channel.IsDefaultChannel(*pm) {
			n += " (default)"
		}
		if d.Channel == channel.Name {
			n += " (shown)"
		}
		channels[i] = n
	}

	return channels
}

// GetSupportedInstallModes returns a string slice representation of install mode
// objects the operator supports.
func (d *OperatorDescribe) GetSupportedInstallModes(pc *operatorsv1.PackageChannel) []string {
	supportedInstallModes := make([]string, 1)
	for _, imode := range pc.CurrentCSVDesc.InstallModes {
		if imode.Supported {
			supportedInstallModes = append(supportedInstallModes, string(imode.Type))
		}
	}

	return supportedInstallModes
}
