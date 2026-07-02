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
func runMSIBuild(msiPath string, tablePaths []string, cabPath string, p model.Product) error {
	msibuildPath, err := exec.LookPath("msibuild")
	if err != nil {
		return fmt.Errorf("msibuild not found: %w", err)
	}

	args := msibuildArgs(msiPath, tablePaths, cabPath, p)
	cmd := exec.Command(msibuildPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("msibuild failed: %w\noutput:\n%s", err, out)
	}
	return nil
}
