package cli

import (
	"fmt"

	"github.com/cubbitgg/cmd-drivers/fsutils"
	"github.com/spf13/cobra"
)

type initOpts struct {
	globalOpts

	fsType string
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
	return nil
}

func (o *initOpts) run(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("driver-init not yet implemented")
}

// NewInitCmd returns the cobra command for driver-init.
func NewInitCmd() *cobra.Command {
	opts := &initOpts{}
	cmd := &cobra.Command{
		Use:           "driver-init",
		Short:         "Partition and format available disks",
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
	cmd.AddCommand(newVersionCmd())
	return cmd
}
