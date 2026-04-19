package cli

import (
	"context"
	"fmt"

	"github.com/cubbitgg/cmd-drivers/fsutils"
	"github.com/cubbitgg/cmd-drivers/logger"
	"github.com/cubbitgg/cmd-drivers/providers"
	"github.com/cubbitgg/cmd-drivers/services"
	"github.com/spf13/cobra"
)

// InitOption configures optional dependencies for NewInitCmd.
// Used by tests to inject mock providers; production callers pass no options.
type InitOption func(*initOpts)

// WithFormatProvider injects a FormatProvider. If not set, the real provider is used.
func WithFormatProvider(p providers.FormatProvider) InitOption {
	return func(o *initOpts) { o.formatProvider = p }
}

// WithInitLSBLK injects an LSBLK implementation. If not set, the real implementation is used.
func WithInitLSBLK(l fsutils.LSBLK) InitOption {
	return func(o *initOpts) { o.lsblk = l }
}

type initOpts struct {
	globalOpts

	// raw flag values
	fsType  string
	label   string
	minSize uint64
	dryRun  bool

	// injectable last-mile providers (nil = use real implementations)
	formatProvider providers.FormatProvider
	lsblk          fsutils.LSBLK
}

func (o *initOpts) complete() {
	o.completeGlobal()
	if o.fsType == "" {
		o.fsType = string(fsutils.FSTypeExt4)
	}
}

func (o *initOpts) validate() error {
	if err := o.validateGlobal(); err != nil {
		return err
	}
	if !fsutils.IsValidFSType(o.fsType) {
		return fmt.Errorf("unsupported filesystem type %q: must be one of %s", o.fsType, fsutils.ValidFSTypeList())
	}
	if o.label != "" {
		if err := fsutils.ValidateLabel(o.label); err != nil {
			return err
		}
	}
	return nil
}

func (o *initOpts) run(cmd *cobra.Command, args []string) error {
	log := logger.InitLogger(o.resolvedLogLevel)
	ctx := logger.WithLogger(context.Background(), log)

	lsblk := o.lsblk
	if lsblk == nil {
		lsblk = fsutils.NewLSBLK()
	}

	format := o.formatProvider
	if format == nil {
		format = providers.NewFormatProvider()
	}

	config := services.InitConfig{
		FSType:  o.fsType,
		Label:   o.label,
		MinSize: o.minSize,
		DryRun:  o.dryRun,
	}

	initializer := services.NewDiskInitializer(config, lsblk, format)
	formatted, err := initializer.Init(ctx)
	if err != nil {
		return err
	}

	if len(formatted) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No unformatted devices found.")
		return nil
	}

	verb := "Formatted"
	if o.dryRun {
		verb = "Would format"
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s %d device(s) with %s:\n", verb, len(formatted), o.fsType)
	for _, dev := range formatted {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", dev)
	}
	return nil
}

// NewInitCmd returns the cobra command for driver-init.
// Pass InitOption values to inject mock providers (for testing only).
func NewInitCmd(options ...InitOption) *cobra.Command {
	opts := &initOpts{}
	for _, o := range options {
		o(opts)
	}
	cmd := &cobra.Command{
		Use:           "driver-init",
		Short:         "Format unformatted block devices",
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
	cmd.Flags().StringVar(&opts.fsType, "fs-type", "", "Filesystem type for formatting (default: ext4; supported: "+fsutils.ValidFSTypeList()+")")
	cmd.Flags().StringVar(&opts.label, "label", "", "Filesystem label (≤10 chars, A-Z and 0-9 only; portable across all supported fs types)")
	cmd.Flags().Uint64Var(&opts.minSize, "min-size", 50*1024*1024, "Minimum device size in bytes (devices smaller than this are skipped)")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Report what would be formatted without making any changes")
	cmd.AddCommand(newVersionCmd())
	return cmd
}
