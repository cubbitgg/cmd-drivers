package display_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/cubbitgg/cmd-drivers/display"
	"github.com/cubbitgg/cmd-drivers/models"
)

func captureStdout(f func()) string {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r) //nolint:errcheck
	return buf.String()
}

func TestUnit_DisplayDevices_EmptyList(t *testing.T) {
	out := captureStdout(func() {
		display.DisplayDevices(nil, "/mnt", "ext4")
	})
	if !strings.Contains(out, "No devices found") {
		t.Errorf("expected 'No devices found', got: %s", out)
	}
}

func TestUnit_DisplayDevices_WithDevices(t *testing.T) {
	devices := []models.DeviceInfo{
		{
			UUID:      "abc-123",
			Device:    "/dev/sda1",
			MountPath: "/mnt/data",
			FSType:    "ext4",
			Status:    models.StatusMounted,
			TotalSize: 100 << 20,
			FreeSpace: 60 << 20,
			UsedSpace: 40 << 20,
		},
	}

	out := captureStdout(func() {
		display.DisplayDevices(devices, "/mnt", "ext4")
	})

	for _, want := range []string{"UUID", "DEVICE", "MOUNT PATH", "abc-123", "/dev/sda1", "/mnt/data", "ext4", "mounted"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q; full output:\n%s", want, out)
		}
	}
}

func TestUnit_DisplayDevices_EmptyMountPath(t *testing.T) {
	devices := []models.DeviceInfo{
		{
			UUID:      "abc-123",
			Device:    "/dev/sdb",
			MountPath: "",
			FSType:    "ext4",
			Status:    models.StatusNotPartitioned,
			TotalSize: 100 << 20,
		},
	}

	out := captureStdout(func() {
		display.DisplayDevices(devices, "/mnt", "ext4")
	})

	if !strings.Contains(out, "N/A") {
		t.Errorf("expected 'N/A' for empty mount path, got: %s", out)
	}
}

func TestUnit_DisplayDevices_LongDeviceName(t *testing.T) {
	longName := strings.Repeat("a", 30)
	devices := []models.DeviceInfo{
		{UUID: "u", Device: longName, FSType: "ext4", Status: models.StatusMounted, TotalSize: 100 << 20},
	}

	out := captureStdout(func() {
		display.DisplayDevices(devices, "/mnt", "ext4")
	})

	// device column is 20 chars max; long name should be truncated with "..."
	if strings.Contains(out, longName) {
		t.Error("expected long device name to be truncated, but full name appeared in output")
	}
	if !strings.Contains(out, "...") {
		t.Error("expected '...' truncation marker in output")
	}
}
