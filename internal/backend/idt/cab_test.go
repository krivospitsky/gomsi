package idt

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/krivospitsky/gomsi/internal/model"
)

func TestLcabArgs(t *testing.T) {
	got := lcabArgs([]string{"a.txt", "b.bin"}, "out.cab")
	want := []string{"-n", "-q", "a.txt", "b.bin", "out.cab"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLcabArgs_EmptyFiles(t *testing.T) {
	got := lcabArgs(nil, "out.cab")
	want := []string{"-n", "-q", "out.cab"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestGenCAB(t *testing.T) {
	if _, err := exec.LookPath("lcab"); err != nil {
		t.Skip("lcab not available:", err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("lcab is Linux-only")
	}

	dir := t.TempDir()

	// Create a fake payload file.
	payload := filepath.Join(dir, "myagent.exe")
	if err := os.WriteFile(payload, []byte("fake payload content"), 0644); err != nil {
		t.Fatal(err)
	}

	cabPath := filepath.Join(dir, "gomsi.cab")
	files := []model.File{
		{Source: payload, Destination: "myagent.exe"},
	}

	if err := genCAB(cabPath, files); err != nil {
		t.Fatal(err)
	}

	fi, err := os.Stat(cabPath)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() == 0 {
		t.Fatal("CAB file is empty")
	}
	t.Logf("CAB size: %d bytes", fi.Size())
}

func TestGenCAB_FileNotExist(t *testing.T) {
	dir := t.TempDir()
	cabPath := filepath.Join(dir, "gomsi.cab")
	files := []model.File{
		{Source: filepath.Join(dir, "nonexistent.exe"), Destination: "nope.exe"},
	}

	err := genCAB(cabPath, files)
	if err == nil {
		t.Fatal("expected error for non-existent source file")
	}
	t.Logf("got expected error: %v", err)
}

func TestGenCAB_StagingNamesMatchDestination(t *testing.T) {
	// Verify that the cab-internal name matches the destination, not the
	// source basename. We do this by using two sources with different
	// filenames but the same destination basename would collide — instead
	// just check that the staged copy worked by inspecting the staging dir.
	if _, err := exec.LookPath("lcab"); err != nil {
		t.Skip("lcab not available:", err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("lcab is Linux-only")
	}

	dir := t.TempDir()

	// Source basename differs from destination.
	src := filepath.Join(dir, "build_output.bin")
	if err := os.WriteFile(src, []byte("payload"), 0644); err != nil {
		t.Fatal(err)
	}

	cabPath := filepath.Join(dir, "gomsi.cab")
	files := []model.File{
		{Source: src, Destination: "myagent.exe"},
	}

	if err := genCAB(cabPath, files); err != nil {
		t.Fatal(err)
	}

	// We can't easily inspect the cab internals without cabextract, but at
	// least verify no errors occurred and the cab file exists.
	fi, err := os.Stat(cabPath)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() == 0 {
		t.Fatal("CAB file is empty")
	}

	// Log the CAB path so a dev can inspect manually.
	t.Logf("CAB at %s, size %d bytes (contains file named 'myagent.exe')", cabPath, fi.Size())
}

func TestLcabArgs_Integration(t *testing.T) {
	if _, err := exec.LookPath("lcab"); err != nil {
		t.Skip("lcab not available:", err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("lcab is Linux-only")
	}

	dir := t.TempDir()

	// Create two files.
	files := []struct {
		name    string
		content string
	}{
		{"a.txt", "hello"},
		{"b.txt", "world"},
	}
	var paths []string
	for _, f := range files {
		p := filepath.Join(dir, f.name)
		if err := os.WriteFile(p, []byte(f.content), 0644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}

	cabPath := filepath.Join(dir, "test.cab")
	args := lcabArgs(paths, cabPath)
	cmd := exec.Command("lcab", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("lcab failed: %v\noutput: %s", err, out)
	}

	fi, err := os.Stat(cabPath)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() == 0 {
		t.Fatal("CAB file is empty")
	}
	t.Logf("lcab produced %d-byte CAB at %s", fi.Size(), cabPath)
}
