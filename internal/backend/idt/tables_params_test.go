package idt

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/krivospitsky/gomsi/internal/model"
)

func testModelWithParams() *model.MSI {
	m := testModel()
	m.Parameters = []model.Parameter{
		{
			Name:     "serverUrl",
			Property: "SERVERURL",
			Type:     "string",
			Title:    "Server URL",
			Required: true,
			Default:  "",
			Validate: "url",
			UI:       "auto",
		},
		{
			Name:     "token",
			Property: "TOKEN",
			Type:     "password",
			Required: false,
			Default:  "s3cr3t",
		},
	}
	return m
}

func TestProperty_Parameters_Golden(t *testing.T) {
	m := testModelWithParams()
	tbl := buildProperty(m)

	got, err := tbl.Render()
	if err != nil {
		t.Fatalf("Property: Render: %v", err)
	}

	path := filepath.Join("testdata", "params", "Property.idt")
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Property: MkdirAll: %v", err)
		}
		if err := os.WriteFile(path, got, 0644); err != nil {
			t.Fatalf("Property: WriteFile: %v", err)
		}
	} else {
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Property: ReadFile: %v", err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("Property.idt: golden mismatch; run `go test -update` to regenerate")
		}
	}
}

func TestProperty_SecureCustomProperties(t *testing.T) {
	m := testModelWithParams()
	tbl := buildProperty(m)

	// Find the SecureCustomProperties row and verify its value.
	var gotSecure string
	for _, row := range tbl.Rows {
		if row[0] == "SecureCustomProperties" {
			gotSecure = row[1]
			break
		}
	}
	if gotSecure == "" {
		t.Fatal("SecureCustomProperties row not found or empty")
	}

	const want = "SERVERURL;TOKEN"
	if gotSecure != want {
		t.Errorf("SecureCustomProperties = %q, want %q", gotSecure, want)
	}

	// Verify both parameter property rows exist.
	paramRows := 0
	for _, row := range tbl.Rows {
		if row[0] == "SERVERURL" {
			paramRows++
			if row[1] != "" {
				t.Errorf("SERVERURL value = %q, want empty", row[1])
			}
		}
		if row[0] == "TOKEN" {
			paramRows++
			if row[1] != "s3cr3t" {
				t.Errorf("TOKEN value = %q, want %q", row[1], "s3cr3t")
			}
		}
	}
	if paramRows != 2 {
		t.Errorf("found %d parameter rows, want 2", paramRows)
	}
}

func TestProperty_NoParameters(t *testing.T) {
	m := testModel() // no parameters
	tbl := buildProperty(m)

	for _, row := range tbl.Rows {
		if row[0] == "SecureCustomProperties" {
			if row[1] != "" {
				t.Errorf("SecureCustomProperties with no params = %q, want empty", row[1])
			}
			return
		}
	}
	t.Error("SecureCustomProperties row not found")
}

func TestProperty_SkipEmptyProperty(t *testing.T) {
	m := testModel()
	m.Parameters = []model.Parameter{
		{Name: "noProp"},         // Property is empty — should be skipped
		{Name: "valid", Property: "VALID", Default: "ok"},
	}
	tbl := buildProperty(m)

	for _, row := range tbl.Rows {
		if row[0] == "SecureCustomProperties" {
			if row[1] != "VALID" {
				t.Errorf("SecureCustomProperties = %q, want %q", row[1], "VALID")
			}
			return
		}
	}
	t.Error("SecureCustomProperties row not found")
}
