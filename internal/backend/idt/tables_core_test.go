package idt

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/krivospitsky/gomsi/internal/model"
)

// testModel returns a deterministic model fixture for table-builder golden tests.
func testModel() *model.MSI {
	return &model.MSI{
		CodePage: 0,
		Product: model.Product{
			Name:         "MyAgent",
			Version:      "1.2.3",
			Manufacturer: "Acme",
			UpgradeCode:  "",
			ProductCode:  "{22222222-2222-2222-2222-222222222222}",
		},
		Install: model.Install{Directory: "MyAgent"},
		Files: []model.File{
			{
				Source:      "",
				Destination: "myagent.exe",
				Size:        12345,
			},
		},
	}
}

func TestCoreTables_Golden(t *testing.T) {
	m := testModel()
	tables := coreTables(m)

	for _, tbl := range tables {
		got, err := tbl.Render()
		if err != nil {
			t.Fatalf("%s: Render: %v", tbl.Name, err)
		}

		path := filepath.Join("testdata", "core", tbl.Name+".idt")
		if *update {
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

func TestCoreTables_ComponentGUIDDeterministic(t *testing.T) {
	m := testModel()
	tables := coreTables(m)

	var compTable *Table
	for _, tbl := range tables {
		if tbl.Name == "Component" {
			compTable = tbl
			break
		}
	}
	if compTable == nil {
		t.Fatal("Component table not found")
	}
	if len(compTable.Rows) == 0 {
		t.Fatal("Component table has no rows")
	}

	// Same model yields the same GUID.
	first := compTable.Rows[0][1] // ComponentId
	m2 := testModel()
	tables2 := coreTables(m2)
	var compTable2 *Table
	for _, tbl := range tables2 {
		if tbl.Name == "Component" {
			compTable2 = tbl
			break
		}
	}
	if compTable2.Rows[0][1] != first {
		t.Errorf("deterministic GUID changed: got %q, want %q", compTable2.Rows[0][1], first)
	}
}

func TestCoreTables_ExplicitCodePage(t *testing.T) {
	m := testModel()
	m.CodePage = 1251

	tables := coreTables(m)
	for _, tbl := range tables {
		cp, err := tbl.effectiveCodePage()
		if err != nil {
			t.Fatalf("%s: effectiveCodePage: %v", tbl.Name, err)
		}
		if cp != 1251 {
			t.Errorf("%s: expected codepage 1251, got %d", tbl.Name, cp)
		}
	}
}

func TestCoreTables_EmptyFiles(t *testing.T) {
	m := testModel()
	m.Files = nil

	tables := coreTables(m)
	// Should not panic; Component, FeatureComponents, File, Media handle zero files.
	for _, tbl := range tables {
		_, err := tbl.Render()
		if err != nil {
			t.Fatalf("%s: Render with empty files: %v", tbl.Name, err)
		}
	}
}
