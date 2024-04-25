package operator

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"

	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
)

type PackageManifest struct {
	operatorsv1.PackageManifest
}

// DefaultChannel is the default argument to specify with GetChannel when you want to get the package's default channel.
const DefaultChannel = ""

// GetChannel returns the specified package channel. DefaultChannel can be used to fetch the package's default channel.
func (pm PackageManifest) GetChannel(channel string) (*PackageChannel, error) {
	if channel == DefaultChannel {
		defaultChannel := pm.GetDefaultChannel()
		if defaultChannel == "" {
			return nil, ErrNoDefaultChannel{pm.GetName()}
		}
		channel = defaultChannel
	}

	var packageChannel *operatorsv1.PackageChannel
	for _, ch := range pm.Status.Channels {
		ch := ch
		if ch.Name == channel {
			packageChannel = &ch
			break
		}
	}
	if packageChannel == nil {
		return nil, ErrChannelNotFound{ChannelName: channel, PackageName: pm.GetName()}
	}
	return &PackageChannel{PackageChannel: *packageChannel}, nil
}

type PackageChannel struct {
	operatorsv1.PackageChannel
}

func (pc PackageChannel) GetSupportedInstallModes() sets.Set[string] {
	supported := sets.New[string]()
	for _, im := range pc.CurrentCSVDesc.InstallModes {
		if im.Supported {
			supported.Insert(string(im.Type))
		}
	}
	return supported
}

type ErrNoDefaultChannel struct {
	PackageName string
}

func (e ErrNoDefaultChannel) Error() string {
	return fmt.Sprintf("package %q does not have a default channel", e.PackageName)
}

type ErrChannelNotFound struct {
	PackageName string
	ChannelName string
}

func (e ErrChannelNotFound) Error() string {
	return fmt.Sprintf("channel %q does not exist for package %q", e.ChannelName, e.PackageName)
}
