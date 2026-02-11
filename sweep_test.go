package cctidy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractToolEntry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		entry         string
		wantTool      string
		wantSpecifier string
	}{
		{
			name:          "Bash tool",
			entry:         "Bash(git -C /repo status)",
			wantTool:      "Bash",
			wantSpecifier: "git -C /repo status",
		},
		{
			name:          "Write tool",
			entry:         "Write(/some/path)",
			wantTool:      "Write",
			wantSpecifier: "/some/path",
		},
		{
			name:          "Read tool",
			entry:         "Read(/some/path)",
			wantTool:      "Read",
			wantSpecifier: "/some/path",
		},
		{
			name:          "mcp tool with underscores",
			entry:         "mcp__github__search_code(query)",
			wantTool:      "mcp__github__search_code",
			wantSpecifier: "query",
		},
		{
			name:     "bare tool name without parens",
			entry:    "Bash",
			wantTool: "",
		},
		{
			name:     "empty string",
			entry:    "",
			wantTool: "",
		},
		{
			name:     "starts with number",
			entry:    "1Tool(arg)",
			wantTool: "",
		},
		{
			name:          "WebFetch tool",
			entry:         "WebFetch(domain:github.com)",
			wantTool:      "WebFetch",
			wantSpecifier: "domain:github.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotTool, gotSpec := extractToolEntry(tt.entry)
			if gotTool != tt.wantTool {
				t.Errorf("extractToolEntry(%q) tool = %q, want %q", tt.entry, gotTool, tt.wantTool)
			}
			if gotSpec != tt.wantSpecifier {
				t.Errorf("extractToolEntry(%q) specifier = %q, want %q", tt.entry, gotSpec, tt.wantSpecifier)
			}
		})
	}
}

