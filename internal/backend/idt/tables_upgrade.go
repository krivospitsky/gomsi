package idt

import (
	"github.com/krivospitsky/gomsi/internal/model"
)

// upgradeTables returns the Upgrade table when the model has an UpgradeCode.
// Returns nil when there is none.
func upgradeTables(m *model.MSI) []*Table {
	if m.Product.UpgradeCode == "" {
		return nil
	}
	return []*Table{buildUpgrade(m)}
}

// ── Upgrade ──────────────────────────────────────────────────────────────────

func buildUpgrade(m *model.MSI) *Table {
	tbl := &Table{
		Name:     "Upgrade",
		CodePage: m.CodePage,
		Columns: []Column{
			{Name: "UpgradeCode", Type: Str(38), PK: true},
			{Name: "VersionMin", Type: NStr(20)},
			{Name: "VersionMax", Type: NStr(20)},
			{Name: "Language", Type: NStr(20)},
			{Name: "Attributes", Type: I2()},
			{Name: "Remove", Type: NStr(255)},
			{Name: "ActionProperty", Type: Str(72), PK: true},
		},
	}

	tbl.AddRow(
		m.Product.UpgradeCode, // UpgradeCode
		"",                    // VersionMin  (null → detect all previous)
		m.Product.Version,     // VersionMax  (current product version)
		"",                    // Language    (null → all languages)
		"0",                   // Attributes  (OnlyDetect off → removal enabled)
		"",                    // Remove      (null → REMOVE=ALL)
		"OLDPRODUCTSFOUND",    // ActionProperty
	)

	return tbl
}
