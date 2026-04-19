package cli_test

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/cubbitgg/cmd-drivers/cli"
	"github.com/cubbitgg/cmd-drivers/fsutils"
	"github.com/cubbitgg/cmd-drivers/providers"
	"github.com/cubbitgg/cmd-drivers/tests/loopdev"
	"github.com/cubbitgg/cmd-drivers/tests/mocks"
)

const loopDevSize = 100 * 1024 * 1024 // 100 MiB sparse file — uses no real disk space

func unformattedLSBLK(devices []fsutils.BlockDevice) *mocks.MockLSBLK {
	return &mocks.MockLSBLK{
		GetBlockDevicesFunc: func(_ context.Context, _ fsutils.FilterFunc) ([]fsutils.BlockDevice, error) {
			return devices, nil
		},
	}
}

func TestE2E_Init_FormatExt4(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}
	loopdev.RequireRoot(t)

	dev := loopdev.Create(t, loopDevSize)

	cmd := cli.NewInitCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--debug", "--fs-type", "ext4", "--min-size", "10"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), dev.DevicePath) {
		t.Errorf("expected %s in output; got:\n%s", dev.DevicePath, buf.String())
	}
	if got := loopdev.FSType(t, dev); got != "ext4" {
		t.Errorf("expected filesystem type ext4, got %q", got)
	}
}

func TestE2E_Init_FormatXFS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}
	loopdev.RequireRoot(t)

	if _, err := exec.LookPath("mkfs.xfs"); err != nil {
		t.Skip("mkfs.xfs not available (install xfsprogs)")
	}

	dev := loopdev.Create(t, loopDevSize)

	cmd := cli.NewInitCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--fs-type", "xfs", "--min-size", "0"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), dev.DevicePath) {
		t.Errorf("expected %s in output; got:\n%s", dev.DevicePath, buf.String())
	}
	if got := loopdev.FSType(t, dev); got != "xfs" {
		t.Errorf("expected filesystem type xfs, got %q", got)
	}
}

func TestE2E_Init_DryRunReal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}
	loopdev.RequireRoot(t)

	dev := loopdev.Create(t, loopDevSize)

	cmd := cli.NewInitCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--dry-run", "--min-size", "0"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Would format") {
		t.Errorf("expected 'Would format' in output; got:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), dev.DevicePath) {
		t.Errorf("expected %s in dry-run output; got:\n%s", dev.DevicePath, buf.String())
	}
	if got := loopdev.FSType(t, dev); got != "" {
		t.Errorf("expected no filesystem after dry-run, got %q", got)
	}
}

func TestE2E_Init_MultipleDevices(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}
	loopdev.RequireRoot(t)

	devs := loopdev.CreateN(t, 2, loopDevSize)

	cmd := cli.NewInitCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--fs-type", "ext4", "--min-size", "0"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, dev := range devs {
		if !strings.Contains(buf.String(), dev.DevicePath) {
			t.Errorf("expected %s in output; got:\n%s", dev.DevicePath, buf.String())
		}
		if got := loopdev.FSType(t, dev); got != "ext4" {
			t.Errorf("expected filesystem type ext4 on %s, got %q", dev.DevicePath, got)
		}
	}
}

func TestE2E_Init_SkipsAlreadyFormatted(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}
	loopdev.RequireRoot(t)

	dev := loopdev.Create(t, loopDevSize)

	// Pre-format the device so init should skip it.
	if out, err := exec.Command("mkfs.ext4", "-F", dev.DevicePath).CombinedOutput(); err != nil {
		t.Fatalf("pre-format failed: %v\noutput: %s", err, out)
	}

	cmd := cli.NewInitCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--fs-type", "ext4", "--min-size", "0"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(buf.String(), dev.DevicePath) {
		t.Errorf("expected %s to be skipped (already formatted); got:\n%s", dev.DevicePath, buf.String())
	}
}

