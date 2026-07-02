package idt

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "update golden files in testdata/")

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

// ── ColType ───────────────────────────────────────────────────────────────────

func TestColType_String(t *testing.T) {
	cases := []struct {
		ct   ColType
		want string
	}{
		{Str(72), "s72"},
		{NStr(255), "S255"},
		{Loc(0), "l0"},
		{NLoc(50), "L50"},
		{Bin(0), "v0"},
		{NBin(0), "V0"},
		{I2(), "i2"},
		{NI2(), "I2"},
		{I4(), "i4"},
		{NI4(), "I4"},
		{Str(0), "s0"},
	}
	for _, c := range cases {
		if got := c.ct.String(); got != c.want {
			t.Errorf("ColType{%c,%d}.String() = %q, want %q", c.ct.kind, c.ct.size, got, c.want)
		}
	}
}

func TestColType_isInt(t *testing.T) {
	if !I2().isInt() {
		t.Error("I2().isInt() = false")
	}
	if !NI4().isInt() {
		t.Error("NI4().isInt() = false")
	}
	if Str(10).isInt() {
		t.Error("Str(10).isInt() = true")
	}
}

func TestColType_nullable(t *testing.T) {
	if NStr(10).nullable() == false {
		t.Error("NStr(10).nullable() = false")
	}
	if Str(10).nullable() == true {
		t.Error("Str(10).nullable() = true")
	}
	if NI2().nullable() == false {
		t.Error("NI2().nullable() = false")
	}
}

// ── Table: inline assertion tests ─────────────────────────────────────────────

func TestRender_SimpleASCII(t *testing.T) {
	tbl := &Table{
		Name: "Property",
		Columns: []Column{
			{Name: "Property", Type: Str(72), PK: true},
			{Name: "Value", Type: NStr(0)},
		},
	}
	tbl.AddRow("ProductName", "MyAgent")
	tbl.AddRow("ProductVersion", "1.0.0")

	got, err := tbl.Render()
	if err != nil {
		t.Fatal(err)
	}

	want := "Property\tValue\r\n" +
		"s72\tS0\r\n" +
		"Property\tProperty\r\n" +
		"ProductName\tMyAgent\r\n" +
		"ProductVersion\t1.0.0\r\n"

	if string(got) != want {
		t.Errorf("Render():\ngot:\n%q\nwant:\n%q", string(got), want)
	}
}

func TestRender_MultiRow(t *testing.T) {
	tbl := &Table{
		Name: "Dirs",
		Columns: []Column{
			{Name: "Directory", Type: Str(72), PK: true},
			{Name: "Directory_Parent", Type: NStr(72)},
			{Name: "DefaultDir", Type: NStr(255)},
		},
	}
	tbl.AddRow("TARGETDIR", "", "SourceDir")
	tbl.AddRow("ProgramFilesFolder", "TARGETDIR", ".")
	tbl.AddRow("INSTALLDIR", "ProgramFilesFolder", "MyApp")

	got, err := tbl.Render()
	if err != nil {
		t.Fatal(err)
	}

	want := "Directory\tDirectory_Parent\tDefaultDir\r\n" +
		"s72\tS72\tS255\r\n" +
		"Dirs\tDirectory\r\n" +
		"TARGETDIR\t\tSourceDir\r\n" +
		"ProgramFilesFolder\tTARGETDIR\t.\r\n" +
		"INSTALLDIR\tProgramFilesFolder\tMyApp\r\n"

	if string(got) != want {
		t.Errorf("Render():\ngot:\n%q\nwant:\n%q", string(got), want)
	}
}

func TestRender_Nulls(t *testing.T) {
	tbl := &Table{
		Name: "Nulls",
		Columns: []Column{
			{Name: "Key", Type: Str(72), PK: true},
			{Name: "IntVal", Type: NI2()},
			{Name: "StrVal", Type: NStr(50)},
		},
	}
	tbl.AddRow("allnull", "", "")
	tbl.AddRow("intval", "42", "")
	tbl.AddRow("strval", "", "hello")
	tbl.AddRow("allval", "7", "test")

	got, err := tbl.Render()
	if err != nil {
		t.Fatal(err)
	}

	want := "Key\tIntVal\tStrVal\r\n" +
		"s72\tI2\tS50\r\n" +
		"Nulls\tKey\r\n" +
		"allnull\t\t\r\n" +
		"intval\t42\t\r\n" +
		"strval\t\thello\r\n" +
		"allval\t7\ttest\r\n"

	if string(got) != want {
		t.Errorf("Render():\ngot:\n%q\nwant:\n%q", string(got), want)
	}
}

