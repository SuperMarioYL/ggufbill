// Package audit renders a 等保 (classified-protection) deployment-audit view
// from one or more manifests: a per-box table of quant, context, hardware,
// operator, procurement_ref, and 等保_level, suitable for dropping into a
// 测评报告 addendum.
//
// m3 (audit) wires this: `ggufbill audit *.ggufbill.json > 等保-部署审计.md`.
// The m1 build only emits manifests, so this file declares the contract and
// returns ErrNotImplemented; the template ships with m3.
package audit

import (
	"errors"

	"github.com/SuperMarioYL/ggufbill/internal/manifest"
)

// Format selects the audit-view output format.
type Format string

const (
	// FormatMarkdown renders a printable 等保-部署审计.md addendum.
	FormatMarkdown Format = "markdown"
	// FormatHTML renders a standalone HTML audit page.
	FormatHTML Format = "html"
)

// ErrNotImplemented marks entry points that ship in a later milestone.
var ErrNotImplemented = errors.New("ggufbill: not implemented in this build (planned for m3: audit)")

// Render renders a 等保 audit view of the given manifests in the chosen
// format. m1 returns ErrNotImplemented; m3 (audit) provides the real
// template (per-box quant/context/hardware/operator/procurement_ref/等保_level).
func Render(ms []*manifest.Manifest, format Format) ([]byte, error) {
	if len(ms) == 0 {
		return nil, errors.New("audit: no manifests provided")
	}
	return nil, ErrNotImplemented
}
