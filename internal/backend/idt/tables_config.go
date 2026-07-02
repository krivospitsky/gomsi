package idt

import (
	"strings"

	"github.com/krivospitsky/gomsi/internal/model"
)

// configTables returns CustomAction and Binary tables when the model has a
// config template. Returns nil when there is none.
func configTables(m *model.MSI) []*Table {
	if m.Config.Template == "" {
		return nil
	}
	return []*Table{
		buildCustomAction(m),
		buildBinary(m),
	}
}

// buildCustomAction returns the CustomAction table with immediate and deferred
// config-writing custom actions.
//
// Immediate SetWriteConfig (Type 51) sets the "WriteConfig" property to a
// Formatted string that resolves INSTALLDIR and parameter property values. The
// installer passes this property as CustomActionData to the deferred CA.
//
// Deferred WriteConfig (Type 3078 = 6|0x400|0x800 — VBScript from Binary,
// InScript, NoImpersonate) runs the VBScript from the Binary table, calling
// function WriteConfig, which reads CustomActionData, replaces sentinels, and
// writes the rendered config file.
func buildCustomAction(m *model.MSI) *Table {
	target := setWriteConfigTarget(m)

	tbl := &Table{
		Name:     "CustomAction",
		CodePage: m.CodePage,
		Columns: []Column{
			{Name: "Action", Type: Str(72), PK: true},
			{Name: "Condition", Type: NStr(255)},
			{Name: "Type", Type: I2()},
			{Name: "Source", Type: NStr(72)},
			{Name: "Target", Type: NStr(255)},
		},
	}

	// Immediate CA: set property WriteConfig (CustomActionData for deferred CA).
	tbl.AddRow(
		"SetWriteConfig", // Action
		"",               // Condition (null; gated via sequence table)
		"51",             // Type: set property from Formatted text
		"WriteConfig",    // Source: property name = deferred CA action → CustomActionData
		target,           // Target: Formatted string
	)

	// Deferred CA: write config file via VBScript from Binary.
	tbl.AddRow(
		"WriteConfig", // Action
		"",            // Condition (null)
		"3078",        // Type: 6|0x400|0x800 = VBScript-binary|InScript|NoImpersonate
		"WriteConfig", // Source: Binary.Name key
		"WriteConfig", // Target: VBScript function name
	)

	return tbl
}

// buildBinary returns the Binary table with the config-writing VBScript.
// The Data column (V0) references the sidecar file loaded by msibuild.
func buildBinary(m *model.MSI) *Table {
	tbl := &Table{
		Name:     "Binary",
		CodePage: m.CodePage,
		Columns: []Column{
			{Name: "Name", Type: Str(72), PK: true},
			{Name: "Data", Type: NBin(0)}, // V0 — binary stream
		},
	}

	tbl.AddRow(
		"WriteConfig",     // Name (PK)
		"WriteConfig.vbs", // Data: sidecar filename; msibuild loads Binary/<this>
	)

	return tbl
}

// setWriteConfigTarget builds the Formatted Target for the SetWriteConfig
// immediate CA. The resolved value at install time becomes CustomActionData
// for the deferred WriteConfig CA.
//
// Format: [INSTALLDIR]<output>|[<prop1>]|[<prop2>]|…
func setWriteConfigTarget(m *model.MSI) string {
	var b strings.Builder
	b.WriteString("[INSTALLDIR]")
	b.WriteString(m.Config.Output)
	for _, p := range m.Parameters {
		b.WriteString("|[")
		b.WriteString(p.Property)
		b.WriteString("]")
	}
	return b.String()
}
