package providers

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/cubbitgg/cmd-drivers/fsutils"
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
	if !fsutils.IsValidFSType(fsType) {
		return fmt.Errorf("unsupported filesystem type %q", fsType)
	}
	cmd := exec.CommandContext(ctx, "mkfs."+fsType, device)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mkfs.%s %q failed: %w\noutput: %s", fsType, device, err, out)
	}
	return nil
}
