package display

import (
	"fmt"
	"strings"

	"github.com/cubbitgg/cmd-drivers/models"
	"github.com/dustin/go-humanize"
)

// DisplayDevices renders a formatted device table to stdout.
// Shared by all CLI commands that need to present device listings.
func DisplayDevices(devices []models.DeviceInfo, filterDir, fsTypes string) {
	if len(devices) == 0 {
		fmt.Printf("No devices found\n")
		return
	}

	fmt.Printf("Block devices (mounted under %s and unmounted with fs=%s):\n\n", filterDir, fsTypes)
	fmt.Printf("%-38s %-20s %-30s %-10s %-15s %-15s %-15s %-15s\n",
		"UUID", "DEVICE", "MOUNT PATH", "FS TYPE", "STATUS", "TOTAL SIZE", "FREE SPACE", "USED SPACE")
	fmt.Println(strings.Repeat("-", 180))

	for _, device := range devices {
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
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
