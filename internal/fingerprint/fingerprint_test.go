package fingerprint

import (
	"testing"
)

func TestNormalize(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{
			in:   "session 550e8400-e29b-41d4-a716-446655440000 failed",
			want: "session <redacted> failed",
		},
		{
			in:   "file /Users/taiki/project/main.go not found",
			want: "file <redacted> not found",
		},
		{
			in:   "at line 42",
			want: "at <redacted>",
		},
	}
	for _, c := range cases {
		got := Normalize(c.in)
		if got != c.want {
			t.Errorf("Normalize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCompute_stable(t *testing.T) {
	// same semantic error → same fingerprint
	fp1 := Compute("claude_code.tool_result", "Bash", "exec_failed", "Error: command not found at line 10")
	fp2 := Compute("claude_code.tool_result", "Bash", "exec_failed", "Error: command not found at line 99")
	if fp1 != fp2 {
		t.Errorf("expected same fingerprint, got %q vs %q", fp1, fp2)
	}
	if len(fp1) != 12 {
		t.Errorf("expected 12 char fingerprint, got %d", len(fp1))
	}
}

func TestCompute_distinct(t *testing.T) {
	fp1 := Compute("claude_code.tool_result", "Bash", "exec_failed", "Error: command not found")
	fp2 := Compute("claude_code.api_error", "Bash", "exec_failed", "Error: command not found")
	if fp1 == fp2 {
		t.Error("different event_name should produce different fingerprint")
	}
}
