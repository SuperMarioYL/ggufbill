// Package llamacpp synthesizes the exact `llama-server` command line that
// reproduces a manifest's deployment on a fresh box.
//
// m2 (inspect) wires this: `ggufbill inspect` reads a manifest, re-verifies
// the GGUF sha256, and prints the llama-server invocation produced here. The
// m1 build only emits manifests, so this file declares the contract and
// returns ErrNotImplemented; the synthesizer ships with m2.
package llamacpp

import (
	"errors"

	"github.com/SuperMarioYL/ggufbill/internal/manifest"
)

// ErrNotImplemented marks entry points that ship in a later milestone.
// Calling code should treat this as "build the m1 sidecar first; this reader
// lands in mN".
var ErrNotImplemented = errors.New("ggufbill: not implemented in this build (planned for m2: inspect)")

// Synthesize builds the exact `llama-server` invocation that reproduces the
// manifest's runtime configuration (context_length → -c, kv_cache_dtype →
// --kv-cache-dtype-q, gpu_layers → -ngl, extra_flags, and the model file).
//
// m1 returns ErrNotImplemented; m2 (inspect) provides the real synthesizer.
func Synthesize(m *manifest.Manifest) (string, error) {
	if m == nil {
		return "", errors.New("llamacpp: nil manifest")
	}
	return "", ErrNotImplemented
}
