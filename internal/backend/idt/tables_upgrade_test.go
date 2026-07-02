package idt

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krivospitsky/gomsi/internal/model"
)

// testModelWithUpgrade returns a model with an UpgradeCode for upgrade golden tests.
func testModelWithUpgrade() *model.MSI {
	m := testModel()
	m.Product.UpgradeCode = "{11111111-1111-1111-1111-111111111111}"
	return m
}

func TestUpgradeTables_Golden(t *testing.T) {
	m := testModelWithUpgrade()

	// Upgrade table.
	upgTbls := upgradeTables(m)
	if len(upgTbls) == 0 {
		t.Fatal("upgradeTables returned nil, expected one Upgrade table")
	}
	got, err := upgTbls[0].Render()
	if err != nil {
		t.Fatalf("Upgrade: Render: %v", err)
	}

	path := filepath.Join("testdata", "upgrade", "Upgrade.idt")
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Upgrade: MkdirAll: %v", err)
		}
		if err := os.WriteFile(path, got, 0644); err != nil {
			t.Fatalf("Upgrade: WriteFile: %v", err)
		}
	} else {
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Upgrade: ReadFile: %v", err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("Upgrade.idt: golden mismatch; run `go test -update` to regenerate")
		}
	}

	// Property table with upgrade (SecureCustomProperties includes OLDPRODUCTSFOUND).
	propTbl := buildProperty(m)
	got, err = propTbl.Render()
	if err != nil {
		t.Fatalf("Property: Render: %v", err)
	}

	path = filepath.Join("testdata", "upgrade", "Property.idt")
	if *update {
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

	// InstallExecuteSequence with upgrade actions.
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

	got, err = seqTbl.Render()
	if err != nil {
		t.Fatalf("InstallExecuteSequence: Render: %v", err)
	}

	path = filepath.Join("testdata", "upgrade", "InstallExecuteSequence.idt")
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

	// Verify expected table name.
	if upgTbls[0].Name != "Upgrade" {
		t.Errorf("upgradeTables[0].Name = %q, want %q", upgTbls[0].Name, "Upgrade")
	}
}

func TestUpgradeTables_NoUpgradeCode(t *testing.T) {
	m := testModel() // UpgradeCode is empty

	// upgradeTables returns nil.
	tables := upgradeTables(m)
	if tables != nil {
		t.Errorf("upgradeTables with empty UpgradeCode = %v, want nil", tables)
	}

	// SecureCustomProperties has no OLDPRODUCTSFOUND.
	propTbl := buildProperty(m)
	for _, row := range propTbl.Rows {
		if row[0] == "SecureCustomProperties" {
			if strings.Contains(row[1], "OLDPRODUCTSFOUND") {
				t.Errorf("SecureCustomProperties = %q, should not contain OLDPRODUCTSFOUND", row[1])
			}
			return
		}
	}
	t.Error("SecureCustomProperties row not found")
}

func TestUpgradeTables_SequenceWithoutUpgrade(t *testing.T) {
	m := testModel() // UpgradeCode is empty

	seqTbl := buildInstallExecuteSequence(m)

	// Should NOT contain FindRelatedProducts or RemoveExistingProducts.
	for _, row := range seqTbl.Rows {
		switch row[0] {
		case "FindRelatedProducts":
			t.Error("FindRelatedProducts should not be present without UpgradeCode")
		case "RemoveExistingProducts":
			t.Error("RemoveExistingProducts should not be present without UpgradeCode")
		}
	}
}

func TestUpgradeSequenceOrdering(t *testing.T) {
	m := testModelWithUpgrade()

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

	// Build a map of action→sequence for easy checking.
	seq := make(map[string]string, len(seqTbl.Rows))
	for _, row := range seqTbl.Rows {
		seq[row[0]] = row[2]
	}

	// FindRelatedProducts (25) must come before InstallInitialize (50).
	if seq["FindRelatedProducts"] != "25" {
		t.Errorf("FindRelatedProducts sequence = %q, want 25", seq["FindRelatedProducts"])
	}

	// RemoveExistingProducts (55) must come after InstallInitialize (50)
	// and before InstallFiles (150).
	if seq["RemoveExistingProducts"] != "55" {
		t.Errorf("RemoveExistingProducts sequence = %q, want 55", seq["RemoveExistingProducts"])
	}
}

func TestUpgrade_AttributesValue(t *testing.T) {
	m := testModelWithUpgrade()
	tbl := buildUpgrade(m)

	if len(tbl.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(tbl.Rows))
	}

	// Attributes is the 5th column (0-indexed: 4).
	const want = "0"
	got := tbl.Rows[0][4]
	if got != want {
		t.Errorf("Upgrade Attributes = %q, want %q", got, want)
	}
}
