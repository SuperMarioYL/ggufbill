// Package gguf provides a minimal, dependency-free reader for the GGUF v3
// file header. It extracts only the deployment-relevant metadata an operator
// needs to record in a manifest: architecture, model name, quantization
// variant, parameter count, and the file's sha256.
//
// It deliberately avoids a llama.cpp dependency — only the on-disk metadata
// block is parsed, never the tensor data. The reader streams the whole file
// through a sha256 hasher in a single pass so the manifest can later be
// re-verified on another box without a second scan.
package gguf

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

// magic identifies a GGUF file: "GGUF" little-endian.
const magic = "GGUF"

// GGUF metadata value types (gguf_metadata_value_type).
const (
	typUint8 uint32 = iota
	typInt8
	typUint16
	typInt16
	typUint32
	typInt32
	typFloat32
	typBool
	typString
	typArray
	typUint64
	typInt64
	typFloat64
)

// Header carries the deployment-relevant fields extracted from a GGUF file.
type Header struct {
	Version      uint32
	Architecture string // general.architecture (e.g. "qwen2")
	Name         string // general.name
	Quant        string // resolved quant name (Q4_K_M, F16, ...)
	ParamsB      float64 // parameter count in billions
	SHA256       string // sha256 of the whole file
	Size         int64  // file size in bytes
	Path         string // path the header was read from
}

// Read parses the GGUF header at path and computes the file's sha256.
// It reads the entire file (streaming through the hasher) so the returned
// SHA256 covers tensor data too — the manifest is meant to be re-verified on
// a different box.
func Read(path string) (*Header, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	hash := sha256.New()
	tr := io.TeeReader(f, hash)
	br := bufio.NewReaderSize(tr, 1<<20) // 1 MiB read-ahead keeps large files fast

	// Header: magic(4) + version(4) + tensor_count(8) + kv_count(8).
	var m [4]byte
	if _, err := io.ReadFull(br, m[:]); err != nil {
		return nil, fmt.Errorf("gguf: read magic: %w", err)
	}
	if string(m[:]) != magic {
		return nil, fmt.Errorf("gguf: not a GGUF file (bad magic %q)", m[:])
	}

	version, err := readU32(br)
	if err != nil {
		return nil, err
	}
	if version < 1 || version > 3 {
		return nil, fmt.Errorf("gguf: unsupported version %d", version)
	}
	tensorCount, err := readU64(br)
	if err != nil {
		return nil, err
	}
	kvCount, err := readU64(br)
	if err != nil {
		return nil, err
	}

	meta := make(map[string]any, kvCount)
	for i := uint64(0); i < kvCount; i++ {
		key, err := readString(br)
		if err != nil {
			return nil, fmt.Errorf("gguf: kv %d key: %w", i, err)
		}
		vtype, err := readU32(br)
		if err != nil {
			return nil, fmt.Errorf("gguf: kv %q type: %w", key, err)
		}
		val, err := readValue(br, vtype)
		if err != nil {
			return nil, fmt.Errorf("gguf: kv %q value: %w", key, err)
		}
		meta[key] = val
	}

	// Drain tensor-info + tensor data through the hasher so the sha256 covers
	// the whole file. We do not need the tensor values, only the metadata, but
	// reading past them completes the single-pass hash.
	if _, err := io.Copy(io.Discard, br); err != nil {
		return nil, fmt.Errorf("gguf: drain: %w", err)
	}

	h := &Header{
		Version: version,
		Path:    path,
		Size:    info.Size(),
		SHA256:  hex.EncodeToString(hash.Sum(nil)),
	}
	h.Architecture = asString(meta["general.architecture"])
	h.Name = asString(meta["general.name"])

	// Quant: prefer an explicit quantization_name string when the GGUF carries
	// one; otherwise resolve from the general.file_type enum.
	if qn := asString(meta["general.quantization_name"]); qn != "" {
		h.Quant = qn
	} else {
		h.Quant = ftypeName(uint32(asNumber(meta["general.file_type"])))
	}

	// Params: prefer general.n_params (recent GGUFs); fall back to summing
	// tensor nelements when the publisher omitted it. The tensor loop below
	// only runs when n_params is absent.
	params, hasParams := meta["general.n_params"]
	if !hasParams {
		params, hasParams = meta["general.n_params_total"]
	}
	if hasParams {
		h.ParamsB = asNumber(params) / 1e9
	} else {
		n, err := sumTensorParams(path, tensorCount)
		if err != nil {
			// Non-fatal: a manifest without a param count is still valid; the
			// operator can edit it. Surface as 0 and continue.
			h.ParamsB = 0
		} else {
			h.ParamsB = float64(n) / 1e9
		}
	}

	return h, nil
}

