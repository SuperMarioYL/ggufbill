package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/SuperMarioYL/ggufbill/internal/gguf"
	"github.com/SuperMarioYL/ggufbill/internal/manifest"
)

// runEmit implements `ggufbill emit <file.gguf> [flags]`: it reads the GGUF
// header, collects runtime/hardware/quirks/audit fields (from flags, or
// interactively when stdin is a TTY), and writes a valid .ggufbill.json
// sidecar.
func runEmit(args []string) error {
	var (
		context   int64
		kv        string
		gpuLayers int
		vram      int
		gpus      stringSlice
		gpuCount  int
		engineBld string
		operator  string
		procRef   string
		dengbao   string
		toolCall  bool
		breakages stringSlice
		extra     stringSlice
		notes     string
		output    string
		nonInter  bool
	)
	fs := flag.NewFlagSet("emit", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Int64Var(&context, "context", 0, "serving context length (tokens)")
	fs.StringVar(&kv, "kv", "", "KV cache dtype (e.g. Q8_0, f16)")
	fs.IntVar(&gpuLayers, "gpu-layers", -1, "layers offloaded to GPU (-1 = all)")
	fs.IntVar(&vram, "vram", 0, "minimum VRAM per GPU (GB)")
	fs.Var(&gpus, "gpu", "target GPU model(s), comma-separated; repeatable")
	fs.IntVar(&gpuCount, "gpu-count", 0, "number of GPUs in the deployment")
	fs.StringVar(&engineBld, "engine-build", "", "llama.cpp build id (e.g. b10181)")
	fs.StringVar(&operator, "operator", "", "operator accountable for this deployment")
	fs.StringVar(&procRef, "procurement-ref", "", "procurement / asset reference for sign-off")
	fs.StringVar(&dengbao, "dengbao-level", "", "等保 (classified protection) level claimed")
	fs.BoolVar(&toolCall, "tool-calling", false, "tool calling is reliable on this config")
	fs.Var(&breakages, "breakages", "known breakages, comma-separated; repeatable")
	fs.Var(&extra, "extra-flags", "extra validated engine flags, comma-separated; repeatable")
	fs.StringVar(&notes, "notes", "", "free-form hardware notes")
	fs.StringVar(&output, "output", "", "output path (default: sidecar next to the .gguf)")
	fs.StringVar(&output, "o", "", "shorthand for --output")
	fs.BoolVar(&nonInter, "non-interactive", false, "never prompt; require all fields via flags")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: ggufbill emit <file.gguf> [flags]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Reads the GGUF header and writes a portable, 等保-auditable")
		fmt.Fprintln(os.Stderr, "deployment manifest (.ggufbill.json) sidecar.")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "flags:")
		fs.PrintDefaults()
	}
	// The stdlib flag package stops at the first non-flag argument, so reorder
	// args to let operators write the natural `emit <file> --flags` order.
	flagArgs, posArgs := splitArgs(fs, args)
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}

	ggufPath := ""
	if len(posArgs) > 0 {
		ggufPath = posArgs[0]
	}
	if ggufPath == "" {
		fs.Usage()
		return fmt.Errorf("emit: a .gguf file path is required")
	}

	hdr, err := gguf.Read(ggufPath)
	if err != nil {
		return fmt.Errorf("emit: read %s: %w", ggufPath, err)
	}

	interactive := isStdinTTY() && !nonInter

	// Resolve the operator-controlled fields. Flags win; otherwise we prompt
	// (when interactive) for the *required* fields only — context, kv, operator.
	// Optional fields fall back to safe defaults so a script or PTY never blocks
	// on a field the operator can also edit in the JSON later.
	if context <= 0 {
		if interactive {
			context = promptInt64("Serving context length (tokens)", 4096)
		} else {
			return fmt.Errorf("emit: --context is required (or run interactively)")
		}
	}
	if kv == "" {
		if interactive {
			kv = prompt("KV cache dtype (e.g. Q8_0, f16)", "f16")
		} else {
			kv = "f16"
		}
	}
	if engineBld == "" {
		engineBld = "unknown"
	}
	if operator == "" {
		if interactive {
			operator = prompt("Operator name", defaultOperator())
		} else {
			return fmt.Errorf("emit: --operator is required (or run interactively)")
		}
	}
	if gpuCount == 0 && len(gpus) > 0 {
		gpuCount = len(gpus)
	}

	m := &manifest.Manifest{
		SchemaRef:       manifest.SchemaURL,
		ManifestVersion: manifest.Version,
		Model: manifest.Model{
			Name:    modelName(hdr),
			Arch:    hdr.Architecture,
			Quant:   hdr.Quant,
			ParamsB: hdr.ParamsB,
			File:    gguf.BaseName(ggufPath),
			SHA256:  hdr.SHA256,
		},
		Runtime: manifest.Runtime{
			Engine:        "llama.cpp",
			EngineBuild:   engineBld,
			ContextLength: context,
			KVCacheDType:  kv,
			GPULayers:     gpuLayers,
			ExtraFlags:    extra,
		},
		Hardware: manifest.Hardware{
			MinVRAMGB:  vram,
			TargetGPUs: gpus,
			GPUCount:   gpuCount,
			Notes:      notes,
		},
		Quirks: manifest.Quirks{
			ToolCalling:    toolCall,
			KnownBreakages: breakages,
		},
		Audit: manifest.Audit{
			EmittedAt:      time.Now().UTC(),
			EmittedBy:      "ggufbill " + version,
			Operator:       operator,
			ProcurementRef: procRef,
			DengbaoLevel:   dengbao,
		},
	}
	if err := m.Validate(); err != nil {
		return fmt.Errorf("emit: %w", err)
	}

	outPath := output
	if outPath == "" {
		outPath = sidecarPath(ggufPath)
	}
	data, err := m.JSON()
	if err != nil {
		return fmt.Errorf("emit: marshal: %w", err)
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("emit: write %s: %w", outPath, err)
	}

	fmt.Fprintf(os.Stderr, "wrote %s\n", outPath)
	fmt.Fprintf(os.Stderr, "  model   %s (%s, %s)\n", m.Model.Name, m.Model.Arch, m.Model.Quant)
	fmt.Fprintf(os.Stderr, "  params  %.1fB  sha256 %s…\n", m.Model.ParamsB, shortSHA(m.Model.SHA256))
	fmt.Fprintf(os.Stderr, "  runtime llama.cpp %s  ctx %d  kv %s  ngl %d\n",
		m.Runtime.EngineBuild, m.Runtime.ContextLength, m.Runtime.KVCacheDType, m.Runtime.GPULayers)
	return nil
}

