package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ensignwesley/restorecheck/internal/config"
)

func TestRunAssertionsExistsNonEmptyFileAndChecksumPass(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "home/app/data/file.txt"), "payload")
	results := RunAssertions(root, []config.Assertion{
		{Name: "file exists", Type: "exists", Path: "/home/app/data/file.txt"},
		{Name: "file has data", Type: "not-empty-file", Path: "/home/app/data/file.txt"},
		{Name: "file checksum matches", Type: "matches-checksum", Path: "/home/app/data/file.txt", Sha256: "239f59ed55e737c77147cf55ad0c1b030b6d7ee748a7426952f9b852d5a935e5"},
	})
	passed, failed := Counts(results)
	if passed != 3 || failed != 0 {
		t.Fatalf("passed=%d failed=%d, want 3/0: %#v", passed, failed, results)
	}
}

func TestRunAssertionsChecksumFailsOnMismatch(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "home/app/data/file.txt"), "payload")
	results := RunAssertions(root, []config.Assertion{
		{Name: "file checksum matches", Type: "matches-checksum", Path: "/home/app/data/file.txt", Sha256: "8810ad581e59f2bc3928b261707a71308f7e139eb04820366dc4d5c18d980225"},
	})
	if results[0].OK {
		t.Fatalf("assertion passed, want failure")
	}
	if want := "sha256 mismatch"; len(results[0].Evidence) < len(want) || results[0].Evidence[:len(want)] != want {
		t.Fatalf("evidence=%q, want prefix %q", results[0].Evidence, want)
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
