package main

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"strings"
	"syscall"

	"github.com/cubbitgg/cmd-drivers/fsutils"
	"github.com/cubbitgg/cmd-drivers/logger"
	"github.com/moby/sys/mountinfo"

	"github.com/dustin/go-humanize"
)

type MountDisplay struct {
	UUID      string
	Device    string
	MountPath string
	FSType    string
	Status    string // "mounted", "partitioned", "not partitioned"
	TotalSize uint64
	FreeSpace uint64
	UsedSpace uint64
}

func main() {
	// Command-line flags
	filterDir := flag.String("dir", "/mnt/cubbit", "Filter mount points under this directory")
	fsTypes := flag.String("fs", "ext4", "Comma-separated list of filesystem types to include")
	minSize := flag.Uint64("min-size", 50*1024*1024, "Minimum mount point size in bytes (default: 50MB)")
	logLevel := flag.String("log-level", "", "Log level (debug, info, warn, error) - defaults to warn")
	debug := flag.Bool("debug", false, "Enable debug logging (shorthand for --log-level=debug)")
	flag.Parse()

	// Determine log level
	level := *logLevel
	if *debug {
		level = "debug"
	}

	// Setup logger
	log := logger.InitLogger(level)

	// Create context with logger
	ctx := logger.WithLogger(context.Background(), log)

	log.Info().
		Str("filter_dir", *filterDir).
		Str("fs_types", *fsTypes).
		Uint64("min_size", *minSize).
		Str("min_size_human", humanize.IBytes(*minSize)).
		Str("log_level", level).
		Msg("Starting mount list application")

	// Parse filesystem types
	fsTypeList := parseFSTypes(*fsTypes)
	log.Debug().
		Strs("fs_types", fsTypeList).
		Msg("Parsed filesystem types")

	// Get mounted devices
	mountedDevices := getMountedDevices(ctx, *filterDir, fsTypeList, *minSize)
	log.Info().
		Int("mounted_count", len(mountedDevices)).
		Msg("Retrieved mounted devices")

	// Get unmounted devices
	unmountedDevices := getUnmountedDevices(ctx, fsTypeList, *minSize, mountedDevices)
	log.Info().
		Int("unmounted_count", len(unmountedDevices)).
		Msg("Retrieved unmounted devices")

	// Combine all devices
	allDevices := append(mountedDevices, unmountedDevices...)

	if len(allDevices) == 0 {
		log.Warn().Msg("No devices found")
		fmt.Printf("No devices found\n")
		return
	}

	// Sort by UUID
	sort.Slice(allDevices, func(i, j int) bool {
		return allDevices[i].UUID < allDevices[j].UUID
	})

	// Display results
	fmt.Printf("Block devices (mounted under %s and unmounted with fs=%s):\n\n", *filterDir, *fsTypes)

	fmt.Printf("%-38s %-20s %-30s %-10s %-15s %-15s %-15s %-15s\n",
		"UUID", "DEVICE", "MOUNT PATH", "FS TYPE", "STATUS", "TOTAL SIZE", "FREE SPACE", "USED SPACE")
	fmt.Println(strings.Repeat("-", 180))

	for _, device := range allDevices {
		mountPath := device.MountPath
		if mountPath == "" {
			mountPath = "N/A"
		}

		freeSpace := "N/A"
		usedSpace := "N/A"
		if device.FreeSpace > 0 || device.UsedSpace > 0 {
			freeSpace = humanize.IBytes(device.FreeSpace)
			usedSpace = humanize.IBytes(device.UsedSpace)
		}

		fmt.Printf("%-38s %-20s %-30s %-10s %-15s %-15s %-15s %-15s\n",
			device.UUID,
			truncate(device.Device, 20),
			truncate(mountPath, 30),
			device.FSType,
			device.Status,
			humanize.IBytes(device.TotalSize),
			freeSpace,
			usedSpace)
	}

	log.Info().
		Int("total_devices", len(allDevices)).
		Msg("Display completed successfully")
}

// getMountedDevices retrieves mounted devices from mountinfo
func getMountedDevices(ctx context.Context, filterDir string, fsTypeList []string, minSize uint64) []MountDisplay {
	log := logger.FromContext(ctx)

	// Create filters
	prefixFilter := mountinfo.PrefixFilter(filterDir)
	fsTypeFilter := mountinfo.FSTypeFilter(fsTypeList...)
	sizeFilter := createSizeFilter(minSize)

	// Compose filters with AND logic
	composedFilter := And(prefixFilter, fsTypeFilter, sizeFilter)

	// Get mounts using the composed filter
	log.Debug().Msg("Getting mounts from system")
	mounts, err := mountinfo.GetMounts(composedFilter)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to read mounts")
		return []MountDisplay{}
	}

	// Create lsblk instance for UUID lookup
	lsblkUtil := fsutils.NewLSBLK()

	var devices []MountDisplay
	for _, mount := range mounts {
		log.Debug().
			Str("device", mount.Source).
			Str("mountpoint", mount.Mountpoint).
			Str("fstype", mount.FSType).
			Msg("Processing mount")

		display := MountDisplay{
			UUID:      getUUID(ctx, mount.Source, lsblkUtil),
			Device:    mount.Source,
			MountPath: mount.Mountpoint,
			FSType:    mount.FSType,
			Status:    "mounted",
		}

		// Get disk space information
		var stat syscall.Statfs_t
		if err := syscall.Statfs(mount.Mountpoint, &stat); err == nil {
			display.TotalSize = stat.Blocks * uint64(stat.Bsize)
			display.FreeSpace = stat.Bfree * uint64(stat.Bsize)
			display.UsedSpace = display.TotalSize - display.FreeSpace

			log.Debug().
				Str("mountpoint", mount.Mountpoint).
				Uint64("total_size", display.TotalSize).
				Uint64("free_space", display.FreeSpace).
				Uint64("used_space", display.UsedSpace).
				Msg("Retrieved disk space information")
		} else {
			log.Warn().
				Err(err).
				Str("mountpoint", mount.Mountpoint).
				Msg("Failed to get disk space information")
		}

		devices = append(devices, display)
	}

	return devices
}

