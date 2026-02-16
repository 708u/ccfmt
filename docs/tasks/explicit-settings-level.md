# Explicit Settings Level

## Summary

Introduce `SettingsLevel` type to replace the implicit
`projectDir == ""` convention for distinguishing user-level
and project-level settings.

## Motivation

The sweeper uses `projectDir` emptiness to decide whether
to scan `~/.claude/` or `<project>/.claude/` for agents,
skills, and commands. This is an implicit convention
that is not self-documenting.

## Design

- Add `SettingsLevel` enum: `UserLevel` (zero value),
  `ProjectLevel`
- Replace `WithBaseDir(dir)` with
  `WithProjectLevel(dir)` which sets both `level` and
  `projectDir`
- `UserLevel` is the default (no option needed)
- `projectDir` remains in `sweepConfig` for path resolution
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
  `projectDir` only)
- `TaskToolSweeper` / `SkillToolSweeper` /
  `MCPToolSweeper`
- `NewBashToolSweeper` signature
- `WithUnsafe` / `WithBashConfig`
- Golden test files
