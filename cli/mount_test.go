package cli_test

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cubbitgg/cmd-drivers/cli"
	"github.com/cubbitgg/cmd-drivers/fsutils"
	"github.com/cubbitgg/cmd-drivers/tests/loopdev"
	"github.com/cubbitgg/cmd-drivers/tests/mocks"
)

const testValidUUID = "550e8400-e29b-41d4-a716-446655440000"

// ---- Integration tests (mocked providers) ----

func TestIntegration_Mount_MissingRequiredUUID(t *testing.T) {
	cmd := cli.NewMountCmd()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing required --uuid, got nil")
	}
	if !strings.Contains(err.Error(), "uuid") {
		t.Errorf("expected 'uuid' in error message, got: %v", err)
	}
}

func TestIntegration_Mount_InvalidUUID(t *testing.T) {
	cmd := cli.NewMountCmd()
	cmd.SetArgs([]string{"--uuid", "not-a-uuid"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for invalid UUID, got nil")
	}
	if !strings.Contains(err.Error(), "not a valid UUID") {
		t.Errorf("expected 'not a valid UUID' in error, got: %v", err)
	}
}

func TestIntegration_Mount_InvalidFSType(t *testing.T) {
	cmd := cli.NewMountCmd()
	cmd.SetArgs([]string{"--uuid", testValidUUID, "--fs-type", "btrfs"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for invalid fs-type, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported filesystem type") {
		t.Errorf("expected 'unsupported filesystem type' in error, got: %v", err)
	}
}

func TestIntegration_Mount_HappyPath(t *testing.T) {
	dir := t.TempDir()
	resolver := &mocks.MockDeviceResolver{
		ResolveUUIDFunc: func(_ context.Context, _ string) (string, error) {
			return "/dev/sdb1", nil
		},
	}
	mountProv := &mocks.MockK8sMountProvider{
		IsLikelyNotMountPointFunc: func(_ string) (bool, error) { return true, nil },
		MountFunc:                 func(_, _, _ string, _ []string) error { return nil },
	}
	lsblk := &mocks.MockLSBLK{
		GetBlockDeviceFunc: func(_ context.Context, _ string) (*fsutils.BlockDevice, error) {
			return &fsutils.BlockDevice{FSType: "ext4"}, nil
		},
	}

	cmd := cli.NewMountCmd(
		cli.WithDeviceResolver(resolver),
		cli.WithK8sMountProvider(mountProv),
		cli.WithMountLSBLK(lsblk),
	)
	cmd.SetArgs([]string{"--uuid", testValidUUID, "--mount-point", dir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIntegration_Mount_DeviceNotFound(t *testing.T) {
	resolver := &mocks.MockDeviceResolver{
		ResolveUUIDFunc: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("device with UUID not found")
		},
	}

	cmd := cli.NewMountCmd(
		cli.WithDeviceResolver(resolver),
		cli.WithK8sMountProvider(&mocks.MockK8sMountProvider{}),
		cli.WithMountLSBLK(&mocks.MockLSBLK{}),
	)
	cmd.SetArgs([]string{"--uuid", testValidUUID, "--mount-point", t.TempDir()})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when device not found, got nil")
	}
}

func TestIntegration_Mount_Unmount(t *testing.T) {
	mountProv := &mocks.MockK8sMountProvider{
		IsLikelyNotMountPointFunc: func(_ string) (bool, error) { return false, nil }, // is mounted
		UnmountFunc:               func(_ string) error { return nil },
	}

	cmd := cli.NewMountCmd(
		cli.WithDeviceResolver(&mocks.MockDeviceResolver{}),
		cli.WithK8sMountProvider(mountProv),
		cli.WithMountLSBLK(&mocks.MockLSBLK{}),
	)
	cmd.SetArgs([]string{"--uuid", testValidUUID, "--mount-point", t.TempDir(), "--unmount"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error on unmount: %v", err)
	}
}

func TestIntegration_Mount_Help(t *testing.T) {
	cmd := cli.NewMountCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error for --help: %v", err)
	}
	if !strings.Contains(buf.String(), "driver-mounter") {
		t.Errorf("expected 'driver-mounter' in help output; got:\n%s", buf.String())
	}
}

func TestIntegration_Mount_Version(t *testing.T) {
	cmd := cli.NewMountCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error for version subcommand: %v", err)
	}
	if !strings.Contains(buf.String(), "Version:") {
		t.Errorf("expected 'Version:' in output; got:\n%s", buf.String())
	}
}

func TestIntegration_Mount_InvalidLogLevel(t *testing.T) {
	cmd := cli.NewMountCmd()
	cmd.SetArgs([]string{"--uuid", testValidUUID, "--log-level", "banana"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for invalid log level, got nil")
	}
	if !strings.Contains(err.Error(), "invalid log level") {
		t.Errorf("expected 'invalid log level' in error, got: %v", err)
	}
}

// ---- Real E2E tests (real loop devices, no mocks) ----

func TestE2E_Mount_MountAndUnmount(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}
	loopdev.RequireRoot(t)

	dev := loopdev.Create(t, loopDevSize)
	loopdev.Format(t, dev, "ext4")

	devUUID := loopdev.UUID(t, dev)
	mountBase := t.TempDir()
	mountTarget := filepath.Join(mountBase, devUUID)

	// Ensure unmount before loop device is detached (LIFO: runs before loopdev cleanup).
	t.Cleanup(func() {
		t.Log("Ensure unmount cleanup")
		exec.Command("umount", mountTarget).Run() //nolint:errcheck
	})

	// Mount
	mountCmd := cli.NewMountCmd()
	mountCmd.SetArgs([]string{"--debug", "--uuid", devUUID, "--mount-point", mountBase})
	if err := mountCmd.Execute(); err != nil {
		t.Fatalf("mount failed: %v", err)
	}
	if got := loopdev.MountPoint(t, dev); got != mountTarget {
		t.Fatalf("expected device mounted at %s, lsblk reports %q", mountTarget, got)
	}

	// Unmount
	umountCmd := cli.NewMountCmd()
	umountCmd.SetArgs([]string{"--uuid", devUUID, "--mount-point", mountBase, "--unmount"})
	if err := umountCmd.Execute(); err != nil {
		t.Fatalf("unmount failed: %v", err)
	}
	if got := loopdev.MountPoint(t, dev); got != "" {
		t.Errorf("expected device unmounted, lsblk reports mountpoint %q", got)
	}
}

func TestE2E_Mount_AutoDetectFSType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}
	loopdev.RequireRoot(t)

	dev := loopdev.Create(t, loopDevSize)
	loopdev.Format(t, dev, "ext4")

	devUUID := loopdev.UUID(t, dev)
	mountBase := t.TempDir()
	mountTarget := filepath.Join(mountBase, devUUID)

	t.Cleanup(func() {
		exec.Command("umount", mountTarget).Run() //nolint:errcheck
	})

	// Mount without --fs-type: auto-detection via lsblk
	cmd := cli.NewMountCmd()
	cmd.SetArgs([]string{"--debug", "--uuid", devUUID, "--mount-point", mountBase})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("mount with auto-detected fstype failed: %v", err)
	}
	if got := loopdev.MountPoint(t, dev); got != mountTarget {
		t.Errorf("expected device mounted at %s, lsblk reports %q", mountTarget, got)
	}
}

