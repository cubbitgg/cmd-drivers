package main

import (
	"fmt"
	"os"

	"github.com/cubbitgg/cmd-drivers/cli"
)

func main() {
	cmd := cli.NewScanCmd()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
