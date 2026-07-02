package idt

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/krivospitsky/gomsi/internal/model"
)

// gcabArgs builds the command-line arguments for gcab. The staged files should
// be full paths; only their basenames are passed as arguments so the cab-internal
// name matches the Destination. The caller must set cmd.Dir to the staging root.
func gcabArgs(stagedFiles []string, cabPath string) []string {
	args := []string{"--create", cabPath}
	for _, f := range stagedFiles {
		args = append(args, filepath.Base(f))
	}
	return args
}

// genCAB creates a cabinet file containing the given payload files. Each file
// is staged in a temporary directory under its Destination name so that the
// cab-internal filename matches the File table's FileName column.
// genCAB assumes gcab is available in PATH — the caller should check first.
func genCAB(cabPath string, files []model.File) error {
	staging, err := os.MkdirTemp("", "gomsi-cab-*")
	if err != nil {
		return fmt.Errorf("create CAB staging dir: %w", err)
	}
	defer os.RemoveAll(staging)

	var staged []string
	for _, f := range files {
		dst := filepath.Join(staging, f.Destination)
		data, err := os.ReadFile(f.Source)
		if err != nil {
			return fmt.Errorf("read %q for CAB: %w", f.Source, err)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return fmt.Errorf("write CAB staging %q: %w", dst, err)
		}
		staged = append(staged, dst)
	}

	gcabPath, err := exec.LookPath("gcab")
	if err != nil {
		return fmt.Errorf("gcab not found: %w", err)
	}

	args := gcabArgs(staged, cabPath)
	cmd := exec.Command(gcabPath, args...)
	cmd.Dir = staging
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gcab failed: %w\noutput:\n%s", err, out)
	}

	return nil
}
