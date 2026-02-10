package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/708u/ccfmt"
	"github.com/alecthomas/kong"
)

var version = "dev"

type CLI struct {
	Target  string           `help:"Path to a specific file to format." short:"t" name:"target"`
	Backup  bool             `help:"Create backup before writing."`
	DryRun  bool             `help:"Show changes without writing." name:"dry-run"`
	Version kong.VersionFlag `help:"Print version."`

	checker ccfmt.PathChecker
	w       io.Writer
}

type targetFile struct {
	path          string
	needsCleaning bool
}

type fileResult struct {
	path       string
	original   []byte
	result     *ccfmt.FormatResult
	backupPath string
}

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ccfmt: %v\n", err)
		os.Exit(1)
	}

	cli := CLI{
		checker: osPathChecker{},
		w:       os.Stdout,
	}
	kong.Parse(&cli,
		kong.Vars{"version": version},
	)

	if err := cli.Run(home); err != nil {
		fmt.Fprintf(os.Stderr, "ccfmt: %v\n", err)
		os.Exit(1)
	}
}

func (c *CLI) Run(home string) error {
	return c.runTargets(c.resolveTargets(home))
}

func (c *CLI) runTargets(targets []targetFile) error {
	single := len(targets) == 1

	for _, tf := range targets {
		r, err := c.formatFile(tf)
		if err != nil {
			if single || !os.IsNotExist(err) {
				return err
			}
			fmt.Fprintf(c.w, "%s: skipped (not found)\n\n", tf.path)
			continue
		}
		printResult(c.w, r, single)
	}
	return nil
}

func (c *CLI) resolveTargets(home string) []targetFile {
	if c.Target != "" {
		return []targetFile{{
			path:          c.Target,
			needsCleaning: isClaudeJSON(c.Target),
		}}
	}
	return defaultTargets(home)
}

func (c *CLI) formatFile(tf targetFile) (*fileResult, error) {
	info, err := os.Stat(tf.path)
	if err != nil {
		return nil, err
	}
	perm := info.Mode().Perm()

	data, err := os.ReadFile(tf.path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", tf.path, err)
	}

	var result *ccfmt.FormatResult
	if tf.needsCleaning {
		f := &ccfmt.Formatter{PathChecker: c.checker}
		result, err = f.Format(data)
	} else {
		result, err = ccfmt.FormatJSON(data)
	}
	if err != nil {
		return nil, fmt.Errorf("formatting %s: %w", tf.path, err)
	}

	var backupPath string
	if !c.DryRun {
		if c.Backup {
			backupPath = fmt.Sprintf("%s.backup.%s",
				tf.path, time.Now().Format("20060102150405"))
			if err := os.WriteFile(backupPath, data, perm); err != nil {
				return nil, fmt.Errorf("creating backup: %w", err)
			}
		}
		if err := os.WriteFile(tf.path, result.Data, perm); err != nil {
			return nil, fmt.Errorf("writing %s: %w", tf.path, err)
		}
	}

	return &fileResult{
		path:       tf.path,
		original:   data,
		result:     result,
		backupPath: backupPath,
	}, nil
}

func defaultTargets(home string) []targetFile {
	cwd, _ := os.Getwd()
	return []targetFile{
		{path: filepath.Join(home, ".claude.json"), needsCleaning: true},
		{path: filepath.Join(home, ".claude", "settings.json"), needsCleaning: false},
		{path: filepath.Join(home, ".claude", "settings.local.json"), needsCleaning: false},
		{path: filepath.Join(cwd, ".claude", "settings.json"), needsCleaning: false},
		{path: filepath.Join(cwd, ".claude", "settings.local.json"), needsCleaning: false},
	}
}

func isClaudeJSON(path string) bool {
	return filepath.Base(path) == ".claude.json"
}

func printResult(w io.Writer, r *fileResult, single bool) {
	if single {
		fmt.Fprint(w, r.result.Stats.Summary(r.backupPath))
		return
	}
	if bytes.Equal(r.original, r.result.Data) {
		fmt.Fprintf(w, "%s:\n  (no changes)\n\n", r.path)
		return
	}
	fmt.Fprintf(w, "%s:\n", r.path)
	for _, line := range splitLines(r.result.Stats.Summary(r.backupPath)) {
		fmt.Fprintf(w, "  %s\n", line)
	}
	fmt.Fprintln(w)
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := range len(s) {
		if s[i] == '\n' {
			line := s[start:i]
			if line != "" {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

type osPathChecker struct{}

func (osPathChecker) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
