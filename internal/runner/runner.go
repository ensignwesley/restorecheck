package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ensignwesley/restorecheck/internal/config"
)

type Options struct {
	Snapshot      string
	Paths         []string
	WorkdirParent string
	KeepWorkdir   bool
}

type Result struct {
	OK         bool
	Status     string
	Repository string
	Snapshot   string
	Workdir    string
	Elapsed    time.Duration
	Assertions []AssertionResult
	Cleanup    string
}

type AssertionResult struct {
	Name     string
	Type     string
	Path     string
	OK       bool
	Evidence string
}

func Run(ctx context.Context, cfg *config.Config, opts Options) (*Result, error) {
	start := time.Now()
	parent := cfg.Temp.Parent
	if opts.WorkdirParent != "" {
		parent = opts.WorkdirParent
	}
	workdir, err := os.MkdirTemp(parent, "restorecheck-*")
	if err != nil {
		return nil, fmt.Errorf("create temp restore dir: %w", err)
	}
	res := &Result{Repository: cfg.Restic.Repository, Snapshot: snapshot(cfg, opts), Workdir: workdir, Status: "error"}
	// Keep cleanup tied to the explicit CLI escape hatch. Config must not be
	// able to silently turn temp restores into disk leaks.
	keep := opts.KeepWorkdir
	defer func() {
		res.Elapsed = time.Since(start)
		if keep {
			res.Cleanup = "preserved temporary restore"
			return
		}
		if err := os.RemoveAll(workdir); err != nil {
			res.Cleanup = "cleanup failed: " + err.Error()
		} else {
			res.Cleanup = "removed temporary restore"
		}
	}()

	if err := restore(ctx, cfg, opts, workdir); err != nil {
		return res, err
	}

	res.Assertions = RunAssertions(workdir, cfg.Assertions)
	res.OK = true
	res.Status = "pass"
	for _, a := range res.Assertions {
		if !a.OK {
			res.OK = false
			res.Status = "fail"
			break
		}
	}
	return res, nil
}

func snapshot(cfg *config.Config, opts Options) string {
	if opts.Snapshot != "" {
		return opts.Snapshot
	}
	return cfg.Restic.Snapshot
}

func restore(ctx context.Context, cfg *config.Config, opts Options, target string) error {
	snap := snapshot(cfg, opts)
	args := []string{"restore", snap, "--target", target}
	if cfg.Restic.Repository != "" {
		args = append(args, "--repo", cfg.Restic.Repository)
	}
	if cfg.Restic.PasswordFile != "" {
		args = append(args, "--password-file", cfg.Restic.PasswordFile)
	}
	paths := cfg.Restic.Paths
	if len(opts.Paths) > 0 {
		paths = opts.Paths
	}
	for _, p := range paths {
		args = append(args, "--path", p)
	}
	cmd := exec.CommandContext(ctx, "restic", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restic restore failed: %w\ncommand: restic %s\noutput: %s", err, strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return nil
}

func RunAssertions(root string, assertions []config.Assertion) []AssertionResult {
	results := make([]AssertionResult, 0, len(assertions))
	for _, a := range assertions {
		result := AssertionResult{Name: a.Name, Type: a.Type, Path: a.Path}
		full := RestoredPath(root, a.Path)
		switch a.Type {
		case "exists":
			_, err := os.Stat(full)
			if err == nil {
				result.OK = true
				result.Evidence = "path exists"
			} else if os.IsNotExist(err) {
				result.Evidence = "path does not exist"
			} else {
				result.Evidence = err.Error()
			}
		case "not-empty-file":
			info, err := os.Stat(full)
			if err != nil {
				if os.IsNotExist(err) {
					result.Evidence = "file does not exist"
				} else {
					result.Evidence = err.Error()
				}
			} else if info.IsDir() {
				result.Evidence = "path is a directory"
			} else if info.Size() == 0 {
				result.Evidence = "file exists but is empty"
			} else {
				result.OK = true
				result.Evidence = fmt.Sprintf("file exists and has %d bytes", info.Size())
			}
		case "matches-checksum":
			actual, err := fileSHA256(full)
			if err != nil {
				if os.IsNotExist(err) {
					result.Evidence = "file does not exist"
				} else {
					result.Evidence = err.Error()
				}
				break
			}
			expected := strings.ToLower(strings.TrimSpace(a.Sha256))
			if actual == expected {
				result.OK = true
				result.Evidence = "sha256 matches " + actual
			} else {
				result.Evidence = fmt.Sprintf("sha256 mismatch: expected %s, got %s", expected, actual)
			}
		case "min-size":
			info, err := os.Stat(full)
			if err != nil {
				if os.IsNotExist(err) {
					result.Evidence = "file does not exist"
				} else {
					result.Evidence = err.Error()
				}
			} else if info.IsDir() {
				result.Evidence = "path is a directory"
			} else if info.Size() < a.Bytes {
				result.Evidence = fmt.Sprintf("file has %d bytes, below minimum %d", info.Size(), a.Bytes)
			} else {
				result.OK = true
				result.Evidence = fmt.Sprintf("file has %d bytes, meeting minimum %d", info.Size(), a.Bytes)
			}
		case "non-empty-dir":
			entries, err := os.ReadDir(full)
			if err != nil {
				if os.IsNotExist(err) {
					result.Evidence = "directory does not exist"
				} else {
					result.Evidence = err.Error()
				}
			} else if len(entries) == 0 {
				result.Evidence = "directory exists but is empty"
			} else {
				result.OK = true
				result.Evidence = fmt.Sprintf("directory contains %d entries", len(entries))
			}
		default:
			result.Evidence = "assertion type not implemented by run yet"
		}
		results = append(results, result)
	}
	return results
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func RestoredPath(root, configured string) string {
	clean := filepath.Clean(configured)
	if filepath.IsAbs(clean) {
		clean = strings.TrimPrefix(clean, string(filepath.Separator))
	}
	return filepath.Join(root, clean)
}

func Counts(results []AssertionResult) (passed, failed int) {
	for _, r := range results {
		if r.OK {
			passed++
		} else {
			failed++
		}
	}
	return passed, failed
}
