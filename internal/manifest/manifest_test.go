package manifest

import (
	"path/filepath"
	"testing"
)

func TestParse_YAML(t *testing.T) {
	path := filepath.Join("testdata", "installer.yaml")
	m, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if m.Product.Name != "MyAgent" {
		t.Errorf("Product.Name = %q, want MyAgent", m.Product.Name)
	}
	if m.Product.Version != "1.2.3" {
		t.Errorf("Product.Version = %q, want 1.2.3", m.Product.Version)
	}
	if m.Install.Directory != "MyAgent" {
		t.Errorf("Install.Directory = %q, want MyAgent", m.Install.Directory)
	}
	if len(m.Files) != 1 || m.Files[0].Source != "dist/myagent.exe" {
		t.Errorf("Files = %+v", m.Files)
	}
	if len(m.Services) != 1 || m.Services[0].Name != "myagent" {
		t.Errorf("Services = %+v", m.Services)
	}
	if len(m.Parameters) != 2 {
		t.Fatalf("Parameters = %+v (len %d)", m.Parameters, len(m.Parameters))
	}
	if m.Parameters[0].Name != "serverUrl" {
		t.Errorf("Parameters[0].Name = %q, want serverUrl", m.Parameters[0].Name)
	}
	if m.Parameters[0].Property != "SERVERURL" {
		t.Errorf("Parameters[0].Property = %q, want SERVERURL", m.Parameters[0].Property)
	}
	if m.Parameters[0].Required != true {
		t.Errorf("Parameters[0].Required = false, want true")
	}
	if m.Parameters[1].Name != "token" || m.Parameters[1].Type != "password" {
		t.Errorf("Parameters[1] = %+v", m.Parameters[1])
	}
	if m.Config.Template != "installer/config.tpl" {
		t.Errorf("Config.Template = %q", m.Config.Template)
	}
	if m.Config.Output != "config.json" {
		t.Errorf("Config.Output = %q", m.Config.Output)
	}
}

func TestParse_AutoCodesResolveToGUIDs(t *testing.T) {
	path := filepath.Join("testdata", "installer.yaml")
	m, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	for _, code := range []string{m.Product.UpgradeCode, m.Product.ProductCode} {
		if code == "auto" || code == "" {
			t.Errorf("code not resolved: %q", code)
		}
		if len(code) != 38 || code[0] != '{' || code[len(code)-1] != '}' {
			t.Errorf("code %q is not a braced GUID", code)
		}
	}
	// Two distinct codes must be generated.
	if m.Product.UpgradeCode == m.Product.ProductCode {
		t.Error("upgrade and product codes are identical")
	}
}

func TestParse_ExplicitCodesPreserved(t *testing.T) {
	path := filepath.Join("testdata", "explicit.yaml")
	m, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	const want = "{11111111-2222-3333-4444-555555555555}"
	if m.Product.UpgradeCode != want {
		t.Errorf("UpgradeCode = %q, want %q", m.Product.UpgradeCode, want)
	}
}
