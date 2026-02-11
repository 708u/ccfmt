package cctidy

import (
	"context"
	"path/filepath"
	"regexp"
	"slices"
)

var absPathRe = regexp.MustCompile(`(?:^|[\s=(])(/[^\s"'):*]+)`)
var relPathRe = regexp.MustCompile(`\./[^\s"'):*]+`)

// ExtractAbsolutePaths extracts all absolute paths from a permission entry string.
func ExtractAbsolutePaths(entry string) []string {
	matches := absPathRe.FindAllStringSubmatch(entry, -1)
	if matches == nil {
		return nil
	}
	paths := make([]string, len(matches))
	for i, m := range matches {
		paths[i] = m[1]
	}
	return paths
}

// ExtractRelativePaths extracts all relative paths from a permission entry string.
func ExtractRelativePaths(entry string) []string {
	matches := relPathRe.FindAllString(entry, -1)
	if matches == nil {
		return nil
	}
	return matches
}

// PruneResult holds statistics from permission pruning.
// Deny entries are intentionally excluded from pruning because they represent
// explicit user prohibitions; removing stale deny rules costs nothing but
// could silently re-enable a previously blocked action.
type PruneResult struct {
	PrunedAllow   int
	PrunedAsk     int
	RelativeWarns []string
}

// PermissionPruner prunes stale permission entries from settings objects.
// By default only absolute paths are evaluated. Use WithBaseDir to enable
// relative path resolution; without it, entries containing relative paths
// are kept unchanged and recorded in PruneResult.RelativeWarns.
type PermissionPruner struct {
	checker        PathChecker
	baseDir        string
	resolveRelPath bool
	result         *PruneResult
}

// PruneOption configures a PermissionPruner.
type PruneOption func(*PermissionPruner)

// WithBaseDir enables relative path resolution against the given directory.
func WithBaseDir(dir string) PruneOption {
	return func(p *PermissionPruner) {
		p.baseDir = dir
		p.resolveRelPath = true
	}
}

// NewPermissionPruner creates a PermissionPruner.
func NewPermissionPruner(checker PathChecker, opts ...PruneOption) *PermissionPruner {
	p := &PermissionPruner{checker: checker}
	for _, o := range opts {
		o(p)
	}
	return p
}

// Prune removes stale allow/ask permission entries from obj.
func (p *PermissionPruner) Prune(ctx context.Context, obj map[string]any) *PruneResult {
	p.result = &PruneResult{}

	raw, ok := obj["permissions"]
	if !ok {
		return p.result
	}
	perms, ok := raw.(map[string]any)
	if !ok {
		return p.result
	}

	type category struct {
		key   string
		count *int
	}
	categories := []category{
		{"allow", &p.result.PrunedAllow},
		{"ask", &p.result.PrunedAsk},
	}

	for _, cat := range categories {
		raw, ok := perms[cat.key]
		if !ok {
			continue
		}
		arr, ok := raw.([]any)
		if !ok {
			continue
		}

		kept := make([]any, 0, len(arr))
		for _, v := range arr {
			entry, ok := v.(string)
			if !ok {
				kept = append(kept, v)
				continue
			}

			if p.shouldPrune(ctx, entry) {
				*cat.count++
				continue
			}

			kept = append(kept, v)
		}
		perms[cat.key] = kept
	}

	return p.result
}

func (p *PermissionPruner) shouldPrune(ctx context.Context, entry string) bool {
	absPaths := ExtractAbsolutePaths(entry)
	if len(absPaths) > 0 {
		return !slices.ContainsFunc(absPaths, func(path string) bool {
			return p.checker.Exists(ctx, path)
		})
	}

	relPaths := ExtractRelativePaths(entry)
	if len(relPaths) > 0 {
		// Without WithBaseDir, relative paths cannot be resolved.
		// Keep the entry and record it as a warning.
		if !p.resolveRelPath {
			p.result.RelativeWarns = append(p.result.RelativeWarns, entry)
			return false
		}
		return !slices.ContainsFunc(relPaths, func(path string) bool {
			return p.checker.Exists(ctx, filepath.Join(p.baseDir, path))
		})
	}

	return false
}
