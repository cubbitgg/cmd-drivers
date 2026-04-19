package providers

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"

	"github.com/cubbitgg/cmd-drivers/fsutils"
	"github.com/cubbitgg/cmd-drivers/logger"
)

// FormatProvider formats a block device with a given filesystem type.
type FormatProvider interface {
	Format(ctx context.Context, device, fsType string) error
}

type realFormatProvider struct{}

// NewFormatProvider returns a FormatProvider that invokes mkfs.<fsType>.
func NewFormatProvider() FormatProvider {
	return &realFormatProvider{}
}

func (p *realFormatProvider) Format(ctx context.Context, device, fsType string) error {
	log := logger.FromContext(ctx)
	if !fsutils.IsValidFSType(fsType) {
		return fmt.Errorf("unsupported filesystem type %q", fsType)
	}
	cmd := exec.CommandContext(ctx, "mkfs."+fsType, device)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mkfs.%s %q failed: %w\noutput: %s", fsType, device, err, out)
	}

	syscall.Sync()

	settleCmd := exec.CommandContext(ctx, "udevadm", "settle", "--timeout=5")
	if out, err := settleCmd.CombinedOutput(); err != nil {
		log.Warn().Err(err).Str("device", device).Bytes("output", out).
			Msg("[format] udevadm settle failed after mkfs; lsblk may see stale data briefly")
	}
	return nil
}
