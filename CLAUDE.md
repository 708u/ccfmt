# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ccfmt is a CLI tool that formats `~/.claude.json`. It performs:

- Recursive key sorting of all JSON objects
- Removal of non-existent project paths (`projects` key)
- Removal of non-existent GitHub repo paths (`githubRepoPaths` key),
  including cleanup of empty repo keys
- Sorting of homogeneous arrays (string, number, bool)

## Commands

```bash
make build    # Build binary to ./ccfmt
make install  # Install to $GOPATH/bin
make test     # Run all tests (unit + integration)

# Update golden file for integration tests
go test -tags integration ./cmd/ -update
```

## Architecture

Two packages:

- **`ccfmt` (root)** - Library package. `Formatter` takes a
  `PathChecker` interface and raw JSON bytes, returns formatted
  bytes + `Stats`. No filesystem I/O.
- **`cmd/`** - CLI entrypoint (`package main`). Uses kong for
  flag parsing. `run()` handles file I/O, backup creation,
  and wiring `Formatter` with `osPathChecker`.

`PathChecker` interface enables testing without real filesystem
access. Tests use `alwaysTrue`, `alwaysFalse`, and `pathSet`
stubs.

Integration tests use `//go:build integration` tag and live in
`cmd/integration_test.go`. Golden test data is in
`cmd/testdata/`.
