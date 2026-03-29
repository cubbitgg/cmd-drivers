package services_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/cubbitgg/cmd-drivers/fsutils"
	"github.com/cubbitgg/cmd-drivers/services"
	"github.com/cubbitgg/cmd-drivers/tests/mocks"
)

func TestUnit_Mount_HappyPath(t *testing.T) {
	dir := t.TempDir()
	uuid := "550e8400-e29b-41d4-a716-446655440000"

	resolver := &mocks.MockDeviceResolver{
		ResolveUUIDFunc: func(_ context.Context, _ string) (string, error) {
			return "/dev/sdb1", nil
		},
	}
	mountProv := &mocks.MockK8sMountProvider{
		IsLikelyNotMountPointFunc: func(_ string) (bool, error) { return true, nil },
		MountFunc: func(source, target, fstype string, options []string) error {
			return nil
		},
	}
	lsblk := &mocks.MockLSBLK{
		GetBlockDeviceFunc: func(_ context.Context, _ string) (*fsutils.BlockDevice, error) {
			return &fsutils.BlockDevice{FSType: "ext4"}, nil
		},
	}

	mounter := services.NewDeviceMounter(
		services.MountConfig{UUID: uuid, MountPoint: dir, FSType: "ext4"},
		resolver, mountProv, lsblk,
	)
	if err := mounter.Mount(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestUnit_Mount_AlreadyMounted(t *testing.T) {
	dir := t.TempDir()
	uuid := "550e8400-e29b-41d4-a716-446655440000"

	resolver := &mocks.MockDeviceResolver{
		ResolveUUIDFunc: func(_ context.Context, _ string) (string, error) {
			return "/dev/sdb1", nil
		},
	}
	mountProv := &mocks.MockK8sMountProvider{
		IsLikelyNotMountPointFunc: func(_ string) (bool, error) { return false, nil }, // already mounted
	}
	lsblk := &mocks.MockLSBLK{}

	mounter := services.NewDeviceMounter(
		services.MountConfig{UUID: uuid, MountPoint: dir, FSType: "ext4"},
		resolver, mountProv, lsblk,
	)
	if err := mounter.Mount(context.Background()); err != nil {
		t.Fatalf("expected no error (idempotent), got: %v", err)
	}
}

func TestUnit_Mount_ResolveError(t *testing.T) {
	resolver := &mocks.MockDeviceResolver{
		ResolveUUIDFunc: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("not found")
		},
	}
	mounter := services.NewDeviceMounter(
		services.MountConfig{UUID: "550e8400-e29b-41d4-a716-446655440000", MountPoint: t.TempDir()},
		resolver, &mocks.MockK8sMountProvider{}, &mocks.MockLSBLK{},
	)
	if err := mounter.Mount(context.Background()); err == nil {
		t.Fatal("expected error from resolver, got nil")
	}
}

func TestUnit_Mount_MountError(t *testing.T) {
	dir := t.TempDir()
	uuid := "550e8400-e29b-41d4-a716-446655440000"

	resolver := &mocks.MockDeviceResolver{
		ResolveUUIDFunc: func(_ context.Context, _ string) (string, error) {
			return "/dev/sdb1", nil
		},
	}
	mountProv := &mocks.MockK8sMountProvider{
		IsLikelyNotMountPointFunc: func(_ string) (bool, error) { return true, nil },
		MountFunc: func(_, _, _ string, _ []string) error {
			return errors.New("mount failed")
		},
	}
	lsblk := &mocks.MockLSBLK{}

	mounter := services.NewDeviceMounter(
		services.MountConfig{UUID: uuid, MountPoint: dir, FSType: "ext4"},
		resolver, mountProv, lsblk,
	)
	if err := mounter.Mount(context.Background()); err == nil {
		t.Fatal("expected mount error, got nil")
	}
}

func TestUnit_Mount_AutoDetectFSType(t *testing.T) {
	dir := t.TempDir()
	uuid := "550e8400-e29b-41d4-a716-446655440000"

	resolver := &mocks.MockDeviceResolver{
		ResolveUUIDFunc: func(_ context.Context, _ string) (string, error) {
			return "/dev/sdb1", nil
		},
	}
	mountProv := &mocks.MockK8sMountProvider{
		IsLikelyNotMountPointFunc: func(_ string) (bool, error) { return true, nil },
		MountFunc: func(_, _, fstype string, _ []string) error {
			if fstype != "ext4" {
				t.Errorf("expected fstype ext4, got %q", fstype)
			}
			return nil
		},
	}
	lsblk := &mocks.MockLSBLK{
		GetBlockDeviceFunc: func(_ context.Context, _ string) (*fsutils.BlockDevice, error) {
			return &fsutils.BlockDevice{FSType: "ext4"}, nil
		},
	}

	mounter := services.NewDeviceMounter(
		services.MountConfig{UUID: uuid, MountPoint: dir}, // no FSType
		resolver, mountProv, lsblk,
	)
	if err := mounter.Mount(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestUnit_Mount_NoFSTypeDetected(t *testing.T) {
	dir := t.TempDir()
	uuid := "550e8400-e29b-41d4-a716-446655440000"

	resolver := &mocks.MockDeviceResolver{
		ResolveUUIDFunc: func(_ context.Context, _ string) (string, error) {
			return "/dev/sdb1", nil
		},
	}
	mountProv := &mocks.MockK8sMountProvider{
		IsLikelyNotMountPointFunc: func(_ string) (bool, error) { return true, nil },
	}
	lsblk := &mocks.MockLSBLK{
		GetBlockDeviceFunc: func(_ context.Context, _ string) (*fsutils.BlockDevice, error) {
			return &fsutils.BlockDevice{FSType: ""}, nil // empty
		},
	}

	mounter := services.NewDeviceMounter(
		services.MountConfig{UUID: uuid, MountPoint: dir},
		resolver, mountProv, lsblk,
	)
	if err := mounter.Mount(context.Background()); err == nil {
		t.Fatal("expected error when fstype cannot be detected, got nil")
	}
}

func TestUnit_Unmount_HappyPath(t *testing.T) {
	dir := t.TempDir()
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	target := filepath.Join(dir, uuid)
	if err := os.MkdirAll(target, 0750); err != nil {
		t.Fatal(err)
	}

	mountProv := &mocks.MockK8sMountProvider{
		IsLikelyNotMountPointFunc: func(_ string) (bool, error) { return false, nil }, // is mounted
		UnmountFunc: func(_ string) error {
			return nil
		},
	}

	mounter := services.NewDeviceMounter(
		services.MountConfig{UUID: uuid, MountPoint: dir},
		&mocks.MockDeviceResolver{}, mountProv, &mocks.MockLSBLK{},
	)
	if err := mounter.Unmount(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestUnit_Unmount_NotMounted(t *testing.T) {
	dir := t.TempDir()
	uuid := "550e8400-e29b-41d4-a716-446655440000"

	mountProv := &mocks.MockK8sMountProvider{
		IsLikelyNotMountPointFunc: func(_ string) (bool, error) { return true, nil }, // not mounted
	}

	mounter := services.NewDeviceMounter(
		services.MountConfig{UUID: uuid, MountPoint: dir},
		&mocks.MockDeviceResolver{}, mountProv, &mocks.MockLSBLK{},
	)
	if err := mounter.Unmount(context.Background()); err != nil {
		t.Fatalf("expected no error (idempotent), got: %v", err)
	}
}

func TestUnit_Unmount_Error(t *testing.T) {
	dir := t.TempDir()
	uuid := "550e8400-e29b-41d4-a716-446655440000"

	mountProv := &mocks.MockK8sMountProvider{
		IsLikelyNotMountPointFunc: func(_ string) (bool, error) { return false, nil },
		UnmountFunc: func(_ string) error {
			return errors.New("unmount failed")
		},
	}

	mounter := services.NewDeviceMounter(
		services.MountConfig{UUID: uuid, MountPoint: dir},
		&mocks.MockDeviceResolver{}, mountProv, &mocks.MockLSBLK{},
	)
	if err := mounter.Unmount(context.Background()); err == nil {
		t.Fatal("expected unmount error, got nil")
	}
}
