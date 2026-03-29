package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// globalOpts holds flags common to all commands (log-level, debug).
// Embed this in each command's opts struct and call completeGlobal()/validateGlobal()
// from the command's own complete()/validate() methods.
type globalOpts struct {
	logLevel         string
	debug            bool
	resolvedLogLevel string
}

// completeGlobal resolves the effective log level: --debug overrides --log-level,
// and an empty log-level defaults to "warn".
func (g *globalOpts) completeGlobal() {
	if g.debug {
		g.resolvedLogLevel = "debug"
		return
	}
	if g.logLevel == "" {
		g.resolvedLogLevel = "warn"
		return
	}
	g.resolvedLogLevel = g.logLevel
}

// validateGlobal checks that the resolved log level is one of the accepted values.
func (g *globalOpts) validateGlobal() error {
	valid := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !valid[g.resolvedLogLevel] {
		return fmt.Errorf("invalid log level %q: must be one of debug, info, warn, error", g.resolvedLogLevel)
	}
	return nil
}

// addGlobalFlags registers --log-level and --debug on the given command,
// binding them to the provided globalOpts fields.
func addGlobalFlags(cmd *cobra.Command, g *globalOpts) {
	cmd.Flags().StringVar(&g.logLevel, "log-level", "", "Log level (debug, info, warn, error) — defaults to warn")
	cmd.Flags().BoolVar(&g.debug, "debug", false, "Enable debug logging (shorthand for --log-level=debug)")
}
