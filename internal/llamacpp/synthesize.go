// Package llamacpp synthesizes the exact `llama-server` command line that
// reproduces a manifest's deployment on a fresh box.
//
// `ggufbill inspect` wires this: it reads a manifest, re-verifies the GGUF
// sha256 against the model file next to the sidecar, and prints the command
// produced here plus a deployment summary.
package llamacpp

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/SuperMarioYL/ggufbill/internal/manifest"
)

// Synthesize builds the exact `llama-server` invocation that reproduces the
// manifest's runtime configuration:
//
//	model.file       -> -m (modelPath: the path verified on disk by inspect,
//	                    quoted when it contains whitespace)
//	context_length   -> -c
//	gpu_layers       -> -ngl (-1 = offload all layers)
//	kv_cache_dtype   -> -ctk <dtype> -ctv <dtype>, omitted for f16 (the
//	                    llama.cpp default) so the reproduced command stays
//	                    minimal
//	extra_flags      -> appended verbatim (operator-validated at emit time)
func Synthesize(m *manifest.Manifest, modelPath string) (string, error) {
	if m == nil {
		return "", errors.New("llamacpp: nil manifest")
	}
	if modelPath == "" {
		return "", errors.New("llamacpp: model path is required")
	}
	if strings.ContainsAny(modelPath, " \t") {
		modelPath = strconv.Quote(modelPath)
	}
	var b strings.Builder
	b.WriteString("llama-server")
	fmt.Fprintf(&b, " -m %s", modelPath)
	fmt.Fprintf(&b, " -c %d", m.Runtime.ContextLength)
	fmt.Fprintf(&b, " -ngl %d", m.Runtime.GPULayers)
	if dt := m.Runtime.KVCacheDType; dt != "" && !strings.EqualFold(dt, "f16") {
		fmt.Fprintf(&b, " -ctk %s -ctv %s", dt, dt)
	}
	for _, f := range m.Runtime.ExtraFlags {
		b.WriteString(" ")
		b.WriteString(f)
	}
	return b.String(), nil
}
