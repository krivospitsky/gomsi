package idt

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/krivospitsky/gomsi/internal/model"
)

func TestWriter_Emit(t *testing.T) {
	dir := t.TempDir()

	// Create a fake payload file.
	payload := filepath.Join(dir, "myagent.exe")
	if err := os.WriteFile(payload, []byte("fake exe content"), 0644); err != nil {
		t.Fatal(err)
	}

	emitDir := filepath.Join(dir, "emit")
	m := &model.MSI{
		Product: model.Product{
			Name:         "TestApp",
			Version:      "2.0.0",
			Manufacturer: "TestCorp",
			UpgradeCode:  "{aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa}",
			ProductCode:  "{bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb}",
		},
		Install: model.Install{Directory: "TestApp"},
		Files: []model.File{
			{Source: payload, Destination: "myagent.exe"},
		},
	}

	w := &Writer{EmitDir: emitDir}
	outPath := filepath.Join(dir, "out.msi")

	if err := w.Write(m, outPath); err != nil {
		t.Fatal(err)
	}

	// Check IDT files exist.
	expectedIDT := []string{
		"Property.idt", "Directory.idt", "Component.idt",
		"Feature.idt", "FeatureComponents.idt", "File.idt",
		"Media.idt", "InstallExecuteSequence.idt", "InstallUISequence.idt",
	}
	for _, name := range expectedIDT {
		p := filepath.Join(emitDir, name)
		fi, err := os.Stat(p)
		if err != nil {
			t.Errorf("missing emitted file: %s (%v)", name, err)
			continue
		}
		if fi.Size() == 0 {
			t.Errorf("emitted file %s is empty", name)
		}
	}

	// CAB is emitted only when lcab is available (Linux).
	if _, err := exec.LookPath("lcab"); err == nil {
		p := filepath.Join(emitDir, "gomsi.cab")
		fi, err := os.Stat(p)
		if err != nil {
			t.Errorf("missing emitted CAB: %v", err)
		} else if fi.Size() == 0 {
			t.Errorf("emitted CAB is empty")
		}
	}
}

func TestWriter_Emit_WithService(t *testing.T) {
	dir := t.TempDir()

	payload := filepath.Join(dir, "myagent.exe")
	if err := os.WriteFile(payload, []byte("fake exe content"), 0644); err != nil {
		t.Fatal(err)
	}

	emitDir := filepath.Join(dir, "emit")
	m := &model.MSI{
		Product: model.Product{
			Name:         "TestApp",
			Version:      "2.0.0",
			Manufacturer: "TestCorp",
			UpgradeCode:  "{aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa}",
			ProductCode:  "{bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb}",
		},
		Install: model.Install{Directory: "TestApp"},
		Files: []model.File{
			{Source: payload, Destination: "myagent.exe"},
		},
		Services: []model.Service{
			{
				Name:        "myagent",
				DisplayName: "My Agent",
				Description: "Monitoring Agent",
				Start:       "auto",
			},
		},
	}

	w := &Writer{EmitDir: emitDir}
	outPath := filepath.Join(dir, "out.msi")

	if err := w.Write(m, outPath); err != nil {
		t.Fatal(err)
	}

	expectedIDT := []string{
		"Property.idt", "Directory.idt", "Component.idt",
		"Feature.idt", "FeatureComponents.idt", "File.idt",
		"Media.idt", "InstallExecuteSequence.idt", "InstallUISequence.idt",
		"ServiceInstall.idt", "ServiceControl.idt",
	}
	for _, name := range expectedIDT {
		p := filepath.Join(emitDir, name)
		fi, err := os.Stat(p)
		if err != nil {
			t.Errorf("missing emitted file: %s (%v)", name, err)
			continue
		}
		if fi.Size() == 0 {
			t.Errorf("emitted file %s is empty", name)
		}
	}

	// Verify the service InstallExecuteSequence has the expected number of rows.
	seqPath := filepath.Join(emitDir, "InstallExecuteSequence.idt")
	data, err := os.ReadFile(seqPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	// Header + PK row + 3 header rows? Actually IDT has col names (line1),
	// col types (line2), PK (line3), then 14 data rows = 17 total lines.
	if lines != 17 {
		t.Errorf("InstallExecuteSequence.idt: got %d lines, want 17 (3 header + 14 data)", lines)
	}
}

func TestWriter_Emit_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	payload := filepath.Join(dir, "payload.bin")
	if err := os.WriteFile(payload, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Emit dir doesn't exist yet.
	emitDir := filepath.Join(dir, "nested", "emit")
	m := &model.MSI{
		Product: model.Product{
			Name:         "X",
			Version:      "1",
			Manufacturer: "Y",
			ProductCode:  "{cccccccc-cccc-cccc-cccc-cccccccccccc}",
		},
		Install: model.Install{Directory: "X"},
		Files: []model.File{
			{Source: payload, Destination: "x.exe"},
		},
	}

	w := &Writer{EmitDir: emitDir}
	if err := w.Write(m, "nopath"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(emitDir, "Property.idt")); err != nil {
		t.Errorf("emit dir was not created: %v", err)
	}
}

func TestWriter_ZeroFiles(t *testing.T) {
	m := &model.MSI{
		Product: model.Product{Name: "X", Version: "1", Manufacturer: "Y", ProductCode: "{00000000-0000-0000-0000-000000000000}"},
		Install: model.Install{Directory: "X"},
	}

	w := &Writer{}
	err := w.Write(m, "x.msi")
	if err == nil {
		t.Fatal("expected error for zero files")
	}
}

func TestWriter_SourceNotExist(t *testing.T) {
	dir := t.TempDir()
	m := &model.MSI{
		Product: model.Product{Name: "X", Version: "1", Manufacturer: "Y", ProductCode: "{00000000-0000-0000-0000-000000000000}"},
		Install: model.Install{Directory: "X"},
		Files: []model.File{
			{Source: filepath.Join(dir, "nonexistent.exe"), Destination: "x.exe"},
		},
	}

	w := &Writer{}
	err := w.Write(m, "x.msi")
	if err == nil {
		t.Fatal("expected error for non-existent source file")
	}
}

func TestWriter_FullBuild(t *testing.T) {
	if _, err := exec.LookPath("msibuild"); err != nil {
		t.Skip("msibuild not available:", err)
	}
	if _, err := exec.LookPath("lcab"); err != nil {
		t.Skip("lcab not available:", err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("msibuild/lcab are Linux-only")
	}

	dir := t.TempDir()
	payload := filepath.Join(dir, "myagent.exe")
	if err := os.WriteFile(payload, []byte("fake exe for full build"), 0644); err != nil {
		t.Fatal(err)
	}

	m := &model.MSI{
		Product: model.Product{
			Name:         "FullTest",
			Version:      "1.0.0",
			Manufacturer: "TestCo",
			UpgradeCode:  "{dddddddd-dddd-dddd-dddd-dddddddddddd}",
			ProductCode:  "{eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee}",
		},
		Install: model.Install{Directory: "FullTest"},
		Files: []model.File{
			{Source: payload, Destination: "myagent.exe"},
		},
	}

	msiPath := filepath.Join(dir, "output.msi")
	w := &Writer{}
	if err := w.Write(m, msiPath); err != nil {
		t.Fatal(err)
	}

	fi, err := os.Stat(msiPath)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() == 0 {
		t.Fatal("MSI output is empty")
	}
	t.Logf("Full MSI build: %s (%d bytes)", msiPath, fi.Size())
}
