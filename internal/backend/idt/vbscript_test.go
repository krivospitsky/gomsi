package idt

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krivospitsky/gomsi/internal/model"
)

func TestTranslateTemplate_Simple(t *testing.T) {
	params := []model.Parameter{
		{Property: "SERVERURL"},
		{Property: "TOKEN"},
	}
	tmpl := `{"url": "{{.SERVERURL}}", "token": "{{.TOKEN}}"}`

	got, err := translateTemplate(tmpl, params)
	if err != nil {
		t.Fatalf("translateTemplate: %v", err)
	}

	want := `{"url": "__GOMSI_SERVERURL__", "token": "__GOMSI_TOKEN__"}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTranslateTemplate_UnknownProp(t *testing.T) {
	params := []model.Parameter{
		{Property: "SERVERURL"},
	}
	tmpl := `{{.UNKNOWN}}`

	_, err := translateTemplate(tmpl, params)
	if err == nil {
		t.Fatal("expected error for unknown property")
	}
}

func TestTranslateTemplate_UnsupportedRange(t *testing.T) {
	params := []model.Parameter{
		{Property: "SERVERURL"},
	}
	tmpl := `{{range .Items}}x{{end}}`

	_, err := translateTemplate(tmpl, params)
	if err == nil {
		t.Fatal("expected error for range construct")
	}
}

func TestTranslateTemplate_UnsupportedIf(t *testing.T) {
	params := []model.Parameter{
		{Property: "SERVERURL"},
	}
	tmpl := `{{if .SERVERURL}}x{{end}}`

	_, err := translateTemplate(tmpl, params)
	if err == nil {
		t.Fatal("expected error for if construct")
	}
}

func TestTranslateTemplate_EmptyField(t *testing.T) {
	params := []model.Parameter{
		{Property: "SERVERURL"},
	}
	tmpl := `{{.}}`

	_, err := translateTemplate(tmpl, params)
	if err == nil {
		t.Fatal("expected error for empty field reference")
	}
}

func TestTranslateTemplate_DotDot(t *testing.T) {
	params := []model.Parameter{
		{Property: "SERVERURL"},
	}
	tmpl := `{{.SERVERURL.SUB}}`

	_, err := translateTemplate(tmpl, params)
	if err == nil {
		t.Fatal("expected error for sub-field reference")
	}
}

func TestTranslateTemplate_NoMatches(t *testing.T) {
	params := []model.Parameter{
		{Property: "SERVERURL"},
	}
	tmpl := `{"key": "value"}`

	got, err := translateTemplate(tmpl, params)
	if err != nil {
		t.Fatalf("translateTemplate: %v", err)
	}
	if got != tmpl {
		t.Errorf("got %q, want %q", got, tmpl)
	}
}

func TestTranslateTemplate_MultipleLines(t *testing.T) {
	params := []model.Parameter{
		{Property: "HOST"},
		{Property: "PORT"},
	}
	tmpl := "host={{.HOST}}\nport={{.PORT}}\n"

	got, err := translateTemplate(tmpl, params)
	if err != nil {
		t.Fatalf("translateTemplate: %v", err)
	}

	want := "host=__GOMSI_HOST__\nport=__GOMSI_PORT__\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildVBScript_EmptySkeleton(t *testing.T) {
	params := []model.Parameter{
		{Property: "SERVERURL"},
	}
	vbs := buildVBScript("", params)

	// Verify the basic structure.
	s := string(vbs)
	if !strings.Contains(s, "Function WriteConfig()") {
		t.Error("missing Function WriteConfig()")
	}
	if !strings.Contains(s, "Session.Property(\"CustomActionData\")") {
		t.Error("missing CustomActionData read")
	}
	if !strings.Contains(s, "__GOMSI_SERVERURL__") {
		t.Error("missing SERVERURL sentinel")
	}
	if !strings.Contains(s, "parts(1)") {
		t.Error("missing parts(1) reference")
	}
	if !strings.Contains(s, "CreateObject(\"Scripting.FileSystemObject\")") {
		t.Error("missing FileSystemObject")
	}
	if !strings.Contains(s, "CreateTextFile(parts(0)") {
		t.Error("missing CreateTextFile")
	}
}

func TestBuildVBScript_NoParams(t *testing.T) {
	vbs := buildVBScript("static content", nil)
	s := string(vbs)

	if !strings.Contains(s, "content = \"static content\"") {
		t.Errorf("expected literal content, got:\n%s", s)
	}

	// No Replace calls since there are no params.
	if strings.Count(s, "Replace(content") != 0 {
		t.Error("expected no Replace calls with no params")
	}
}

func TestBuildVBScript_MultipleParams(t *testing.T) {
	params := []model.Parameter{
		{Property: "A"},
		{Property: "B"},
		{Property: "C"},
	}
	vbs := buildVBScript("x", params)
	s := string(vbs)

	if !strings.Contains(s, "Replace(content, \"__GOMSI_A__\", parts(1))") {
		t.Error("missing sentinel A → parts(1)")
	}
	if !strings.Contains(s, "Replace(content, \"__GOMSI_B__\", parts(2))") {
		t.Error("missing sentinel B → parts(2)")
	}
	if !strings.Contains(s, "Replace(content, \"__GOMSI_C__\", parts(3))") {
		t.Error("missing sentinel C → parts(3)")
	}
}

func TestBuildVBScript_QuoteEscaping(t *testing.T) {
	skeleton := `{"message": "hello"}`
	vbs := buildVBScript(skeleton, nil)
	s := string(vbs)

	// The skeleton contains double quotes — they must be escaped as "" in VBScript.
	if !strings.Contains(s, "\"\"message\"\": \"\"hello\"\"") {
		t.Errorf("quotes not properly escaped, got:\n%s", s)
	}
}

func TestGenerateVBScript_Golden(t *testing.T) {
	m := testModelWithConfig()
	vbs, err := generateVBScript(m)
	if err != nil {
		t.Fatalf("generateVBScript: %v", err)
	}

	path := filepath.Join("testdata", "config", "WriteConfig.vbs")
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(path, vbs, 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	} else {
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if !bytes.Equal(vbs, want) {
			t.Errorf("WriteConfig.vbs: golden mismatch; run `go test -update` to regenerate")
		}
	}
}

func TestGenerateVBScript_FileNotExist(t *testing.T) {
	m := testModel()
	m.Config = model.Config{Template: filepath.Join(t.TempDir(), "nonexistent.tpl"), Output: "x"}
	_, err := generateVBScript(m)
	if err == nil {
		t.Fatal("expected error for non-existent template file")
	}
}
