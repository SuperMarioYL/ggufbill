package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/SuperMarioYL/ggufbill/internal/llamacpp"
	"github.com/SuperMarioYL/ggufbill/internal/manifest"
)

// runInspect implements `ggufbill inspect <manifest>`: it validates the
// sidecar, re-verifies the model's sha256 against the GGUF next to it, and
// prints the exact `llama-server` invocation plus a deployment summary — the
// m2 milestone (inspect/reproduce).
//
// Verification contract: a model file PRESENT next to the manifest with a
// different digest is fatal (that is the tamper/mis-ship case the sidecar
// exists to catch); a MISSING model file degrades to a warning, because audit
// preparation legitimately reads manifests without the weights on disk.
func runInspect(args []string) error {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: ggufbill inspect <manifest.ggufbill.json>")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Validates the sidecar, re-verifies the GGUF sha256 against the")
		fmt.Fprintln(os.Stderr, "model file next to it, and prints the exact llama-server")
		fmt.Fprintln(os.Stderr, "invocation plus a deployment summary.")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("inspect: exactly one manifest path is required")
	}
	path := fs.Arg(0)

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("inspect: %w", err)
	}
	m, err := manifest.Parse(data)
	if err != nil {
		return fmt.Errorf("inspect: parse %s: %w", path, err)
	}
	if err := m.Validate(); err != nil {
		return fmt.Errorf("inspect: %s: %w", path, err)
	}
	if m.ManifestVersion != manifest.Version {
		fmt.Fprintf(os.Stderr, "inspect: warning: manifest_version %s, this tool emits %s\n",
			m.ManifestVersion, manifest.Version)
	}

	// The sidecar records only the model basename; the verified file lives
	// next to the manifest.
	modelPath := filepath.Join(filepath.Dir(path), m.Model.File)
	verified := false
	if st, err := os.Stat(modelPath); err == nil && !st.IsDir() {
		sum, err := fileSHA256(modelPath)
		if err != nil {
			return fmt.Errorf("inspect: hash %s: %w", modelPath, err)
		}
		if sum != m.Model.SHA256 {
			return fmt.Errorf("inspect: sha256 mismatch for %s: manifest records %s, file is %s",
				modelPath, m.Model.SHA256, sum)
		}
		verified = true
	} else {
		fmt.Fprintf(os.Stderr, "inspect: warning: %s not found — sha256 unverified\n", modelPath)
	}

	cmd, err := llamacpp.Synthesize(m, modelPath)
	if err != nil {
		return fmt.Errorf("inspect: %w", err)
	}

	fmt.Println(cmd)
	fmt.Println()
	fmt.Println("deployment summary")
	fmt.Printf("  model     %s (%s, %s, %.1fB)\n", m.Model.Name, m.Model.Arch, m.Model.Quant, m.Model.ParamsB)
	fmt.Printf("  file      %s  sha256 %s…  %s\n",
		m.Model.File, shortSHA(m.Model.SHA256), map[bool]string{true: "verified", false: "unverified"}[verified])
	fmt.Printf("  runtime   %s %s  ctx %d  kv %s  ngl %d\n",
		m.Runtime.Engine, dashIfEmpty(m.Runtime.EngineBuild), m.Runtime.ContextLength,
		m.Runtime.KVCacheDType, m.Runtime.GPULayers)
	fmt.Printf("  hardware  %s\n", inspectHardwareSummary(&m.Hardware))
	fmt.Printf("  quirks    %s\n", inspectQuirksSummary(&m.Quirks))
	fmt.Printf("  audit     operator %s  ref %s  等保 %s  emitted %s by %s\n",
		m.Audit.Operator, dashIfEmpty(m.Audit.ProcurementRef), dashIfEmpty(m.Audit.DengbaoLevel),
		m.Audit.EmittedAt.UTC().Format("2006-01-02"), m.Audit.EmittedBy)
	return nil
}

// dashIfEmpty renders absent optional fields as "-" in the summary.
func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// fileSHA256 returns the hex sha256 of the file at path, streamed so large
// GGUFs never load into memory.
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func inspectHardwareSummary(h *manifest.Hardware) string {
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
	summary := strings.Join(parts, "; ")
	if h.Notes != "" {
		summary += " (" + h.Notes + ")"
	}
	return summary
}

func inspectQuirksSummary(q *manifest.Quirks) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("tool_calling=%t", q.ToolCalling))
	for _, b := range q.KnownBreakages {
		parts = append(parts, "breakage: "+b)
	}
	return strings.Join(parts, "; ")
}
