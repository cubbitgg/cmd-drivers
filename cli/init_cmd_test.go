package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cubbitgg/cmd-drivers/cli"
)

func TestE2E_Init_DefaultFSType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	cmd := cli.NewInitCmd()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected 'not yet implemented' error, got nil")
	}
	// complete() defaults to ext4, so validation should pass and run() returns "not yet implemented"
	if !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("expected 'not yet implemented' in error, got: %v", err)
	}
}

func TestE2E_Init_ValidFSType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	for _, fsType := range []string{"ext4", "xfs", "vfat", "ntfs"} {
		t.Run(fsType, func(t *testing.T) {
			cmd := cli.NewInitCmd()
			cmd.SetArgs([]string{"--fs-type", fsType})

			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected 'not yet implemented' error, got nil")
			}
			if !strings.Contains(err.Error(), "not yet implemented") {
				t.Errorf("expected 'not yet implemented' for fs-type=%s, got: %v", fsType, err)
			}
		})
	}
}

func TestE2E_Init_InvalidFSType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	cmd := cli.NewInitCmd()
	cmd.SetArgs([]string{"--fs-type", "zfs"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for unsupported fs type, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported filesystem type") {
		t.Errorf("expected 'unsupported filesystem type' in error, got: %v", err)
	}
}

func TestE2E_Init_Help(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	cmd := cli.NewInitCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error for --help: %v", err)
	}
	if !strings.Contains(buf.String(), "driver-init") {
		t.Errorf("expected 'driver-init' in help output; got:\n%s", buf.String())
	}
}

func TestE2E_Init_Version(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	cmd := cli.NewInitCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error for version subcommand: %v", err)
	}
	if !strings.Contains(buf.String(), "Version:") {
		t.Errorf("expected 'Version:' in output; got:\n%s", buf.String())
	}
}

func TestE2E_Init_InvalidLogLevel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	cmd := cli.NewInitCmd()
	cmd.SetArgs([]string{"--log-level", "banana"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for invalid log level, got nil")
	}
	if !strings.Contains(err.Error(), "invalid log level") {
		t.Errorf("expected 'invalid log level' in error, got: %v", err)
	}
}
