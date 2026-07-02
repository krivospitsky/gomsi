// Package manifest parses YAML/JSON manifests into the internal MSI model.
package manifest

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/krivospitsky/gomsi/internal/model"
)

// rawManifest mirrors the manifest file structure as authored by users.
// The "service" key is singular in the manifest but maps to the model's
// Services slice to keep the door open for multiple services later.
type rawManifest struct {
	Product    rawProduct             `yaml:"product" json:"product"`
	Install    rawInstall             `yaml:"install" json:"install"`
	Files      []rawFile              `yaml:"files" json:"files"`
	Service    *rawService            `yaml:"service" json:"service"`
	Parameters map[string]rawParameter `yaml:"parameters" json:"parameters"`
	Config     *rawConfig             `yaml:"config" json:"config"`
}

type rawProduct struct {
	Name         string `yaml:"name" json:"name"`
	Version      string `yaml:"version" json:"version"`
	Manufacturer string `yaml:"manufacturer" json:"manufacturer"`
	UpgradeCode  string `yaml:"upgradeCode" json:"upgradeCode"`
	ProductCode  string `yaml:"productCode" json:"productCode"`
}

type rawInstall struct {
	Directory string `yaml:"directory" json:"directory"`
}

type rawFile struct {
	Source      string `yaml:"source" json:"source"`
	Destination string `yaml:"destination" json:"destination"`
}

type rawService struct {
	Name        string `yaml:"name" json:"name"`
	DisplayName string `yaml:"displayName" json:"displayName"`
	Description string `yaml:"description" json:"description"`
	Start       string `yaml:"start" json:"start"`
}

type rawParameter struct {
	Property string `yaml:"property" json:"property"`
	Type     string `yaml:"type" json:"type"`
	Title    string `yaml:"title" json:"title"`
	Required bool   `yaml:"required" json:"required"`
	Default  string `yaml:"default" json:"default"`
	Validate string `yaml:"validate" json:"validate"`
	UI       string `yaml:"ui" json:"ui"`
}

type rawConfig struct {
	Template string `yaml:"template" json:"template"`
	Output   string `yaml:"output" json:"output"`
}

// Parse reads a manifest file (YAML or JSON, chosen by extension) and
// returns the corresponding internal model. A code value of "auto" for
// UpgradeCode or ProductCode is resolved into a freshly generated GUID.
func Parse(path string) (*model.MSI, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest %q: %w", path, err)
	}

	var raw rawManifest
	switch lower := strings.ToLower(filepath.Ext(path)); lower {
	case ".json":
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse manifest %q as JSON: %w", path, err)
		}
	default: // .yaml, .yml, or anything else: treat as YAML
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse manifest %q as YAML: %w", path, err)
		}
	}

	return convert(raw)
}

func convert(r rawManifest) (*model.MSI, error) {
	m := &model.MSI{
		Product: model.Product{
			Name:         r.Product.Name,
			Version:      r.Product.Version,
			Manufacturer: r.Product.Manufacturer,
		},
		Install: model.Install{Directory: r.Install.Directory},
	}

	uc, err := resolveCode(r.Product.UpgradeCode)
	if err != nil {
		return nil, err
	}
	pc, err := resolveCode(r.Product.ProductCode)
	if err != nil {
		return nil, err
	}
	m.Product.UpgradeCode = uc
	m.Product.ProductCode = pc

	for _, f := range r.Files {
		m.Files = append(m.Files, model.File{Source: f.Source, Destination: f.Destination})
	}

	if r.Service != nil {
		m.Services = append(m.Services, model.Service{
			Name:        r.Service.Name,
			DisplayName: r.Service.DisplayName,
			Description: r.Service.Description,
			Start:       r.Service.Start,
		})
	}

	// Preserve manifest order by iterating the YAML map deterministically:
	// yaml.v3 decodes into map[string]T without order, so we sort by key to
	// make builds reproducible (a core design principle).
	for _, name := range sortedKeys(r.Parameters) {
		p := r.Parameters[name]
		m.Parameters = append(m.Parameters, model.Parameter{
			Name:     name,
			Property: p.Property,
			Type:     p.Type,
			Title:    p.Title,
			Required: p.Required,
			Default:  p.Default,
			Validate: p.Validate,
			UI:       p.UI,
		})
	}

	if r.Config != nil {
		m.Config = model.Config{Template: r.Config.Template, Output: r.Config.Output}
	}

	return m, nil
}

// resolveCode turns a code value into a GUID, generating one for "auto".
func resolveCode(code string) (string, error) {
	if code == "" || code == "auto" {
		return newGUID()
	}
	return code, nil
}

// newGUID generates an MSI-style braced GUID (v4).
func newGUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate GUID: %w", err)
	}
	// RFC 4122 v4 variant.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	s := hex.EncodeToString(b[:])
	return fmt.Sprintf("{%s-%s-%s-%s-%s}", s[0:8], s[8:12], s[12:16], s[16:20], s[20:32]), nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple insertion sort keeps the bootstrap dependency-free; map sizes
	// here are tiny.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	return keys
}
