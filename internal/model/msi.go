// Package model defines the backend-agnostic internal representation of an
// MSI package. Parsers, templating, and parameter handling operate on this
// model; backend writers consume it. Nothing in this package may leak
// backend-specific (e.g. IDT) types.
package model

// MSI is the central abstraction for a package under construction.
type MSI struct {
	Product    Product
	Install    Install
	Files      []File
	Services   []Service
	Parameters []Parameter
	Config     Config
	// CodePage controls the Windows-125x codepage for non-ASCII strings in
	// the generated IDT tables. 0 (default) means auto-detect: CP1251 for
	// Cyrillic text, CP1252 for Latin-1 supplement. Common explicit values:
	// 1251 (Cyrillic/Russian), 1252 (Western European). Any string not
	// representable in the selected codepage causes the build to fail.
	CodePage int
}

// Product describes the product identity written into the MSI summary.
type Product struct {
	Name         string
	Version      string
	Manufacturer string
	UpgradeCode  string
	ProductCode  string
}

// Install describes where the package installs on the target system.
type Install struct {
	// Directory is the folder name created under Program Files.
	Directory string
}

// File is a single file shipped by the package.
type File struct {
	Source      string
	Destination string
}

// Service describes a Windows service registered by the package.
type Service struct {
	Name        string
	DisplayName string
	Description string
	// Start controls the service start mode. Common values: "auto", "manual".
	Start string
}

// Parameter is a first-class install parameter. Each parameter maps
// simultaneously to an MSI Property, a msiexec CLI argument, a UI dialog
// field, and a template variable.
type Parameter struct {
	// Name is the manifest key for the parameter (e.g. "serverUrl").
	Name string

	// Property is the MSI property name (e.g. "SERVERURL").
	Property string

	// Type is the value type, e.g. "string" or "password".
	Type string

	// Title is a human-readable label shown in the UI.
	Title string

	// Required marks the parameter as mandatory.
	Required bool

	// Default is the value used when none is supplied.
	Default string

	// Validate is an optional validation rule, e.g. "url".
	Validate string

	// UI controls visibility: "auto", "always", or "never".
	UI string
}

// Config describes how the on-disk config file is rendered at install time.
type Config struct {
	// Template is the path to a Go text/template file.
	Template string
	// Output is the rendered file name placed next to the binary.
	Output string
}