func TestRender_ControlChars(t *testing.T) {
	tbl := &Table{
		Name: "Ctl",
		Columns: []Column{
			{Name: "Key", Type: Str(72), PK: true},
			{Name: "Val", Type: NStr(0)},
		},
	}
	tbl.AddRow("null", "a\x00b")
	tbl.AddRow("bs", "a\x08b")
	tbl.AddRow("tab", "a\x09b")
	tbl.AddRow("lf", "a\x0Ab")
	tbl.AddRow("ff", "a\x0Cb")
	tbl.AddRow("cr", "a\x0Db")
	tbl.AddRow("mixed", "\x00\x08\x09\x0A\x0C\x0D")

	got, err := tbl.Render()
	if err != nil {
		t.Fatal(err)
	}

	want := "Key\tVal\r\n" +
		"s72\tS0\r\n" +
		"Ctl\tKey\r\n" +
		"null\ta21b\r\n" +
		"bs\ta27b\r\n" +
		"tab\ta16b\r\n" +
		"lf\ta25b\r\n" +
		"ff\ta24b\r\n" +
		"cr\ta17b\r\n" +
		"mixed\t212716252417\r\n"

	if string(got) != want {
		t.Errorf("Render():\ngot:\n%q\nwant:\n%q", string(got), want)
	}
}

func TestRender_NonASCII_Cyrillic1251(t *testing.T) {
	tbl := &Table{
		Name: "rus",
		Columns: []Column{
			{Name: "Key", Type: Str(72), PK: true},
			{Name: "Val", Type: NStr(0)},
		},
	}
	tbl.AddRow("productName", "МойАгент")

	got, err := tbl.Render()
	if err != nil {
		t.Fatal(err)
	}

	// CP1251 bytes for "МойАгент":
	// М=U+041C→0xCC  о=U+043E→0xEE  й=U+0439→0xE9
	// А=U+0410→0xC0  г=U+0433→0xE3  е=U+0435→0xE5  н=U+043D→0xED  т=U+0442→0xF2
	valBytes := []byte{0xCC, 0xEE, 0xE9, 0xC0, 0xE3, 0xE5, 0xED, 0xF2}

	var want bytes.Buffer
	want.WriteString("Key\tVal\r\n")
	want.WriteString("s72\tS0\r\n")
	want.WriteString("1251\trus\tKey\r\n")
	want.WriteString("productName\t")
	want.Write(valBytes)
	want.WriteString("\r\n")

	if !bytes.Equal(got, want.Bytes()) {
		t.Errorf("Render():\ngot:\n%x\nwant:\n%x", got, want.Bytes())
	}
}

func TestRender_NonASCII_Latin1252(t *testing.T) {
	tbl := &Table{
		Name: "lat",
		Columns: []Column{
			{Name: "Key", Type: Str(72), PK: true},
			{Name: "Val", Type: NStr(0)},
		},
	}
	tbl.AddRow("company", "Acmé SARL")

	got, err := tbl.Render()
	if err != nil {
		t.Fatal(err)
	}

	// CP1252 bytes for "Acmé SARL": é=U+00E9→0xE9
	valBytes := []byte("Acm")           // 41 63 6D
	valBytes = append(valBytes, 0xE9)   // é
	valBytes = append(valBytes, []byte(" SARL")...) // 20 53 41 52 4C

	var want bytes.Buffer
	want.WriteString("Key\tVal\r\n")
	want.WriteString("s72\tS0\r\n")
	want.WriteString("1252\tlat\tKey\r\n")
	want.WriteString("company\t")
	want.Write(valBytes)
	want.WriteString("\r\n")

	if !bytes.Equal(got, want.Bytes()) {
		t.Errorf("Render():\ngot:\n%x\nwant:\n%x", got, want.Bytes())
	}
}

