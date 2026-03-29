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

type scanOpts struct {
	globalOpts

	// raw flag values
	dir     string
	fs      string
	minSize uint64

	// resolved by validate()
	fsTypes []fsutils.FSType
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

	config := services.ScanConfig{
		DirPrefix: o.dir,
		FSTypes:   fsTypesStr,
		MinSize:   o.minSize,
	}

	statfsProvider := providers.NewStatfsProvider()
	scanner := services.NewScanner(
		config,
		providers.NewMountInfoProvider(o.dir, fsTypesStr, o.minSize, statfsProvider),
		statfsProvider,
		fsutils.NewLSBLK(),
	)

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
func NewScanCmd() *cobra.Command {
	opts := &scanOpts{}
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
