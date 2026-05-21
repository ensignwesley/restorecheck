package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/ensignwesley/restorecheck/internal/config"
)

const version = "0.1.0-dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
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

func printUsage() {
	fmt.Print(`restorecheck proves that restic backups can become usable files again.

Usage:
  restorecheck init [--output restorecheck.yml] [--force]
  restorecheck check-config [--config restorecheck.yml]
  restorecheck version

Restore execution is intentionally not implemented yet.
`)
}
