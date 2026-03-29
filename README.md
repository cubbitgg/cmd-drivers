# cmd-drivers

A collection of Linux CLI tools for managing block devices — scanning, mounting, and initializing disks. Designed for use inside a Kubernetes CSI local volume driver.

## Tools

| Command | Status | Description |
|---|---|---|
| `driver-scan` | ready | Discover and report mounted/unmounted block devices |
| `driver-mounter` | ready | Mount/unmount a disk by UUID; idempotent (CSI-safe) |
| `driver-init` | ready | Format unformatted block devices with a configurable filesystem |

## Prerequisites

- Linux (relies on `/proc/self/mountinfo`, `lsblk`, `syscall.Statfs`, and mount syscalls)
- Go 1.25+
- `lsblk` (part of `util-linux`, available on all major distros)
- `mkfs.<fstype>` (e2fsprogs, xfsprogs, dosfstools, ntfs-3g) — required only by `driver-init`

## Build

```bash
make build          # build all three binaries into bin/
make build-scan     # build only driver-scan
```

Binaries are placed in `bin/`. The version is derived automatically from `git describe`.

## Usage

### driver-scan

Scans the system for block devices — both mounted (filtered by directory, filesystem type, and minimum size) and unmounted — and prints a summary table.

```bash
bin/driver-scan [flags]
```

| Flag | Default | Description |
|---|---|---|
| `--dir` | `/mnt/cubbit` | Only report mounts under this directory |
| `--fs-type` | `ext4` | Comma-separated list of filesystem types to include |
| `--min-size` | `52428800` (50 MB) | Minimum device size in bytes |
| `--log-level` | `warn` | Log verbosity: `debug`, `info`, `warn`, `error` |
| `--debug` | `false` | Shorthand for `--log-level=debug` |

**Example:**

```bash
bin/driver-scan --dir /mnt --fs-type ext4,vfat --min-size 16000000
```

**Output:**

```
Block devices (mounted under /mnt and unmounted with fs=ext4,vfat):

UUID                                   DEVICE               MOUNT PATH                     FS TYPE    STATUS          TOTAL SIZE      FREE SPACE      USED SPACE
--------------------------------------------------------------------------------------------------------------------
a1b2c3d4-...                           /dev/sda1            /mnt/data                      ext4       mounted         931 GiB         820 GiB         111 GiB
```

---

### driver-mounter

Mounts or unmounts a block device identified by UUID. Designed for use in CSI `NodePublishVolume` / `NodeUnpublishVolume` calls. Both operations are idempotent.

```bash
bin/driver-mounter --uuid <UUID> [flags]
```

| Flag | Default | Description |
|---|---|---|
| `--uuid` | *(required)* | Filesystem UUID of the device to mount |
| `--mount-point` | `/mnt/cubbit` | Base directory; device is mounted at `<mount-point>/<uuid>` |
| `--fs-type` | *(auto-detect)* | Filesystem type (e.g. `ext4`); detected via `lsblk` if omitted |
| `--options` | | Comma-separated mount options (e.g. `noatime,discard`) |
| `--unmount` | `false` | Unmount instead of mount |
| `--log-level` | `warn` | Log verbosity |

**Examples:**

```bash
# Mount a device
bin/driver-mounter --uuid 550e8400-e29b-41d4-a716-446655440000

# Mount with explicit filesystem type and options
bin/driver-mounter --uuid 550e8400-e29b-41d4-a716-446655440000 --fs-type ext4 --options noatime,discard

# Unmount
bin/driver-mounter --uuid 550e8400-e29b-41d4-a716-446655440000 --unmount
```

UUID resolution order:
1. `/dev/disk/by-uuid/<uuid>` symlink (fast, no subprocess)
2. `lsblk` enumeration (fallback)

---

### driver-init

Finds all unformatted block devices and formats them. A device is considered unformatted when it has no filesystem type and no UUID. Disks with existing partitions are skipped.

```bash
bin/driver-init [flags]
```

| Flag | Default | Description |
|---|---|---|
| `--fs-type` | `ext4` | Filesystem to create (`ext4`, `xfs`, `vfat`, `ntfs`) |
| `--min-size` | `52428800` (50 MB) | Skip devices smaller than this (bytes) |
| `--dry-run` | `false` | Report what would be formatted without making changes |
| `--log-level` | `warn` | Log verbosity |

**Examples:**

```bash
# Preview what would be formatted
bin/driver-init --dry-run

# Format with xfs, skip devices smaller than 100 GB
bin/driver-init --fs-type xfs --min-size 107374182400

# Format with default settings (ext4, 50 MB minimum)
bin/driver-init
```

> **Warning:** `driver-init` is a destructive operation. Use `--dry-run` first to verify targets.

---

### version subcommand

Every tool exposes a `version` subcommand:

```bash
bin/driver-scan version
bin/driver-mounter version
bin/driver-init version
```

Use `--short` for the version number only:

```bash
bin/driver-scan version --short
```

## Project Structure

```
cmd-drivers/
  cmd/
    scan/           driver-scan entry point
    mounter/        driver-mounter entry point
    init/           driver-init entry point
  cli/              cobra commands (complete/validate/run pattern)
  models/           shared data types (DeviceInfo, MountEntry, ...)
  services/         business logic (DeviceScanner, DeviceMounter, DiskInitializer)
  providers/        OS abstractions (MountInfoProvider, StatfsProvider, K8sMountProvider, DeviceResolver, FormatProvider)
  display/          shared table renderer
  fsutils/          lsblk wrapper and filter combinators
  logger/           zerolog-based structured logging
  tests/mocks/      func-field test doubles for all provider interfaces
  docs/             additional documentation
```

## Development

See [docs/development.md](docs/development.md) for:
- Setting up loop devices as test fixtures
- Testing driver-mounter and driver-init locally
- Full `lsblk` command reference
- Running tests and linting

```bash
make test     # run tests with coverage
make lint     # run golangci-lint
make help     # list all available targets
```

## License

TBD
