# gh-otel-harness

Claude Code の失敗履歴を OpenObserve から取得し、ハーネスリポジトリへ GitHub Issue として起票する `gh` 拡張機能。

## インストール

```bash
gh extension install UtakataKyosui/gh-otel-harness
```

## セットアップ

```bash
gh otel-harness configure
```

`~/.config/gh-otel-harness/config.toml` を対話式で作成します。

### 環境変数による上書き

| 変数 | 説明 |
|---|---|
| `OO_AUTH` | OpenObserve 認証ヘッダー (`Basic <base64>`) |
| `HARNESS_REPO` | 起票先リポジトリ (`owner/repo`) |

## 使い方

### TUI モード（既定）

```bash
gh otel-harness
gh otel-harness --since 7d
gh otel-harness --type tool_error,refusal
gh otel-harness --project my-project
```

space で複数選択、enter で一括起票。

### 一覧表示

```bash
gh otel-harness list
gh otel-harness list --since 7d -j   # JSON 出力
```

### 単発起票

```bash
gh otel-harness open <event-id>
gh otel-harness open <event-id> --dry-run   # title/body を確認
```

### Claude Code に指示を渡す

```bash
# プロンプトを生成して Claude Code に渡す
gh otel-harness prompt <event-id> | claude --print

# ファイルに保存してから渡す
gh otel-harness prompt <event-id> > /tmp/harness.md
```

`prompt` サブコマンドは Claude Code が読み取れる Markdown を stdout に出力します。
Claude Code はそのプロンプトに従って fingerprint 重複チェック → ハーネステスト追加 → Issue 起票まで自律的に実行できます。

## 検出対象

| カテゴリ | 条件 |
|---|---|
| `tool_error` | `claude_code.tool_result` かつ `success = false` |
| `refusal` | `claude_code.tool_decision` かつ `decision = reject`、または `severityText = ERROR` |
| `tool_anomaly` | `api_error` / `api_retries_exhausted` / `internal_error` イベント |

## 重複除去

Issue body の HTML コメントに fingerprint を埋め込みます:

```
<!-- gh-otel-harness:fingerprint:abc123def456 -->
```

起票前に `gh search issues` でこの fingerprint を検索し、既存 Issue があればスキップします。

## 設定ファイル

`~/.config/gh-otel-harness/config.toml`:

```toml
[openobserve]
endpoint = "http://localhost:5080"
org = "default"
stream = "default"
auth = "Basic <base64(email:password)>"

[harness]
repo = "owner/claude-harness"
default_labels = ["claude-code", "telemetry-derived"]

[query]
default_since = "24h"
project_filter = ""
```

## 前提条件

- [Claude Code の OpenTelemetry 設定](https://code.claude.com/docs/en/monitoring-usage) が有効であること
- OpenObserve にテレメトリが蓄積されていること (`CLAUDE_CODE_ENABLE_TELEMETRY=1`)
- `gh auth login` で GitHub 認証済みであること
