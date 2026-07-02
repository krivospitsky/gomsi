// Package idt is the MVP backend. It renders the model into msitools .idt
// table files and a CAB archive, then shells out to msibuild to assemble the
// final .msi.
//
// This is a bootstrap stub: the table emission is not yet implemented. The
// scaffold is in place so the CLI can wire up a backend end-to-end while the
// detail of each MSI table is filled in incrementally.
package idt

import (
	"errors"

	"github.com/krivospitsky/gomsi/internal/backend"
	"github.com/krivospitsky/gomsi/internal/model"
)

// Writer implements backend.Writer using the msitools (IDT + CAB + msibuild) flow.
type Writer struct{}

// New returns an IDT-backed writer.
func New() *Writer { return &Writer{} }

// Compile-time check that the IDT writer satisfies the backend contract.
var _ backend.Writer = (*Writer)(nil)

// Write renders the model to an MSI file. Not yet implemented.
func (w *Writer) Write(m *model.MSI, outputPath string) error {
	_ = m
	_ = outputPath
	return errors.New("idt.Writer.Write: not implemented (MVP backend pending)")
}