// sumTensorParams re-opens the file, skips the header, and sums nelements
// across all tensor-info records. Used only when general.n_params is absent.
func sumTensorParams(path string, tensorCount uint64) (uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	br := bufio.NewReaderSize(f, 1<<20)
	// Skip magic(4) + version(4) + tensor_count(8); kv_count is read next.
	if _, err := io.CopyN(io.Discard, br, 4+4+8); err != nil {
		return 0, err
	}
	// Skip the KV block to reach tensor info.
	kvCount, err := readU64(br)
	if err != nil {
		return 0, err
	}
	for i := uint64(0); i < kvCount; i++ {
		if _, err := readString(br); err != nil {
			return 0, err
		}
		vt, err := readU32(br)
		if err != nil {
			return 0, err
		}
		if err := skipValue(br, vt); err != nil {
			return 0, err
		}
	}
	var total uint64
	for i := uint64(0); i < tensorCount; i++ {
		if _, err := readString(br); err != nil {
			return 0, err
		}
		nDims, err := readU32(br)
		if err != nil {
			return 0, err
		}
		if nDims > 8 {
			return 0, fmt.Errorf("gguf: implausible n_dims %d", nDims)
		}
		ne := uint64(1)
		for d := uint32(0); d < nDims; d++ {
			// GGUF v2/v3 pack each ne dimension as uint64 (gguf-py "Q").
			dim, err := readU64(br)
			if err != nil {
				return 0, err
			}
			ne *= dim
		}
		// type(4) + offset(8) follow the dims.
		if _, err := readU32(br); err != nil {
			return 0, err
		}
		if _, err := readU64(br); err != nil {
			return 0, err
		}
		total += ne
	}
	return total, nil
}

// --- low-level readers -------------------------------------------------------

func readU8(r io.Reader) (uint8, error) {
	var b [1]byte
	_, err := io.ReadFull(r, b[:])
	return b[0], err
}

func readU32(r io.Reader) (uint32, error) {
	var v uint32
	err := binary.Read(r, binary.LittleEndian, &v)
	return v, err
}

func readU64(r io.Reader) (uint64, error) {
	var v uint64
	err := binary.Read(r, binary.LittleEndian, &v)
	return v, err
}

func readF32(r io.Reader) (float32, error) {
	var v float32
	err := binary.Read(r, binary.LittleEndian, &v)
	return v, err
}

func readF64(r io.Reader) (float64, error) {
	var v float64
	err := binary.Read(r, binary.LittleEndian, &v)
	return v, err
}