// modelName prefers the GGUF's general.name; falls back to the file's base name
// (sans .gguf) when the publisher omitted it.
func modelName(h *gguf.Header) string {
	if h.Name != "" {
		return h.Name
	}
	base := strings.TrimSuffix(filepath.Base(h.Path), ".gguf")
	if base == "" {
		return "unknown"
	}
	return base
}

// sidecarPath returns the default output path: the input .gguf path with its
// .gguf suffix replaced by .ggufbill.json (a true sidecar next to the model).
func sidecarPath(ggufPath string) string {
	if strings.HasSuffix(ggufPath, ".gguf") {
		return strings.TrimSuffix(ggufPath, ".gguf") + ".ggufbill.json"
	}
	return ggufPath + ".ggufbill.json"
}

func shortSHA(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

func defaultOperator() string {
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return "operator"
}

// --- interactive helpers -----------------------------------------------------

func isStdinTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func prompt(label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func promptInt64(label string, def int64) int64 {
	s := prompt(label, strconv.FormatInt(def, 10))
	if s == "" {
		return def
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// stringSlice is a flag.Value that accumulates comma-separated values across
// repeated flag occurrences.
type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error {
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			*s = append(*s, p)
		}
	}
	return nil
}

// splitArgs separates interspersed flags from positional args so operators
// can write `emit <file> --flags` (the stdlib flag package otherwise stops at
// the first non-flag token). It inspects fs to tell bool flags (which take no
// value) from value flags, and treats negative numbers as values so
// `--gpu-layers -1` works.
func splitArgs(fs *flag.FlagSet, args []string) (flagArgs, posArgs []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			posArgs = append(posArgs, args[i+1:]...)
			break
		}
		if a == "-" {
			posArgs = append(posArgs, a)
			continue
		}
		if len(a) > 1 && a[0] == '-' {
			name := strings.TrimLeft(a, "-")
			if eq := strings.IndexByte(name, '='); eq >= 0 {
				flagArgs = append(flagArgs, a)
				continue
			}
			if isBoolFlag(fs, name) {
				flagArgs = append(flagArgs, a)
				continue
			}
			// Value flag: consume the next arg unless it is itself a flag
			// (negative numbers count as values, not flags).
			if i+1 < len(args) && !looksLikeFlag(args[i+1]) {
				flagArgs = append(flagArgs, a, args[i+1])
				i++
			} else {
				flagArgs = append(flagArgs, a)
			}
			continue
		}
		posArgs = append(posArgs, a)
	}
	return flagArgs, posArgs
}

func isBoolFlag(fs *flag.FlagSet, name string) bool {
	fl := fs.Lookup(name)
	if fl == nil {
		return false
	}
	if bf, ok := fl.Value.(interface{ IsBoolFlag() bool }); ok {
		return bf.IsBoolFlag()
	}
	return false
}

// looksLikeFlag reports whether s starts with "-" but is not a bare number
// (so "-1" is a value, "--context" is a flag).
func looksLikeFlag(s string) bool {
	if len(s) < 2 || s[0] != '-' {
		return false
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return false
	}
	return true
}
