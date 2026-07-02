// Package main is the gomsi command-line entrypoint.
package main

import (
	"fmt"
	"os"

	"github.com/krivospitsky/gomsi/internal/cli"
)

func main() {
	if err := cli.Execute(os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "gomsi: %v\n", err)
		os.Exit(1)
	}
}
