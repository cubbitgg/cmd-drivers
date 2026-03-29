package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/cubbitgg/cmd-drivers/cli"
	"github.com/cubbitgg/cmd-drivers/fsutils"
	"github.com/cubbitgg/cmd-drivers/models"
	"github.com/cubbitgg/cmd-drivers/tests/mocks"
)

// captureStdout captures anything written to os.Stdout during f().
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

// mountedDeviceMocks returns mocks that produce one mounted device and no unmounted devices.
func mountedDeviceMocks() (*mocks.MockMountInfoProvider, *mocks.MockStatfsProvider, *mocks.MockLSBLK) {
	mip := &mocks.MockMountInfoProvider{
		GetMountsFunc: func(_ context.Context) ([]models.MountEntry, error) {
			return []models.MountEntry{
				{Source: "/dev/sda1", Mountpoint: "/mnt/cubbit/data", FSType: "ext4"},
			}, nil
		},
	}
	sp := &mocks.MockStatfsProvider{
		StatfsFunc: func(_ string) (*models.StatfsResult, error) {
			return &models.StatfsResult{TotalSize: 100 << 20, FreeSpace: 60 << 20}, nil
		},
	}
	lb := &mocks.MockLSBLK{
		GetUUIDFunc: func(_ context.Context, _ string) (string, error) {
			return "test-uuid-1234", nil
		},
		GetBlockDevicesFunc: func(_ context.Context, _ fsutils.FilterFunc) ([]fsutils.BlockDevice, error) {
			return []fsutils.BlockDevice{}, nil
		},
	}
	return mip, sp, lb
}

// emptyMocks returns mocks that produce no devices.
func emptyMocks() (*mocks.MockMountInfoProvider, *mocks.MockStatfsProvider, *mocks.MockLSBLK) {
	mip := &mocks.MockMountInfoProvider{
		GetMountsFunc: func(_ context.Context) ([]models.MountEntry, error) {
			return []models.MountEntry{}, nil
		},
	}
	sp := &mocks.MockStatfsProvider{}
	lb := &mocks.MockLSBLK{
		GetBlockDevicesFunc: func(_ context.Context, _ fsutils.FilterFunc) ([]fsutils.BlockDevice, error) {
			return []fsutils.BlockDevice{}, nil
		},
	}
	return mip, sp, lb
}

func TestE2E_Scan_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	mip, sp, lb := mountedDeviceMocks()
	cmd := cli.NewScanCmd(
		cli.WithMountInfoProvider(mip),
		cli.WithStatfsProvider(sp),
		cli.WithLSBLK(lb),
	)
	cmd.SetArgs([]string{"--dir", "/mnt/cubbit", "--fs-type", "ext4", "--min-size", "0"})

	var out string
	var execErr error
	out = captureStdout(func() {
		execErr = cmd.Execute()
	})

	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	for _, want := range []string{"test-uuid-1234", "/dev/sda1", "/mnt/cubbit/data", "ext4", "mounted"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q; full output:\n%s", want, out)
		}
	}
}

func TestE2E_Scan_NoDevicesFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	mip, sp, lb := emptyMocks()
	cmd := cli.NewScanCmd(
		cli.WithMountInfoProvider(mip),
		cli.WithStatfsProvider(sp),
		cli.WithLSBLK(lb),
	)
	cmd.SetArgs([]string{"--dir", "/mnt/cubbit", "--fs-type", "ext4", "--min-size", "0"})

	var out string
	var execErr error
	out = captureStdout(func() {
		execErr = cmd.Execute()
	})

	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if !strings.Contains(out, "No devices found") {
		t.Errorf("expected 'No devices found'; full output:\n%s", out)
	}
}

func TestE2E_Scan_MultipleFSTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	mip, sp, lb := emptyMocks()
	cmd := cli.NewScanCmd(
		cli.WithMountInfoProvider(mip),
		cli.WithStatfsProvider(sp),
		cli.WithLSBLK(lb),
	)
	cmd.SetArgs([]string{"--dir", "/mnt/cubbit", "--fs-type", "ext4,xfs", "--min-size", "0"})

	captureStdout(func() {
		if err := cmd.Execute(); err != nil {
			t.Errorf("unexpected error for multi fs-type: %v", err)
		}
	})
}

func TestE2E_Scan_ScanError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	mip := &mocks.MockMountInfoProvider{
		GetMountsFunc: func(_ context.Context) ([]models.MountEntry, error) {
			return nil, errors.New("mountinfo unavailable")
		},
	}
	cmd := cli.NewScanCmd(
		cli.WithMountInfoProvider(mip),
		cli.WithStatfsProvider(&mocks.MockStatfsProvider{}),
		cli.WithLSBLK(&mocks.MockLSBLK{
			GetBlockDevicesFunc: func(_ context.Context, _ fsutils.FilterFunc) ([]fsutils.BlockDevice, error) {
				return nil, nil
			},
		}),
	)
	cmd.SetArgs([]string{"--dir", "/mnt/cubbit", "--fs-type", "ext4", "--min-size", "0"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error from scan failure, got nil")
	}
	if !strings.Contains(err.Error(), "scan failed") {
		t.Errorf("expected 'scan failed' in error, got: %v", err)
	}
}

func TestE2E_Scan_InvalidFSType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	cmd := cli.NewScanCmd()
	cmd.SetArgs([]string{"--fs-type", "btrfs"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for unsupported fs type, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported filesystem type") {
		t.Errorf("expected 'unsupported filesystem type' in error, got: %v", err)
	}
}

func TestE2E_Scan_EmptyDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	cmd := cli.NewScanCmd()
	cmd.SetArgs([]string{"--dir", "", "--fs-type", "ext4"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for empty dir, got nil")
	}
	if !strings.Contains(err.Error(), "--dir must not be empty") {
		t.Errorf("expected '--dir must not be empty' in error, got: %v", err)
	}
}

func TestE2E_Scan_InvalidLogLevel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	cmd := cli.NewScanCmd()
	cmd.SetArgs([]string{"--log-level", "banana"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for invalid log level, got nil")
	}
	if !strings.Contains(err.Error(), "invalid log level") {
		t.Errorf("expected 'invalid log level' in error, got: %v", err)
	}
}

func TestE2E_Scan_DebugOverridesLogLevel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	mip, sp, lb := emptyMocks()
	cmd := cli.NewScanCmd(
		cli.WithMountInfoProvider(mip),
		cli.WithStatfsProvider(sp),
		cli.WithLSBLK(lb),
	)
	cmd.SetArgs([]string{"--debug", "--log-level", "info", "--dir", "/mnt/cubbit", "--fs-type", "ext4", "--min-size", "0"})

	captureStdout(func() {
		if err := cmd.Execute(); err != nil {
			t.Errorf("expected no error when --debug overrides --log-level, got: %v", err)
		}
	})
}

func TestE2E_Scan_Help(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	cmd := cli.NewScanCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error for --help: %v", err)
	}
	if !strings.Contains(buf.String(), "driver-scan") {
		t.Errorf("expected 'driver-scan' in help output; got:\n%s", buf.String())
	}
}

func TestE2E_Scan_Version(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	cmd := cli.NewScanCmd()
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

func TestE2E_Scan_VersionShort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	cmd := cli.NewScanCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"version", "--short"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error for version --short: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if out == "" {
		t.Error("expected non-empty version output for --short")
	}
	// --short should print a single line (no labels like "Commit:", "Built:", etc.)
	if strings.Contains(out, "Commit:") || strings.Contains(out, "Built:") {
		t.Errorf("--short output should be version only, got:\n%s", out)
	}
}
