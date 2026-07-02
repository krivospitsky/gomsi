package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/krivospitsky/gomsi/internal/backend"
	"github.com/krivospitsky/gomsi/internal/backend/idt"
	"github.com/krivospitsky/gomsi/internal/manifest"
)

// outputFlag is bound to the build command's -o flag.
var outputFlag string

// buildCmd is the "gomsi build" subcommand.
var buildCmd = &cobra.Command{
	Use:   "build <manifest>",
	Short: "Build an MSI from a YAML/JSON manifest",
	Long: "Parse a YAML/JSON manifest and render it into an MSI package. " +
		"The output path defaults to <name>-<version>.msi.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := manifest.Parse(args[0])
		if err != nil {
			return err
		}

		outPath := outputFlag
		if outPath == "" {
			name := m.Product.Name
			if name == "" {
				name = "package"
			}
			ver := m.Product.Version
			if ver == "" {
				ver = "0.0.0"
			}
			outPath = fmt.Sprintf("%s-%s.msi", name, ver)
		}
		if abs, err := filepath.Abs(outPath); err == nil {
			outPath = abs
		}

		var w backend.Writer = idt.New()
		if err := w.Write(m, outPath); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", outPath)
		return nil
	},
}

func init() {
	buildCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "output MSI path (defaults to <name>-<version>.msi)")
	rootCmd.AddCommand(buildCmd)
}
