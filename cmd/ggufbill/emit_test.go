package main

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestPrompter builds a prompter over fully buffered input (answers "pasted"
// in one chunk) with output discarded, mirroring how a terminal paste delivers
// every line before the first question is answered.
func newTestPrompter(input string) *prompter {
	return &prompter{in: bufio.NewReader(strings.NewReader(input)), out: io.Discard}
}

// TestPrompterConsumesPastedLines guards the v0.1.0 defect where every prompt
// built a fresh bufio.Reader over stdin: a pasted multi-line answer was
// buffered by the first reader and discarded, so later questions silently fell
// back to their defaults. One shared reader must consume all lines in order.
func TestPrompterConsumesPastedLines(t *testing.T) {
	p := newTestPrompter("4096\nQ8_0\nops\n")
	if got := p.askInt64("Serving context length (tokens)", 8192); got != 4096 {
		t.Errorf("context = %d, want 4096 (first pasted line)", got)
	}
	if got := p.ask("KV cache dtype (e.g. Q8_0, f16)", "f16"); got != "Q8_0" {
		t.Errorf("kv = %q, want Q8_0 (second pasted line)", got)
	}
	if got := p.ask("Operator name", "operator"); got != "ops" {
		t.Errorf("operator = %q, want ops (third pasted line)", got)
	}
}

// TestPrompterDefaultsOnEmptyAndEOF keeps the fallback contract: an empty line
// accepts the shown default, an unparseable or non-positive number keeps the
// default, and exhausted input (EOF) also yields the default instead of
// blocking or returning garbage.
func TestPrompterDefaultsOnEmptyAndEOF(t *testing.T) {
	p := newTestPrompter("\nnot-a-number\n")
	if got := p.ask("KV cache dtype (e.g. Q8_0, f16)", "f16"); got != "f16" {
		t.Errorf("empty line = %q, want default f16", got)
	}
	if got := p.askInt64("Serving context length (tokens)", 4096); got != 4096 {
		t.Errorf("unparseable number = %d, want default 4096", got)
	}
	if got := p.ask("Operator name", "ops"); got != "ops" {
		t.Errorf("EOF = %q, want default ops", got)
	}
}

// TestEmitRefusesToOverwriteExistingSidecar guards the v0.4.0 fix: emit used to
// replace any existing sidecar with that run's flag defaults, silently
// destroying the operator-validated fields (extra flags, quirks, procurement
// ref, 等保 level) they had edited in place. The second emit must fail and leave
// the file untouched.
func TestEmitRefusesToOverwriteExistingSidecar(t *testing.T) {
	dir := t.TempDir()
	model := copyFixtureTo(t, dir, "qwen3-27b-q4_k_m.gguf")
	out := filepath.Join(dir, "custom.ggufbill.json")
	args := []string{model, "--context", "4096", "--kv", "Q8_0", "--operator", "ops-a",
		"--non-interactive", "--output", out}
	if err := runEmit(args); err != nil {
		t.Fatalf("first emit: %v", err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(raw), `"tool_calling": false`, `"tool_calling": true`, 1)
	if edited == string(raw) {
		t.Fatalf("fixture sidecar has no tool_calling field to edit:\n%s", raw)
	}
	if err := os.WriteFile(out, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	second := []string{model, "--context", "8192", "--kv", "Q4_0", "--operator", "ops-b",
		"--non-interactive", "--output", out}
	err = runEmit(second)
	if err == nil || !strings.Contains(err.Error(), "refusing to overwrite") || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("err = %v, want a refusal naming --force", err)
	}
	after, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != edited {
		t.Errorf("existing sidecar was modified despite the refusal:\n%s", after)
	}
}

// TestEmitForceOverwritesExistingSidecar keeps the escape hatch: --force
// restores the old overwrite behavior for deliberate re-emits.
func TestEmitForceOverwritesExistingSidecar(t *testing.T) {
	dir := t.TempDir()
	model := copyFixtureTo(t, dir, "qwen3-27b-q4_k_m.gguf")
	out := filepath.Join(dir, "custom.ggufbill.json")
	args := []string{model, "--context", "4096", "--kv", "Q8_0", "--operator", "ops-a",
		"--non-interactive", "--output", out}
	if err := runEmit(args); err != nil {
		t.Fatalf("first emit: %v", err)
	}
	forced := []string{model, "--context", "8192", "--kv", "Q4_0", "--operator", "ops-b",
		"--non-interactive", "--output", out, "--force"}
	if err := runEmit(forced); err != nil {
		t.Fatalf("forced emit: %v", err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"context_length": 8192`, `"kv_cache_dtype": "Q4_0"`, `"operator": "ops-b"`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("forced emit did not rewrite %s:\n%s", want, raw)
		}
	}
}
