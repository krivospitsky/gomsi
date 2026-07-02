package idt

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/krivospitsky/gomsi/internal/model"
)

func TestMsibuildArgs(t *testing.T) {
	p := model.Product{
		Name:         "MyAgent",
		Manufacturer: "Acme",
		ProductCode:  "{22222222-2222-2222-2222-222222222222}",
	}

	got := msibuildArgs(
		"/out/my.msi",
		[]string{"/tmp/Property.idt", "/tmp/Directory.idt"},
		"/tmp/gomsi.cab",
		p,
	)

	want := []string{
		"/out/my.msi",
		"-i", "/tmp/Property.idt",
		"-i", "/tmp/Directory.idt",
		"-a", "gomsi.cab", "/tmp/gomsi.cab",
		"-s", "MyAgent", "Acme", ";1033", "{22222222-2222-2222-2222-222222222222}",
	}

	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d\n  got:  %#v\n  want: %#v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestMsibuildArgs_NoCab(t *testing.T) {
	p := model.Product{
		Name:         "Test",
		Manufacturer: "Mfr",
		ProductCode:  "{00000000-0000-0000-0000-000000000000}",
	}

	got := msibuildArgs("test.msi", []string{"t.idt"}, "", p)

	want := []string{
		"test.msi",
		"-i", "t.idt",
		"-s", "Test", "Mfr", ";1033", "{00000000-0000-0000-0000-000000000000}",
	}

	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRunMSIBuild(t *testing.T) {
	if _, err := exec.LookPath("msibuild"); err != nil {
		t.Skip("msibuild not available:", err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("msibuild is Linux-only")
	}

	dir := t.TempDir()

	// Create a minimal Property.idt that msibuild can import.
	propIDT := filepath.Join(dir, "Property.idt")
	if err := os.WriteFile(propIDT, []byte("Property\tValue\r\ns72\tS0\r\nProperty\tProperty\r\nProductName\tTest\r\nProductVersion\t1.0.0\r\n"), 0644); err != nil {
		t.Fatal(err)
	}

	msiPath := filepath.Join(dir, "test.msi")
	tablePaths := []string{propIDT}

	runMSIBuild(msiPath, tablePaths, "", model.Product{
		Name:         "Test",
		Manufacturer: "Mfr",
		ProductCode:  "{00000000-0000-0000-0000-000000000000}",
	}, dir)

	fi, err := os.Stat(msiPath)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() == 0 {
		t.Fatal("MSI file is empty")
	}
	t.Logf("MSI produced: %s (%d bytes)", msiPath, fi.Size())
}
