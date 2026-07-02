package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/krivospitsky/gomsi/internal/backend/idt"
	"github.com/krivospitsky/gomsi/internal/manifest"
)

// outputFlag is bound to the build command's -o flag.
var outputFlag string

// emitFlag is bound to the build command's --emit flag.
var emitFlag string

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

		// Resolve file sources and config template relative to the manifest file's directory.
		manifestDir := filepath.Dir(args[0])
		for i := range m.Files {
			if !filepath.IsAbs(m.Files[i].Source) {
				m.Files[i].Source = filepath.Join(manifestDir, m.Files[i].Source)
			}
		}
		if m.Config.Template != "" && !filepath.IsAbs(m.Config.Template) {
			m.Config.Template = filepath.Join(manifestDir, m.Config.Template)
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

		w := idt.New()
		w.EmitDir = emitFlag

		if err := w.Write(m, outPath); err != nil {
			return err
		}

		if emitFlag != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "emitted IDT + CAB to %s\n", emitFlag)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", outPath)
		}
		return nil
	},
}

func init() {
	buildCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "output MSI path (defaults to <name>-<version>.msi)")
	buildCmd.Flags().StringVarP(&emitFlag, "emit", "", "", "stop after emitting IDT + CAB to the given directory (skip msibuild)")
	rootCmd.AddCommand(buildCmd)
}
