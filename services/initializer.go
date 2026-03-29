package services

import (
	"context"
	"fmt"

	"github.com/cubbitgg/cmd-drivers/fsutils"
	"github.com/cubbitgg/cmd-drivers/logger"
	"github.com/cubbitgg/cmd-drivers/providers"
)

// InitConfig holds parameters for a disk initialization run.
type InitConfig struct {
	FSType  string // filesystem to create (e.g. "ext4")
	MinSize uint64 // skip devices smaller than this (bytes)
	DryRun  bool   // report what would be done without formatting
}

// DiskInitializer finds unformatted block devices and formats them.
type DiskInitializer interface {
	Init(ctx context.Context) ([]string, error) // returns list of formatted (or would-format) device paths
}

type diskInitializer struct {
	config InitConfig
	lsblk  fsutils.LSBLK
	format providers.FormatProvider
}

// NewDiskInitializer creates a DiskInitializer with injected dependencies.
func NewDiskInitializer(config InitConfig, lsblk fsutils.LSBLK, format providers.FormatProvider) DiskInitializer {
	return &diskInitializer{config: config, lsblk: lsblk, format: format}
}

// Init finds all unformatted disks that meet the size threshold and formats them.
// A device is considered unformatted when it has no filesystem type and no UUID.
// Disks that already have partitions (children) are skipped.
// Returns the list of device paths that were formatted (or would be, on dry-run).
func (d *diskInitializer) Init(ctx context.Context) ([]string, error) {
	log := logger.FromContext(ctx)

	devices, err := d.lsblk.GetBlockDevices(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("enumerate block devices: %w", err)
	}

	var targets []string
	for _, bd := range devices {
		// Skip disks that already have partitions — only format bare unpartitioned disks
		if len(bd.Children) > 0 {
			log.Debug().Str("device", bd.Name).Msg("Skipping: disk has partitions")
			continue
		}
		// Skip if already has filesystem or UUID
		if bd.FSType != "" || bd.UUID != "" {
			log.Debug().Str("device", bd.Name).Str("fstype", bd.FSType).Msg("Skipping: already has filesystem")
			continue
		}
		// Skip mounted devices
		if bd.Mountpoint != "" {
			log.Debug().Str("device", bd.Name).Msg("Skipping: device is mounted")
			continue
		}
		if uint64(bd.Size) < d.config.MinSize {
			log.Debug().Str("device", bd.Name).Int64("size", int64(bd.Size)).Msg("Skipping: below minimum size")
			continue
		}

		targets = append(targets, bd.Name)
	}

	if len(targets) == 0 {
		log.Info().Msg("No unformatted devices found")
		return nil, nil
	}

	for _, dev := range targets {
		if d.config.DryRun {
			log.Info().Str("device", dev).Str("fstype", d.config.FSType).Msg("[dry-run] Would format device")
			continue
		}
		log.Info().Str("device", dev).Str("fstype", d.config.FSType).Msg("Formatting device")
		if err := d.format.Format(ctx, dev, d.config.FSType); err != nil {
			return targets, fmt.Errorf("format %q: %w", dev, err)
		}
		log.Info().Str("device", dev).Msg("Format successful")
	}

	return targets, nil
}
