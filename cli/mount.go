package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/cubbitgg/cmd-drivers/fsutils"
	"github.com/cubbitgg/cmd-drivers/logger"
	"github.com/cubbitgg/cmd-drivers/providers"
	"github.com/cubbitgg/cmd-drivers/services"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// MountOption configures optional dependencies for NewMountCmd.
// Used by tests to inject mock providers; production callers pass no options.
type MountOption func(*mountOpts)

// WithK8sMountProvider injects a K8sMountProvider. If not set, the real provider is used.
func WithK8sMountProvider(p providers.K8sMountProvider) MountOption {
	return func(o *mountOpts) { o.k8sMountProvider = p }
}

// WithDeviceResolver injects a DeviceResolver. If not set, the real resolver is used.
func WithDeviceResolver(p providers.DeviceResolver) MountOption {
	return func(o *mountOpts) { o.deviceResolver = p }
}

// WithMountLSBLK injects an LSBLK implementation. If not set, the real implementation is used.
func WithMountLSBLK(l fsutils.LSBLK) MountOption {
	return func(o *mountOpts) { o.lsblk = l }
}

type mountOpts struct {
	globalOpts

	// raw flag values
	uuid       string
	mountPoint string
	fsType     string
	options    string // comma-separated, parsed in complete()
	unmount    bool

	// resolved by complete()
	mountOptions []string

	// injectable last-mile providers (nil = use real implementations)
	k8sMountProvider providers.K8sMountProvider
	deviceResolver   providers.DeviceResolver
	lsblk            fsutils.LSBLK
}

func (o *mountOpts) complete() {
	o.completeGlobal()
	if o.options != "" {
		for _, opt := range strings.Split(o.options, ",") {
			if trimmed := strings.TrimSpace(opt); trimmed != "" {
				o.mountOptions = append(o.mountOptions, trimmed)
			}
		}
	}
}

func (o *mountOpts) validate() error {
	if err := o.validateGlobal(); err != nil {
		return err
	}
	if _, err := uuid.Parse(o.uuid); err != nil {
		return fmt.Errorf("--uuid %q is not a valid UUID: %w", o.uuid, err)
	}
	if o.fsType != "" && !fsutils.IsValidFSType(o.fsType) {
		return fmt.Errorf("unsupported filesystem type %q: must be one of %s", o.fsType, fsutils.ValidFSTypeList())
	}
	return nil
}

func (o *mountOpts) run(cmd *cobra.Command, args []string) error {
	log := logger.InitLogger(o.resolvedLogLevel)
	ctx := logger.WithLogger(context.Background(), log)

	lsblk := o.lsblk
	if lsblk == nil {
		lsblk = fsutils.NewLSBLK()
	}

	resolver := o.deviceResolver
	if resolver == nil {
		resolver = providers.NewDeviceResolver(lsblk)
	}

	k8sMount := o.k8sMountProvider
	if k8sMount == nil {
		k8sMount = providers.NewK8sMountProvider()
	}

	config := services.MountConfig{
		UUID:       o.uuid,
		MountPoint: o.mountPoint,
		FSType:     o.fsType,
		Options:    o.mountOptions,
	}

	mounter := services.NewDeviceMounter(config, resolver, k8sMount, lsblk)

	if o.unmount {
		return mounter.Unmount(ctx)
	}
	return mounter.Mount(ctx)
}

// NewMountCmd returns the cobra command for driver-mounter.
// Pass MountOption values to inject mock providers (for testing only).
func NewMountCmd(options ...MountOption) *cobra.Command {
	opts := &mountOpts{}
	for _, o := range options {
		o(opts)
	}
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
	cmd.Flags().StringVar(&opts.mountPoint, "mount-point", DefaultMountDir, "Mount point base path (device mounted at <mount-point>/<uuid>)")
	cmd.Flags().StringVar(&opts.fsType, "fs-type", "", "Filesystem type for mounting (auto-detected if omitted; supported: "+fsutils.ValidFSTypeList()+")")
	cmd.Flags().StringVar(&opts.options, "options", "", "Comma-separated mount options (e.g. noatime,discard)")
	cmd.Flags().BoolVar(&opts.unmount, "unmount", false, "Unmount the device instead of mounting it")
	_ = cmd.MarkFlagRequired("uuid")
	cmd.AddCommand(newVersionCmd())
	return cmd
}
