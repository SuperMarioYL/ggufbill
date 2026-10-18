package audit

import (
	"strings"
	"testing"
	"time"

	"github.com/SuperMarioYL/ggufbill/internal/manifest"
)

func testManifest(name, operator, procRef, dengbao string) *manifest.Manifest {
	return &manifest.Manifest{
		SchemaRef:       manifest.SchemaURL,
		ManifestVersion: manifest.Version,
		Model: manifest.Model{
			Name: name, Arch: "qwen2", Quant: "Q4_K_M", ParamsB: 27,
			File: name + "-Q4_K_M.gguf", SHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
		Runtime: manifest.Runtime{
			Engine: "llama.cpp", EngineBuild: "b10181", ContextLength: 73728,
			KVCacheDType: "Q8_0", GPULayers: -1,
		},
		Hardware: manifest.Hardware{
			MinVRAMGB: 12, TargetGPUs: []string{"RTX-3060-12GB"}, GPUCount: 4,
		},
		Audit: manifest.Audit{
			EmittedAt:      time.Date(2026, 10, 19, 8, 0, 0, 0, time.UTC),
			EmittedBy:      "ggufbill 0.3.0",
			Operator:       operator,
			ProcurementRef: procRef,
			DengbaoLevel:   dengbao,
		},
	}
}

func TestRenderMarkdown(t *testing.T) {
	ms := []*manifest.Manifest{
		testManifest("Qwen3-27B", "ops-a", "CG-2026-001", "三级"),
		testManifest("DeepSeek-V4-Flash", "ops-b", "", ""),
	}
	out, err := Render(ms, FormatMarkdown)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	s := string(out)
	for _, want := range []string{
		"# GGUF 部署审计报告",
		"| 1 | Qwen3-27B-Q4_K_M.gguf | Qwen3-27B | Q4_K_M | 27.0 | 73728 | Q8_0 | -1 |",
		"min 12 GB/GPU; 4× RTX-3060-12GB",
		"ops-a",
		"CG-2026-001",
		"三级",
		// Dashes for absent optional audit fields.
		"| 2 | DeepSeek-V4-Flash-Q4_K_M.gguf | DeepSeek-V4-Flash | Q4_K_M | 27.0 | 73728 | Q8_0 | -1 |",
		"## 清单指纹",
		"sha256 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"engine llama.cpp b10181",
		"emitted by ggufbill 0.3.0",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("markdown missing %q\ngot:\n%s", want, s)
		}
	}
}

// TestRenderHTMLEscapesOperatorInput guards the HTML view: operator-supplied
// strings are free-form and must be escaped, not injected.
func TestRenderHTMLEscapesOperatorInput(t *testing.T) {
	ms := []*manifest.Manifest{testManifest("Qwen3-27B", `ops & <co>`, "CG-1", "三级")}
	out, err := Render(ms, FormatHTML)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	s := string(out)
	for _, want := range []string{
		"<!doctype html>",
		"<table>",
		"ops &amp; &lt;co&gt;",
		"CG-1",
		"三级",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("html missing %q\ngot:\n%s", want, s)
		}
	}
	if strings.Contains(s, "<script>") {
		t.Errorf("html contains unescaped input:\n%s", s)
	}
}

func TestRenderRejectsBadInput(t *testing.T) {
	if _, err := Render(nil, FormatMarkdown); err == nil {
		t.Error("nil list: expected error, got nil")
	}
	if _, err := Render([]*manifest.Manifest{nil}, FormatMarkdown); err == nil {
		t.Error("nil manifest: expected error, got nil")
	}
	if _, err := Render([]*manifest.Manifest{testManifest("m", "o", "", "")}, "yaml"); err == nil {
		t.Error("unknown format: expected error, got nil")
	}
}
