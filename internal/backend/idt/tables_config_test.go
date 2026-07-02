package idt

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/krivospitsky/gomsi/internal/model"
)

func testModelWithConfig() *model.MSI {
	m := testModelWithParams()
	m.Config = model.Config{Template: "testdata/config/template.tpl", Output: "config.json"}
	return m
}

func TestConfigTables_Golden(t *testing.T) {
	m := testModelWithConfig()
	tables := configTables(m)
	if tables == nil {
		t.Fatal("configTables returned nil")
	}

	for _, tbl := range tables {
		got, err := tbl.Render()
		if err != nil {
			t.Fatalf("%s: Render: %v", tbl.Name, err)
		}

		path := filepath.Join("testdata", "config", tbl.Name+".idt")
		if *update {
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatalf("%s: MkdirAll: %v", tbl.Name, err)
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
				t.Errorf("%s.idt: golden mismatch; run `go test -update` to regenerate", tbl.Name)
			}
		}
	}
}

func TestConfigTables_Nil(t *testing.T) {
	m := testModel()
	tables := configTables(m)
	if tables != nil {
		t.Error("configTables without config = non-nil, want nil")
	}
}

func TestConfigTables_EmptyTemplate(t *testing.T) {
	m := testModelWithParams()
	m.Config = model.Config{Template: ""}
	tables := configTables(m)
	if tables != nil {
		t.Error("configTables with empty Template = non-nil, want nil")
	}
}

func TestConfigTables_SequenceWithConfig(t *testing.T) {
	m := testModelWithConfig()

	var seqTbl *Table
	for _, tbl := range coreTables(m) {
		if tbl.Name == "InstallExecuteSequence" {
			seqTbl = tbl
			break
		}
	}
	if seqTbl == nil {
		t.Fatal("InstallExecuteSequence table not found")
	}

	// Build a set of actions for easy checking.
	actions := make(map[string]struct{}, len(seqTbl.Rows))
	for _, row := range seqTbl.Rows {
		actions[row[0]] = struct{}{}
	}

	if _, ok := actions["SetWriteConfig"]; !ok {
		t.Error("InstallExecuteSequence missing SetWriteConfig")
	}
	if _, ok := actions["WriteConfig"]; !ok {
		t.Error("InstallExecuteSequence missing WriteConfig")
	}
}

func TestConfigTables_SequenceWithoutConfig(t *testing.T) {
	m := testModel()

	var seqTbl *Table
	for _, tbl := range coreTables(m) {
		if tbl.Name == "InstallExecuteSequence" {
			seqTbl = tbl
			break
		}
	}
	if seqTbl == nil {
		t.Fatal("InstallExecuteSequence table not found")
	}

	for _, row := range seqTbl.Rows {
		switch row[0] {
		case "SetWriteConfig":
			t.Error("SetWriteConfig should not be present without config")
		case "WriteConfig":
			t.Error("WriteConfig should not be present without config")
		}
	}
}

func TestConfigTables_SequenceOrdering(t *testing.T) {
	m := testModelWithConfig()

	var seqTbl *Table
	for _, tbl := range coreTables(m) {
		if tbl.Name == "InstallExecuteSequence" {
			seqTbl = tbl
			break
		}
	}
	if seqTbl == nil {
		t.Fatal("InstallExecuteSequence table not found")
	}

	seq := make(map[string]string, len(seqTbl.Rows))
	for _, row := range seqTbl.Rows {
		seq[row[0]] = row[2]
	}

	if seq["SetWriteConfig"] != "151" {
		t.Errorf("SetWriteConfig sequence = %q, want 151", seq["SetWriteConfig"])
	}
	if seq["WriteConfig"] != "205" {
		t.Errorf("WriteConfig sequence = %q, want 205", seq["WriteConfig"])
	}
	if seq["InstallFiles"] != "150" {
		t.Errorf("InstallFiles sequence = %q, want 150", seq["InstallFiles"])
	}
	if seq["InstallFinalize"] != "210" {
		t.Errorf("InstallFinalize sequence = %q, want 210", seq["InstallFinalize"])
	}
}

func TestSetWriteConfigTarget(t *testing.T) {
	m := testModelWithConfig()
	got := setWriteConfigTarget(m)
	want := "[INSTALLDIR]config.json|[SERVERURL]|[TOKEN]"
	if got != want {
		t.Errorf("setWriteConfigTarget = %q, want %q", got, want)
	}
}

func TestSetWriteConfigTarget_NoParams(t *testing.T) {
	m := testModelWithConfig()
	m.Parameters = nil
	got := setWriteConfigTarget(m)
	want := "[INSTALLDIR]config.json"
	if got != want {
		t.Errorf("setWriteConfigTarget with no params = %q, want %q", got, want)
	}
}

