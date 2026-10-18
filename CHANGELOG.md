# Changelog

All notable changes to ggufbill are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions follow
[SemVer](https://semver.org/). The manifest **format** version (`manifest_version`,
currently `0.1`) is versioned separately from the tool and does not change in
this release.

## [Unreleased]

## [0.3.0] - 2026-10-19

### Added

- `ggufbill inspect <manifest>` (plan milestone m2): validates the sidecar,
  re-verifies the model's whole-file sha256 against the GGUF next to the
  manifest — a digest mismatch fails with a non-zero exit, a missing model
  file is reported as `unverified` — and prints the exact `llama-server`
  invocation plus a deployment summary. The invocation maps
  `context_length` to `-c`, `gpu_layers` to `-ngl`, `kv_cache_dtype` to
  `-ctk <dtype> -ctv <dtype>` (omitted for `f16`, the llama.cpp default),
  and appends `extra_flags` verbatim.
- `ggufbill audit [--format markdown|html] <manifest>…` (plan milestone m3):
  validates every manifest and renders the 等保 deployment-audit view — a
  per-box table of model/quant/params, context, KV dtype, GPU layers,
  hardware, operator, procurement_ref and 等保_level, plus per-manifest
  fingerprints (sha256, engine build, emitting tool) — to stdout so
  `audit *.ggufbill.json > 等保-部署审计.md` works. Previously both
  subcommands were stubs that printed a not-implemented notice and exited 0.

### Fixed

- The GGUF reader no longer fabricates `quant: "F32"` for files that carry
  neither `general.quantization_name` nor `general.file_type`: the fallback
  resolved the missing `general.file_type` to the zero ftype, whose table
  entry is "F32". An unlabeled quant now reads `unknown` instead of a false
  label in the manifest.
- The GGUF version gate rejects GGUF v1 cleanly. The gate used to accept
  version 1 while every subsequent read assumed the v2/v3 layout (u64 counts,
  u64-length strings), so v1 files — which pack counts and string lengths as
  u32 — died with a confusing mid-block misparse error instead of an
  unsupported-version rejection.

## [0.2.0] - 2026-09-14

### Fixed

- `ggufbill emit` no longer records `params_b: 0` for GGUF files that lack
  `general.n_params` (older GGUFs). The tensor-info fallback misparsed the
  header — it skipped past `kv_count` and then read the first KV key's length
  as if it were `kv_count`, and read tensor dimensions as 32-bit where the
  GGUF spec (and gguf-py) write 64-bit — so the fallback always failed and the
  parameter count silently degraded to 0. The fallback now walks the KV block
  correctly and sums tensor elements as `uint64`.
- Interactive `ggufbill emit` no longer swallows pasted multi-line answers.
  Every prompt used to build a fresh buffered reader over stdin, so a pasted
  answer was consumed by the first prompt's buffer and every line after the
  first was silently discarded, making later questions fall back to their
  defaults. All prompts of one run now share a single reader.

### Quality

- Version lockstep: `VERSION`, the `ggufbill` binary version, and
  `web/site.json` `meta.implementation_version` are pinned together by a
  consistency test (`TestVersionLockstep`), so a bump that touches only one
  surface fails CI.
- `schemas/manifest.v0.1.json` is pinned byte-for-byte to the Go manifest
  structs (`manifest.Schema()`), keeping the committed schema an honest
  reflection of the single source of truth.
- Added this CHANGELOG.

## [0.1.0] - 2026-08-23

### Added

- `ggufbill emit <file.gguf>` — reads the GGUF header (architecture, name,
  quant, parameter count, whole-file sha256) and writes a portable,
  等保-auditable `.ggufbill.json` deployment-manifest sidecar (m1).
- Manifest JSON Schema `schemas/manifest.v0.1.json` and the
  `examples/Qwen3-27B-Q4_K_M.ggufbill.json` example manifest.

[Unreleased]: https://github.com/SuperMarioYL/ggufbill/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/SuperMarioYL/ggufbill/releases/tag/v0.3.0
[0.2.0]: https://github.com/SuperMarioYL/ggufbill/releases/tag/v0.2.0
[0.1.0]: https://github.com/SuperMarioYL/ggufbill/releases/tag/v0.1.0
