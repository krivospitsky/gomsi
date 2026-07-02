package idt

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/krivospitsky/gomsi/internal/model"
)

// testModelWithService returns a model for service-table golden tests.
func testModelWithService() *model.MSI {
	m := testModel()
	m.Services = []model.Service{
		{
			Name:        "myagent",
			DisplayName: "My Agent",
			Description: "Monitoring Agent",
			Start:       "auto",
		},
	}
	return m
}

func TestServiceTables_Golden(t *testing.T) {
	m := testModelWithService()

	// Service-specific tables.
	svcTbls := serviceTables(m)
	expected := []string{"ServiceInstall", "ServiceControl"}
	for _, tbl := range svcTbls {
		got, err := tbl.Render()
		if err != nil {
			t.Fatalf("%s: Render: %v", tbl.Name, err)
		}

		path := filepath.Join("testdata", "service", tbl.Name+".idt")
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

	// Also verify the service-augmented InstallExecuteSequence.
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

	path := filepath.Join("testdata", "service", "InstallExecuteSequence.idt")
	if *update {
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

	// Verify expected table names match.
	for i, name := range expected {
		if svcTbls[i].Name != name {
			t.Errorf("serviceTables[%d].Name = %q, want %q", i, svcTbls[i].Name, name)
		}
	}
}

func TestServiceTables_EmptyServices(t *testing.T) {
	m := testModel()
	m.Services = nil

	tables := serviceTables(m)
	if tables != nil {
		t.Errorf("serviceTables with nil services = %v, want nil", tables)
	}

	m.Services = []model.Service{}
	tables = serviceTables(m)
	if tables != nil {
		t.Errorf("serviceTables with empty slice = %v, want nil", tables)
	}
}

func TestStartTypeMapping(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"auto", "2"},
		{"manual", "3"},
		{"disabled", "4"},
		{"", "3"},
		{"unknown", "3"},
	}

	for _, c := range cases {
		got := startType(c.input)
		if got != c.want {
			t.Errorf("startType(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestServiceTables_ComponentViaFirstFile(t *testing.T) {
	m := testModel()
	m.Services = []model.Service{{Name: "svc", Start: "auto"}}
	m.Files = []model.File{
		{Source: "", Destination: "first.exe", Size: 100},
		{Source: "", Destination: "second.exe", Size: 200},
	}

	tables := serviceTables(m)
	if len(tables) == 0 {
		t.Fatal("expected service tables")
	}

	// Both ServiceInstall and ServiceControl should reference the first file's component.
	for _, tbl := range tables {
		expectedCmp := componentName("first.exe")
		if len(tbl.Rows) != 1 {
			t.Fatalf("%s: expected 1 row", tbl.Name)
		}
		row := tbl.Rows[0]

		var gotCmp string
		if tbl.Name == "ServiceInstall" {
			// Component_ is the 12th column (0-indexed: 11).
			gotCmp = row[11]
		} else {
			// ServiceControl: Component_ is the 6th column (0-indexed: 5).
			gotCmp = row[5]
		}
		if gotCmp != expectedCmp {
			t.Errorf("%s Component_ = %q, want %q", tbl.Name, gotCmp, expectedCmp)
		}
	}
}