// readString reads a GGUF string: uint64 length + that many bytes.
func readString(r io.Reader) (string, error) {
	n, err := readU64(r)
	if err != nil {
		return "", err
	}
	if n > 1<<28 { // 256 MiB guard against a corrupt length field
		return "", fmt.Errorf("gguf: string length %d implausible", n)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

// readValue decodes one metadata value of the given type.
func readValue(r io.Reader, typ uint32) (any, error) {
	switch typ {
	case typUint8:
		return readU8(r)
	case typInt8:
		v, err := readU8(r)
		return int8(v), err
	case typUint16:
		var v uint16
		err := binary.Read(r, binary.LittleEndian, &v)
		return v, err
	case typInt16:
		var v int16
		err := binary.Read(r, binary.LittleEndian, &v)
		return v, err
	case typUint32:
		return readU32(r)
	case typInt32:
		v, err := readU32(r)
		return int32(v), err
	case typFloat32:
		return readF32(r)
	case typBool:
		b, err := readU8(r)
		return b != 0, err
	case typString:
		return readString(r)
	case typArray:
		elemType, err := readU32(r)
		if err != nil {
			return nil, err
		}
		count, err := readU64(r)
		if err != nil {
			return nil, err
		}
		if count > 1<<28 {
			return nil, fmt.Errorf("gguf: array length %d implausible", count)
		}
		out := make([]any, 0, count)
		for i := uint64(0); i < count; i++ {
			v, err := readValue(r, elemType)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	case typUint64:
		return readU64(r)
	case typInt64:
		v, err := readU64(r)
		return int64(v), err
	case typFloat64:
		return readF64(r)
	default:
		return nil, fmt.Errorf("gguf: unknown value type %d", typ)
	}
}

// skipValue advances past a value of the given type without decoding it.
func skipValue(r io.Reader, typ uint32) error {
	switch typ {
	case typUint8, typInt8, typBool:
		_, err := readU8(r)
		return err
	case typUint16, typInt16:
		var buf [2]byte
		_, err := io.ReadFull(r, buf[:])
		return err
	case typUint32, typInt32, typFloat32:
		var buf [4]byte
		_, err := io.ReadFull(r, buf[:])
		return err
	case typUint64, typInt64, typFloat64:
		var buf [8]byte
		_, err := io.ReadFull(r, buf[:])
		return err
	case typString:
		n, err := readU64(r)
		if err != nil {
			return err
		}
		if _, err := io.CopyN(io.Discard, r, int64(n)); err != nil {
			return err
		}
		return nil
	case typArray:
		elemType, err := readU32(r)
		if err != nil {
			return err
		}
		count, err := readU64(r)
		if err != nil {
			return err
		}
		for i := uint64(0); i < count; i++ {
			if err := skipValue(r, elemType); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("gguf: unknown value type %d", typ)
	}
}

// --- value coercion ---------------------------------------------------------

func asString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

// asNumber coerces any numeric metadata value to float64.
func asNumber(v any) float64 {
	switch n := v.(type) {
	case uint8:
		return float64(n)
	case int8:
		return float64(n)
	case uint16:
		return float64(n)
	case int16:
		return float64(n)
	case uint32:
		return float64(n)
	case int32:
		return float64(n)
	case uint64:
		return float64(n)
	case int64:
		return float64(n)
	case float32:
		return float64(n)
	case float64:
		return n
	case bool:
		if n {
			return 1
		}
		return 0
	case []any:
		if len(n) > 0 {
			return asNumber(n[0])
		}
	}
	return 0
}

// ftypeName maps a ggml_ftype enum value to the canonical quant short name.
// Unknown values fall back to a stable synthetic label so the manifest is
// never empty.
func ftypeName(ft uint32) string {
	if name, ok := ftypeNames[ft]; ok {
		return name
	}
	return "FTYPE_" + strconv.FormatUint(uint64(ft), 10)
}

var ftypeNames = map[uint32]string{
	0:  "F32",
	1:  "F16",
	2:  "Q4_0",
	3:  "Q4_1",
	4:  "Q4_1_F16",
	7:  "Q8_0",
	8:  "Q5_0",
	9:  "Q5_1",
	10: "Q2_K",
	11: "Q3_K_S",
	12: "Q3_K_M",
	13: "Q3_K_L",
	14: "Q4_K_S",
	15: "Q4_K_M",
	16: "Q5_K_S",
	17: "Q5_K_M",
	18: "Q6_K",
	19: "IQ2_XXS",
	20: "IQ2_XS",
	21: "Q2_K_S",
	22: "IQ3_XXS",
	23: "IQ3_XS",
	24: "IQ3_S",
	25: "IQ3_M",
	26: "IQ4_XS",
	27: "IQ4_NL",
	28: "IQ8_XXN",
	29: "IQ2_S",
	30: "IQ2_M",
	31: "BF16",
	32: "Q4_0_4_4",
	33: "Q4_0_4_8",
	35: "TQ1_0",
	36: "TQ2_0",
}

// BaseName returns the portable base name of a GGUF path (no directory),
// so a manifest records the filename an operator should look for, not the
// emitting machine's directory layout.
func BaseName(path string) string {
	return filepath.Base(path)
}

// ErrNotGGUF is returned when a file does not start with the GGUF magic.
var ErrNotGGUF = errors.New("not a GGUF file")
