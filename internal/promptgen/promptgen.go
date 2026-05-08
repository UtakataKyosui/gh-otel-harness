package promptgen

import (
	"fmt"
	"strings"
	"time"

	"github.com/UtakataKyosui/gh-c2-harness/internal/classify"
	"github.com/UtakataKyosui/gh-c2-harness/internal/fingerprint"
	"github.com/UtakataKyosui/gh-c2-harness/internal/issue"
)

// Generate produces a Markdown prompt that Claude Code can consume to:
// 1. Add a harness test case
// 2. Create a GitHub Issue (with dedup check)
func Generate(e *classify.Event, harnessRepo string, labels []string) string {
	fp := fingerprint.Compute(e.EventName, e.ToolName, e.ErrorType, e.Body)
	issueBody := issue.BuildBody(e, fp)
	issueTitle := e.Title()

	var b strings.Builder
	fmt.Fprintf(&b, "# Task: Claude Code 失敗ケースをハーネスに取り込む\n\n")
	fmt.Fprintf(&b, "## 対象イベント\n\n")
	fmt.Fprintf(&b, "| フィールド | 値 |\n|---|---|\n")
	fmt.Fprintf(&b, "| category | `%s` |\n", e.Category)
	fmt.Fprintf(&b, "| event_name | `%s` |\n", e.EventName)
	fmt.Fprintf(&b, "| tool_name | `%s` |\n", e.ToolName)
	fmt.Fprintf(&b, "| error_type | `%s` |\n", e.ErrorType)
	fmt.Fprintf(&b, "| session_id | `%s` |\n", e.SessionID)
	fmt.Fprintf(&b, "| project | `%s` |\n", e.ProjectName)
	fmt.Fprintf(&b, "| timestamp | `%s` |\n", e.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(&b, "| fingerprint | `%s` |\n\n", fp)

	fmt.Fprintf(&b, "## あなたが行うこと\n\n")
	fmt.Fprintf(&b, "1. **重複チェック**: 起票前に必ず以下を実行してください。\n")
	fmt.Fprintf(&b, "   ```bash\n   rtk gh search issues \"fingerprint:%s\" repo:%s in:body\n   ```\n", fp, harnessRepo)
	fmt.Fprintf(&b, "   - ヒットした場合は既存 Issue にコメントを追加し、新規起票しない。\n")
	fmt.Fprintf(&b, "   - ヒットしない場合のみ手順 2-4 を進める。\n\n")

	fmt.Fprintf(&b, "2. **ハーネスケース追加**: リポジトリ `%s` の `cases/%s/` ディレクトリに\n", harnessRepo, fp)
	fmt.Fprintf(&b, "   このエラーを再現するためのメモ・再現スクリプト・または Markdown を追加する。\n\n")

	labelArgs := ""
	for _, l := range labels {
		labelArgs += fmt.Sprintf(" --label %q", l)
	}

	fmt.Fprintf(&b, "3. **Issue 起票**: 以下のコマンドを実行する。\n")
	fmt.Fprintf(&b, "   ```bash\n")
	fmt.Fprintf(&b, "   rtk gh issue create -R %s --title %q%s --body \"$(cat <<'BODY'\n", harnessRepo, issueTitle, labelArgs)
	fmt.Fprintf(&b, "%s\nBODY\n)\"\n", issueBody)
	fmt.Fprintf(&b, "   ```\n\n")

	fmt.Fprintf(&b, "4. **完了**: 起票した Issue の番号と URL を返してください。\n\n")

	fmt.Fprintf(&b, "## 制約\n\n")
	fmt.Fprintf(&b, "- fingerprint チェックを省略しない\n")
	fmt.Fprintf(&b, "- Issue body の末尾の HTML コメント (`<!-- gh-c2-harness:fingerprint:... -->`) を必ず含める\n")
	fmt.Fprintf(&b, "- `rtk gh` を使い、`gh` を直接呼ばない\n")

	return b.String()
}
