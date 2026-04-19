package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cubbitgg/cmd-drivers/fsutils"
	"github.com/cubbitgg/cmd-drivers/logger"
	"github.com/cubbitgg/cmd-drivers/providers"
)

// MountConfig holds parameters for a mount or unmount operation.
type MountConfig struct {
	UUID       string   // filesystem UUID of the device to mount
	MountPoint string   // base directory; the device will be mounted at <MountPoint>/<UUID>
	FSType     string   // filesystem type (e.g. "ext4"); auto-detected via lsblk if empty
	Options    []string // mount options (e.g. ["noatime", "discard"])
}

// DeviceMounter mounts or unmounts a block device identified by UUID.
type DeviceMounter interface {
	Mount(ctx context.Context) error
	Unmount(ctx context.Context) error
}

type deviceMounter struct {
	config   MountConfig
	resolver providers.DeviceResolver
	mount    providers.K8sMountProvider
	lsblk    fsutils.LSBLK
}

// NewDeviceMounter creates a DeviceMounter with injected dependencies.
func NewDeviceMounter(
	config MountConfig,
	resolver providers.DeviceResolver,
	mount providers.K8sMountProvider,
	lsblk fsutils.LSBLK,
) DeviceMounter {
	return &deviceMounter{
		config:   config,
		resolver: resolver,
		mount:    mount,
		lsblk:    lsblk,
	}
}

// Mount resolves the UUID to a device path, creates the target directory, and mounts the device.
// The operation is idempotent: if the target is already mounted, Mount returns nil.
func (m *deviceMounter) Mount(ctx context.Context) error {
	log := logger.FromContext(ctx)

	devicePath, err := m.resolver.ResolveUUID(ctx, m.config.UUID)
	if err != nil {
		return fmt.Errorf("resolve UUID %q: %w", m.config.UUID, err)
	}

	target := filepath.Join(m.config.MountPoint, m.config.UUID)

	if err := os.MkdirAll(target, 0750); err != nil {
		return fmt.Errorf("create mount target %q: %w", target, err)
	}

	notMounted, err := m.mount.IsLikelyNotMountPoint(target)
	if err != nil {
		return fmt.Errorf("check mount point %q: %w", target, err)
	}
	if !notMounted {
		log.Info().Str("target", target).Msg("[mounter] Already mounted, skipping")
		return nil
	}

	fsType := m.config.FSType
	if fsType == "" {
		dev, err := m.lsblk.GetBlockDevice(ctx, devicePath)
		if err != nil {
			return fmt.Errorf("detect filesystem type for %q: %w", devicePath, err)
		}
		fsType = dev.FSType
		if fsType == "" {
			return fmt.Errorf("cannot determine filesystem type for %q: device has no filesystem", devicePath)
		}
	}

	log.Info().
		Str("device", devicePath).
		Str("target", target).
		Str("fstype", fsType).
		Strs("options", m.config.Options).
		Msg("[mounter] Mounting device")

	if err := m.mount.Mount(devicePath, target, fsType, m.config.Options); err != nil {
		return err
	}

	log.Info().Str("device", devicePath).Str("target", target).Msg("[mounter] Mount successful")
	return nil
}

// Unmount unmounts the device at <MountPoint>/<UUID> and removes the empty directory.
// The operation is idempotent: if the target is not mounted, Unmount returns nil.
func (m *deviceMounter) Unmount(ctx context.Context) error {
	log := logger.FromContext(ctx)

	target := filepath.Join(m.config.MountPoint, m.config.UUID)

	if _, err := os.Stat(target); os.IsNotExist(err) {
		log.Info().Str("target", target).Msg("[mounter] Mount point does not exist, nothing to unmount")
		return nil
	}

	notMounted, err := m.mount.IsLikelyNotMountPoint(target)
	if err != nil {
		return fmt.Errorf("check mount point %q: %w", target, err)
	}
	if notMounted {
		log.Info().Str("target", target).Msg("[mounter] Not mounted, skipping")
		return nil
	}

	log.Info().Str("target", target).Msg("[mounter] Unmounting device")

	if err := m.mount.Unmount(target); err != nil {
		return err
	}

	// Best-effort: remove the now-empty directory.
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		log.Warn().Str("target", target).Err(err).Msg("[mounter] Could not remove mount directory after unmount")
	}

	log.Info().Str("target", target).Msg("[mounter] Unmount successful")
	return nil
}
