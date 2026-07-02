package idt

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/krivospitsky/gomsi/internal/backend"
	"github.com/krivospitsky/gomsi/internal/model"
)

// Writer implements backend.Writer using the msitools (IDT + CAB + msibuild) flow.
//
// If EmitDir is set, Write stops after emitting IDT files and (when lcab is
// available) the CAB into that directory, skipping the msibuild step. This is
// useful for development on platforms without msitools/lcab (e.g. Windows).
type Writer struct {
	EmitDir string
}

// New returns an IDT-backed writer.
func New() *Writer { return &Writer{} }

// Compile-time check that the IDT writer satisfies the backend contract.
var _ backend.Writer = (*Writer)(nil)

// Write renders the model to an MSI file.
func (w *Writer) Write(m *model.MSI, outputPath string) error {
	if len(m.Files) == 0 {
		return fmt.Errorf("at least one file is required")
	}

	if err := statFiles(m); err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "gomsi-build-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	tables := coreTables(m)
	tables = append(tables, serviceTables(m)...)
	tables = append(tables, upgradeTables(m)...)
	tables = append(tables, configTables(m)...)

	var tablePaths []string
	for _, tbl := range tables {
		p := filepath.Join(tempDir, tbl.Name+".idt")
		if err := tbl.WriteFile(p); err != nil {
			return fmt.Errorf("write %s.idt: %w", tbl.Name, err)
		}
		tablePaths = append(tablePaths, p)
	}

	// If config is defined, generate the VBScript CA and write the stream
	// sidecar that msibuild loads for the Binary table.
	var configSidecar string
	if m.Config.Template != "" {
		vbs, err := generateVBScript(m)
		if err != nil {
			return fmt.Errorf("generate VBScript CA: %w", err)
		}
		sidecarDir := filepath.Join(tempDir, "Binary")
		if err := os.MkdirAll(sidecarDir, 0755); err != nil {
			return fmt.Errorf("create Binary sidecar dir: %w", err)
		}
		configSidecar = filepath.Join(sidecarDir, "WriteConfig.vbs")
		if err := os.WriteFile(configSidecar, vbs, 0644); err != nil {
			return fmt.Errorf("write VBScript sidecar: %w", err)
		}
	}

	// Generate CAB when lcab is available. In emit mode, skip gracefully
	// when lcab is absent (e.g. Windows dev).
	cabPath := ""
	_, lcabErr := exec.LookPath("lcab")
	if lcabErr == nil {
		cabPath = filepath.Join(tempDir, "gomsi.cab")
		if err := genCAB(cabPath, m.Files); err != nil {
			return fmt.Errorf("generate CAB: %w", err)
		}
	} else if w.EmitDir == "" {
		return fmt.Errorf("lcab not found: required for full MSI build: %w", lcabErr)
	}

	if w.EmitDir != "" {
		return emitToDir(w.EmitDir, tablePaths, cabPath, configSidecar)
	}

	if err := runMSIBuild(outputPath, tablePaths, cabPath, m.Product, tempDir); err != nil {
		return fmt.Errorf("msibuild: %w", err)
	}

	return nil
}

// statFiles reads the size of each payload file and fills m.Files[i].Size.
func statFiles(m *model.MSI) error {
	for i := range m.Files {
		fi, err := os.Stat(m.Files[i].Source)
		if err != nil {
			return fmt.Errorf("stat %q: %w", m.Files[i].Source, err)
		}
		m.Files[i].Size = fi.Size()
	}
	return nil
}

// emitToDir copies all generated IDT files, the CAB (when cabPath is
// non-empty), and the VBScript config sidecar (when configSidecar is
// non-empty) to the given directory.
func emitToDir(dir string, tablePaths []string, cabPath string, configSidecar string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create emit dir: %w", err)
	}

	for _, src := range tablePaths {
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		dst := filepath.Join(dir, filepath.Base(src))
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return err
		}
	}

	if cabPath != "" {
		data, err := os.ReadFile(cabPath)
		if err != nil {
			return err
		}
		dst := filepath.Join(dir, filepath.Base(cabPath))
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return err
		}
	}

	if configSidecar != "" {
		vbsDir := filepath.Join(dir, "Binary")
		if err := os.MkdirAll(vbsDir, 0755); err != nil {
			return fmt.Errorf("create Binary emit dir: %w", err)
		}
		data, err := os.ReadFile(configSidecar)
		if err != nil {
			return err
		}
		dst := filepath.Join(vbsDir, filepath.Base(configSidecar))
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return err
		}
	}

	return nil
}
