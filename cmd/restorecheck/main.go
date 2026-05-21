package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ensignwesley/restorecheck/internal/config"
	"github.com/ensignwesley/restorecheck/internal/runner"
)

const version = "0.1.0-dev"

type exitError struct {
	code int
	err  error
}

func (e exitError) Error() string { return e.err.Error() }

func main() {
	if err := run(os.Args[1:]); err != nil {
		var ee exitError
		if errors.As(err, &ee) {
			if ee.code == 2 {
				fmt.Fprintln(os.Stderr, "restorecheck: ERROR")
				fmt.Fprintln(os.Stderr, "reason:", ee.err)
			}
			os.Exit(ee.code)
		}
		fmt.Fprintln(os.Stderr, "restorecheck: ERROR")
		fmt.Fprintln(os.Stderr, "reason:", err)
		os.Exit(2)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return runCheckConfig([]string{})
	}
	switch args[0] {
	case "init":
		return runInit(args[1:])
	case "check-config":
		return runCheckConfig(args[1:])
	case "run":
		return runRestore(args[1:])
	case "version":
		fmt.Println("restorecheck", version)
		return nil
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	output := fs.String("output", "restorecheck.yml", "config file to write")
	force := fs.Bool("force", false, "overwrite an existing config file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if _, err := os.Stat(*output); err == nil && !*force {
		return fmt.Errorf("%s already exists; use --force to overwrite", *output)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.WriteFile(*output, []byte(config.StarterConfig), 0644); err != nil {
		return err
	}
	fmt.Println("wrote", *output)
	return nil
}

func runCheckConfig(args []string) error {
	fs := flag.NewFlagSet("check-config", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	path := fs.String("config", "restorecheck.yml", "config file to validate")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := config.Load(*path)
	if err != nil {
		return err
	}
	fmt.Printf("restorecheck: config OK (%d assertions)\n", len(cfg.Assertions))
	return nil
}

func runRestore(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	path := fs.String("config", "restorecheck.yml", "config file to use")
	pathShort := fs.String("c", "", "config file to use")
	snapshot := fs.String("snapshot", "", "snapshot ID or latest")
	workdir := fs.String("workdir", "", "parent directory for temporary restore")
	keep := fs.Bool("keep-workdir", false, "preserve temporary restore directory")
	jsonOut := fs.Bool("json", false, "emit JSON")
	var paths repeatedFlag
	fs.Var(&paths, "path", "restore path override; repeatable")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *pathShort != "" {
		*path = *pathShort
	}
	cfg, err := config.Load(*path)
	if err != nil {
		return err
	}
	res, err := runner.Run(context.Background(), cfg, runner.Options{
		Snapshot:      *snapshot,
		Paths:         paths,
		WorkdirParent: *workdir,
		KeepWorkdir:   *keep,
	})
	if err != nil {
		if res != nil {
			printResult(res, *jsonOut)
		}
		return exitError{code: 2, err: err}
	}
	printResult(res, *jsonOut)
	if !res.OK {
		return exitError{code: 1, err: errors.New("one or more assertions failed")}
	}
	return nil
}

type repeatedFlag []string

func (r *repeatedFlag) String() string { return strings.Join(*r, ",") }
func (r *repeatedFlag) Set(v string) error {
	*r = append(*r, v)
	return nil
}

func printResult(res *runner.Result, asJSON bool) {
	if asJSON {
		b, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(b))
		return
	}
	passed, failed := runner.Counts(res.Assertions)
	fmt.Printf("restorecheck: %s\n", strings.ToUpper(res.Status))
	fmt.Printf("repository: %s\n", res.Repository)
	fmt.Printf("snapshot: %s\n", res.Snapshot)
	fmt.Printf("restored: %s\n", res.Workdir)
	fmt.Printf("assertions: %d passed, %d failed\n\n", passed, failed)
	for _, a := range res.Assertions {
		mark := "✓"
		if !a.OK {
			mark = "✗"
		}
		fmt.Printf("%s %-32s %s\n", mark, a.Name, a.Path)
		if a.Evidence != "" {
			fmt.Printf("  evidence: %s\n", a.Evidence)
		}
	}
	if len(res.Assertions) > 0 {
		fmt.Println()
	}
	fmt.Println("cleanup:", res.Cleanup)
	fmt.Printf("elapsed: %.1fs\n", res.Elapsed.Round(100*time.Millisecond).Seconds())
}

func printUsage() {
	fmt.Print(`restorecheck proves that restic backups can become usable files again.

Usage:
  restorecheck init [--output restorecheck.yml] [--force]
  restorecheck check-config [--config restorecheck.yml]
  restorecheck run [--config restorecheck.yml] [--snapshot latest|ID] [--path PATH] [--workdir DIR] [--keep-workdir] [--json]
  restorecheck version

Run flow:
  1. restore selected restic snapshot into a temporary directory
  2. run configured assertions against restored files
  3. print evidence and exit 0/1/2

Implemented run assertions today: exists, not-empty-file.
`)
}
