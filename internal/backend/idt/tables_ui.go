package idt

import (
	"fmt"

	"github.com/krivospitsky/gomsi/internal/model"
)

// hasVisibleParam returns true when at least one parameter's UI visibility
// is not "never". Empty/missing UI field is treated as "auto" (visible).
func hasVisibleParam(m *model.MSI) bool {
	for _, p := range m.Parameters {
		if p.UI != "never" {
			return true
		}
	}
	return false
}

// uiTables returns the UI-related tables when the model has visible
// parameters. Returns nil when there are none.
func uiTables(m *model.MSI) []*Table {
	if !hasVisibleParam(m) {
		return nil
	}
	return []*Table{
		buildTextStyle(m),
		buildDialog(m),
		buildControl(m),
		buildControlEvent(m),
	}
}

// applyUIProperties appends Property rows for the auto-generated UI wizard
// to the Property table built by core.
func applyUIProperties(propTbl *Table, m *model.MSI) {
	propTbl.AddRow("DefaultUIFont", `{\DlgFont8}`)
	propTbl.AddRow("ButtonText_Next", "&Next >")
	propTbl.AddRow("ButtonText_Back", "< &Back")
	propTbl.AddRow("ButtonText_Cancel", "Cancel")
	propTbl.AddRow("ButtonText_Finish", "&Finish")
}

// applyUISequence modifies the InstallUISequence table for the auto-generated
// UI wizard: changes ExecuteAction's sequence from 4 to 1299 so it runs after
// the dialogs, and appends WelcomeDlg and ExitDlg entries.
func applyUISequence(seqTbl *Table, m *model.MSI) {
	for _, row := range seqTbl.Rows {
		if row[0] == "ExecuteAction" {
			row[2] = "1299"
			break
		}
	}
	seqTbl.AddRow("WelcomeDlg", "", "50")
	seqTbl.AddRow("ExitDlg", "", "1300")
}

// ── TextStyle ──────────────────────────────────────────────────────────────────

func buildTextStyle(m *model.MSI) *Table {
	tbl := &Table{
		Name:     "TextStyle",
		CodePage: m.CodePage,
		Columns: []Column{
			{Name: "TextStyle", Type: Str(72), PK: true},
			{Name: "FaceName", Type: Str(31)},
			{Name: "Size", Type: I2()},
			{Name: "Color", Type: NI4()},
			{Name: "StyleBits", Type: NI2()},
		},
	}
	tbl.AddRow("DlgFont8", "Verdana", "8", "", "")
	return tbl
}

// ── Dialog ─────────────────────────────────────────────────────────────────────

func buildDialog(m *model.MSI) *Table {
	tbl := &Table{
		Name:     "Dialog",
		CodePage: m.CodePage,
		Columns: []Column{
			{Name: "Dialog", Type: Str(72), PK: true},
			{Name: "HCentering", Type: I2()},
			{Name: "VCentering", Type: I2()},
			{Name: "Width", Type: I2()},
			{Name: "Height", Type: I2()},
			{Name: "Attributes", Type: NI4()},
			{Name: "Title", Type: NStr(128)},
			{Name: "Control_First", Type: Str(72)},
			{Name: "Control_Default", Type: NStr(72)},
			{Name: "Control_Cancel", Type: NStr(72)},
		},
	}

	title := fmt.Sprintf("[ProductName] Setup")

	tbl.AddRow("WelcomeDlg", "50", "50", "370", "270", "3", title, "Title", "NextBtn", "CancelBtn")
	tbl.AddRow("ParametersDlg", "50", "50", "370", "270", "3", title, "Title", "NextBtn", "CancelBtn")
	tbl.AddRow("VerifyReadyDlg", "50", "50", "370", "270", "3", title, "Title", "NextBtn", "CancelBtn")
	tbl.AddRow("ExitDlg", "50", "50", "370", "270", "3", title, "FinishBtn", "FinishBtn", "")

	return tbl
}

// ── Control ────────────────────────────────────────────────────────────────────

const (
	attrVisible  = 0x00000001
	attrEnabled  = 0x00000002
	attrPassword = 0x00200000
	attrTransparent = 0x00010000
)

// visibleParams returns the subset of parameters with ui != "never".
func visibleParams(m *model.MSI) []model.Parameter {
	var out []model.Parameter
	for _, p := range m.Parameters {
		if p.UI != "never" {
			out = append(out, p)
		}
	}
	return out
}

// editAttr returns the Control Attributes value for an Edit control.
// Password-type parameters get the password input bit.
func editAttr(p model.Parameter) string {
	attr := attrVisible | attrEnabled
	if p.Type == "password" {
		attr |= attrPassword
	}
	return fmt.Sprintf("%d", attr)
}

