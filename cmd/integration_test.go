//go:build integration

package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/708u/ccfmt"
)

type alwaysTrue struct{}

func (alwaysTrue) Exists(string) bool { return true }

var (
	update                        = flag.Bool("update", false, "update golden files")
	mockChecker ccfmt.PathChecker = alwaysTrue{}
)

func TestGolden(t *testing.T) {
	input, err := os.ReadFile("testdata/input.json")
	if err != nil {
		t.Fatalf("reading input: %v", err)
	}

	f := &ccfmt.Formatter{PathChecker: alwaysTrue{}}
	result, err := f.Format(input)
	if err != nil {
		t.Fatalf("format: %v", err)
	}

	goldenPath := "testdata/golden.json"
	if *update {
		if err := os.WriteFile(goldenPath, result.Data, 0o644); err != nil {
			t.Fatalf("updating golden: %v", err)
		}
		t.Log("golden file updated")
		return
	}

	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading golden (run with -update to generate): %v", err)
	}

	if !bytes.Equal(result.Data, golden) {
		t.Errorf("output differs from golden:\ngot:\n%s\nwant:\n%s", result.Data, golden)
	}
}

func TestIntegrationPathCleaning(t *testing.T) {
	dir := t.TempDir()

	existingProject := filepath.Join(dir, "project-a")
	existingRepo := filepath.Join(dir, "repo-path")
	os.Mkdir(existingProject, 0o755)
	os.Mkdir(existingRepo, 0o755)

	goneProject := filepath.Join(dir, "gone-project")
	goneRepo := filepath.Join(dir, "gone-repo")

	input := `{
  "projects": {
    "` + existingProject + `": {"key": "value"},
    "` + goneProject + `": {"key": "value"}
  },
  "githubRepoPaths": {
    "org/repo-a": ["` + existingRepo + `", "` + goneRepo + `"],
    "org/repo-b": ["` + goneRepo + `"]
  }
}`

	file := filepath.Join(dir, "claude.json")
	os.WriteFile(file, []byte(input), 0o644)

	var buf bytes.Buffer
	cli := &CLI{File: file}
	if err := run(cli, osPathChecker{}, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(file)
	got := string(data)

	if !strings.Contains(got, existingProject) {
		t.Error("existing project path was removed")
	}
	if strings.Contains(got, goneProject) {
		t.Error("non-existent project path was not removed")
	}

	if !strings.Contains(got, existingRepo) {
		t.Error("existing repo path was removed")
	}
	if strings.Contains(got, goneRepo) {
		t.Error("non-existent repo path was not removed")
	}

	if strings.Contains(got, "repo-b") {
		t.Error("empty repo key was not removed")
	}

	output := buf.String()
	if !strings.Contains(output, "Projects: 2 -> 1 (removed 1)") {
		t.Errorf("unexpected projects output: %s", output)
	}
	if !strings.Contains(output, "removed 2 paths, 1 empty repos") {
		t.Errorf("unexpected repo paths output: %s", output)
	}
}

func TestRun(t *testing.T) {
	input := `{"z": 1, "a": 2}`
	wantJSON := "{\n  \"a\": 2,\n  \"githubRepoPaths\": {},\n  \"projects\": {},\n  \"z\": 1\n}\n"

	t.Run("normal flow writes file without backup", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "test.json")
		os.WriteFile(file, []byte(input), 0o644)

		var buf bytes.Buffer
		cli := &CLI{File: file}
		if err := run(cli, mockChecker, &buf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, _ := os.ReadFile(file)
		if string(data) != wantJSON {
			t.Errorf("file content mismatch:\ngot:\n%s\nwant:\n%s", data, wantJSON)
		}

		matches, _ := filepath.Glob(filepath.Join(dir, "test.json.backup.*"))
		if len(matches) != 0 {
			t.Errorf("backup created without --backup flag")
		}

		output := buf.String()
		if !strings.Contains(output, "Keys sorted recursively.") {
			t.Errorf("output missing expected line: %s", output)
		}
	})

	t.Run("dry-run does not modify file", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "test.json")
		os.WriteFile(file, []byte(input), 0o644)

		var buf bytes.Buffer
		cli := &CLI{File: file, DryRun: true}
		if err := run(cli, mockChecker, &buf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, _ := os.ReadFile(file)
		if string(data) != input {
			t.Errorf("file was modified in dry-run mode")
		}

		matches, _ := filepath.Glob(filepath.Join(dir, "test.json.backup.*"))
		if len(matches) != 0 {
			t.Errorf("backup created in dry-run mode")
		}

		output := buf.String()
		if strings.Contains(output, "Backup:") {
			t.Errorf("dry-run output should not contain backup line")
		}
	})

	t.Run("backup flag creates backup", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "test.json")
		os.WriteFile(file, []byte(input), 0o644)

		var buf bytes.Buffer
		cli := &CLI{File: file, Backup: true}
		if err := run(cli, mockChecker, &buf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, _ := os.ReadFile(file)
		if string(data) != wantJSON {
			t.Errorf("file content mismatch")
		}

		matches, _ := filepath.Glob(filepath.Join(dir, "test.json.backup.*"))
		if len(matches) != 1 {
			t.Fatalf("expected 1 backup file, got %d", len(matches))
		}
		backup, _ := os.ReadFile(matches[0])
		if string(backup) != input {
			t.Errorf("backup content mismatch: got %q, want %q", backup, input)
		}

		output := buf.String()
		if !strings.Contains(output, "Backup:") {
			t.Errorf("output missing backup line: %s", output)
		}
	})

	t.Run("preserves file permissions", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "test.json")
		os.WriteFile(file, []byte(input), 0o600)

		var buf bytes.Buffer
		cli := &CLI{File: file}
		if err := run(cli, mockChecker, &buf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		info, _ := os.Stat(file)
		if info.Mode().Perm() != 0o600 {
			t.Errorf("permission changed: got %o, want 600", info.Mode().Perm())
		}
	})

	t.Run("file not found error", func(t *testing.T) {
		var buf bytes.Buffer
		cli := &CLI{File: "/nonexistent/path/test.json"}
		err := run(cli, mockChecker, &buf)
		if err == nil {
			t.Fatal("expected error for missing file")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error should mention 'not found': %v", err)
		}
	})
}
