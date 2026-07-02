// Package backend defines the contract between the internal MSI model and
// concrete MSI producers. The model never references a backend directly;
// each backend implements Writer.
package backend

import "github.com/krivospitsky/gomsi/internal/model"

// Writer produces an MSI package file from a model.
//
// Implementations must not require the caller to supply backend-specific
// types: the model is the only contract.
type Writer interface {
	// Write renders the package described by m to the output path.
	Write(m *model.MSI, outputPath string) error
}
