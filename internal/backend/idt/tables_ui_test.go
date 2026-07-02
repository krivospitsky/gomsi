package idt

import (
	"bytes"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/krivospitsky/gomsi/internal/model"
)

// testModelWithUI returns a model fixture with visible parameters for UI tests.
func testModelWithUI() *model.MSI {
	m := testModel()
	m.Parameters = []model.Parameter{
		{Name: "serverUrl", Property: "SERVERURL", Type: "string", Title: "Server URL", Required: true, Default: "http://localhost:8080"},
		{Name: "token", Property: "TOKEN", Type: "password", Title: "Auth Token", Required: false, Default: ""},
		{Name: "debug", Property: "DEBUG", Type: "string", Title: "", Required: false, Default: "false", UI: "never"},
	}
	return m
}

func TestHasVisibleParam(t *testing.T) {
	cases := []struct {
		name string
		params []model.Parameter
		want bool
	}{
		{"nil", nil, false},
		{"empty", []model.Parameter{}, false},
		{"visible", []model.Parameter{{Name: "x", Property: "X"}}, true},
		{"never", []model.Parameter{{Name: "x", Property: "X", UI: "never"}}, false},
		{"auto", []model.Parameter{{Name: "x", Property: "X", UI: "auto"}}, true},
		{"always", []model.Parameter{{Name: "x", Property: "X", UI: "always"}}, true},
		{"mixed", []model.Parameter{
			{Name: "a", Property: "A", UI: "never"},
			{Name: "b", Property: "B", UI: "auto"},
		}, true},
		{"all_never", []model.Parameter{
			{Name: "a", Property: "A", UI: "never"},
			{Name: "b", Property: "B", UI: "never"},
		}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := &model.MSI{Parameters: c.params}
			got := hasVisibleParam(m)
			if got != c.want {
				t.Errorf("hasVisibleParam = %v, want %v", got, c.want)
			}
		})
	}
}

func TestUITables_NilWhenNoVisibleParams(t *testing.T) {
	m := &model.MSI{Parameters: []model.Parameter{
		{Name: "x", Property: "X", UI: "never"},
	}}
	tables := uiTables(m)
	if tables != nil {
		t.Errorf("expected nil, got %d tables", len(tables))
	}
}

func TestUITables_NilWhenEmpty(t *testing.T) {
	m := &model.MSI{}
	tables := uiTables(m)
	if tables != nil {
		t.Errorf("expected nil, got %d tables", len(tables))
	}
}

func TestUITables_Golden(t *testing.T) {
	m := testModelWithUI()
	tables := uiTables(m)
	if tables == nil {
		t.Fatal("expected non-nil for model with visible params")
	}

	for _, tbl := range tables {
		got, err := tbl.Render()
		if err != nil {
			t.Fatalf("%s: Render: %v", tbl.Name, err)
		}

		path := filepath.Join("testdata", "ui", tbl.Name+".idt")
		if *update {
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, got, 0644); err != nil {
				t.Fatalf("%s: WriteFile: %v", tbl.Name, err)
			}
		} else {
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%s: ReadFile: %v", tbl.Name, err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("%s.idt: golden mismatch; run `go test -update` to regenerate\n  got:\n%s\n  want:\n%s",
					tbl.Name, hex.Dump(got), hex.Dump(want))
			}
		}
	}
}

func TestApplyUIProperties(t *testing.T) {
	m := testModel()
	m.Parameters = []model.Parameter{
		{Name: "x", Property: "X"},
	}

	// Build Property table as core would, then apply UI properties.
	tables := coreTables(m)
	propTbl := findTable(tables, "Property")
	applyUIProperties(propTbl, m)

	rendered, err := propTbl.Render()
	if err != nil {
		t.Fatal(err)
	}

	s := string(rendered)
	if !containsLine(s, "DefaultUIFont\t{\\DlgFont8}") {
		t.Errorf("Property.idt missing DefaultUIFont")
	}
	if !containsLine(s, "ButtonText_Next\t&Next >") {
		t.Errorf("Property.idt missing ButtonText_Next")
	}
	if !containsLine(s, "ButtonText_Back\t< &Back") {
		t.Errorf("Property.idt missing ButtonText_Back")
	}
	if !containsLine(s, "ButtonText_Cancel\tCancel") {
		t.Errorf("Property.idt missing ButtonText_Cancel")
	}
	if !containsLine(s, "ButtonText_Finish\t&Finish") {
		t.Errorf("Property.idt missing ButtonText_Finish")
	}
}

func TestApplyUISequence(t *testing.T) {
	m := testModel()
	tables := coreTables(m)
	seqTbl := findTable(tables, "InstallUISequence")
	applyUISequence(seqTbl, m)

	rendered, err := seqTbl.Render()
	if err != nil {
		t.Fatal(err)
	}

	s := string(rendered)
	if !containsLine(s, "ExecuteAction\t\t1299") {
		t.Errorf("InstallUISequence missing ExecuteAction at seq 1299")
	}
	if !containsLine(s, "WelcomeDlg\t\t50") {
		t.Errorf("InstallUISequence missing WelcomeDlg at seq 50")
	}
	if !containsLine(s, "ExitDlg\t\t1300") {
		t.Errorf("InstallUISequence missing ExitDlg at seq 1300")
	}
}