func TestRender_SpecialCharsPlain(t *testing.T) {
	// Characters like backslash, pipe, quotes, brackets, braces are NOT
	// escaped by the IDT format — they pass through verbatim.
	tbl := &Table{
		Name: "Specials",
		Columns: []Column{
			{Name: "Key", Type: Str(72), PK: true},
			{Name: "Raw", Type: NStr(0)},
		},
	}
	tbl.AddRow("backslash", "a\\b")
	tbl.AddRow("pipe", "a|b")
	tbl.AddRow("quotes", `a"b'c`)
	tbl.AddRow("brackets", "a[b]c(d)e")
	tbl.AddRow("braces", "a{b}c")

	got, err := tbl.Render()
	if err != nil {
		t.Fatal(err)
	}

	want := "Key\tRaw\r\n" +
		"s72\tS0\r\n" +
		"Specials\tKey\r\n" +
		"backslash\ta\\b\r\n" +
		"pipe\ta|b\r\n" +
		"quotes\ta\"b'c\r\n" +
		"brackets\ta[b]c(d)e\r\n" +
		"braces\ta{b}c\r\n"

	if string(got) != want {
		t.Errorf("Render():\ngot:\n%q\nwant:\n%q", string(got), want)
	}
}

func TestRender_NoPK(t *testing.T) {
	tbl := &Table{
		Name: "NoPK",
		Columns: []Column{
			{Name: "Col1", Type: Str(72)},
			{Name: "Col2", Type: NStr(0)},
		},
	}
	tbl.AddRow("a", "b")

	got, err := tbl.Render()
	if err != nil {
		t.Fatal(err)
	}

	want := "Col1\tCol2\r\n" +
		"s72\tS0\r\n" +
		"NoPK\r\n" +
		"a\tb\r\n"

	if string(got) != want {
		t.Errorf("Render():\ngot:\n%q\nwant:\n%q", string(got), want)
	}
}

func TestRender_Empty(t *testing.T) {
	tbl := &Table{
		Name: "Empty",
		Columns: []Column{
			{Name: "K", Type: Str(10), PK: true},
		},
	}
	// zero rows
	got, err := tbl.Render()
	if err != nil {
		t.Fatal(err)
	}

	want := "K\r\n" +
		"s10\r\n" +
		"Empty\tK\r\n"

	if string(got) != want {
		t.Errorf("Render():\ngot:\n%q\nwant:\n%q", string(got), want)
	}
}

// ── Table: builder helpers for golden tests ───────────────────────────────────

func buildGoldenProperty() *Table {
	tbl := &Table{
		Name: "Property",
		Columns: []Column{
			{Name: "Property", Type: Str(72), PK: true},
			{Name: "Value", Type: NStr(0)},
		},
	}
	tbl.AddRow("ProductName", "MyAgent")
	tbl.AddRow("ProductVersion", "1.2.3")
	tbl.AddRow("Manufacturer", "Acme Inc")
	tbl.AddRow("ProductLanguage", "1033")
	tbl.AddRow("ProductCode", "{12345678-1234-1234-1234-123456789abc}")
	return tbl
}

func buildGoldenControlChars() *Table {
	tbl := &Table{
		Name: "CtlChars",
		Columns: []Column{
			{Name: "K", Type: Str(72), PK: true},
			{Name: "V", Type: NStr(0)},
		},
	}
	tbl.AddRow("null", "\x00mid\x00")
	tbl.AddRow("bs", "\x08back\x08")
	tbl.AddRow("tab", "a\tb\tc")
	tbl.AddRow("lf", "line1\x0Aline2")
	tbl.AddRow("ff", "page\x0Cbreak")
	tbl.AddRow("cr", "win\r\n")
	tbl.AddRow("all", "\x00\x08\x09\x0A\x0C\x0D")
	return tbl
}

func buildGoldenNonASCII_RU() *Table {
	tbl := &Table{
		Name: "RussianTbl",
		Columns: []Column{
			{Name: "K", Type: Str(72), PK: true},
			{Name: "V", Type: NStr(0)},
		},
	}
	tbl.AddRow("productName", "МойАгент")
	tbl.AddRow("description", "Настройка приложения")
	return tbl
}

