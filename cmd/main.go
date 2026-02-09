package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/708u/ccfmt"
	"github.com/alecthomas/kong"
)

var version = "dev"

type CLI struct {
	File    string           `help:"Path to JSON file." default:"${default_file}" short:"f"`
	Backup  bool             `help:"Create backup before writing."`
	DryRun  bool             `help:"Show changes without writing." name:"dry-run"`
	Version kong.VersionFlag `help:"Print version."`
}

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ccfmt: %v\n", err)
		os.Exit(1)
	}

	var cli CLI
	kong.Parse(&cli,
		kong.Vars{"default_file": home + "/.claude.json"},
		kong.Vars{"version": version},
	)

	if err := run(&cli, osPathChecker{}, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "ccfmt: %v\n", err)
		os.Exit(1)
	}
}

func run(cli *CLI, checker ccfmt.PathChecker, w io.Writer) error {
	info, err := os.Stat(cli.File)
	if err != nil {
		return fmt.Errorf("%s not found", cli.File)
	}
	perm := info.Mode().Perm()

	data, err := os.ReadFile(cli.File)
	if err != nil {
		return fmt.Errorf("reading %s: %w", cli.File, err)
	}

	f := &ccfmt.Formatter{PathChecker: checker}
	result, err := f.Format(data)
	if err != nil {
		return err
	}

	var backupPath string
	if !cli.DryRun {
		if cli.Backup {
			backupPath = fmt.Sprintf("%s.backup.%s",
				cli.File, time.Now().Format("20060102150405"))
			if err := os.WriteFile(backupPath, data, perm); err != nil {
				return fmt.Errorf("creating backup: %w", err)
			}
		}
		if err := os.WriteFile(cli.File, result.Data, perm); err != nil {
			return fmt.Errorf("writing %s: %w", cli.File, err)
		}
	}

	fmt.Fprint(w, result.Stats.Summary(backupPath))
	return nil
}

type osPathChecker struct{}

func (osPathChecker) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
