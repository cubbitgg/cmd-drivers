package cli

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

type mountOpts struct {
	globalOpts

	uuid       string
	mountPoint string
}

func (o *mountOpts) complete() {
	o.completeGlobal()
}

func (o *mountOpts) validate() error {
	if err := o.validateGlobal(); err != nil {
		return err
	}
	if _, err := uuid.Parse(o.uuid); err != nil {
		return fmt.Errorf("--uuid %q is not a valid UUID: %w", o.uuid, err)
	}
	return nil
}

func (o *mountOpts) run(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("driver-mounter not yet implemented")
}

// NewMountCmd returns the cobra command for driver-mounter.
func NewMountCmd() *cobra.Command {
	opts := &mountOpts{}
	cmd := &cobra.Command{
		Use:           "driver-mounter",
		Short:         "Mount a disk by UUID to a mount point",
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
	cmd.Flags().StringVar(&opts.uuid, "uuid", "", "UUID of the disk to mount")
	cmd.Flags().StringVar(&opts.mountPoint, "mount-point", DefaultMountDir, "Mount point base path (a UUID subdirectory will be created under it)")
	_ = cmd.MarkFlagRequired("uuid")
	cmd.AddCommand(newVersionCmd())
	return cmd
}
