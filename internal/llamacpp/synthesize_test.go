package llamacpp

import (
	"strings"
	"testing"

	"github.com/SuperMarioYL/ggufbill/internal/manifest"
)

func testManifest() *manifest.Manifest {
	return &manifest.Manifest{
		Model: manifest.Model{
			Name: "Qwen3-27B", Arch: "qwen2", Quant: "Q4_K_M", ParamsB: 27,
			File: "Qwen3-27B-Q4_K_M.gguf", SHA256: "0123456789abcdef",
		},
		Runtime: manifest.Runtime{
			Engine: "llama.cpp", EngineBuild: "b10181", ContextLength: 4096,
			KVCacheDType: "Q8_0", GPULayers: 99, ExtraFlags: []string{"--flash-attn"},
		},
	}
}

func TestSynthesize(t *testing.T) {
	cmd, err := Synthesize(testManifest(), "/models/Qwen3-27B-Q4_K_M.gguf")
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	want := "llama-server -m /models/Qwen3-27B-Q4_K_M.gguf -c 4096 -ngl 99 -ctk Q8_0 -ctv Q8_0 --flash-attn"
	if cmd != want {
		t.Errorf("cmd = %q, want %q", cmd, want)
	}
}

// TestSynthesizeF16OmitsKVFlags keeps the reproduced command minimal: f16 is
// the llama.cpp KV cache default, so no -ctk/-ctv flags are emitted for it.
func TestSynthesizeF16OmitsKVFlags(t *testing.T) {
	m := testManifest()
	m.Runtime.KVCacheDType = "f16"
	cmd, err := Synthesize(m, "/models/m.gguf")
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	if strings.Contains(cmd, "-ctk") || strings.Contains(cmd, "-ctv") {
		t.Errorf("cmd = %q, want no -ctk/-ctv for f16", cmd)
	}
}

// TestSynthesizeQuotesSpacedModelPath keeps the invocation exact when the
// model path contains whitespace.
func TestSynthesizeQuotesSpacedModelPath(t *testing.T) {
	cmd, err := Synthesize(testManifest(), "/models/My Models/m.gguf")
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	if want := `-m "/models/My Models/m.gguf"`; !strings.Contains(cmd, want) {
		t.Errorf("cmd = %q, want the model path quoted as %q", cmd, want)
	}
}

func TestSynthesizeRejectsNilManifestAndEmptyPath(t *testing.T) {
	if _, err := Synthesize(nil, "/models/m.gguf"); err == nil {
		t.Error("nil manifest: expected error, got nil")
	}
	if _, err := Synthesize(testManifest(), ""); err == nil {
		t.Error("empty model path: expected error, got nil")
	}
}
