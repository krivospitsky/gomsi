package idt

import (
	"fmt"
	"os/exec"

	"github.com/krivospitsky/gomsi/internal/model"
)

// msibuildArgs returns the command-line argument vector for msibuild.
// The first element is the MSI output path (msibuild expects it first).
func msibuildArgs(msiPath string, tablePaths []string, cabPath string, p model.Product) []string {
	args := make([]string, 0, 3+2*len(tablePaths)+6)
	args = append(args, msiPath)
	for _, tp := range tablePaths {
		args = append(args, "-i", tp)
	}
	if cabPath != "" {
		args = append(args, "-a", "gomsi.cab", cabPath)
	}
	args = append(args, "-s", p.Name, p.Manufacturer, ";1033", p.ProductCode)
	return args
}

// runMSIBuild invokes msibuild to assemble the MSI package from the given
// table IDT files and embedded cabinet.
//
// workDir is the working directory for the msibuild process. It must be set
// to the temp build directory so that Binary table stream references (loaded
// via g_build_filename("Binary", cellValue)) resolve correctly.
func runMSIBuild(msiPath string, tablePaths []string, cabPath string, p model.Product, workDir string) error {
	msibuildPath, err := exec.LookPath("msibuild")
	if err != nil {
		return fmt.Errorf("msibuild not found: %w", err)
	}

	args := msibuildArgs(msiPath, tablePaths, cabPath, p)
	cmd := exec.Command(msibuildPath, args...)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("msibuild failed: %w\noutput:\n%s", err, out)
	}
	return nil
}
