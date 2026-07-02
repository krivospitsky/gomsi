package idt

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/krivospitsky/gomsi/internal/model"
)

// coreTables returns all phase-2 IDT tables in canonical emission order.
func coreTables(m *model.MSI) []*Table {
	return []*Table{
		buildProperty(m),
		buildDirectory(m),
		buildComponent(m),
		buildFeature(m),
		buildFeatureComponents(m),
		buildFile(m),
		buildMedia(m),
		buildInstallExecuteSequence(m),
		buildInstallUISequence(m),
	}
}

// ── Property ──────────────────────────────────────────────────────────────────

func buildProperty(m *model.MSI) *Table {
	tbl := &Table{
		Name:     "Property",
		CodePage: m.CodePage,
		Columns: []Column{
			{Name: "Property", Type: Str(72), PK: true},
			{Name: "Value", Type: NStr(0)},
		},
	}

	addProp := func(name, value string) {
		tbl.AddRow(name, value)
	}

	addProp("ProductName", m.Product.Name)
	addProp("ProductVersion", m.Product.Version)
	addProp("Manufacturer", m.Product.Manufacturer)
	addProp("ProductLanguage", "1033")
	addProp("ProductCode", m.Product.ProductCode)
	if m.Product.UpgradeCode != "" {
		addProp("UpgradeCode", m.Product.UpgradeCode)
	}

	// Phase 4: emit a Property row per parameter so each maps to an MSI
	// public property settable via msiexec SERVERURL=... or the UI.
	// Required parameters are best-effort in MVP — no client-side
	// enforcement; the default value (possibly empty) is used as-is.
	secureProps := make([]string, 0, len(m.Parameters))
	for _, p := range m.Parameters {
		if p.Property == "" {
			continue
		}
		addProp(p.Property, p.Default)
		secureProps = append(secureProps, p.Property)
	}

	// SecureCustomProperties tells the installer which public properties
	// to pass to the deferred/machine context (e.g. for VBScript CAs).
	// Future phases (upgrade) may append to this list.
	addProp("SecureCustomProperties", strings.Join(secureProps, ";"))

	return tbl
}

// ── Directory ─────────────────────────────────────────────────────────────────

func buildDirectory(m *model.MSI) *Table {
	tbl := &Table{
		Name:     "Directory",
		CodePage: m.CodePage,
		Columns: []Column{
			{Name: "Directory", Type: Str(72), PK: true},
			{Name: "Directory_Parent", Type: NStr(72)},
			{Name: "DefaultDir", Type: NStr(255)},
		},
	}

	tbl.AddRow("TARGETDIR", "", "SourceDir")
	tbl.AddRow("ProgramFilesFolder", "TARGETDIR", ".")
	tbl.AddRow("INSTALLDIR", "ProgramFilesFolder", m.Install.Directory)

	return tbl
}

// ── Component ─────────────────────────────────────────────────────────────────

func buildComponent(m *model.MSI) *Table {
	tbl := &Table{
		Name:     "Component",
		CodePage: m.CodePage,
		Columns: []Column{
			{Name: "Component", Type: Str(72), PK: true},
			{Name: "ComponentId", Type: NStr(38)},
			{Name: "Directory_", Type: NStr(72)},
			{Name: "Attributes", Type: I2()},
			{Name: "Condition", Type: NStr(255)},
			{Name: "KeyPath", Type: NStr(72)},
		},
	}

	for _, f := range m.Files {
		comp := componentName(f.Destination)
		fileID := fileID(f.Destination)
		guid := deterministicGUID(m.Product.Name + "|" + comp)
		tbl.AddRow(comp, guid, "INSTALLDIR", "0", "", fileID)
	}

	return tbl
}

// ── Feature ───────────────────────────────────────────────────────────────────

func buildFeature(m *model.MSI) *Table {
	tbl := &Table{
		Name:     "Feature",
		CodePage: m.CodePage,
		Columns: []Column{
			{Name: "Feature", Type: Str(38), PK: true},
			{Name: "Feature_Parent", Type: NStr(38)},
			{Name: "Title", Type: NStr(64)},
			{Name: "Description", Type: NStr(255)},
			{Name: "Display", Type: I2()},
			{Name: "Level", Type: I2()},
			{Name: "Attributes", Type: I2()},
		},
	}

	tbl.AddRow("Complete", "", "", "", "0", "1", "0")

	return tbl
}

// ── FeatureComponents ─────────────────────────────────────────────────────────

func buildFeatureComponents(m *model.MSI) *Table {
	tbl := &Table{
		Name:     "FeatureComponents",
		CodePage: m.CodePage,
		Columns: []Column{
			{Name: "Feature_", Type: Str(38), PK: true},
			{Name: "Component_", Type: Str(72), PK: true},
		},
	}

	for _, f := range m.Files {
		tbl.AddRow("Complete", componentName(f.Destination))
	}

	return tbl
}

