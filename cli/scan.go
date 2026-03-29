package cli

import (
	"context"
	"fmt"

	"github.com/cubbitgg/cmd-drivers/display"
	"github.com/cubbitgg/cmd-drivers/fsutils"
	"github.com/cubbitgg/cmd-drivers/logger"
	"github.com/cubbitgg/cmd-drivers/providers"
	"github.com/cubbitgg/cmd-drivers/services"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

// ScanOption configures optional dependencies for NewScanCmd.
// Used by tests to inject mock providers; production callers pass no options.
type ScanOption func(*scanOpts)

// WithStatfsProvider injects a StatfsProvider. If not set, the real provider is used.
func WithStatfsProvider(p services.StatfsProvider) ScanOption {
	return func(o *scanOpts) { o.statfsProvider = p }
}

// WithMountInfoProvider injects a MountInfoProvider. If not set, the real provider is used.
func WithMountInfoProvider(p services.MountInfoProvider) ScanOption {
	return func(o *scanOpts) { o.mountInfoProvider = p }
}

// WithLSBLK injects an LSBLK implementation. If not set, the real implementation is used.
func WithLSBLK(l fsutils.LSBLK) ScanOption {
	return func(o *scanOpts) { o.lsblk = l }
}

type scanOpts struct {
	globalOpts

	// raw flag values
	dir     string
	fs      string
	minSize uint64

	// resolved by validate()
	fsTypes []fsutils.FSType

	// injectable last-mile providers (nil = use real implementations)
	statfsProvider    services.StatfsProvider
	mountInfoProvider services.MountInfoProvider
	lsblk             fsutils.LSBLK
}

func (o *scanOpts) complete() {
	o.completeGlobal()
}

func (o *scanOpts) validate() error {
	if err := o.validateGlobal(); err != nil {
		return err
	}
	if o.dir == "" {
		return fmt.Errorf("--dir must not be empty")
	}
	fsTypes, err := fsutils.ParseFSTypes(o.fs)
	if err != nil {
		return err
	}
	if len(fsTypes) == 0 {
		return fmt.Errorf("--fs must include at least one filesystem type")
	}
	o.fsTypes = fsTypes
	return nil
}

func (o *scanOpts) run(cmd *cobra.Command, args []string) error {
	log := logger.InitLogger(o.resolvedLogLevel)
	ctx := logger.WithLogger(context.Background(), log)

	fsTypesStr := fsutils.FSTypesToStrings(o.fsTypes)

	log.Info().
		Str("filter_dir", o.dir).
		Strs("fs_types", fsTypesStr).
		Uint64("min_size", o.minSize).
		Str("min_size_human", humanize.IBytes(o.minSize)).
		Str("log_level", o.resolvedLogLevel).
		Msg("Starting mount list application")

	statfs := o.statfsProvider
	if statfs == nil {
		statfs = providers.NewStatfsProvider()
	}

	mounts := o.mountInfoProvider
	if mounts == nil {
		mounts = providers.NewMountInfoProvider(o.dir, fsTypesStr, o.minSize, statfs)
	}

	lsblk := o.lsblk
	if lsblk == nil {
		lsblk = fsutils.NewLSBLK()
	}

	config := services.ScanConfig{
		DirPrefix: o.dir,
		FSTypes:   fsTypesStr,
		MinSize:   o.minSize,
	}

	scanner := services.NewScanner(config, mounts, statfs, lsblk)

	devices, err := scanner.ScanAll(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Scan failed")
		return fmt.Errorf("scan failed: %w", err)
	}

	log.Info().Int("total_devices", len(devices)).Msg("Scan completed")

	display.DisplayDevices(devices, o.dir, o.fs)
	return nil
}

// NewScanCmd returns the cobra command for driver-scan.
// Pass ScanOption values to inject mock providers (for testing only).
func NewScanCmd(options ...ScanOption) *cobra.Command {
	opts := &scanOpts{}
	for _, o := range options {
		o(opts)
	}
	cmd := &cobra.Command{
		Use:           "driver-scan",
		Short:         "Scan for mounted and unmounted block devices",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.complete()
			if err := opts.validate(); err != nil {
				return err
			}
			return opts.run(cmd, args)
		},
	}
	addGlobalFlags(cmd, &opts.globalOpts)
	cmd.Flags().StringVar(&opts.dir, "dir", DefaultMountDir, "Filter mount points under this directory")
	cmd.Flags().StringVar(&opts.fs, "fs", string(fsutils.FSTypeExt4), "Comma-separated list of filesystem types to include ("+fsutils.ValidFSTypeList()+")")
	cmd.Flags().Uint64Var(&opts.minSize, "min-size", 50*1024*1024, "Minimum mount point size in bytes (default: 50MB)")
	cmd.AddCommand(newVersionCmd())
	return cmd
}
