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

const testValidUUID = "550e8400-e29b-41d4-a716-446655440000"

func TestE2E_Mount_MissingRequiredUUID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

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

func TestE2E_Mount_InvalidUUID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

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

func TestE2E_Mount_InvalidFSType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

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

func TestE2E_Mount_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

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

func TestE2E_Mount_DeviceNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

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

func TestE2E_Mount_Unmount(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

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

func TestE2E_Mount_Help(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

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

func TestE2E_Mount_Version(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

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

func TestE2E_Mount_InvalidLogLevel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

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
