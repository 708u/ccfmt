# TaskToolSweeper の実装と always-active 化

## 目的

Claude Code の permission に蓄積される `Task(AgentName)` 形式の
エントリを sweep する機能を追加する。カスタム agent ファイルの削除や
plugin のアンインストール後もエントリが残り続ける問題を解決する。

既存の `ToolSweeper` interface と `PermissionSweeper` アーキテクチャに
`TaskToolSweeper` を追加する形で実装した。

Task sweep は決定論的なチェック (built-in, plugin, .md 存在確認) のみで
あるため、Bash sweep のような opt-in 設計は不要と判断し、
`TaskSweepConfig`, `--sweep-task`, `[sweep.task]` を全て削除して
Read/Edit/Write/MCP と同様の always-active にリファクタリングした。

## 作業内容

### Session 1: TaskToolSweeper 実装

1. `config.go` に `TaskSweepConfig` / `rawTaskSweepConfig` 型を追加
2. `sweep.go` に `TaskToolSweeper` を実装
3. `cmd/cctidy/main.go` に `--sweep-task` CLI 配線
4. テスト追加 (unit, config merge, integration)
5. golden file 更新
6. docs 更新

### Session 2: always-active リファクタリング

1. `config.go`: `TaskSweepConfig`, `rawTaskSweepConfig` 型を削除、
   `SweepToolConfig`/`rawSweepToolConfig` から `Task` フィールド削除、
   `rawToConfig()`, `mergeRawConfigs()`, `MergeConfig()` から
   Task マージロジック削除
2. `sweep.go`: `TaskToolSweeper` から `excludes` フィールド削除、
   `NewTaskToolSweeper` シグネチャ簡素化 (cfg パラメータ削除)、
   `WithTaskSweep` オプション削除、`NewPermissionSweeper` で
   Task sweeper を常時登録
3. `cmd/cctidy/main.go`: `SweepTask` CLI フラグ削除、
   `taskSweepConfig()` メソッド削除、Task 配線コード削除
4. `sweep_test.go`: "excluded agent" テスト削除、
   `NewTaskToolSweeper` 呼び出し更新、
   "task entries kept when sweep-task not enabled" 削除
5. `config_test.go`: Task 関連 TOML パース/マージテスト削除
6. `integration_test.go`: `WithTaskSweep` 削除、
   `TestIntegrationTaskSweepDisabledByDefault` 削除、
   `TestIntegrationTaskSweepWithConfig` (3 subtests) 削除
7. docs 更新: `--sweep-task`, `[sweep.task]` セクション削除、
   Task を enabled に変更、Exclude Patterns 削除
8. plugin.json バージョン 0.1.1 -> 0.1.2

### Session 3: レビュー指摘対応と frontmatter 対応計画

1. `builtinAgents` を `[]string` から `map[string]bool` に変更し、
   `NewTaskToolSweeper` での毎回の map 再構築を排除
2. agent lookup を settings level にスコープ:
   project-level は project agents dir のみ、
   user-level は home agents dir のみチェック
3. `resolveTargets` で `-t` 指定時の user-level 判定を追加
   (`isUserLevelSettings` を inline 化)
4. markdownlint warnings 修正
   (重複見出し MD024、テーブル整列 MD060)
5. claude-code-guide agent で built-in agents リスト
   (6種) の網羅性を確認
6. frontmatter ベース agent 名解決の設計を計画
   (AgentNameSet パターン、plan mode で策定)

## 変更ファイル

- sweep.go:106-115, 304-352, 407-440
- cmd/cctidy/main.go:203-222, 262-280
- sweep_test.go:587-640
- cmd/cctidy/integration_test.go:825-960
- docs/reference/permission-sweeping.md:129-173, 184-188

## 利用した Skill

- /commit-push-update-pr
- /continue
- /export-session

## Pull Request

<https://github.com/708u/cctidy/pull/33>

## 未完了タスク

- [ ] frontmatter ベースの agent 名解決を実装
  (`AgentNameSet` / `LoadAgentNames` パターン、
  plan file: `~/.claude/plans/proud-munching-hummingbird.md`)
- [ ] settings level 明示化のリファクタリング
  (調査済み、タスク: `docs/tasks/explicit-settings-level.md`)
