package loopdev

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Device holds the sparse file path and the loop device path (e.g. "/dev/loop5").
type Device struct {
	FilePath   string // path to the sparse backing file
	DevicePath string // e.g. "/dev/loop5"
}

// RequireRoot skips the test if not running as root.
func RequireRoot(t *testing.T) {
	t.Helper()
	if os.Getuid() != 0 {
		t.Skip("test requires root (loop device creation needs CAP_SYS_ADMIN)")
	}
}

// Create creates a sparse file of the given size and attaches it as a loop device.
// It registers t.Cleanup() to detach the loop device and remove the sparse file.
// Returns a Device with the file and device paths.
func Create(t *testing.T, size int64) Device {
	t.Helper()
	t.Logf("Creating loop device with size %d bytes", size)

	f, err := os.CreateTemp("", "loopdev-e2e-*.img")
	if err != nil {
		t.Fatalf("loopdev: create temp file: %v", err)
	}
	filePath := f.Name()

	if err := f.Truncate(size); err != nil {
		f.Close()
		os.Remove(filePath)
		t.Fatalf("loopdev: truncate sparse file: %v", err)
	}
	f.Close()

	out, err := exec.Command("losetup", "--find", "--show", filePath).CombinedOutput()
	if err != nil {
		os.Remove(filePath)
		t.Fatalf("loopdev: losetup --find --show: %v\noutput: %s", err, out)
	}
	devicePath := strings.TrimSpace(string(out))
	t.Logf("Created loop device %s for file %s", devicePath, filePath)

	t.Cleanup(func() {
		t.Logf("Cleaning up: detaching loop device %s and removing file %s", devicePath, filePath)
		exec.Command("losetup", "--detach", devicePath).Run() //nolint:errcheck
		os.Remove(filePath)
	})

	return Device{
		FilePath:   filePath,
		DevicePath: devicePath,
	}
}

// CreateN creates n loop devices each of the given size.
func CreateN(t *testing.T, n int, size int64) []Device {
	t.Helper()
	devices := make([]Device, n)
	for i := range devices {
		devices[i] = Create(t, size)
	}
	return devices
}
