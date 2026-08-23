package gguf

import (
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

func TestBaseName(t *testing.T) {
	if got := BaseName("/tmp/models/Qwen3-27B-Q4_K_M.gguf"); got != "Qwen3-27B-Q4_K_M.gguf" {
		t.Errorf("BaseName = %q", got)
	}
}
