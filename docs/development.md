# Development Guide

## Local Test Setup with Loop Devices

Loop devices let you simulate real block devices on any Linux machine without needing physical disks.

### Create test images

```bash
# Small disk (~21 MB) — will be filtered out by --min-size default
dd if=/dev/zero of=/tmp/mydisk-tiny.img bs=4096 count=5360

# Medium disk (~60 MB) — passes default min-size filter
dd if=/dev/zero of=/tmp/mydisk-medium.img bs=4096 count=15360

# Sparse large disk (20 GB, takes no real space)
dd if=/dev/zero of=/tmp/mydisk-large.img bs=1 count=0 seek=20G
```

### Testing driver-scan

```bash
# Format as ext4 and attach as a loop device
mkfs.ext4 /tmp/mydisk-medium.img
sudo losetup -f --show /tmp/mydisk-medium.img   # prints e.g. /dev/loop0

# Mount it under the default scan path
sudo mkdir -p /mnt/cubbit/test
sudo mount /dev/loop0 /mnt/cubbit/test

# Scan
make run-scan ARGS="--dir /mnt/cubbit --fs-type ext4 --min-size 16000000 --debug"

# Cleanup
sudo umount /mnt/cubbit/test
sudo losetup -d /dev/loop0
rm /tmp/mydisk-medium.img
```

### Testing driver-mounter

```bash
# Create and attach an unformatted loop device
dd if=/dev/zero of=/tmp/test-mount.img bs=4096 count=15360
sudo losetup -f --show /tmp/test-mount.img      # e.g. /dev/loop0

# Format it so it has a UUID
sudo mkfs.ext4 /dev/loop0

# Get the UUID assigned by mkfs
sudo lsblk -o NAME,UUID /dev/loop0

# Mount by UUID (creates /mnt/cubbit/<uuid>)
sudo bin/driver-mounter --uuid <UUID> --log-level debug

# Verify
mount | grep <UUID>

# Unmount
sudo bin/driver-mounter --uuid <UUID> --unmount

# Cleanup
sudo losetup -d /dev/loop0
rm /tmp/test-mount.img
```

### Testing driver-init

```bash
# Create a bare (unformatted) loop device
dd if=/dev/zero of=/tmp/test-init.img bs=4096 count=15360
sudo losetup -f --show /tmp/test-init.img       # e.g. /dev/loop0

# Preview what driver-init would format
sudo bin/driver-init --dry-run --min-size 0 --log-level debug

# Format it
sudo bin/driver-init --min-size 0 --log-level debug

# Verify a UUID was assigned
sudo lsblk -o NAME,UUID /dev/loop0

# Cleanup
sudo losetup -d /dev/loop0
rm /tmp/test-init.img
```

---

## lsblk Reference

All three drivers rely on `lsblk` to enumerate block devices. The exact command issued internally is:

```bash
lsblk --paths --json --bytes \
  --output NAME,TYPE,SIZE,ROTA,SERIAL,WWN,VENDOR,MODEL,REV,MOUNTPOINT,FSTYPE,UUID,PARTUUID
```

### Fields used

| Field | Description |
|---|---|
| `NAME` | Full device path (e.g. `/dev/sda`, `/dev/sda1`) |
| `TYPE` | Device type: `disk`, `part`, `loop`, `lvm`, etc. |
| `SIZE` | Size in bytes (parsed as integer) |
| `MOUNTPOINT` | Current mount point, empty if unmounted |
| `FSTYPE` | Filesystem type (e.g. `ext4`, `vfat`) |
| `UUID` | Filesystem UUID |
| `PARTUUID` | Partition UUID (fallback when UUID is absent) |

### Requirements

- `lsblk` must be available on `$PATH` (part of `util-linux`, present on all major Linux distros)
- Must be run as root or with sufficient permissions to read device metadata

---

## Running Tests

```bash
make test                          # run all tests with coverage report
go test ./... -v                   # verbose, includes E2E tests
go test ./... -v -short            # skip E2E tests (unit + integration only)
go test ./services/... -run Unit   # run only unit tests in a package
```

### Test tiers

| Prefix | Description | Deps mocked |
|---|---|---|
| `TestUnit_` | No OS/network/disk interaction | All |
| `TestIntegration_` | Real OS calls (`/proc/self/mountinfo`, `syscall.Statfs`) | None |
| `TestE2E_` | Full cobra pipeline (flag parsing → complete → validate → run) | Last-mile providers |

E2E tests inject mock providers via functional options (`WithLSBLK`, `WithK8sMountProvider`, etc.) so they run without real devices.

---

## Linting

```bash
make lint     # runs golangci-lint (must be installed separately)
```

Install golangci-lint: https://golangci-lint.run/usage/install/