// ── File ──────────────────────────────────────────────────────────────────────

func buildFile(m *model.MSI) *Table {
	tbl := &Table{
		Name:     "File",
		CodePage: m.CodePage,
		Columns: []Column{
			{Name: "File", Type: Str(72), PK: true},
			{Name: "Component_", Type: NStr(72)},
			{Name: "FileName", Type: NStr(255)},
			{Name: "FileSize", Type: I4()},
			{Name: "Version", Type: NStr(72)},
			{Name: "Language", Type: NStr(20)},
			{Name: "Attributes", Type: I4()},
			{Name: "Sequence", Type: I2()},
		},
	}

	for i, f := range m.Files {
		tbl.AddRow(
			fileID(f.Destination),
			componentName(f.Destination),
			f.Destination,
			fmt.Sprintf("%d", f.Size),
			"",
			"",
			"0",
			fmt.Sprintf("%d", i+1),
		)
	}

	return tbl
}

// ── Media ─────────────────────────────────────────────────────────────────────

func buildMedia(m *model.MSI) *Table {
	tbl := &Table{
		Name:     "Media",
		CodePage: m.CodePage,
		Columns: []Column{
			{Name: "DiskId", Type: I2(), PK: true},
			{Name: "LastSequence", Type: I2()},
			{Name: "DiskPrompt", Type: NStr(64)},
			{Name: "Cabinet", Type: NStr(255)},
			{Name: "VolumeLabel", Type: NStr(32)},
		},
	}

	tbl.AddRow("1", fmt.Sprintf("%d", len(m.Files)), "", "gomsi.cab", "")

	return tbl
}

// ── InstallExecuteSequence ────────────────────────────────────────────────────

func buildInstallExecuteSequence(m *model.MSI) *Table {
	tbl := &Table{
		Name:     "InstallExecuteSequence",
		CodePage: m.CodePage,
		Columns: []Column{
			{Name: "Action", Type: Str(72), PK: true},
			{Name: "Condition", Type: NStr(255)},
			{Name: "Sequence", Type: I2()},
		},
	}

	entries := []struct {
		action    string
		condition string
		sequence  int
	}{
		{"CostInitialize", "", 1},
		{"FileCost", "", 2},
		{"CostFinalize", "", 3},
		{"InstallValidate", "", 10},
		{"InstallInitialize", "", 50},
		{"ProcessComponents", "", 60},
	}

	// Service actions: stop and delete before InstallFiles (to free the
	// running binary), install after InstallFiles.
	if len(m.Services) > 0 {
		entries = append(entries,
			[]struct {
				action    string
				condition string
				sequence  int
			}{
				{"StopServices", "", 140},
				{"DeleteServices", "", 145},
			}...,
		)
	}

	entries = append(entries, []struct {
		action    string
		condition string
		sequence  int
	}{
		{"InstallFiles", "", 150},
	}...)

	if len(m.Services) > 0 {
		entries = append(entries, struct {
			action    string
			condition string
			sequence  int
		}{"InstallServices", "", 155})
	}

	entries = append(entries, []struct {
		action    string
		condition string
		sequence  int
	}{
		{"RegisterProduct", "", 180},
		{"PublishFeatures", "", 190},
		{"PublishProduct", "", 200},
		{"InstallFinalize", "", 210},
	}...)

	for _, e := range entries {
		tbl.AddRow(e.action, e.condition, fmt.Sprintf("%d", e.sequence))
	}

	return tbl
}

// ── InstallUISequence ─────────────────────────────────────────────────────────

func buildInstallUISequence(m *model.MSI) *Table {
	tbl := &Table{
		Name:     "InstallUISequence",
		CodePage: m.CodePage,
		Columns: []Column{
			{Name: "Action", Type: Str(72), PK: true},
			{Name: "Condition", Type: NStr(255)},
			{Name: "Sequence", Type: I2()},
		},
	}

	entries := []struct {
		action    string
		condition string
		sequence  int
	}{
		{"CostInitialize", "", 1},
		{"FileCost", "", 2},
		{"CostFinalize", "", 3},
		{"ExecuteAction", "", 4},
	}

	for _, e := range entries {
		tbl.AddRow(e.action, e.condition, fmt.Sprintf("%d", e.sequence))
	}

	return tbl
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// componentName returns the IDT component name for a destination filename.
func componentName(dest string) string { return "C_" + dest }

// fileID returns the IDT File column value for a destination filename.
func fileID(dest string) string { return "F_" + dest }

// deterministicGUID generates a v4-ish braced GUID from a seed string.
func deterministicGUID(seed string) string {
	h := sha256.Sum256([]byte(seed))
	b := h[:16]
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	s := hex.EncodeToString(b)
	return fmt.Sprintf("{%s-%s-%s-%s-%s}", s[0:8], s[8:12], s[12:16], s[16:20], s[20:32])
}
