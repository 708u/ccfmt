# Explicit Settings Level

## Summary

Introduce `SettingsLevel` type to replace the implicit
`baseDir == ""` convention for distinguishing user-level
and project-level settings.

## Motivation

The sweeper uses `baseDir` emptiness to decide whether
to scan `~/.claude/` or `<project>/.claude/` for agents,
skills, and commands. This is an implicit convention
that is not self-documenting.

## Design

- Add `SettingsLevel` enum: `UserLevel` (zero value),
  `ProjectLevel`
- Replace `WithBaseDir(dir)` with
  `WithProjectLevel(dir)` which sets both `level` and
  `baseDir`
- `UserLevel` is the default (no option needed)
- `baseDir` remains in `sweepConfig` for path resolution
  by `ReadEditToolSweeper` and `BashToolSweeper`
- `level` is used only in `NewPermissionSweeper` for
  `claudeDir` derivation

## Changes

| File | Change |
| --- | --- |
| `sweep.go` | `SettingsLevel` type, `sweepConfig.level`, `WithProjectLevel`, `claudeDir` switch |
| `cmd/cctidy/main.go` | `WithBaseDir` -> `WithProjectLevel` |
| `sweep_test.go` | `WithBaseDir` -> `WithProjectLevel` |
| `cmd/cctidy/integration_test.go` | `WithBaseDir` -> `WithProjectLevel` |

## Unchanged

- `ReadEditToolSweeper` / `BashToolSweeper` (use
  `baseDir` only)
- `TaskToolSweeper` / `SkillToolSweeper` /
  `MCPToolSweeper`
- `NewBashToolSweeper` signature
- `WithUnsafe` / `WithBashConfig`
- Golden test files
