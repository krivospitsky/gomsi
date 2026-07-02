// Package idt is the MVP backend. It renders the model into msitools .idt
// table files and a CAB archive, then shells out to msibuild to assemble the
// final .msi.
//
// This file is the ONLY place with IDT serialization logic. Table-builder
// files (tables_core.go, tables_service.go, …) use the types and functions
// here; they never assemble tab/string content directly.
package idt

import (
	"bytes"
	"fmt"
	"os"
	"strings"
)

// ── Column type ──────────────────────────────────────────────────────────────

// ColType is an IDT column type definition string such as "s72" or "I4".
type ColType struct {
	kind byte   // s S l L v V i I
	size int    // 0–255 for strings, 2 or 4 for integers
}

func (c ColType) String() string { return fmt.Sprintf("%c%d", c.kind, c.size) }

func (c ColType) isInt() bool     { return c.kind == 'i' || c.kind == 'I' }
func (c ColType) nullable() bool  { return c.kind >= 'A' && c.kind <= 'Z' }

// Column-type constructors.  n is the max character count for string/binary
// columns (0 = unlimited); integer types ignore n.
func Str(n int) ColType   { return ColType{'s', n} }
func NStr(n int) ColType  { return ColType{'S', n} }
func Loc(n int) ColType   { return ColType{'l', n} }
func NLoc(n int) ColType  { return ColType{'L', n} }
func Bin(n int) ColType   { return ColType{'v', n} }
func NBin(n int) ColType  { return ColType{'V', n} }
func I2() ColType         { return ColType{'i', 2} }
func NI2() ColType        { return ColType{'I', 2} }
func I4() ColType         { return ColType{'i', 4} }
func NI4() ColType        { return ColType{'I', 4} }

// ── Column ───────────────────────────────────────────────────────────────────

// Column describes one column of an IDT table.
type Column struct {
	Name string
	Type ColType
	PK   bool // true if this column is part of the primary key
}

// ── Table ────────────────────────────────────────────────────────────────────

// Table is an IDT table under construction. Table-builder files populate
// Columns and Rows, then call Render or WriteFile.
//
// CodePage controls codepage encoding:
//   - 0 (default): auto-detect — ASCII if all cells are ≤ 0x7F; else
//     CP1251 (Cyrillic, tried first) or CP1252 (Latin), whichever fits.
//   - 1251 or 1252: force the encoding.
type Table struct {
	Name     string
	CodePage int
	Columns  []Column
	Rows     [][]string
}

// AddRow appends a row. Callers must supply exactly len(Columns) values.
func (t *Table) AddRow(values ...string) { t.Rows = append(t.Rows, values) }

// Render serialises the table to the IDT archive format (CRLF line endings).
func (t *Table) Render() ([]byte, error) {
	if len(t.Columns) == 0 {
		return nil, fmt.Errorf("table %q: no columns", t.Name)
	}
	for i, row := range t.Rows {
		if len(row) != len(t.Columns) {
			return nil, fmt.Errorf("table %q: row %d has %d values, want %d",
				t.Name, i+1, len(row), len(t.Columns))
		}
	}

	cp, err := t.effectiveCodePage()
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	// ── Row 1: column names ──
	for i, col := range t.Columns {
		if i > 0 {
			buf.WriteByte('\t')
		}
		buf.WriteString(col.Name)
	}
	buf.WriteString("\r\n")

	// ── Row 2: column type definitions ──
	for i, col := range t.Columns {
		if i > 0 {
			buf.WriteByte('\t')
		}
		buf.WriteString(col.Type.String())
	}
	buf.WriteString("\r\n")

	// ── Row 3: [code-page] table-name [PK columns] ──
	if cp > 0 {
		fmt.Fprintf(&buf, "%d\t%s", cp, t.Name)
	} else {
		buf.WriteString(t.Name)
	}
	for _, col := range t.Columns {
		if col.PK {
			fmt.Fprintf(&buf, "\t%s", col.Name)
		}
	}
	buf.WriteString("\r\n")

	// ── Rows 4+: data ──
	for _, row := range t.Rows {
		for ci, cell := range row {
			if ci > 0 {
				buf.WriteByte('\t')
			}
			if cell == "" {
				continue // empty field  →  NULL
			}
			col := t.Columns[ci]
			if col.Type.isInt() {
				buf.WriteString(cell)
			} else {
				s := escapeControl(cell)
				if cp > 0 {
					enc, err := cpEncode(cp, s)
					if err != nil {
						return nil, fmt.Errorf("table %q: encode cell %d: %w", t.Name, ci, err)
					}
					buf.Write(enc)
				} else {
					buf.WriteString(s)
				}
			}
		}
		buf.WriteString("\r\n")
	}

	return buf.Bytes(), nil
}

