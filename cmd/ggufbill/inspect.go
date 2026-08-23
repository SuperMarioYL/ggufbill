package main

import (
	"flag"
	"fmt"
	"os"
)

// runInspect is the entry point for `ggufbill inspect <manifest>`.
//
// m1 build: not yet implemented. The full behavior ships in milestone m2
// (inspect/reproduce): schema-validate the sidecar against
// schemas/manifest.v0.1.json, re-verify the GGUF sha256, and print the exact
// `llama-server` invocation plus a human-readable deployment summary. The m1
// build only emits manifests; the reader/validator lands with m2.
func runInspect(args []string) {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: ggufbill inspect <manifest.ggufbill.json>")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "  (m2) schema-validate the sidecar, re-verify the GGUF sha256,")
		fmt.Fprintln(os.Stderr, "  and print the exact llama-server invocation + deployment summary.")
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	fmt.Println("ggufbill inspect is not implemented in this build (m1 = emit only).")
	fmt.Println()
	fmt.Println("Planned for milestone m2 (inspect/reproduce):")
	fmt.Println("  - schema-validate the sidecar against schemas/manifest.v0.1.json")
	fmt.Println("  - re-verify the GGUF sha256 against the model file on disk")
	fmt.Println("  - print the exact `llama-server` invocation + a deployment summary")
	fmt.Println()
	fmt.Println("See the roadmap in README.md for the release plan.")
}