func TestWriter_Emit_WithUI(t *testing.T) {
	dir := t.TempDir()

	payload := filepath.Join(dir, "myagent.exe")
	if err := os.WriteFile(payload, []byte("fake exe"), 0644); err != nil {
		t.Fatal(err)
	}

	emitDir := filepath.Join(dir, "emit")
	m := &model.MSI{
		Product: model.Product{
			Name:         "UITest",
			Version:      "1.0.0",
			Manufacturer: "TestCo",
			ProductCode:  "{cccccccc-cccc-cccc-cccc-cccccccccccc}",
		},
		Install: model.Install{Directory: "UITest"},
		Files: []model.File{
			{Source: payload, Destination: "myagent.exe"},
		},
		Parameters: []model.Parameter{
			{Name: "serverUrl", Property: "SERVERURL", Type: "string", Title: "Server URL"},
			{Name: "token", Property: "TOKEN", Type: "password", Title: "Token"},
		},
	}

	w := &Writer{EmitDir: emitDir}
	if err := w.Write(m, "out.msi"); err != nil {
		t.Fatal(err)
	}

	// Verify UI IDT files exist.
	for _, name := range []string{"TextStyle.idt", "Dialog.idt", "Control.idt", "ControlEvent.idt"} {
		p := filepath.Join(emitDir, name)
		fi, err := os.Stat(p)
		if err != nil {
			t.Errorf("missing UI file: %s (%v)", name, err)
			continue
		}
		if fi.Size() == 0 {
			t.Errorf("UI file %s is empty", name)
		}
	}

	// Verify core files still present.
	for _, name := range []string{"Property.idt", "Directory.idt", "InstallExecuteSequence.idt", "InstallUISequence.idt"} {
		p := filepath.Join(emitDir, name)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("missing core file: %s (%v)", name, err)
		}
	}

	// Verify Property.idt contains UI additions.
	propData, err := os.ReadFile(filepath.Join(emitDir, "Property.idt"))
	if err != nil {
		t.Fatal(err)
	}
	propStr := string(propData)
	if !strings.Contains(propStr, "DefaultUIFont\t{\\DlgFont8}") {
		t.Error("Property.idt missing DefaultUIFont")
	}
	if !strings.Contains(propStr, "ButtonText_Next\t&Next >") {
		t.Error("Property.idt missing ButtonText_Next")
	}

	// Verify InstallUISequence.idt has UI entries.
	seqData, err := os.ReadFile(filepath.Join(emitDir, "InstallUISequence.idt"))
	if err != nil {
		t.Fatal(err)
	}
	seqStr := string(seqData)
	if !strings.Contains(seqStr, "ExecuteAction\t\t1299") {
		t.Error("InstallUISequence.idt: ExecuteAction not at seq 1299")
	}
	if !strings.Contains(seqStr, "WelcomeDlg\t\t50") {
		t.Error("InstallUISequence.idt missing WelcomeDlg")
	}
	if !strings.Contains(seqStr, "ExitDlg\t\t1300") {
		t.Error("InstallUISequence.idt missing ExitDlg")
	}

	// Verify password Edit has the password attribute.
	ctrlData, err := os.ReadFile(filepath.Join(emitDir, "Control.idt"))
	if err != nil {
		t.Fatal(err)
	}
	ctrlStr := string(ctrlData)
	if !strings.Contains(ctrlStr, "Edit_TOKEN") {
		t.Error("Control.idt missing Edit_TOKEN for password param")
	}
	// Password attribute = 0x00200000 | Visible(1) | Enabled(2) = 0x00200003 = 2097155
	if !strings.Contains(ctrlStr, "2097155") {
		// Check the hex encoding in IDT: the attr value is decimal.
		t.Error("Control.idt: password Edit missing password attribute value 2097155")
	}
}

func TestWriter_FullBuild_WithUI(t *testing.T) {
	if _, err := exec.LookPath("msibuild"); err != nil {
		t.Skip("msibuild not available:", err)
	}
	if _, err := exec.LookPath("gcab"); err != nil {
		t.Skip("gcab not available:", err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("msibuild/gcab are Linux-only")
	}

	dir := t.TempDir()
	payload := filepath.Join(dir, "myagent.exe")
	if err := os.WriteFile(payload, []byte("fake exe for full UI build"), 0644); err != nil {
		t.Fatal(err)
	}

	m := &model.MSI{
		Product: model.Product{
			Name:         "FullUI",
			Version:      "1.0.0",
			Manufacturer: "TestCo",
			ProductCode:  "{ffffffff-ffff-ffff-ffff-ffffffffffff}",
		},
		Install: model.Install{Directory: "FullUI"},
		Files: []model.File{
			{Source: payload, Destination: "myagent.exe"},
		},
		Parameters: []model.Parameter{
			{Name: "host", Property: "HOST", Type: "string", Title: "Host"},
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
	t.Logf("Full MSI build with UI: %s (%d bytes)", msiPath, fi.Size())
}

// containsLine reports whether s contains the exact line text (without trailing \r).
func containsLine(s, line string) bool {
	for _, l := range bytes.Split([]byte(s), []byte{0x0A}) {
		trimmed := bytes.TrimRight(l, "\r")
		if string(trimmed) == line {
			return true
		}
	}
	return false
}
