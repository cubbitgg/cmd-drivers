# cmd-drivers

A collection of Linux CLI tools for managing block devices — scanning, mounting, and initializing disks.

## Tools

| Command | Status | Description |
|---|---|---|
| `driver-scan` | ready | Discover and report mounted/unmounted block devices |
| `driver-mounter` | planned | Mount a disk by UUID to a configurable mount point |
| `driver-init` | planned | Partition and format an available disk |

## Prerequisites

- Linux (relies on `/proc/self/mountinfo`, `lsblk`, and `syscall.Statfs`)
- Go 1.23+
- `lsblk` (part of `util-linux`, available on all major distros)

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
| `-dir` | `/mnt/cubbit` | Only report mounts under this directory |
| `-fs` | `ext4` | Comma-separated list of filesystem types to include |
| `-min-size` | `52428800` (50 MB) | Minimum device size in bytes |
| `-log-level` | `warn` | Log verbosity: `debug`, `info`, `warn`, `error` |
| `-debug` | `false` | Shorthand for `-log-level debug` |

**Example:**

```bash
bin/driver-scan -dir /mnt -fs ext4,vfat -min-size 16000000
```

**Output:**

```
Block devices (mounted under /mnt and unmounted with fs=ext4,vfat):

UUID                                   DEVICE               MOUNT PATH                     FS TYPE    STATUS          TOTAL SIZE      FREE SPACE      USED SPACE
--------------------------------------------------------------------------------------------------------------------
a1b2c3d4-...                           /dev/sda1            /mnt/data                      ext4       mounted         931 GiB         820 GiB         111 GiB
```

## Project Structure

```
cmd-drivers/
  cmd/
    scan/           driver-scan entry point
    mounter/        driver-mounter entry point (stub)
    init/           driver-init entry point (stub)
  models/           shared data types (DeviceInfo, MountEntry, ...)
  services/         business logic (DeviceScanner)
  providers/        OS abstractions (MountInfoProvider, StatfsProvider)
  display/          shared table renderer
  fsutils/          lsblk wrapper and filter combinators
  logger/           zerolog-based structured logging
  docs/             additional documentation
```

## Development

See [docs/development.md](docs/development.md) for:
- Setting up loop devices as test fixtures
- Full `lsblk` command reference
- Running tests and linting

```bash
make test     # run tests with coverage
make lint     # run golangci-lint
make help     # list all available targets
```

## License

TBD
