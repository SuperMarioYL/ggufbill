package main

import (
	"flag"
	"fmt"
	"os"
)

// runAudit is the entry point for `ggufbill audit <manifest>…`.
//
// m1 build: not yet implemented. The full behavior ships in milestone m3
// (audit report): emit a 等保 audit view (markdown + HTML) with a per-box
// table of quant/context/hardware/operator/procurement_ref/等保_level, ready
// to drop into a 测评报告 addendum. The m1 build only emits manifests; the
// audit template lands with m3.
func runAudit(args []string) {
	fs := flag.NewFlagSet("audit", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var format string
	fs.StringVar(&format, "format", "markdown", "audit view format: markdown | html")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: ggufbill audit [--format markdown|html] <manifest>…")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "  (m3) emit a 等保 deployment audit view (markdown / HTML)")
		fmt.Fprintln(os.Stderr, "  with per-box quant/context/hardware/operator/procurement_ref/等保_level.")
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	fmt.Println("ggufbill audit is not implemented in this build (m1 = emit only).")
	fmt.Println()
	fmt.Println("Planned for milestone m3 (audit report):")
	fmt.Println("  - read one or more .ggufbill.json manifests")
	fmt.Println("  - emit a 等保 deployment audit view (markdown + HTML)")
	fmt.Println("  - per-box: quant / context / hardware / operator / procurement_ref / 等保_level")
	fmt.Println("  - drops into a 测评报告 addendum for procurement sign-off")
	fmt.Println()
	fmt.Println("See the roadmap in README.md for the release plan.")
}