// WriteFile renders the table and writes it to path.
func (t *Table) WriteFile(path string) error {
	data, err := t.Render()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// effectiveCodePage returns the actual code page to use, auto-detecting when
// CodePage is 0.
func (t *Table) effectiveCodePage() (int, error) {
	if t.CodePage != 0 {
		return t.CodePage, nil
	}

	// Quick scan: any non-ASCII rune?
	hasNonASCII := false
outer:
	for _, row := range t.Rows {
		for _, cell := range row {
			for _, r := range cell {
				if r > 0x7F {
					hasNonASCII = true
					break outer
				}
			}
		}
	}
	if !hasNonASCII {
		return 0, nil
	}

	// Try 1251 (Cyrillic, Russian-first) then 1252 (Latin).
	for _, cp := range []int{1251, 1252} {
		enc := cpEnc(cp)
		ok := true
		for _, row := range t.Rows {
			for _, cell := range row {
				for _, r := range cell {
					if r > 0x7F {
						if _, found := enc[rune(r)]; !found {
							ok = false
							break
						}
					}
				}
				if !ok {
					break
				}
			}
			if !ok {
				break
			}
		}
		if ok {
			return cp, nil
		}
	}

	return 0, fmt.Errorf("table %q: non-ASCII runes not representable in CP1251 or CP1252", t.Name)
}

// ── Codepage encoding ────────────────────────────────────────────────────────

var (
	cp1251Enc map[rune]byte
	cp1252Enc map[rune]byte
)

func cpEnc(cp int) map[rune]byte {
	switch cp {
	case 1251:
		return cp1251Enc
	case 1252:
		return cp1252Enc
	default:
		return nil
	}
}

func cpEncode(cp int, s string) ([]byte, error) {
	enc := cpEnc(cp)
	if enc == nil {
		return nil, fmt.Errorf("unsupported codepage %d", cp)
	}
	buf := make([]byte, 0, len(s))
	for _, r := range s {
		if r < 0x80 {
			buf = append(buf, byte(r))
			continue
		}
		b, ok := enc[r]
		if !ok {
			return nil, fmt.Errorf("rune %U not representable in codepage %d", r, cp)
		}
		buf = append(buf, b)
	}
	return buf, nil
}

func init() { initCodePages() }

func initCodePages() {
	// ── CP1252 (Latin / Western European) ──
	cp1252Enc = make(map[rune]byte)
	// ASCII is handled by cpEncode's fast path; only 0x80-0xFF.
	// Latin-1 supplement: byte 0xA0–0xFF → U+00A0–U+00FF.
	for i := 0xA0; i <= 0xFF; i++ {
		b := byte(i)
		cp1252Enc[rune(b)] = b
	}
	// Override 0x80–0x9F with Windows-1252 specific characters.
	cp1252Def := []struct {
		b byte
		r rune
	}{
		{0x80, 0x20AC}, // €
		{0x82, 0x201A}, // ‚
		{0x83, 0x0192}, // ƒ
		{0x84, 0x201E}, // „
		{0x85, 0x2026}, // …
		{0x86, 0x2020}, // †
		{0x87, 0x2021}, // ‡
		{0x88, 0x02C6}, // ˆ
		{0x89, 0x2030}, // ‰
		{0x8A, 0x0160}, // Š
		{0x8B, 0x2039}, // ‹
		{0x8C, 0x0152}, // Œ
		{0x8E, 0x017D}, // Ž
		{0x91, 0x2018}, // ‘
		{0x92, 0x2019}, // ’
		{0x93, 0x201C}, // “
		{0x94, 0x201D}, // ”
		{0x95, 0x2022}, // •
		{0x96, 0x2013}, // –
		{0x97, 0x2014}, // —
		{0x98, 0x02DC}, // ˜
		{0x99, 0x2122}, // ™
		{0x9A, 0x0161}, // š
		{0x9B, 0x203A}, // ›
		{0x9C, 0x0153}, // œ
		{0x9E, 0x017E}, // ž
		{0x9F, 0x0178}, // Ÿ
	}
	for _, e := range cp1252Def {
		cp1252Enc[e.r] = e.b
	}

	// ── CP1251 (Cyrillic) ──
	cp1251Enc = make(map[rune]byte)
	// 0x80–0x9F specific entries.
	cp1251Def := []struct {
		b byte
		r rune
	}{
		{0x80, 0x0402}, // Ђ
		{0x81, 0x0403}, // Ѓ
		{0x82, 0x201A}, // ‚
		{0x83, 0x0453}, // ѓ
		{0x84, 0x201E}, // „
		{0x85, 0x2026}, // …
		{0x86, 0x2020}, // †
		{0x87, 0x2021}, // ‡
		{0x88, 0x20AC}, // €
		{0x89, 0x2030}, // ‰
		{0x8A, 0x0409}, // Љ
		{0x8B, 0x2039}, // ‹
		{0x8C, 0x040A}, // Њ
		{0x8D, 0x040C}, // Ќ
		{0x8E, 0x040B}, // Ћ
		{0x8F, 0x040F}, // Џ
		{0x90, 0x0452}, // ђ
		{0x91, 0x2018}, // '
		{0x92, 0x2019}, // '
		{0x93, 0x201C}, // "
		{0x94, 0x201D}, // "
		{0x95, 0x2022}, // •
		{0x96, 0x2013}, // –
		{0x97, 0x2014}, // —
		{0x98, 0x2122}, // ™
		{0x99, 0x0459}, // љ
		{0x9A, 0x203A}, // ›
		{0x9B, 0x045A}, // њ
		{0x9C, 0x045C}, // ќ
		{0x9D, 0x045B}, // ћ
		{0x9E, 0x045F}, // џ
	}
	for _, e := range cp1251Def {
		cp1251Enc[e.r] = e.b
	}
	// 0xA0–0xBF mixed entries.
	cp1251Mixed := []struct {
		b byte
		r rune
	}{
		{0xA0, 0x00A0}, //  
		{0xA1, 0x040E}, // Ў
		{0xA2, 0x045E}, // ў
		{0xA3, 0x0408}, // Ј
		{0xA4, 0x00A4}, // ¤
		{0xA5, 0x0490}, // Ґ
		{0xA6, 0x00A6}, // ¦
		{0xA7, 0x00A7}, // §
		{0xA8, 0x0401}, // Ё
		{0xA9, 0x00A9}, // ©
		{0xAA, 0x0404}, // Є
		{0xAB, 0x00AB}, // «
		{0xAC, 0x00AC}, // ¬
		{0xAD, 0x00AD}, // ­
		{0xAE, 0x00AE}, // ®
		{0xAF, 0x0407}, // Ї
		{0xB0, 0x00B0}, // °
		{0xB1, 0x00B1}, // ±
		{0xB2, 0x0406}, // І
		{0xB3, 0x0456}, // і
		{0xB4, 0x0491}, // ґ
		{0xB5, 0x00B5}, // µ
		{0xB6, 0x00B6}, // ¶
		{0xB7, 0x00B7}, // ·
		{0xB8, 0x0451}, // ё
		{0xB9, 0x2116}, // №
		{0xBA, 0x0454}, // є
		{0xBB, 0x00BB}, // »
		{0xBC, 0x0458}, // ј
		{0xBD, 0x0405}, // Ѕ
		{0xBE, 0x0455}, // ѕ
		{0xBF, 0x0457}, // ї
	}
	for _, e := range cp1251Mixed {
		cp1251Enc[e.r] = e.b
	}
	// 0xC0–0xFF: Cyrillic U+0410–U+044F linearly.
	for i := 0xC0; i <= 0xFF; i++ {
		b := byte(i)
		cp1251Enc[rune(0x0410)+rune(i-0xC0)] = b
	}
}

// ── Control-character escaping ───────────────────────────────────────────────

// escapeControl replaces control bytes with their decimal IDT escape sequence.
// The bytes that conflict with the IDT field / line delimiters are:
//
//	NULL(0x00)→21  BS(0x08)→27  HT(0x09)→16  LF(0x0A)→25  FF(0x0C)→24  CR(0x0D)→17
func escapeControl(s string) string {
	has := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case 0x00, 0x08, 0x09, 0x0A, 0x0C, 0x0D:
			has = true
			break
		}
		if has {
			break
		}
	}
	if !has {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 10)
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case 0x00:
			b.WriteString("21")
		case 0x08:
			b.WriteString("27")
		case 0x09:
			b.WriteString("16")
		case 0x0A:
			b.WriteString("25")
		case 0x0C:
			b.WriteString("24")
		case 0x0D:
			b.WriteString("17")
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
