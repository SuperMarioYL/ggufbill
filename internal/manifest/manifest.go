// Package manifest defines the GGUF deployment manifest — a JSON sidecar
// (`.ggufbill.json`) that travels with a `.gguf` and declares everything an
// operator needs to reproduce a llama.cpp deployment and pass a 等保
// (classified-protection) audit: model identity, runtime config, hardware
// target, known quirks, and audit provenance.
//
// The manifest is the product's new primitive — an SBOM-analogue for model
// deployment, not a wrapper over llama.cpp CLI flags. The Go struct is the
// single source of truth; schemas/manifest.v0.1.json is its reflection via
// invopop/jsonschema.
package manifest

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/invopop/jsonschema"
)

// SchemaURL is the canonical location of the manifest JSON Schema. Every
// written sidecar carries it in its "$schema" field so any reader can locate
// the spec without guessing the format.
const SchemaURL = "https://ggufbill.org/v0.1/manifest.schema.json"

// Version is the manifest format version emitted by this package.
const Version = "0.1"

// Manifest is the top-level deployment-manifest object.
type Manifest struct {
	SchemaRef       string   `json:"$schema" jsonschema:"description=Canonical JSON Schema location for this manifest format"`
	ManifestVersion string   `json:"manifest_version" jsonschema:"description=Manifest format version"`
	Model           Model    `json:"model" jsonschema:"description=Model identity read from the GGUF header"`
	Runtime         Runtime  `json:"runtime" jsonschema:"description=llama.cpp runtime configuration the operator validated"`
	Hardware        Hardware `json:"hardware" jsonschema:"description=Target GPU hardware the deployment was tuned for"`
	Quirks          Quirks   `json:"quirks" jsonschema:"description=Known behavioral quirks of this quant on this config"`
	Audit           Audit    `json:"audit" jsonschema:"description=等保 / procurement audit provenance"`
}

// Model holds the identity fields read directly from the GGUF header.
type Model struct {
	Name    string  `json:"name" jsonschema:"description=Model name from general.name"`
	Arch    string  `json:"arch" jsonschema:"description=Model architecture from general.architecture"`
	Quant   string  `json:"quant" jsonschema:"description=Quantization variant e.g. Q4_K_M"`
	ParamsB float64 `json:"params_b" jsonschema:"description=Parameter count in billions"`
	File    string  `json:"file" jsonschema:"description=Portable GGUF filename basename without directory"`
	SHA256  string  `json:"sha256" jsonschema:"description=SHA-256 of the whole .gguf file"`
}

// Runtime holds the llama.cpp serving configuration the operator validated.
type Runtime struct {
	Engine        string   `json:"engine" jsonschema:"description=Inference engine, e.g. llama.cpp"`
	EngineBuild   string   `json:"engine_build" jsonschema:"description=Engine build identifier e.g. b10181"`
	ContextLength int64    `json:"context_length" jsonschema:"description=Serving context length in tokens"`
	KVCacheDType  string   `json:"kv_cache_dtype" jsonschema:"description=KV cache dtype e.g. Q8_0 or f16"`
	GPULayers     int      `json:"gpu_layers" jsonschema:"description=Layers offloaded to GPU (-1 = all)"`
	ExtraFlags    []string `json:"extra_flags,omitempty" jsonschema:"description=Additional validated engine flags"`
}

// Hardware holds the target GPU hardware the deployment was tuned for.
type Hardware struct {
	MinVRAMGB  int      `json:"min_vram_gb" jsonschema:"description=Minimum VRAM per GPU in GB"`
	TargetGPUs []string `json:"target_gpus" jsonschema:"description=GPU models the config was validated on"`
	GPUCount   int      `json:"gpu_count" jsonschema:"description=Number of GPUs in the deployment"`
	Notes      string   `json:"notes,omitempty" jsonschema:"description=Free-form hardware notes"`
}

// Quirks captures known behavioral quirks of this quant on this config.
type Quirks struct {
	ToolCalling     bool     `json:"tool_calling" jsonschema:"description=Whether tool calling is reliable on this config"`
	KnownBreakages  []string `json:"known_breakages,omitempty" jsonschema:"description=Known failure modes on this quant/config"`
}

// Audit holds the 等保 / procurement audit provenance for the deployment.
type Audit struct {
	EmittedAt       time.Time `json:"emitted_at" jsonschema:"description=When the manifest was emitted (RFC3339)"`
	EmittedBy       string    `json:"emitted_by" jsonschema:"description=ggufbill version that emitted the manifest"`
	Operator        string    `json:"operator" jsonschema:"description=Operator accountable for this deployment"`
	ProcurementRef  string    `json:"procurement_ref,omitempty" jsonschema:"description=Procurement / asset reference for sign-off"`
	DengbaoLevel    string    `json:"等保_level,omitempty" jsonschema:"description=等保 (classified protection) level claimed for this deployment"`
}

// Schema returns the JSON Schema for the manifest, generated from the Go
// structs via invopop/jsonschema. It is the single source of truth for the
// committed schemas/manifest.v0.1.json and for runtime validation (m2).
func Schema() *jsonschema.Schema {
	r := new(jsonschema.Reflector)
	r.DoNotReference = true
	s := r.Reflect(&Manifest{})
	s.ID = jsonschema.ID(SchemaURL)
	s.Title = "GGUF Deployment Manifest"
	s.Description = "A portable sidecar (.ggufbill.json) that travels with a .gguf and declares the llama.cpp runtime, hardware target, known quirks, and 等保 audit provenance needed to reproduce and audit a deployment."
	return s
}

// JSON encodes the manifest as indented JSON with a trailing newline.
func (m *Manifest) JSON() ([]byte, error) {
	if m == nil {
		return nil, errors.New("manifest: nil")
	}
	if m.SchemaRef == "" {
		m.SchemaRef = SchemaURL
	}
	if m.ManifestVersion == "" {
		m.ManifestVersion = Version
	}
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// Parse decodes manifest JSON. It is the inverse of JSON.
func Parse(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// Validate performs a structural check that the manifest carries the fields a
// reader or auditor depends on. Full JSON-Schema validation against Schema()
// and GGUF sha256 re-verification land in m2 (inspect); for m1 this guards
// emit against writing an obviously broken sidecar.
func (m *Manifest) Validate() error {
	if m == nil {
		return errors.New("manifest: nil")
	}
	if m.ManifestVersion == "" {
		return errors.New("manifest: manifest_version is required")
	}
	if m.Model.Name == "" || m.Model.Arch == "" || m.Model.Quant == "" ||
		m.Model.File == "" || m.Model.SHA256 == "" {
		return errors.New("manifest: model block incomplete (name, arch, quant, file, sha256 all required)")
	}
	if m.Runtime.Engine == "" {
		return errors.New("manifest: runtime.engine is required")
	}
	if m.Runtime.ContextLength <= 0 {
		return errors.New("manifest: runtime.context_length must be positive")
	}
	if m.Audit.Operator == "" {
		return errors.New("manifest: audit.operator is required")
	}
	if m.Audit.EmittedAt.IsZero() {
		return errors.New("manifest: audit.emitted_at is required")
	}
	return nil
}
