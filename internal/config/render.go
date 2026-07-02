// Package config renders on-disk configuration files from a Go text/template
// using install parameters as template variables.
package config

import (
	"fmt"
	"io"
	"os"
	"text/template"

	"github.com/krivospitsky/gomsi/internal/model"
)

// Render executes the template referenced by cfg using params as template
// data and writes the result to outDir/cfg.Output.
//
// Parameter values are exposed to the template by their Property name (the
// same identifier used for MSI properties and msiexec CLI arguments), so a
// template may reference {{.SERVERURL}}.
func Render(cfg model.Config, params []model.Parameter, overrides map[string]string, outDir string) error {
	if cfg.Template == "" {
		return nil
	}

	tmpl, err := template.ParseFiles(cfg.Template)
	if err != nil {
		return fmt.Errorf("parse template %q: %w", cfg.Template, err)
	}

	data := make(map[string]string, len(params))
	for _, p := range params {
		if v, ok := overrides[p.Property]; ok {
			data[p.Property] = v
		} else {
			data[p.Property] = p.Default
		}
	}

	outPath := cfg.Output
	if outDir != "" {
		outPath = outDir + string(os.PathSeparator) + outPath
	}

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create config output %q: %w", outPath, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("execute template %q: %w", cfg.Template, err)
	}
	return nil
}

// RenderTo executes the template and writes to w instead of a file. Useful
// for testing and dry-runs.
func RenderTo(tmpl *template.Template, data map[string]string, w io.Writer) error {
	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}
	return nil
}
