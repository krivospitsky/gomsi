package idt

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/krivospitsky/gomsi/internal/model"
)

// generateVBScript reads the Go template referenced by m.Config.Template,
// translates {{.PROPERTY}} references to __GOMSI_PROPERTY__ sentinels, wraps
// the result in a VBScript function, and returns the VBScript bytes.
func generateVBScript(m *model.MSI) ([]byte, error) {
	data, err := os.ReadFile(m.Config.Template)
	if err != nil {
		return nil, fmt.Errorf("read config template %q: %w", m.Config.Template, err)
	}

	translated, err := translateTemplate(string(data), m.Parameters)
	if err != nil {
		return nil, fmt.Errorf("translate template: %w", err)
	}

	return buildVBScript(translated, m.Parameters), nil
}

// translateTemplate converts {{.PROPERTY}} patterns to __GOMSI_PROPERTY__
// sentinels and validates that no unsupported template constructs are used.
func translateTemplate(tmpl string, params []model.Parameter) (string, error) {
	knownProps := make(map[string]bool, len(params))
	for _, p := range params {
		knownProps[p.Property] = true
	}

	re := regexp.MustCompile(`\{\{([^}]*)\}\}`)
	var result strings.Builder
	lastEnd := 0

	for _, m := range re.FindAllStringSubmatchIndex(tmpl, -1) {
		start, end := m[0], m[1]
		inner := strings.TrimSpace(tmpl[m[2]:m[3]])

		result.WriteString(tmpl[lastEnd:start])
		lastEnd = end

		if !strings.HasPrefix(inner, ".") {
			return "", fmt.Errorf(
				"unsupported template construct %q at position %d: only {{.PROPERTY}} substitutions are supported",
				tmpl[start:end], start)
		}

		fieldName := strings.TrimSpace(inner[1:])
		if fieldName == "" {
			return "", fmt.Errorf(
				"empty property reference %q at position %d", tmpl[start:end], start)
		}

		if !knownProps[fieldName] {
			return "", fmt.Errorf(
				"unknown property %q at position %d: must match a parameter 'property' name",
				fieldName, start)
		}

		result.WriteString("__GOMSI_")
		result.WriteString(fieldName)
		result.WriteString("__")
	}

	result.WriteString(tmpl[lastEnd:])
	return result.String(), nil
}

// buildVBScript wraps the translated skeleton into a VBScript function that
// reads CustomActionData, splits by '|', replaces each __GOMSI_<PROP>__
// sentinel with the corresponding parameter value from parts(), and writes
// the rendered config file via FileSystemObject.
func buildVBScript(skeleton string, params []model.Parameter) []byte {
	var b strings.Builder

	b.WriteString("Option Explicit\n\n")
	b.WriteString("Function WriteConfig()\n")
	b.WriteString("    On Error Resume Next\n")
	b.WriteString("    Dim data, parts, fso, ts, content\n")
	b.WriteString("    data = Session.Property(\"CustomActionData\")\n")
	b.WriteString("    parts = Split(data, \"|\")\n")
	b.WriteString("    content = ")

	// Embed skeleton as VBScript string literals joined by vbCrLf.
	lines := strings.Split(skeleton, "\n")
	for i, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		escaped := strings.ReplaceAll(line, "\"", "\"\"")
		if i == 0 {
			b.WriteString("\"")
			b.WriteString(escaped)
			b.WriteString("\"")
		} else {
			b.WriteString(" & vbCrLf & \"")
			b.WriteString(escaped)
			b.WriteString("\"")
		}
	}
	b.WriteString("\n")

	// Replace sentinels: parts(0)=outputPath, parts(1..N)=parameter values.
	for i, p := range params {
		idx := i + 1
		fmt.Fprintf(&b,
			"    content = Replace(content, \"__GOMSI_%s__\", parts(%d))\n",
			p.Property, idx)
	}

	b.WriteString("    Set fso = CreateObject(\"Scripting.FileSystemObject\")\n")
	b.WriteString("    Set ts = fso.CreateTextFile(parts(0), True, False)\n")
	b.WriteString("    ts.Write content\n")
	b.WriteString("    ts.Close\n")
	b.WriteString("    If Err.Number <> 0 Then\n")
	b.WriteString("        WriteConfig = 3\n")
	b.WriteString("    Else\n")
	b.WriteString("        WriteConfig = 1\n")
	b.WriteString("    End If\n")
	b.WriteString("End Function\n")

	return []byte(b.String())
}