func buildGoldenNonASCII_Latin() *Table {
	tbl := &Table{
		Name: "LatinTbl",
		Columns: []Column{
			{Name: "K", Type: Str(72), PK: true},
			{Name: "V", Type: NStr(0)},
		},
	}
	tbl.AddRow("company", "Acmé SARL")
	tbl.AddRow("product", "café")
	return tbl
}

func buildGoldenNulls() *Table {
	tbl := &Table{
		Name: "NullTbl",
		Columns: []Column{
			{Name: "K", Type: Str(72), PK: true},
			{Name: "I", Type: NI2()},
			{Name: "S", Type: NStr(0)},
		},
	}
	tbl.AddRow("allnull", "", "")
	tbl.AddRow("i_only", "42", "")
	tbl.AddRow("s_only", "", "hello")
	tbl.AddRow("both", "7", "world")
	return tbl
}

func buildGoldenNoPK() *Table {
	tbl := &Table{
		Name: "NoPK",
		Columns: []Column{
			{Name: "A", Type: Str(10)},
			{Name: "B", Type: NI2()},
		},
	}
	tbl.AddRow("x", "1")
	tbl.AddRow("y", "2")
	return tbl
}

// ── Golden file test ─────────────────────────────────────────────────────────

func TestGolden(t *testing.T) {
	tables := map[string]*Table{
		"Property":   buildGoldenProperty(),
		"CtlChars":   buildGoldenControlChars(),
		"NonASCII_RU": buildGoldenNonASCII_RU(),
		"NonASCII_Latin": buildGoldenNonASCII_Latin(),
		"Nulls":      buildGoldenNulls(),
		"NoPK":       buildGoldenNoPK(),
	}

	for name, tbl := range tables {
		got, err := tbl.Render()
		if err != nil {
			t.Fatalf("%s: Render: %v", name, err)
		}

		path := filepath.Join("testdata", name+".idt")
		if *update {
			if err := os.WriteFile(path, got, 0644); err != nil {
				t.Fatalf("%s: WriteFile: %v", name, err)
			}
		} else {
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%s: ReadFile %s: %v", name, path, err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("%s.idt: golden mismatch; run `go test -update` to regenerate", name)
			}
		}
	}
}

// ── Error cases ───────────────────────────────────────────────────────────────

func TestRender_NoColumns(t *testing.T) {
	tbl := &Table{Name: "bad"}
	_, err := tbl.Render()
	if err == nil {
		t.Fatal("expected error for no columns")
	}
}

func TestRender_RowCountMismatch(t *testing.T) {
	tbl := &Table{
		Name: "bad",
		Columns: []Column{
			{Name: "A", Type: Str(10), PK: true},
			{Name: "B", Type: NStr(0)},
		},
	}
	tbl.AddRow("a") // missing B
	_, err := tbl.Render()
	if err == nil {
		t.Fatal("expected error for column count mismatch")
	}
}

func TestRender_MixedScriptError(t *testing.T) {
	// A table with both Cyrillic and Latin-1 supplement runes cannot be
	// represented in either CP1251 or CP1252.
	tbl := &Table{
		Name: "mixed",
		Columns: []Column{
			{Name: "K", Type: Str(72), PK: true},
			{Name: "V", Type: NStr(0)},
		},
	}
	tbl.AddRow("r1", "МойАгент café") // Cyrillic + Latin-1 supplement é
	_, err := tbl.Render()
	if err == nil {
		t.Fatal("expected error for mixed scripts not representable in any codepage")
	}
}

func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	tbl := &Table{
		Name: "TestFile",
		Columns: []Column{
			{Name: "K", Type: Str(10), PK: true},
			{Name: "V", Type: NStr(0)},
		},
	}
	tbl.AddRow("hello", "world")

	path := filepath.Join(dir, "TestFile.idt")
	if err := tbl.WriteFile(path); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	want := "K\tV\r\ns10\tS0\r\nTestFile\tK\r\nhello\tworld\r\n"
	if string(got) != want {
		t.Errorf("WriteFile: got %q, want %q", string(got), want)
	}
}
