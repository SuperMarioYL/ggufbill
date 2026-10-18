package gguf

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFixture(t *testing.T) {
	h, err := Read("testdata/qwen3-27b-q4_k_m.gguf")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if h.Architecture != "qwen2" {
		t.Errorf("Architecture = %q, want qwen2", h.Architecture)
	}
	if h.Name != "Qwen3-27B" {
		t.Errorf("Name = %q, want Qwen3-27B", h.Name)
	}
	if h.Quant != "Q4_K_M" {
		t.Errorf("Quant = %q, want Q4_K_M", h.Quant)
	}
	if h.ParamsB != 27 {
		t.Errorf("ParamsB = %v, want 27", h.ParamsB)
	}
	if h.Version != 3 {
		t.Errorf("Version = %d, want 3", h.Version)
	}
	if len(h.SHA256) != 64 {
		t.Errorf("SHA256 = %q, want 64 hex chars", h.SHA256)
	}
	if h.Size == 0 {
		t.Error("Size = 0, want non-zero")
	}
}

func TestFtypeName(t *testing.T) {
	cases := map[uint32]string{
		0:   "F32",
		1:   "F16",
		7:   "Q8_0",
		15:  "Q4_K_M",
		18:  "Q6_K",
		31:  "BF16",
		999: "FTYPE_999",
	}
	for ft, want := range cases {
		if got := ftypeName(ft); got != want {
			t.Errorf("ftypeName(%d) = %q, want %q", ft, got, want)
		}
	}
}

func TestReadNotGGUF(t *testing.T) {
	if _, err := Read("reader.go"); err == nil {
		t.Fatal("expected error for non-GGUF file, got nil")
	}
}

func TestReadMissing(t *testing.T) {
	if _, err := Read("testdata/does-not-exist.gguf"); err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// TestReadParamsFallback covers GGUFs that predate general.n_params: the
// parameter count must fall back to summing tensor nelements, which requires
// walking the KV block once and reading each ne dimension as uint64 (GGUF
// v2/v3 width). Regression test for the v0.1.0 misparse — a doubled kv_count
// read plus 32-bit dimension fields — which silently degraded params_b to 0.
func TestReadParamsFallback(t *testing.T) {
	var buf bytes.Buffer
	w := fixtureWriter{w: &buf}
	w.bytes("GGUF")
	w.u32(3) // version
	w.u64(1) // tensor_count
	w.u64(2) // kv_count
	w.str("general.architecture")
	w.u32(typString)
	w.str("qwen2")
	w.str("general.name")
	w.u32(typString)
	w.str("LegacyModel")
	// One tensor with 5e9 elements: the first dim exceeds uint32 range, so a
	// 32-bit dim read misaligns the rest of the record.
	w.tensor("token_embd.weight", 5_000_000_000)
	if w.err != nil {
		t.Fatalf("build fixture: %v", w.err)
	}

	path := filepath.Join(t.TempDir(), "legacy.gguf")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	h, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if h.Architecture != "qwen2" || h.Name != "LegacyModel" {
		t.Errorf("header = %q/%q, want qwen2/LegacyModel", h.Architecture, h.Name)
	}
	if h.ParamsB != 5 {
		t.Errorf("ParamsB = %g, want 5 (5e9 summed tensor elements)", h.ParamsB)
	}
}

func TestBaseName(t *testing.T) {
	if got := BaseName("/tmp/models/Qwen3-27B-Q4_K_M.gguf"); got != "Qwen3-27B-Q4_K_M.gguf" {
		t.Errorf("BaseName = %q", got)
	}
}

// TestReadQuantUnknownWhenUnlabeled guards the v0.2.0 defect where a GGUF
// carrying neither general.quantization_name nor general.file_type resolved
// its quant by falling through to ftypeName(0) — fabricating "F32" for a file
// whose quant is simply unlabeled. "unknown" is the honest value.
func TestReadQuantUnknownWhenUnlabeled(t *testing.T) {
	var buf bytes.Buffer
	w := fixtureWriter{w: &buf}
	w.bytes("GGUF")
	w.u32(3)
	w.u64(0) // tensor_count
	w.u64(1) // kv_count
	w.str("general.name")
	w.u32(typString)
	w.str("BareModel")
	if w.err != nil {
		t.Fatalf("build fixture: %v", w.err)
	}

	path := filepath.Join(t.TempDir(), "bare.gguf")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	h, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if h.Quant != "unknown" {
		t.Errorf("Quant = %q, want unknown (no quant metadata to resolve from)", h.Quant)
	}
}

// TestReadV1Rejected guards the v0.2.0 defect where the version gate accepted
// GGUF v1 but every subsequent read assumed the v2/v3 layout (u64 counts,
// u64-length strings), so a v1 file died with a confusing mid-block misparse
// error instead of a clean unsupported-version rejection.
func TestReadV1Rejected(t *testing.T) {
	var buf bytes.Buffer
	w := fixtureWriter{w: &buf}
	w.bytes("GGUF")
	w.u32(1) // version 1 — v1 layout: u32 counts, u32-length strings
	w.u32(0) // tensor_count
	w.u32(1) // kv_count
	w.u32(9)
	w.bytes("general.n") // first (u32-length) KV key, truncated arbitrarily
	if w.err != nil {
		t.Fatalf("build fixture: %v", w.err)
	}

	path := filepath.Join(t.TempDir(), "v1.gguf")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	h, err := Read(path)
	if err == nil {
		t.Fatalf("Read = %+v, want an unsupported-version error", h)
	}
	if !strings.Contains(err.Error(), "unsupported GGUF version") {
		t.Errorf("err = %v, want a clean unsupported-version rejection", err)
	}
}