func TestE2E_Mount_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}
	loopdev.RequireRoot(t)

	dev := loopdev.Create(t, loopDevSize)
	loopdev.Format(t, dev, "ext4")

	devUUID := loopdev.UUID(t, dev)
	mountBase := t.TempDir()
	mountTarget := filepath.Join(mountBase, devUUID)

	t.Cleanup(func() {
		exec.Command("umount", mountTarget).Run() //nolint:errcheck
	})

	args := []string{"--debug", "--uuid", devUUID, "--mount-point", mountBase}

	// First mount
	cmd1 := cli.NewMountCmd()
	cmd1.SetArgs(args)
	if err := cmd1.Execute(); err != nil {
		t.Fatalf("first mount failed: %v", err)
	}

	// Second mount — idempotent, must succeed silently
	cmd2 := cli.NewMountCmd()
	cmd2.SetArgs(args)
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("second mount (idempotent) failed: %v", err)
	}

	if got := loopdev.MountPoint(t, dev); got != mountTarget {
		t.Errorf("expected device mounted at %s, lsblk reports %q", mountTarget, got)
	}
}

func TestE2E_Mount_UnmountNotMounted(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}
	loopdev.RequireRoot(t)

	dev := loopdev.Create(t, loopDevSize)
	loopdev.Format(t, dev, "ext4")

	devUUID := loopdev.UUID(t, dev)
	mountBase := t.TempDir()

	// Unmount without having mounted — idempotent, must succeed silently
	cmd := cli.NewMountCmd()
	cmd.SetArgs([]string{"--debug", "--uuid", devUUID, "--mount-point", mountBase, "--unmount"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unmount of non-mounted device failed: %v", err)
	}
}
