package cli_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cubbitgg/cmd-drivers/cli"
	"github.com/cubbitgg/cmd-drivers/fsutils"
	"github.com/cubbitgg/cmd-drivers/tests/mocks"
)

func unformattedLSBLK(devices []fsutils.BlockDevice) *mocks.MockLSBLK {
	return &mocks.MockLSBLK{
		GetBlockDevicesFunc: func(_ context.Context, _ fsutils.FilterFunc) ([]fsutils.BlockDevice, error) {
			return devices, nil
		},
	}
}

func TestE2E_Init_NoDevicesFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

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

func TestE2E_Init_FormatsDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	devices := []fsutils.BlockDevice{
		{Name: "/dev/sdb", Type: "disk", Size: 100 * 1024 * 1024},
	}
	format := &mocks.MockFormatProvider{
		FormatFunc: func(_ context.Context, _, _ string) error { return nil },
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

func TestE2E_Init_DryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

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

func TestE2E_Init_FormatError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	devices := []fsutils.BlockDevice{
		{Name: "/dev/sdb", Type: "disk", Size: 100 * 1024 * 1024},
	}
	format := &mocks.MockFormatProvider{
		FormatFunc: func(_ context.Context, _, _ string) error {
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

func TestE2E_Init_InvalidFSType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

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

func TestE2E_Init_ValidFSType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

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

func TestE2E_Init_Help(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

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

func TestE2E_Init_Version(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

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

func TestE2E_Init_InvalidLogLevel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

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
