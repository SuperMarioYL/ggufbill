// Package audit renders a 等保 (classified-protection) deployment-audit view
// from one or more manifests: a per-box table of quant, context, hardware,
// operator, procurement_ref, and 等保_level, suitable for dropping into a
// 测评报告 addendum.
//
// `ggufbill audit *.ggufbill.json > 等保-部署审计.md` wires this. The report
// renders only what the manifests declare — it audits the recorded inputs and
// provenance, not whether a deployment actually runs or complies.
package audit

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"

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

// Render renders a 等保 audit view of the given manifests in the chosen
// format. Manifests appear in the order given; every entry must be non-nil
// and already validated by the caller.
func Render(ms []*manifest.Manifest, format Format) ([]byte, error) {
	if len(ms) == 0 {
		return nil, fmt.Errorf("audit: no manifests provided")
	}
	for i, m := range ms {
		if m == nil {
			return nil, fmt.Errorf("audit: manifest %d is nil", i)
		}
	}
	switch format {
	case FormatMarkdown:
		return renderMarkdown(ms)
	case FormatHTML:
		return renderHTML(ms)
	default:
		return nil, fmt.Errorf("audit: unknown format %q (markdown | html)", string(format))
	}
}

// row is the per-box line shared by the markdown and HTML tables.
type row struct {
	Index          int
	File           string
	Model          string
	Quant          string
	Params         string
	Context        int64
	KV             string
	GPULayers      int
	Hardware       string
	Operator       string
	ProcurementRef string
	DengbaoLevel   string
	EmittedAt      string
}

func buildRows(ms []*manifest.Manifest) []row {
	rows := make([]row, 0, len(ms))
	for i, m := range ms {
		rows = append(rows, row{
			Index:          i + 1,
			File:           m.Model.File,
			Model:          m.Model.Name,
			Quant:          m.Model.Quant,
			Params:         fmt.Sprintf("%.1f", m.Model.ParamsB),
			Context:        m.Runtime.ContextLength,
			KV:             m.Runtime.KVCacheDType,
			GPULayers:      m.Runtime.GPULayers,
			Hardware:       hardwareSummary(&m.Hardware),
			Operator:       m.Audit.Operator,
			ProcurementRef: dashIfEmpty(m.Audit.ProcurementRef),
			DengbaoLevel:   dashIfEmpty(m.Audit.DengbaoLevel),
			EmittedAt:      m.Audit.EmittedAt.UTC().Format(time.RFC3339),
		})
	}
	return rows
}

// fingerprints lists the per-manifest identity lines under the table: what
// file, which bytes, which engine build, and which tool recorded it.
func fingerprints(ms []*manifest.Manifest) []string {
	lines := make([]string, 0, len(ms))
	for _, m := range ms {
		lines = append(lines, fmt.Sprintf("%s — sha256 %s, engine %s %s, emitted by %s",
			m.Model.File, m.Model.SHA256, m.Runtime.Engine, dashIfEmpty(m.Runtime.EngineBuild), m.Audit.EmittedBy))
	}
	return lines
}

// hardwareSummary condenses the hardware block into one cell.
func hardwareSummary(h *manifest.Hardware) string {
	var parts []string
	if h.MinVRAMGB > 0 {
		parts = append(parts, fmt.Sprintf("min %d GB/GPU", h.MinVRAMGB))
	}
	if len(h.TargetGPUs) > 0 {
		count := h.GPUCount
		if count <= 0 {
			count = len(h.TargetGPUs)
		}
		parts = append(parts, fmt.Sprintf("%d× %s", count, strings.Join(h.TargetGPUs, "/")))
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, "; ")
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

const mdHeader = `# GGUF 部署审计报告

等保部署审计视图,由 ggufbill audit 从随包清单生成;生成时间 %s。记录的是清单声明与来源指纹,不构成部署可用性或合规性结论。

`

func renderMarkdown(ms []*manifest.Manifest) ([]byte, error) {
	var b strings.Builder
	fmt.Fprintf(&b, mdHeader, time.Now().UTC().Format(time.RFC3339))
	b.WriteString("| # | 清单 | 模型 | 量化 | 参数(B) | 上下文 | KV 类型 | GPU 层 | 硬件 | 操作者 | 采购参考 | 等保 | 记录时间 |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, r := range buildRows(ms) {
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %s | %d | %s | %d | %s | %s | %s | %s | %s |\n",
			r.Index, r.File, r.Model, r.Quant, r.Params, r.Context, r.KV,
			r.GPULayers, r.Hardware, r.Operator, r.ProcurementRef, r.DengbaoLevel, r.EmittedAt)
	}
	b.WriteString("\n## 清单指纹\n\n")
	for _, line := range fingerprints(ms) {
		fmt.Fprintf(&b, "- %s\n", line)
	}
	return []byte(b.String()), nil
}

var htmlTmpl = template.Must(template.New("audit").Parse(`<!doctype html>
<html lang="zh">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>GGUF 部署审计报告</title>
<style>
body { font-family: -apple-system, "PingFang SC", "Microsoft YaHei", sans-serif; margin: 2rem; color: #1a1a1a; }
table { border-collapse: collapse; width: 100%; font-size: 14px; }
th, td { border: 1px solid #ccc; padding: 6px 10px; text-align: left; }
th { background: #f4f4f4; }
</style>
</head>
<body>
<h1>GGUF 部署审计报告</h1>
<p>等保部署审计视图,由 ggufbill audit 从随包清单生成;生成时间 {{.GeneratedAt}}。记录的是清单声明与来源指纹,不构成部署可用性或合规性结论。</p>
<table>
<tr><th>#</th><th>清单</th><th>模型</th><th>量化</th><th>参数(B)</th><th>上下文</th><th>KV 类型</th><th>GPU 层</th><th>硬件</th><th>操作者</th><th>采购参考</th><th>等保</th><th>记录时间</th></tr>
{{range .Rows}}
<tr><td>{{.Index}}</td><td>{{.File}}</td><td>{{.Model}}</td><td>{{.Quant}}</td><td>{{.Params}}</td><td>{{.Context}}</td><td>{{.KV}}</td><td>{{.GPULayers}}</td><td>{{.Hardware}}</td><td>{{.Operator}}</td><td>{{.ProcurementRef}}</td><td>{{.DengbaoLevel}}</td><td>{{.EmittedAt}}</td></tr>
{{end}}
</table>
<h2>清单指纹</h2>
<ul>
{{range .Prints}}<li>{{.}}</li>
{{end}}
</ul>
</body>
</html>
`))

func renderHTML(ms []*manifest.Manifest) ([]byte, error) {
	var buf bytes.Buffer
	data := struct {
		GeneratedAt string
		Rows        []row
		Prints      []string
	}{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Rows:        buildRows(ms),
		Prints:      fingerprints(ms),
	}
	if err := htmlTmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("audit: render html: %w", err)
	}
	return buf.Bytes(), nil
}
