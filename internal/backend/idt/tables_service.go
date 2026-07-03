package idt

import (
	"github.com/krivospitsky/gomsi/internal/model"
)

// serviceTables returns ServiceInstall and ServiceControl tables when the
// model declares services. Returns nil when there are none.
func serviceTables(m *model.MSI) []*Table {
	if len(m.Services) == 0 {
		return nil
	}

	tables := make([]*Table, 0, 2*len(m.Services))
	for _, s := range m.Services {
		tables = append(tables, buildServiceInstall(m, s), buildServiceControl(m, s))
	}
	return tables
}

// startType maps a model.Service.Start string to an MSI StartType integer.
func startType(s string) string {
	switch s {
	case "auto":
		return "2"
	case "manual":
		return "3"
	case "disabled":
		return "4"
	default:
		return "3"
	}
}

// ── ServiceInstall ────────────────────────────────────────────────────────────

func buildServiceInstall(m *model.MSI, s model.Service) *Table {
	tbl := &Table{
		Name:     "ServiceInstall",
		CodePage: m.CodePage,
		Columns: []Column{
			{Name: "ServiceInstall", Type: Str(72), PK: true},
			{Name: "Name", Type: Loc(255)},
			{Name: "DisplayName", Type: NLoc(255)},
			{Name: "ServiceType", Type: I2()},
			{Name: "StartType", Type: I2()},
			{Name: "ErrorControl", Type: I2()},
			{Name: "LoadOrderGroup", Type: NStr(255)},
			{Name: "Dependencies", Type: NStr(255)},
			{Name: "StartName", Type: NStr(255)},
			{Name: "Password", Type: NStr(255)},
			{Name: "Arguments", Type: NStr(255)},
			{Name: "Component_", Type: Str(72)},
			{Name: "Description", Type: NLoc(255)},
		},
	}

	var cmp string
	// If no files exist, use a placeholder.
	if len(m.Files) > 0 {
		cmp = componentName(m.Files[0].Destination)
	}

	tbl.AddRow(
		s.Name,           // ServiceInstall (PK)
		s.Name,           // Name
		s.DisplayName,    // DisplayName (nullable)
		"16",             // ServiceType (SERVICE_WIN32_OWN_PROCESS)
		startType(s.Start), // StartType
		"1",              // ErrorControl (SERVICE_ERROR_NORMAL)
		"",               // LoadOrderGroup (null)
		"",               // Dependencies (null)
		"",               // StartName (null)
		"",               // Password (null)
		"",               // Arguments (null)
		cmp,              // Component_
		s.Description,    // Description (nullable)
	)

	return tbl
}

// ── ServiceControl ────────────────────────────────────────────────────────────

func buildServiceControl(m *model.MSI, s model.Service) *Table {
	tbl := &Table{
		Name:     "ServiceControl",
		CodePage: m.CodePage,
		Columns: []Column{
			{Name: "ServiceControl", Type: Str(72), PK: true},
			{Name: "Name", Type: Loc(255)},
			{Name: "Event", Type: I2()},
			{Name: "Arguments", Type: NStr(255)},
			{Name: "Wait", Type: NI2()},
			{Name: "Component_", Type: Str(72)},
		},
	}

	var cmp string
	if len(m.Files) > 0 {
		cmp = componentName(m.Files[0].Destination)
	}

	tbl.AddRow(
		s.Name,     // ServiceControl (PK)
		s.Name,     // Name
		"162",      // Event: install Stop(2) | uninstall Stop(32) | uninstall Delete(128) = 162
		"",         // Arguments (null)
		"1",        // Wait = true
		cmp,        // Component_
	)

	return tbl
}