func TestContainsGlob(t *testing.T) {
	t.Parallel()
	tests := []struct {
		s    string
		want bool
	}{
		{"**/*.ts", true},
		{"/path/to/file", false},
		{"src/[a-z]/*.go", true},
		{"file?.txt", true},
		{"normal/path", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			t.Parallel()
			if got := containsGlob(tt.s); got != tt.want {
				t.Errorf("containsGlob(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestReadEditToolSweeperShouldSweep(t *testing.T) {
	t.Parallel()

	t.Run("absolute path with // prefix is resolved", func(t *testing.T) {
		t.Parallel()
		p := &ReadEditToolSweeper{checker: alwaysFalse{}}
		result := p.ShouldSweep(t.Context(), "//dead/path")
		if !result.Sweep {
			t.Error("should sweep non-existent absolute path")
		}
	})

	t.Run("existing absolute path is kept", func(t *testing.T) {
		t.Parallel()
		p := &ReadEditToolSweeper{checker: checkerFor("/alive/path")}
		result := p.ShouldSweep(t.Context(), "//alive/path")
		if result.Sweep {
			t.Error("should not sweep existing absolute path")
		}
	})

	t.Run("home-relative path with homeDir is resolved", func(t *testing.T) {
		t.Parallel()
		p := &ReadEditToolSweeper{
			checker: alwaysFalse{},
			homeDir: "/home/user",
		}
		result := p.ShouldSweep(t.Context(), "~/config.json")
		if !result.Sweep {
			t.Error("should sweep non-existent home-relative path")
		}
	})

	t.Run("existing home-relative path is kept", func(t *testing.T) {
		t.Parallel()
		p := &ReadEditToolSweeper{
			checker: checkerFor("/home/user/config.json"),
			homeDir: "/home/user",
		}
		result := p.ShouldSweep(t.Context(), "~/config.json")
		if result.Sweep {
			t.Error("should not sweep existing home-relative path")
		}
	})

	t.Run("home-relative path without homeDir is skipped", func(t *testing.T) {
		t.Parallel()
		p := &ReadEditToolSweeper{checker: alwaysFalse{}}
		result := p.ShouldSweep(t.Context(), "~/config.json")
		if result.Sweep {
			t.Error("should skip home-relative path without homeDir")
		}
	})

	t.Run("relative path with baseDir is resolved", func(t *testing.T) {
		t.Parallel()
		p := &ReadEditToolSweeper{
			checker: alwaysFalse{},
			baseDir: "/project",
		}
		result := p.ShouldSweep(t.Context(), "./src/main.go")
		if !result.Sweep {
			t.Error("should sweep non-existent relative path")
		}
	})

	t.Run("relative path without baseDir is skipped", func(t *testing.T) {
		t.Parallel()
		p := &ReadEditToolSweeper{checker: alwaysFalse{}}
		result := p.ShouldSweep(t.Context(), "./src/main.go")
		if result.Sweep {
			t.Error("should skip relative path without baseDir")
		}
	})

	t.Run("glob pattern is skipped", func(t *testing.T) {
		t.Parallel()
		p := &ReadEditToolSweeper{checker: alwaysFalse{}}
		result := p.ShouldSweep(t.Context(), "**/*.ts")
		if result.Sweep {
			t.Error("should skip glob pattern")
		}
	})

	t.Run("parent-relative path with baseDir is resolved", func(t *testing.T) {
		t.Parallel()
		p := &ReadEditToolSweeper{
			checker: alwaysFalse{},
			baseDir: "/project",
		}
		result := p.ShouldSweep(t.Context(), "../other/file.go")
		if !result.Sweep {
			t.Error("should sweep non-existent parent-relative path")
		}
	})

	t.Run("slash-prefixed path with baseDir is resolved", func(t *testing.T) {
		t.Parallel()
		p := &ReadEditToolSweeper{
			checker: checkerFor("/project/src/file.go"),
			baseDir: "/project",
		}
		result := p.ShouldSweep(t.Context(), "/src/file.go")
		if result.Sweep {
			t.Error("should not sweep existing path resolved with baseDir")
		}
	})
}

func TestSweepPermissions(t *testing.T) {
	t.Parallel()

	t.Run("dead absolute path entry is removed", func(t *testing.T) {
		t.Parallel()
		obj := map[string]any{
			"permissions": map[string]any{
				"allow": []any{"Read(//dead/path)"},
			},
		}
		result := NewPermissionSweeper(alwaysFalse{}).Sweep(t.Context(), obj)
		allow := obj["permissions"].(map[string]any)["allow"].([]any)
		if len(allow) != 0 {
			t.Errorf("allow should be empty, got %v", allow)
		}
		if result.SweptAllow != 1 {
			t.Errorf("SweptAllow = %d, want 1", result.SweptAllow)
		}
	})

	t.Run("existing absolute path entry is kept", func(t *testing.T) {
		t.Parallel()
		obj := map[string]any{
			"permissions": map[string]any{
				"allow": []any{"Read(//alive/path)"},
			},
		}
		result := NewPermissionSweeper(checkerFor("/alive/path")).Sweep(t.Context(), obj)
		allow := obj["permissions"].(map[string]any)["allow"].([]any)
		if len(allow) != 1 {
			t.Errorf("allow should have 1 entry, got %v", allow)
		}
		if result.SweptAllow != 0 {
			t.Errorf("SweptAllow = %d, want 0", result.SweptAllow)
		}
	})

	t.Run("home-relative path with homeDir is swept when dead", func(t *testing.T) {
		t.Parallel()
		obj := map[string]any{
			"permissions": map[string]any{
				"allow": []any{"Read(~/dead/config)"},
			},
		}
		result := NewPermissionSweeper(alwaysFalse{}, WithHomeDir("/home/user")).Sweep(t.Context(), obj)
		allow := obj["permissions"].(map[string]any)["allow"].([]any)
		if len(allow) != 0 {
			t.Errorf("allow should be empty, got %v", allow)
		}
		if result.SweptAllow != 1 {
			t.Errorf("SweptAllow = %d, want 1", result.SweptAllow)
		}
	})

	t.Run("home-relative path with homeDir is kept when exists", func(t *testing.T) {
		t.Parallel()
		obj := map[string]any{
			"permissions": map[string]any{
				"allow": []any{"Read(~/config)"},
			},
		}
		result := NewPermissionSweeper(checkerFor("/home/user/config"), WithHomeDir("/home/user")).Sweep(t.Context(), obj)
		allow := obj["permissions"].(map[string]any)["allow"].([]any)
		if len(allow) != 1 {
			t.Errorf("allow should have 1 entry, got %v", allow)
		}
		if result.SweptAllow != 0 {
			t.Errorf("SweptAllow = %d, want 0", result.SweptAllow)
		}
	})

	t.Run("relative path without baseDir is kept", func(t *testing.T) {
		t.Parallel()
		obj := map[string]any{
			"permissions": map[string]any{
				"allow": []any{"Edit(/src/file.go)"},
			},
		}
		result := NewPermissionSweeper(alwaysFalse{}).Sweep(t.Context(), obj)
		allow := obj["permissions"].(map[string]any)["allow"].([]any)
		if len(allow) != 1 {
			t.Errorf("allow should have 1 entry, got %v", allow)
		}
		if result.SweptAllow != 0 {
			t.Errorf("SweptAllow = %d, want 0", result.SweptAllow)
		}
	})

	t.Run("relative path with baseDir is swept when dead", func(t *testing.T) {
		t.Parallel()
		obj := map[string]any{
			"permissions": map[string]any{
				"allow": []any{"Edit(./src/file.go)"},
			},
		}
		result := NewPermissionSweeper(alwaysFalse{}, WithBaseDir("/project")).Sweep(t.Context(), obj)
		allow := obj["permissions"].(map[string]any)["allow"].([]any)
		if len(allow) != 0 {
			t.Errorf("allow should be empty, got %v", allow)
		}
		if result.SweptAllow != 1 {
			t.Errorf("SweptAllow = %d, want 1", result.SweptAllow)
		}
	})

	t.Run("relative path with baseDir is kept when exists", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "file.go"), []byte(""), 0o644)

		obj := map[string]any{
			"permissions": map[string]any{
				"allow": []any{"Edit(./file.go)"},
			},
		}
		result := NewPermissionSweeper(checkerFor(filepath.Join(dir, "file.go")), WithBaseDir(dir)).Sweep(t.Context(), obj)
		allow := obj["permissions"].(map[string]any)["allow"].([]any)
		if len(allow) != 1 {
			t.Errorf("allow should have 1 entry, got %v", allow)
		}
		if result.SweptAllow != 0 {
			t.Errorf("SweptAllow = %d, want 0", result.SweptAllow)
		}
	})

	t.Run("glob pattern entry is kept", func(t *testing.T) {
		t.Parallel()
		obj := map[string]any{
			"permissions": map[string]any{
				"allow": []any{"Read(**/*.ts)"},
			},
		}
		result := NewPermissionSweeper(alwaysFalse{}).Sweep(t.Context(), obj)
		allow := obj["permissions"].(map[string]any)["allow"].([]any)
		if len(allow) != 1 {
			t.Errorf("allow should have 1 entry, got %v", allow)
		}
		if result.SweptAllow != 0 {
			t.Errorf("SweptAllow = %d, want 0", result.SweptAllow)
		}
	})

	t.Run("unregistered tool entries are kept", func(t *testing.T) {
		t.Parallel()
		obj := map[string]any{
			"permissions": map[string]any{
				"allow": []any{
					"Bash(git -C /dead/path status)",
					"WebFetch(domain:example.com)",
				},
			},
		}
		result := NewPermissionSweeper(alwaysFalse{}).Sweep(t.Context(), obj)
		allow := obj["permissions"].(map[string]any)["allow"].([]any)
		if len(allow) != 2 {
			t.Errorf("allow should have 2 entries, got %v", allow)
		}
		if result.SweptAllow != 0 {
			t.Errorf("SweptAllow = %d, want 0", result.SweptAllow)
		}
	})

	t.Run("missing permissions key is no-op", func(t *testing.T) {
		t.Parallel()
		obj := map[string]any{"key": "value"}
		result := NewPermissionSweeper(alwaysTrue{}).Sweep(t.Context(), obj)
		if result.SweptAllow != 0 || result.SweptAsk != 0 {
			t.Errorf("expected zero counts, got allow=%d ask=%d",
				result.SweptAllow, result.SweptAsk)
		}
		if len(result.Warns) != 0 {
			t.Errorf("expected no warnings, got %v", result.Warns)
		}
	})

	t.Run("deny entries are never swept", func(t *testing.T) {
		t.Parallel()
		obj := map[string]any{
			"permissions": map[string]any{
				"allow": []any{"Read(//dead/allow)"},
				"deny":  []any{"Read(//dead/deny)"},
				"ask":   []any{"Edit(//dead/ask)"},
			},
		}
		result := NewPermissionSweeper(alwaysFalse{}).Sweep(t.Context(), obj)
		if result.SweptAllow != 1 {
			t.Errorf("SweptAllow = %d, want 1", result.SweptAllow)
		}
		if result.SweptAsk != 1 {
			t.Errorf("SweptAsk = %d, want 1", result.SweptAsk)
		}
		deny := obj["permissions"].(map[string]any)["deny"].([]any)
		if len(deny) != 1 {
			t.Errorf("deny should be kept unchanged, got %v", deny)
		}
	})
}
