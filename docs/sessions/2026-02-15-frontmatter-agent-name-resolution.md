# Frontmatter-based agent name resolution

## 目的

`TaskToolSweeper` が `Task(AgentName)` permission entries を
sweep する際、ファイル名ベースのみで agent の生存判定を行っていた。
agent `.md` ファイルは YAML frontmatter の `name` フィールドで
ファイル名と異なる名前を定義できるため、frontmatter の `name` を
チェック対象にする。

## 作業内容

### 実装

1. goldmark + goldmark-frontmatter を依存に追加
2. `agent.go` を新規作成: `AgentNameSet` 型、`LoadAgentNames`
   関数、`parseAgentName` 関数を実装
3. `TaskToolSweeper` をリファクタリング: `PathChecker` /
   `homeDir` / `baseDir` から `AgentNameSet` ベースに変更
4. `NewPermissionSweeper` 内で settings level に応じた agents
   ディレクトリの解決と `LoadAgentNames` 呼び出しを実装
5. goldmark parser の初期化をパッケージレベル変数に引き上げ
6. plugin version bump (0.1.2 -> 0.1.3)

### レビュー中の修正

- `strings.HasSuffix` -> `filepath.Ext` に変更
- parser 初期化を `parseAgentName` 呼び出しごとから
  パッケージレベル変数 `fmParser` に引き上げ
- plugin agent のコロンチェックにコメント追加
- frontmatter `name` がファイル名をオーバーライドする仕様を確認
  (claude-code-guide agent で調査)
- `name` は required field であり、ファイル名は agent 識別に
  使われないことが判明
- `LoadAgentNames` からファイル名ベースの登録を完全に削除

### 重要な設計判断

- `name` フィールドは required → ファイル名 fallback は誤検知の
  原因になるため削除
- `len(agents) == 0` (agents dir 不在) のとき保守的に keep
- frontmatter `name` が唯一の agent 識別子

## 変更ファイル

- @agent.go
- @agent_test.go
- sweep.go:304-340
- sweep_test.go:562-646, 956-1005
- cmd/cctidy/integration_test.go:58-67, 840-845, 858-860, 903-912, 935-940, 953-955, 979-990
- @docs/reference/permission-sweeping.md
- @go.mod

## 利用したSkill

- /commit-push-update-pr
- /export-session

## Pull Request

<https://github.com/708u/cctidy/pull/33>

## 未完了タスク

- [ ] sweep_test.go の `TestTaskToolSweeperShouldSweep` で
  `file-agent` を含む AgentNameSet を使っているテストケース
  ("frontmatter name is kept") の修正
- [ ] integration test の frontmatter テストケース更新
  (ファイル名 fallback 削除に合わせて)
- [ ] docs/reference/permission-sweeping.md の Agent Name
  Resolution セクションからファイル名関連の記述を削除
- [ ] 未コミットの変更 (agent.go, agent_test.go, sweep.go) を
  コミット・プッシュ
