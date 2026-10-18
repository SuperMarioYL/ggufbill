// Command ggufbill emits and inspects portable GGUF deployment manifests — a
// .ggufbill.json sidecar that travels with a .gguf and makes a llama.cpp
// deployment reproducible and 等保-auditable across boxes.
//
// Usage:
//
//	ggufbill emit   <file.gguf>        # write a .ggufbill.json sidecar (m1)
//	ggufbill inspect <manifest>        # (m2) validate + reproduce llama-server
//	ggufbill audit   <manifest>…       # (m3) 等保 audit view
//	ggufbill version
package main

import (
	"fmt"
	"os"
)

// version is the ggufbill build version. It defaults to the repo VERSION file
// value; the release workflow injects the tagged version via -ldflags
// -X main.version. Keep the default in sync with the VERSION file —
// TestVersionLockstep fails if they drift.
var version = "0.3.0"

func main() {
	args := os.Args
	if len(args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd, rest := args[1], args[2:]
	switch cmd {
	case "emit":
		if err := runEmit(rest); err != nil {
			fmt.Fprintf(os.Stderr, "ggufbill: %v\n", err)
			os.Exit(1)
		}
	case "inspect":
		if err := runInspect(rest); err != nil {
			fmt.Fprintf(os.Stderr, "ggufbill: %v\n", err)
			os.Exit(1)
		}
	case "audit":
		if err := runAudit(rest); err != nil {
			fmt.Fprintf(os.Stderr, "ggufbill: %v\n", err)
			os.Exit(1)
		}
	case "version", "-v", "--version":
		fmt.Printf("ggufbill %s\n", version)
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "ggufbill: unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `ggufbill — GGUF deployment-manifest sidecar

Usage:
  ggufbill emit   <file.gguf>  [flags]   write a .ggufbill.json sidecar (m1)
  ggufbill inspect <manifest>            (m2) validate + reproduce llama-server
  ggufbill audit   <manifest>…           (m3) 等保 audit view
  ggufbill version

emit reads the GGUF header (arch, quant, params, sha256) and writes a portable,
等保-auditable deployment manifest. Run "ggufbill emit -h" for the
runtime/hardware/audit flags.

Manifest format: https://ggufbill.org/v0.1/manifest.schema.json
`)
}
