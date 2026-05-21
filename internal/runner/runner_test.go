package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ensignwesley/restorecheck/internal/config"
)

func TestRunAssertionsExistsAndNonEmptyFilePass(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "home/app/data/file.txt"), "payload")
	results := RunAssertions(root, []config.Assertion{
		{Name: "file exists", Type: "exists", Path: "/home/app/data/file.txt"},
		{Name: "file has data", Type: "not-empty-file", Path: "/home/app/data/file.txt"},
	})
	passed, failed := Counts(results)
	if passed != 2 || failed != 0 {
		t.Fatalf("passed=%d failed=%d, want 2/0: %#v", passed, failed, results)
	}
}

func TestRunAssertionsNonEmptyFileFailsForEmptyFile(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "home/app/data/empty.txt"), "")
	results := RunAssertions(root, []config.Assertion{
		{Name: "file has data", Type: "not-empty-file", Path: "/home/app/data/empty.txt"},
	})
	if results[0].OK {
		t.Fatalf("assertion passed, want failure")
	}
	if results[0].Evidence != "file exists but is empty" {
		t.Fatalf("evidence=%q", results[0].Evidence)
	}
}

func TestRestoredPathTreatsAbsoluteConfigPathAsRestoreRelative(t *testing.T) {
	got := RestoredPath("/tmp/restore", "/home/app/data/file.txt")
	want := "/tmp/restore/home/app/data/file.txt"
	if got != want {
		t.Fatalf("RestoredPath=%q, want %q", got, want)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
