package cli

import (
	"fmt"
	"runtime"

	"github.com/cubbitgg/cmd-drivers/version"
	"github.com/spf13/cobra"
)

type versionOpts struct {
	short bool
}

func (o *versionOpts) run(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	if o.short {
		fmt.Fprintf(out, "%s\n", version.Version)
		return nil
	}
	fmt.Fprintf(out, "Version:    %s\n", version.Version)
	fmt.Fprintf(out, "Commit:     %s\n", version.CommitHash)
	fmt.Fprintf(out, "Built:      %s\n", version.BuildDate)
	fmt.Fprintf(out, "Go version: %s\n", runtime.Version())
	fmt.Fprintf(out, "OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	return nil
}

// newVersionCmd returns the version subcommand, shared by all driver commands.
func newVersionCmd() *cobra.Command {
	opts := &versionOpts{}
	cmd := &cobra.Command{
		Use:          "version",
		Short:        "Print version information",
		SilenceUsage: true,
		Example: `  # Show full version information
  driver-scan version

  # Show only the version number
  driver-scan version --short`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run(cmd, args)
		},
	}
	cmd.Flags().BoolVar(&opts.short, "short", false, "Print only the version number")
	return cmd
}
