package gguf

import (
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestWriteFixture regenerates the synthetic Qwen3-27B Q4_K_M fixture used by
// the reader tests and the demo tape. It only runs when
// GGUFBILL_WRITE_FIXTURE=1 is set so a normal `go test` run is a no-op.
//
//	GGUFBILL_WRITE_FIXTURE=1 go test ./internal/gguf -run TestWriteFixture -v
//
// The fixture is a metadata-only GGUF v3 (no tensor data) carrying the fields a
// real Unsloth Qwen3-27B Q4_K_M release would expose. It is small on purpose
// so the demo can ship it in-repo.
func TestWriteFixture(t *testing.T) {
	if os.Getenv("GGUFBILL_WRITE_FIXTURE") != "1" {
		t.Skip("set GGUFBILL_WRITE_FIXTURE=1 to regenerate the test fixture")
	}
	if err := os.MkdirAll("testdata", 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("testdata", "qwen3-27b-q4_k_m.gguf")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	w := &fixtureWriter{w: f}
	w.bytes("GGUF")
	w.u32(3) // version
	w.u64(0) // tensor_count — metadata-only fixture; n_params carries the count
	w.u64(7) // kv_count

	// (key, value-type, value) tuples matching a real Qwen3-27B Q4_K_M GGUF.
	kv := []struct {
		key   string
		vtype uint32
		value any
	}{
		{"general.architecture", typString, "qwen2"},
		{"general.name", typString, "Qwen3-27B"},
		{"general.organization", typString, "Qwen"},
		{"general.file_type", typUint32, uint32(15)}, // GGML_FTYPE_MOSTLY_Q4_K_M
		{"general.quantization_version", typInt32, int32(2)},
		{"general.quantization_name", typString, "Q4_K_M"},
		{"general.n_params", typUint64, uint64(27_000_000_000)},
	}
	for _, p := range kv {
		w.str(p.key)
		w.u32(p.vtype)
		switch v := p.value.(type) {
		case string:
			w.str(v)
		case uint32:
			w.u32(v)
		case int32:
			w.u32(uint32(v))
		case uint64:
			w.u64(v)
		default:
			t.Fatalf("unsupported fixture value type %T for %q", v, p.key)
		}
	}
	if w.err != nil {
		t.Fatalf("write fixture: %v", w.err)
	}
	t.Logf("wrote %s (%d bytes)", path, w.n)
}

// fixtureWriter is a tiny little-endian GGUF writer used only to regenerate the
// committed test fixture. It lives in a test file so production never links it.
type fixtureWriter struct {
	w   io.Writer
	n   int64
	err error
}

func (x *fixtureWriter) bytes(s string) {
	if x.err != nil {
		return
	}
	n, err := x.w.Write([]byte(s))
	x.n += int64(n)
	x.err = err
}

func (x *fixtureWriter) u32(v uint32) {
	if x.err != nil {
		return
	}
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	n, err := x.w.Write(buf[:])
	x.n += int64(n)
	x.err = err
}

func (x *fixtureWriter) u64(v uint64) {
	if x.err != nil {
		return
	}
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], v)
	n, err := x.w.Write(buf[:])
	x.n += int64(n)
	x.err = err
}

func (x *fixtureWriter) str(s string) {
	x.u64(uint64(len(s)))
	x.bytes(s)
}
