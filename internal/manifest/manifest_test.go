package manifest

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSchema(t *testing.T) {
	s := Schema()
	if s == nil {
		t.Fatal("Schema() = nil")
	}
	if string(s.ID) != SchemaURL {
		t.Errorf("ID = %q, want %q", s.ID, SchemaURL)
	}
	if s.Title != "GGUF Deployment Manifest" {
		t.Errorf("Title = %q", s.Title)
	}
	// The 等保_level property must round-trip through the schema.
	out, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "\"等保_level\"") {
		t.Error("schema missing 等保_level property")
	}
	if !strings.Contains(string(out), "\"$schema\"") {
		t.Error("schema missing $schema property")
	}
}

func TestManifestRoundTrip(t *testing.T) {
	src := &Manifest{
		SchemaRef:       SchemaURL,
		ManifestVersion: Version,
		Model: Model{
			Name: "Qwen3-27B", Arch: "qwen2", Quant: "Q4_K_M",
			ParamsB: 27, File: "qwen3-27b-q4_k_m.gguf",
			SHA256: "8c68c1bac72d868cd5f2946b6c8f1afd6bcbde38bbc63fe6bc12c788f110ad0a",
		},
		Runtime: Runtime{
			Engine: "llama.cpp", EngineBuild: "b10181",
			ContextLength: 32768, KVCacheDType: "Q8_0", GPULayers: -1,
		},
		Hardware: Hardware{MinVRAMGB: 16, TargetGPUs: []string{"RTX-3060-12GB"}, GPUCount: 1},
		Quirks:   Quirks{ToolCalling: true, KnownBreakages: []string{"none"}},
		Audit: Audit{
			EmittedBy: "ggufbill 0.1.0", Operator: "ops",
			ProcurementRef: "ref-1", DengbaoLevel: "等保三级",
		},
	}
	src.Audit.EmittedAt = time.Now().UTC()
	if err := src.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	data, err := src.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "\"等保_level\": \"等保三级\"") {
		t.Errorf("JSON missing 等保_level:\n%s", data)
	}
	got, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.Model.SHA256 != src.Model.SHA256 {
		t.Errorf("round-trip sha256 = %q", got.Model.SHA256)
	}
	if got.Audit.DengbaoLevel != "等保三级" {
		t.Errorf("round-trip 等保_level = %q", got.Audit.DengbaoLevel)
	}
}

func TestValidateRejectsIncomplete(t *testing.T) {
	cases := []struct {
		name string
		m    *Manifest
	}{
		{"empty model", &Manifest{ManifestVersion: Version, Runtime: Runtime{Engine: "llama.cpp", ContextLength: 1}, Audit: Audit{Operator: "o"}}},
		{"no engine", &Manifest{ManifestVersion: Version, Model: Model{Name: "n", Arch: "a", Quant: "q", File: "f", SHA256: "s"}, Runtime: Runtime{ContextLength: 1}, Audit: Audit{Operator: "o"}}},
		{"no operator", &Manifest{ManifestVersion: Version, Model: Model{Name: "n", Arch: "a", Quant: "q", File: "f", SHA256: "s"}, Runtime: Runtime{Engine: "llama.cpp", ContextLength: 1}}},
	}
	for _, c := range cases {
		if err := c.m.Validate(); err == nil {
			t.Errorf("%s: expected validation error, got nil", c.name)
		}
	}
	// A nil manifest must error explicitly.
	if err := (*Manifest)(nil).Validate(); err == nil {
		t.Error("nil manifest should fail validation")
	} else if !strings.Contains(err.Error(), "nil") {
		t.Errorf("nil manifest error should mention nil, got %q", err.Error())
	}
}