func TestIntegration_Init_NoDevicesFound(t *testing.T) {
	cmd := cli.NewInitCmd(
		cli.WithInitLSBLK(unformattedLSBLK(nil)),
		cli.WithFormatProvider(&mocks.MockFormatProvider{}),
	)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "No unformatted devices found") {
		t.Errorf("expected 'No unformatted devices found' in output; got:\n%s", buf.String())
	}
}

func TestIntegration_Init_FormatsDevice(t *testing.T) {
	devices := []fsutils.BlockDevice{
		{Name: "/dev/sdb", Type: "disk", Size: 100 * 1024 * 1024},
	}
	format := &mocks.MockFormatProvider{
		FormatFunc: func(_ context.Context, _ string, _ providers.FormatOptions) error { return nil },
	}

	cmd := cli.NewInitCmd(
		cli.WithInitLSBLK(unformattedLSBLK(devices)),
		cli.WithFormatProvider(format),
	)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--min-size", "0"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "/dev/sdb") {
		t.Errorf("expected /dev/sdb in output; got:\n%s", buf.String())
	}
}

func TestIntegration_Init_DryRun(t *testing.T) {
	devices := []fsutils.BlockDevice{
		{Name: "/dev/sdb", Type: "disk", Size: 100 * 1024 * 1024},
	}

	cmd := cli.NewInitCmd(
		cli.WithInitLSBLK(unformattedLSBLK(devices)),
		cli.WithFormatProvider(&mocks.MockFormatProvider{}), // nil FormatFunc — panics if called
	)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--dry-run", "--min-size", "0"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Would format") {
		t.Errorf("expected 'Would format' in dry-run output; got:\n%s", buf.String())
	}
}

func TestIntegration_Init_FormatError(t *testing.T) {
	devices := []fsutils.BlockDevice{
		{Name: "/dev/sdb", Type: "disk", Size: 100 * 1024 * 1024},
	}
	format := &mocks.MockFormatProvider{
		FormatFunc: func(_ context.Context, _ string, _ providers.FormatOptions) error {
			return errors.New("mkfs failed")
		},
	}

	cmd := cli.NewInitCmd(
		cli.WithInitLSBLK(unformattedLSBLK(devices)),
		cli.WithFormatProvider(format),
	)
	cmd.SetArgs([]string{"--min-size", "0"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected format error, got nil")
	}
}

func TestIntegration_Init_InvalidFSType(t *testing.T) {
	cmd := cli.NewInitCmd()
	cmd.SetArgs([]string{"--fs-type", "zfs"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for unsupported fs type, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported filesystem type") {
		t.Errorf("expected 'unsupported filesystem type' in error, got: %v", err)
	}
}

func TestIntegration_Init_ValidFSType(t *testing.T) {
	for _, fsType := range []string{"ext4", "xfs", "vfat", "ntfs"} {
		t.Run(fsType, func(t *testing.T) {
			cmd := cli.NewInitCmd(
				cli.WithInitLSBLK(unformattedLSBLK(nil)),
				cli.WithFormatProvider(&mocks.MockFormatProvider{}),
			)
			cmd.SetArgs([]string{"--fs-type", fsType})

			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error for fs-type=%s: %v", fsType, err)
			}
		})
	}
}

func TestIntegration_Init_Help(t *testing.T) {
	cmd := cli.NewInitCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error for --help: %v", err)
	}
	if !strings.Contains(buf.String(), "driver-init") {
		t.Errorf("expected 'driver-init' in help output; got:\n%s", buf.String())
	}
}

func TestIntegration_Init_Version(t *testing.T) {
	cmd := cli.NewInitCmd()
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

func TestIntegration_Init_InvalidLogLevel(t *testing.T) {
	cmd := cli.NewInitCmd()
	cmd.SetArgs([]string{"--log-level", "banana"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for invalid log level, got nil")
	}
	if !strings.Contains(err.Error(), "invalid log level") {
		t.Errorf("expected 'invalid log level' in error, got: %v", err)
	}
}
