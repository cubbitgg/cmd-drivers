package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cubbitgg/cmd-drivers/cli"
)

const testValidUUID = "550e8400-e29b-41d4-a716-446655440000"

func TestE2E_Mount_MissingRequiredUUID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	cmd := cli.NewMountCmd()
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing required --uuid, got nil")
	}
	if !strings.Contains(err.Error(), "uuid") {
		t.Errorf("expected 'uuid' in error message, got: %v", err)
	}
}

func TestE2E_Mount_InvalidUUID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	cmd := cli.NewMountCmd()
	cmd.SetArgs([]string{"--uuid", "not-a-uuid"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for invalid UUID, got nil")
	}
	if !strings.Contains(err.Error(), "not a valid UUID") {
		t.Errorf("expected 'not a valid UUID' in error, got: %v", err)
	}
}

func TestE2E_Mount_ValidUUID_NotImplemented(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	cmd := cli.NewMountCmd()
	cmd.SetArgs([]string{"--uuid", testValidUUID})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected 'not yet implemented' error, got nil")
	}
	if !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("expected 'not yet implemented' in error, got: %v", err)
	}
}

func TestE2E_Mount_Help(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	cmd := cli.NewMountCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error for --help: %v", err)
	}
	if !strings.Contains(buf.String(), "driver-mounter") {
		t.Errorf("expected 'driver-mounter' in help output; got:\n%s", buf.String())
	}
}

func TestE2E_Mount_Version(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	cmd := cli.NewMountCmd()
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

func TestE2E_Mount_InvalidLogLevel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	cmd := cli.NewMountCmd()
	cmd.SetArgs([]string{"--uuid", testValidUUID, "--log-level", "banana"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validation error for invalid log level, got nil")
	}
	if !strings.Contains(err.Error(), "invalid log level") {
		t.Errorf("expected 'invalid log level' in error, got: %v", err)
	}
}
