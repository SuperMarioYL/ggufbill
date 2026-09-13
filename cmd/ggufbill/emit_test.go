package main

import (
	"bufio"
	"io"
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
