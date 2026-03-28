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

### Format and attach

```bash
# Format as ext4
mkfs.ext4 /tmp/mydisk-medium.img

# Attach as a loop device (prints the device path, e.g. /dev/loop0)
sudo losetup -f --show /tmp/mydisk-medium.img

# Mount it under the default scan path
sudo mkdir -p /mnt/cubbit/test
sudo mount /dev/loop0 /mnt/cubbit/test
```

### Verify with driver-scan

```bash
make run-scan ARGS="-dir /mnt/cubbit -fs ext4 -min-size 16000000 -debug"
```

### Cleanup

```bash
sudo umount /mnt/cubbit/test
sudo losetup -d /dev/loop0
rm /tmp/mydisk-medium.img
```

---

## lsblk Reference

`driver-scan` and related tools rely on `lsblk` to enumerate block devices and retrieve UUIDs. The exact command issued internally is:

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
make test         # runs go test ./... with coverage report
make lint         # runs golangci-lint (must be installed separately)
```

Install golangci-lint: https://golangci-lint.run/usage/install/
