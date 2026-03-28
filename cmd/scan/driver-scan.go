package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/cubbitgg/cmd-drivers/display"
	"github.com/cubbitgg/cmd-drivers/fsutils"
	"github.com/cubbitgg/cmd-drivers/logger"
	"github.com/cubbitgg/cmd-drivers/providers"
	"github.com/cubbitgg/cmd-drivers/services"
	"github.com/dustin/go-humanize"
)

func main() {
	filterDir := flag.String("dir", "/mnt/cubbit", "Filter mount points under this directory")
	fsTypes := flag.String("fs", "ext4", "Comma-separated list of filesystem types to include")
	minSize := flag.Uint64("min-size", 50*1024*1024, "Minimum mount point size in bytes (default: 50MB)")
	logLevel := flag.String("log-level", "", "Log level (debug, info, warn, error) - defaults to warn")
	debug := flag.Bool("debug", false, "Enable debug logging (shorthand for --log-level=debug)")
	flag.Parse()

	level := *logLevel
	if *debug {
		level = "debug"
	}

	log := logger.InitLogger(level)
	ctx := logger.WithLogger(context.Background(), log)

	fsTypeList := parseFSTypes(*fsTypes)

	log.Info().
		Str("filter_dir", *filterDir).
		Str("fs_types", *fsTypes).
		Uint64("min_size", *minSize).
		Str("min_size_human", humanize.IBytes(*minSize)).
		Str("log_level", level).
		Msg("Starting mount list application")

	config := services.ScanConfig{
		DirPrefix: *filterDir,
		FSTypes:   fsTypeList,
		MinSize:   *minSize,
	}

	statfsProvider := providers.NewStatfsProvider()
	scanner := services.NewScanner(
		config,
		providers.NewMountInfoProvider(*filterDir, fsTypeList, *minSize, statfsProvider),
		statfsProvider,
		fsutils.NewLSBLK(),
	)

	devices, err := scanner.ScanAll(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Scan failed")
		fmt.Printf("Error: %v\n", err)
		return
	}

	log.Info().Int("total_devices", len(devices)).Msg("Scan completed")

	display.DisplayDevices(devices, *filterDir, *fsTypes)
}

func parseFSTypes(fsTypes string) []string {
	parts := strings.Split(fsTypes, ",")
	result := make([]string, 0, len(parts))
	for _, fs := range parts {
		if trimmed := strings.TrimSpace(fs); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
