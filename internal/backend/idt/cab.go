package idt

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/krivospitsky/gomsi/internal/model"
)

// lcabArgs builds the command-line arguments for lcab. The staged files should
// be basenames already matching the desired cab-internal names.
func lcabArgs(stagedFiles []string, cabPath string) []string {
	args := []string{"-n", "-q"}
	args = append(args, stagedFiles...)
	args = append(args, cabPath)
	return args
}

// genCAB creates a cabinet file containing the given payload files. Each file
// is staged in a temporary directory under its Destination name so that the
// cab-internal filename matches the File table's FileName column.
// genCAB assumes lcab is available in PATH — the caller should check first.
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

	lcabPath, err := exec.LookPath("lcab")
	if err != nil {
		return fmt.Errorf("lcab not found: %w", err)
	}

	args := lcabArgs(staged, cabPath)
	cmd := exec.Command(lcabPath, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("lcab failed: %w\noutput:\n%s", err, out)
	}

	return nil
}
