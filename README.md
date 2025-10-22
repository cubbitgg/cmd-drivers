# CMD-DRIVERS

## Commands

```bash
# create a file empty of 21MB
dd if=/dev/zero of=mydisk-tiny.img bs=4096 count=5360
# create a file empty of 60MB
dd if=/dev/zero of=mydisk-noformat.img bs=4096 count=15360

# format the file as we do with a disk
mkfs.ext4 mydisk-tiny.img

# create a loop device with a file
sudo losetup -f --show mydisk-tiny.img


lsblk --paths --json  --bytes --output "NAME,TYPE,SIZE,ROTA,SERIAL,WWN,VENDOR,MODEL,REV,MOUNTPOINT,FSTYPE,UUID,PARTUUID"
```

## How to use

```bash
go run cmd/scan/driver-scan.go -log-level error -dir /mnt -min-size 16000000 -fs ext4,vfat,ntfs -debug
```
