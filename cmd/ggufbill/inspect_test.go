package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runInspectCapture swaps stdout/stderr with pipes, runs runInspect, and
// returns what it printed. The globals are restored before returning.
func runInspectCapture(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW
	defer func() {
		os.Stdout, os.Stderr = oldOut, oldErr
	}()
	runErr := runInspect(args)
	outW.Close()
	errW.Close()
	outBytes, _ := io.ReadAll(outR)
	errBytes, _ := io.ReadAll(errR)
	return string(outBytes), string(errBytes), runErr
}

// copyFixtureTo places a copy of the committed GGUF fixture at dir/name, so a
// manifest emitted elsewhere can find "the model next to the sidecar".
func copyFixtureTo(t *testing.T, dir, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "internal", "gguf", "testdata", "qwen3-27b-q4_k_m.gguf"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestInspectEndToEnd drives the m2 happy path: emit a sidecar from the
// committed fixture, place the model next to it, inspect — the invocation is
// printed and the digest verifies.
func TestInspectEndToEnd(t *testing.T) {
	dir := t.TempDir()
	model := copyFixtureTo(t, dir, "qwen3-27b-q4_k_m.gguf")
	sidecar := filepath.Join(dir, "demo.ggufbill.json")
	if err := runEmit([]string{model, "--context", "4096", "--kv", "f16", "--operator", "demo-operator",
		"--engine-build", "b10181", "--gpu", "RTX-3060-12GB", "--vram", "12",
		"--procurement-ref", "CG-2026-001", "--dengbao-level", "三级", "--non-interactive",
		"--output", sidecar}); err != nil {
		t.Fatalf("emit: %v", err)
	}

	out, _, err := runInspectCapture(t, sidecar)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	for _, want := range []string{
		"llama-server -m " + model + " -c 4096 -ngl -1",
		"deployment summary",
		"Qwen3-27B (qwen2, Q4_K_M, 27.0B)",
		"verified",
		"operator demo-operator  ref CG-2026-001  等保 三级",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("inspect output missing %q\ngot:\n%s", want, out)
		}
	}
}

// TestInspectSha256Mismatch is the tamper case: the model file next to the
// sidecar carries different bytes than the manifest recorded — inspect must
// fail loudly instead of printing a reproduced command for wrong weights.
func TestInspectSha256Mismatch(t *testing.T) {
	dir := t.TempDir()
	model := copyFixtureTo(t, dir, "qwen3-27b-q4_k_m.gguf")
	sidecar := filepath.Join(dir, "demo.ggufbill.json")
	if err := runEmit([]string{model, "--context", "4096", "--kv", "f16", "--operator", "demo-operator",
		"--non-interactive", "--output", sidecar}); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if err := os.WriteFile(model, []byte("tampered bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := runInspectCapture(t, sidecar)
	if err == nil || !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Errorf("err = %v, want a sha256 mismatch failure", err)
	}
}

// TestInspectMissingModelDegradesToWarning keeps the audit-prep path open: a
// manifest without its weights still prints the invocation, marked unverified.
func TestInspectMissingModelDegradesToWarning(t *testing.T) {
	dir := t.TempDir()
	model := copyFixtureTo(t, dir, "qwen3-27b-q4_k_m.gguf")
	sidecar := filepath.Join(dir, "demo.ggufbill.json")
	if err := runEmit([]string{model, "--context", "4096", "--kv", "f16", "--operator", "demo-operator",
		"--non-interactive", "--output", sidecar}); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if err := os.Remove(model); err != nil {
		t.Fatal(err)
	}
	out, errOut, err := runInspectCapture(t, sidecar)
	if err != nil {
		t.Fatalf("inspect with missing model: %v", err)
	}
	if !strings.Contains(errOut, "sha256 unverified") {
		t.Errorf("stderr = %q, want a sha256 unverified warning", errOut)
	}
	if !strings.Contains(out, "unverified") || strings.Contains(out, "  verified") {
		t.Errorf("summary should mark the digest unverified, got:\n%s", out)
	}
}

func TestInspectRejectsBadManifests(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := runInspectCapture(t, filepath.Join(dir, "absent.ggufbill.json")); err == nil {
		t.Error("missing manifest: expected error, got nil")
	}
	broken := filepath.Join(dir, "broken.ggufbill.json")
	if err := os.WriteFile(broken, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runInspectCapture(t, broken); err == nil {
		t.Error("unparseable manifest: expected error, got nil")
	}
	if _, _, err := runInspectCapture(t); err == nil {
		t.Error("no manifest path: expected error, got nil")
	}
}