// getUnmountedDevices retrieves unmounted devices from lsblk
func getUnmountedDevices(ctx context.Context, fsTypeList []string, minSize uint64, mountedDevices []MountDisplay) []MountDisplay {
	log := logger.FromContext(ctx)

	// Create lsblk instance
	lsblkUtil := fsutils.NewLSBLK()

	// Create filter: must have no mountpoint, match fstype, and be disk/part/loop type
	lsblkFilter := fsutils.And(
		// Skip devices that are mounted
		func(bd *fsutils.BlockDevice) (name string, skip bool, stop bool) {
			return "MountpointEmpty", bd.Mountpoint != "", false
		},
		// Must match one of the filesystem types
		fsutils.FSTypeFilter(fsTypeList...),
		// Only show disk, part, and loop types
		fsutils.TypeFilter("disk", "part", "loop"),
	)

	// Get all block devices
	log.Debug().Msg("Getting block devices from lsblk")
	blockDevices, err := lsblkUtil.GetBlockDevices(ctx, lsblkFilter)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to get block devices")
		return []MountDisplay{}
	}

	var devices []MountDisplay
	for _, bd := range blockDevices {
		// Check if this is a disk with partitions (skip it)
		if bd.Type == "disk" && len(bd.Children) > 0 {
			log.Debug().
				Str("device", bd.Name).
				Int("children", len(bd.Children)).
				Msg("Skipping disk with partitions")
			continue
		}

		// Determine status
		status := "not partitioned"
		if bd.Type == "part" {
			status = "partitioned"
		}

		// Get size
		totalSize := uint64(bd.Size)

		// Skip if smaller than minimum size
		if totalSize < minSize {
			log.Debug().
				Str("device", bd.Name).
				Uint64("size", totalSize).
				Uint64("min_size", minSize).
				Msg("Skipping device smaller than minimum size")
			continue
		}

		uuid := bd.UUID
		if uuid == "" {
			uuid = bd.PartUUID
		}
		if uuid == "" {
			uuid = "N/A"
		}

		display := MountDisplay{
			UUID:      uuid,
			Device:    bd.Name,
			MountPath: "",
			FSType:    bd.FSType,
			Status:    status,
			TotalSize: totalSize,
			FreeSpace: 0,
			UsedSpace: 0,
		}

		log.Debug().
			Str("device", bd.Name).
			Str("type", bd.Type).
			Str("status", status).
			Msg("Added unmounted device")

		devices = append(devices, display)
	}

	return devices
}

// And combines multiple FilterFunc with AND logic.
// All filters must pass (not skip) for the entry to be included.
// If any filter returns stop=true, processing stops immediately.
func And(filters ...mountinfo.FilterFunc) mountinfo.FilterFunc {
	return func(info *mountinfo.Info) (skip bool, stop bool) {
		for _, filter := range filters {
			skip, stop = filter(info)
			if skip || stop {
				return skip, stop
			}
		}
		return false, false
	}
}

// createSizeFilter creates a filter that skips mount points smaller than minSize
func createSizeFilter(minSize uint64) mountinfo.FilterFunc {
	return func(info *mountinfo.Info) (skip bool, stop bool) {
		// Get disk space information
		var stat syscall.Statfs_t
		if err := syscall.Statfs(info.Mountpoint, &stat); err != nil {
			// If we can't get stats, skip this mount
			return true, false
		}

		totalSize := stat.Blocks * uint64(stat.Bsize)

		// Skip if smaller than minimum size
		if totalSize < minSize {
			return true, false
		}

		return false, false
	}
}

// parseFSTypes parses a comma-separated list of filesystem types
func parseFSTypes(fsTypes string) []string {
	parts := strings.Split(fsTypes, ",")
	result := make([]string, 0, len(parts))

	for _, fs := range parts {
		trimmed := strings.TrimSpace(fs)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// getUUID retrieves the UUID for a device using lsblk
func getUUID(ctx context.Context, device string, lsblkUtil fsutils.LSBLK) string {
	log := logger.FromContext(ctx)

	uuid, err := lsblkUtil.GetUUID(ctx, device)
	if err != nil {
		log.Debug().
			Err(err).
			Str("device", device).
			Msg("Failed to get UUID")
		return "N/A"
	}

	if uuid == "" {
		log.Debug().
			Str("device", device).
			Msg("No UUID found for device")
		return "N/A"
	}

	return uuid
}

// truncate truncates a string to a maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