func buildControl(m *model.MSI) *Table {
	tbl := &Table{
		Name:     "Control",
		CodePage: m.CodePage,
		Columns: []Column{
			{Name: "Dialog_", Type: Str(72), PK: true},
			{Name: "Control", Type: Str(50), PK: true},
			{Name: "Type", Type: Str(50)},
			{Name: "X", Type: I2()},
			{Name: "Y", Type: I2()},
			{Name: "Width", Type: I2()},
			{Name: "Height", Type: I2()},
			{Name: "Attributes", Type: NI4()},
			{Name: "Property", Type: NStr(50)},
			{Name: "Text", Type: NStr(0)},
			{Name: "Control_Next", Type: NStr(50)},
			{Name: "Help", Type: NStr(0)},
		},
	}

	// ── WelcomeDlg ──
	addControl(tbl, "WelcomeDlg", "Title", "Text", 20, 10, 330, 20,
		fmt.Sprintf("%d", attrVisible|attrEnabled|attrTransparent),
		"", "[DefaultUIFont]Welcome to [ProductName] Setup", "Body")
	addControl(tbl, "WelcomeDlg", "Body", "Text", 20, 35, 330, 195,
		fmt.Sprintf("%d", attrVisible|attrEnabled|attrTransparent),
		"", "[DefaultUIFont]This wizard will install [ProductName] on your computer. Click Next to continue or Cancel to exit.", "BottomLine")
	addControl(tbl, "WelcomeDlg", "BottomLine", "Line", 0, 234, 370, 1,
		fmt.Sprintf("%d", attrVisible|attrEnabled),
		"", "", "NextBtn")
	addControl(tbl, "WelcomeDlg", "NextBtn", "PushButton", 236, 243, 56, 17,
		fmt.Sprintf("%d", attrVisible|attrEnabled),
		"", "[ButtonText_Next]", "CancelBtn")
	addControl(tbl, "WelcomeDlg", "CancelBtn", "PushButton", 304, 243, 56, 17,
		fmt.Sprintf("%d", attrVisible|attrEnabled),
		"", "[ButtonText_Cancel]", "Title")

	// ── ParametersDlg ──
	params := visibleParams(m)
	yOff := 55
	addControl(tbl, "ParametersDlg", "Title", "Text", 20, 10, 330, 20,
		fmt.Sprintf("%d", attrVisible|attrEnabled|attrTransparent),
		"", "[DefaultUIFont]Configure [ProductName]", "Body")
	addControl(tbl, "ParametersDlg", "Body", "Text", 20, 35, 330, 15,
		fmt.Sprintf("%d", attrVisible|attrEnabled|attrTransparent),
		"", "[DefaultUIFont]Set the following parameters:", "Label_"+params[0].Property)

	for i, p := range params {
		labelID := "Label_" + p.Property
		editID := "Edit_" + p.Property
		y := yOff + 35*i

		label := p.Title
		if label == "" {
			label = p.Name
		}

		nextID := editID
		if i == len(params)-1 {
			nextID = "BackBtn"
		}

		addControl(tbl, "ParametersDlg", labelID, "Text", 20, y, 110, 15,
			fmt.Sprintf("%d", attrVisible|attrEnabled|attrTransparent),
			"", "[DefaultUIFont]"+label, nextID)

		var editNextID string
		if i < len(params)-1 {
			editNextID = "Label_" + params[i+1].Property
		} else {
			editNextID = "BackBtn"
		}

		addControl(tbl, "ParametersDlg", editID, "Edit", 140, y, 210, 17,
			editAttr(p), p.Property, "", editNextID)
	}

	addControl(tbl, "ParametersDlg", "BottomLine", "Line", 0, 234, 370, 1,
		fmt.Sprintf("%d", attrVisible|attrEnabled),
		"", "", "BackBtn")
	addControl(tbl, "ParametersDlg", "BackBtn", "PushButton", 180, 243, 56, 17,
		fmt.Sprintf("%d", attrVisible|attrEnabled),
		"", "[ButtonText_Back]", "NextBtn")
	addControl(tbl, "ParametersDlg", "NextBtn", "PushButton", 236, 243, 56, 17,
		fmt.Sprintf("%d", attrVisible|attrEnabled),
		"", "[ButtonText_Next]", "CancelBtn")
	addControl(tbl, "ParametersDlg", "CancelBtn", "PushButton", 304, 243, 56, 17,
		fmt.Sprintf("%d", attrVisible|attrEnabled),
		"", "[ButtonText_Cancel]", "Title")

	// ── VerifyReadyDlg ──
	addControl(tbl, "VerifyReadyDlg", "Title", "Text", 20, 10, 330, 20,
		fmt.Sprintf("%d", attrVisible|attrEnabled|attrTransparent),
		"", "[DefaultUIFont]Ready to Install [ProductName]", "Body")
	addControl(tbl, "VerifyReadyDlg", "Body", "Text", 20, 35, 330, 195,
		fmt.Sprintf("%d", attrVisible|attrEnabled|attrTransparent),
		"", "[DefaultUIFont]Click Next to begin the installation.", "BottomLine")
	addControl(tbl, "VerifyReadyDlg", "BottomLine", "Line", 0, 234, 370, 1,
		fmt.Sprintf("%d", attrVisible|attrEnabled),
		"", "", "BackBtn")
	addControl(tbl, "VerifyReadyDlg", "BackBtn", "PushButton", 180, 243, 56, 17,
		fmt.Sprintf("%d", attrVisible|attrEnabled),
		"", "[ButtonText_Back]", "NextBtn")
	addControl(tbl, "VerifyReadyDlg", "NextBtn", "PushButton", 236, 243, 56, 17,
		fmt.Sprintf("%d", attrVisible|attrEnabled),
		"", "[ButtonText_Next]", "CancelBtn")
	addControl(tbl, "VerifyReadyDlg", "CancelBtn", "PushButton", 304, 243, 56, 17,
		fmt.Sprintf("%d", attrVisible|attrEnabled),
		"", "[ButtonText_Cancel]", "Title")

	// ── ExitDlg ──
	addControl(tbl, "ExitDlg", "Title", "Text", 20, 10, 330, 20,
		fmt.Sprintf("%d", attrVisible|attrEnabled|attrTransparent),
		"", "[DefaultUIFont]Completed [ProductName] Setup", "Body")
	addControl(tbl, "ExitDlg", "Body", "Text", 20, 35, 330, 195,
		fmt.Sprintf("%d", attrVisible|attrEnabled|attrTransparent),
		"", "[DefaultUIFont][ProductName] has been installed successfully.", "BottomLine")
	addControl(tbl, "ExitDlg", "BottomLine", "Line", 0, 234, 370, 1,
		fmt.Sprintf("%d", attrVisible|attrEnabled),
		"", "", "FinishBtn")
	addControl(tbl, "ExitDlg", "FinishBtn", "PushButton", 236, 243, 56, 17,
		fmt.Sprintf("%d", attrVisible|attrEnabled),
		"", "[ButtonText_Finish]", "Title")

	return tbl
}

