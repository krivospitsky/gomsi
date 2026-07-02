// Package cli implements the gomsi command-line interface using cobra.
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the top-level gomsi command.
var rootCmd = &cobra.Command{
	Use:   "gomsi",
	Short: "Linux-first MSI generator for Go binaries",
	Long: "gomsi builds Windows MSI installers for Go binaries on Linux, " +
		"without the Windows SDK. Think of it as \"nfpm for MSI\".",
	SilenceUsage: true,
}

// Execute runs the root command, writing output to the given streams.
// It wires os.Args[1:] by default.
func Execute(stdout, stderr io.Writer) error {
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs(os.Args[1:])
	return rootCmd.Execute()
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.PersistentFlags().BoolP("help", "h", false, "show help for a command")

	// Override the default help to keep output minimal and on-brand.
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		out := cmd.OutOrStdout()
		fmt.Fprint(out, cmd.Long)
		fmt.Fprintln(out)
		fmt.Fprintln(out)
		fmt.Fprintf(out, "Usage:\n  %s\n", cmd.UseLine())
		if cmd.HasAvailableSubCommands() {
			fmt.Fprintln(out, "\nCommands:")
			for _, c := range cmd.Commands() {
				if c.IsAvailableCommand() {
					fmt.Fprintf(out, "  %-10s %s\n", c.Name(), c.Short)
				}
			}
		}
		fmt.Fprintln(out, "\nFlags:")
		fmt.Fprint(out, cmd.Flags().FlagUsages())
	})
}