func TestCustomAction_Schema(t *testing.T) {
	m := testModelWithConfig()
	tbl := buildCustomAction(m)

	if tbl.Name != "CustomAction" {
		t.Errorf("table Name = %q, want CustomAction", tbl.Name)
	}

	if len(tbl.Columns) != 5 {
		t.Fatalf("CustomAction has %d columns, want 5", len(tbl.Columns))
	}

	expectedCols := []string{"Action", "Condition", "Type", "Source", "Target"}
	for i, col := range tbl.Columns {
		if col.Name != expectedCols[i] {
			t.Errorf("column[%d] = %q, want %q", i, col.Name, expectedCols[i])
		}
	}

	if len(tbl.Rows) != 2 {
		t.Fatalf("CustomAction has %d rows, want 2", len(tbl.Rows))
	}

	if tbl.Rows[0][0] != "SetWriteConfig" {
		t.Errorf("row[0].Action = %q, want SetWriteConfig", tbl.Rows[0][0])
	}
	if tbl.Rows[1][0] != "WriteConfig" {
		t.Errorf("row[1].Action = %q, want WriteConfig", tbl.Rows[1][0])
	}
}

func TestBinary_Schema(t *testing.T) {
	m := testModelWithConfig()
	tbl := buildBinary(m)

	if tbl.Name != "Binary" {
		t.Errorf("table Name = %q, want Binary", tbl.Name)
	}

	if len(tbl.Columns) != 2 {
		t.Fatalf("Binary has %d columns, want 2", len(tbl.Columns))
	}

	if tbl.Columns[0].Name != "Name" {
		t.Errorf("column[0] = %q, want Name", tbl.Columns[0].Name)
	}
	if tbl.Columns[1].Name != "Data" {
		t.Errorf("column[1] = %q, want Data", tbl.Columns[1].Name)
	}

	if len(tbl.Rows) != 1 {
		t.Fatalf("Binary has %d rows, want 1", len(tbl.Rows))
	}

	if tbl.Rows[0][0] != "WriteConfig" {
		t.Errorf("row[0].Name = %q, want WriteConfig", tbl.Rows[0][0])
	}
	if tbl.Rows[0][1] != "WriteConfig.vbs" {
		t.Errorf("row[0].Data = %q, want WriteConfig.vbs", tbl.Rows[0][1])
	}
}

// TestConfigTables_SequenceGolden generates a golden file for the full
// InstallExecuteSequence when a config is present.
func TestConfigTables_SequenceGolden(t *testing.T) {
	m := testModelWithConfig()

	var seqTbl *Table
	for _, tbl := range coreTables(m) {
		if tbl.Name == "InstallExecuteSequence" {
			seqTbl = tbl
			break
		}
	}
	if seqTbl == nil {
		t.Fatal("InstallExecuteSequence table not found")
	}

	got, err := seqTbl.Render()
	if err != nil {
		t.Fatalf("InstallExecuteSequence: Render: %v", err)
	}

	path := filepath.Join("testdata", "config", "InstallExecuteSequence.idt")
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("InstallExecuteSequence: MkdirAll: %v", err)
		}
		if err := os.WriteFile(path, got, 0644); err != nil {
			t.Fatalf("InstallExecuteSequence: WriteFile: %v", err)
		}
	} else {
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("InstallExecuteSequence: ReadFile: %v", err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("InstallExecuteSequence.idt: golden mismatch; run `go test -update` to regenerate")
		}
	}
}

func TestConfigTables_InstallExecuteSequenceCondition(t *testing.T) {
	m := testModelWithConfig()

	var seqTbl *Table
	for _, tbl := range coreTables(m) {
		if tbl.Name == "InstallExecuteSequence" {
			seqTbl = tbl
			break
		}
	}
	if seqTbl == nil {
		t.Fatal("InstallExecuteSequence table not found")
	}

	cond := make(map[string]string, len(seqTbl.Rows))
	for _, row := range seqTbl.Rows {
		cond[row[0]] = row[1]
	}

	if cond["SetWriteConfig"] != "NOT REMOVE~=\"ALL\"" {
		t.Errorf("SetWriteConfig condition = %q, want NOT REMOVE~=\"ALL\"", cond["SetWriteConfig"])
	}
	if cond["WriteConfig"] != "NOT REMOVE~=\"ALL\"" {
		t.Errorf("WriteConfig condition = %q, want NOT REMOVE~=\"ALL\"", cond["WriteConfig"])
	}

	// Other rows (e.g. InstallFiles) should have no condition.
	if cond["InstallFiles"] != "" {
		t.Errorf("InstallFiles condition = %q, want empty", cond["InstallFiles"])
	}
}