func addControl(tbl *Table, dialog, control, ctype string, x, y, w, h int, attr, prop, text, next string) {
	tbl.AddRow(dialog, control, ctype,
		fmt.Sprintf("%d", x), fmt.Sprintf("%d", y),
		fmt.Sprintf("%d", w), fmt.Sprintf("%d", h),
		attr, prop, text, next, "")
}

// ── ControlEvent ───────────────────────────────────────────────────────────────

func buildControlEvent(m *model.MSI) *Table {
	tbl := &Table{
		Name:     "ControlEvent",
		CodePage: m.CodePage,
		Columns: []Column{
			{Name: "Dialog_", Type: Str(72), PK: true},
			{Name: "Control_", Type: Str(50), PK: true},
			{Name: "Event", Type: NStr(128), PK: true},
			{Name: "Argument", Type: NStr(255)},
			{Name: "Condition", Type: NStr(255)},
			{Name: "Ordering", Type: NI2()},
		},
	}

	addEvent := func(dialog, control, event, arg, cond string) {
		tbl.AddRow(dialog, control, event, arg, cond, "1")
	}

	// WelcomeDlg
	addEvent("WelcomeDlg", "NextBtn", "NewDialog", "ParametersDlg", "1")
	addEvent("WelcomeDlg", "CancelBtn", "EndDialog", "Exit", "1")

	// ParametersDlg
	addEvent("ParametersDlg", "BackBtn", "NewDialog", "WelcomeDlg", "1")
	addEvent("ParametersDlg", "NextBtn", "NewDialog", "VerifyReadyDlg", "1")
	addEvent("ParametersDlg", "CancelBtn", "EndDialog", "Exit", "1")

	// VerifyReadyDlg
	addEvent("VerifyReadyDlg", "BackBtn", "NewDialog", "ParametersDlg", "1")
	addEvent("VerifyReadyDlg", "NextBtn", "EndDialog", "Return", "1")
	addEvent("VerifyReadyDlg", "CancelBtn", "EndDialog", "Exit", "1")

	// ExitDlg
	addEvent("ExitDlg", "FinishBtn", "EndDialog", "Exit", "1")

	return tbl
}
