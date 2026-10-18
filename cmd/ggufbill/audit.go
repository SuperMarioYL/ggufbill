package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/SuperMarioYL/ggufbill/internal/audit"
	"github.com/SuperMarioYL/ggufbill/internal/manifest"
)

// runAudit implements `ggufbill audit [--format markdown|html] <manifest>…`:
// it validates every manifest and renders the 等保 deployment-audit view (the
// m3 milestone) to stdout, so `audit *.ggufbill.json > 等保-部署审计.md`
// works. Any invalid manifest fails the run with a non-zero exit.
func runAudit(args []string) error {
	fs := flag.NewFlagSet("audit", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var format string
	fs.StringVar(&format, "format", "markdown", "audit view format: markdown | html")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: ggufbill audit [--format markdown|html] <manifest>…")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Emits a 等保 deployment audit view (markdown / HTML) with a")
		fmt.Fprintln(os.Stderr, "per-box table of quant/context/hardware/operator/")
		fmt.Fprintln(os.Stderr, "procurement_ref/等保_level to stdout.")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	paths := fs.Args()
	if len(paths) == 0 {
		fs.Usage()
		return fmt.Errorf("audit: at least one manifest path is required")
	}

	var ms []*manifest.Manifest
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("audit: %w", err)
		}
		m, err := manifest.Parse(data)
		if err != nil {
			return fmt.Errorf("audit: parse %s: %w", p, err)
		}
		if err := m.Validate(); err != nil {
			return fmt.Errorf("audit: %s: %w", p, err)
		}
		ms = append(ms, m)
	}

	out, err := audit.Render(ms, audit.Format(format))
	if err != nil {
		return fmt.Errorf("audit: %w", err)
	}
	if _, err := os.Stdout.Write(out); err != nil {
		return fmt.Errorf("audit: write report: %w", err)
	}
	return nil
}
