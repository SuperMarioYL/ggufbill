**English** | [简体中文](./README.md)

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./assets/hero-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="./assets/hero-light.svg">
  <img src="./assets/hero-light.svg" width="880" alt="ggufbill — portable GGUF deployment-manifest sidecar">
</picture>

<p align="center"><sub>ggufbill is the deployment-manifest sidecar that travels with a GGUF and makes every deployment reproducible and 等保-auditable.</sub></p>

<p align="center">
  <a href="./LICENSE"><img src="https://img.shields.io/badge/license-MIT-0071E3" alt="license"></a>
  <a href="https://github.com/SuperMarioYL/ggufbill/releases"><img src="https://img.shields.io/github/v/release/SuperMarioYL/ggufbill?label=release&color=0071E3" alt="release"></a>
  <img src="https://img.shields.io/github/actions/workflow/status/SuperMarioYL/ggufbill/ci.yml?branch=main&label=CI" alt="CI">
  <img src="https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white" alt="Go">
</p>

**Turn the hand-written llama.cpp config buried in Reddit threads into a machine-readable, 等保-auditable manifest that travels with your GGUF.**

<h2><img src="https://api.iconify.design/tabler:topology-star-3.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> Architecture</h2>

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./assets/atlas-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="./assets/atlas-light.svg">
  <img src="./assets/atlas-light.svg" width="880" alt="Architecture: GGUF model → ggufbill emit → .ggufbill.json manifest → reproduce / 等保 audit">
</picture>

One Go binary, three subcommands, one sidecar format, no daemons. `ggufbill emit` reads the GGUF header (arch, quant, params, sha256) and writes a `.ggufbill.json` sidecar that travels with the `.gguf` across boxes and rebuilds — so operators reproduce a known-good deployment and 等保 / 信创 procurement get a checkable per-box artifact.

<h2><img src="https://api.iconify.design/tabler:list.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> Contents</h2>

- [Why ggufbill](#why-ggufbill)
- [Install & quickstart](#install--quickstart)
- [Usage](#usage)
- [Demo](#demo)
- [Manifest format](#manifest-format)
- [Pricing](#pricing)
- [Roadmap](#roadmap)
- [License](#license)

<h2><img src="https://api.iconify.design/tabler:bulb.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> Why ggufbill</h2>

Today an operator downloads a GGUF and reverse-engineers the serving config from Reddit prose — context length, KV-cache dtype, multi-GPU topology all live in narrative comments that don't survive a box rebuild, and 等保 audit has no per-box artifact to inspect. r/LocalLLaMA even ran a [petition](https://www.reddit.com/r/LocalLLaMA/comments/1vqnbhe/petition_to_add_a_rule_for_people_to_add_their/) asking authors to disclose quant config, and the 4×3060 DeepSeek V4 [post](https://www.reddit.com/r/LocalLLaMA/comments/1vrqf4f/running_deepseek_v4_flash_q4_k_xl_at_100_toks/) is itself a hand-written manifest in prose — engine build, quant, KV-cache dtype, context window, multi-GPU topology are all fields with no schema. ggufbill turns those fields into a JSON sidecar that travels with the GGUF: it survives rebuilds, stays engine-agnostic, and is 等保-checkable.

<h2><img src="https://api.iconify.design/tabler:rocket.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> Install & quickstart</h2>

Single static binary, ~10s to install. Cold clone to first visible result in three commands:

```bash
git clone https://github.com/SuperMarioYL/ggufbill && cd ggufbill
go install github.com/SuperMarioYL/ggufbill/cmd/ggufbill@latest
ggufbill emit internal/gguf/testdata/qwen3-27b-q4_k_m.gguf --context 32768 --kv Q8_0 --vram 16 --gpu RTX-3060-12GB --operator ops -o Qwen3-27B-Q4_K_M.ggufbill.json
```

<details><summary>Sample output</summary>

```
wrote Qwen3-27B-Q4_K_M.ggufbill.json
  model   Qwen3-27B (qwen2, Q4_K_M)
  params  27.0B  sha256 8c68c1bac72d…
  runtime llama.cpp unknown  ctx 32768  kv Q8_0  ngl -1
```
```json
{
  "$schema": "https://ggufbill.org/v0.1/manifest.schema.json",
  "manifest_version": "0.1",
  "model": { "name": "Qwen3-27B", "arch": "qwen2", "quant": "Q4_K_M", "params_b": 27, "file": "qwen3-27b-q4_k_m.gguf", "sha256": "8c68c1…" },
  "audit": { "emitted_by": "ggufbill 0.1.0", "operator": "ops" }
}
```
</details>

The repo ships a synthetic Qwen3-27B Q4_K_M GGUF (`internal/gguf/testdata/`, ~318 bytes, metadata-only) so the command above runs without downloading a 16 GB model first. For a real deployment, point it at your own `.gguf`.

<h2><img src="https://api.iconify.design/tabler:terminal-2.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> Usage</h2>

```bash
# Read the GGUF header, interactively fill runtime/hardware/audit fields, write the sidecar
ggufbill emit ./models/Qwen3-27B-Q4_K_M.gguf

# Pass every field at once (script/demo friendly)
ggufbill emit ./DeepSeek-V4-Flash-Q4_K_XL.gguf \
  --context 360864 --kv Q8_0 --vram 12 --gpu RTX-3060-12GB --gpu-count 4 \
  --engine-build b10181 --operator ops-lee \
  --procurement-ref proc-2026-GPU-014 --dengbao-level dengbao-3 \
  --tool-calling --notes "4×3060 12GB, per the r/LocalLLaMA 1M-token post"

# (m2) validate the sidecar + reproduce the llama-server invocation
ggufbill inspect Qwen3-27B-Q4_K_M.ggufbill.json

# (m3) 等保 deployment audit view
ggufbill audit *.ggufbill.json > dengbao-deployment-audit.md
```

Common `emit` flags:

| flag | meaning | example |
|---|---|---|
| `--context` | serving context length (tokens) | `32768` |
| `--kv` | KV cache dtype | `Q8_0`, `f16` |
| `--vram` | min VRAM per GPU (GB) | `16` |
| `--gpu` | target GPU model(s), comma-separated / repeatable | `RTX-3060-12GB` |
| `--gpu-count` | number of GPUs | `4` |
| `--engine-build` | llama.cpp build id | `b10181` |
| `--operator` | operator accountable (required) | `ops` |
| `--procurement-ref` | procurement / asset ref | `proc-2026-GPU-014` |
| `--dengbao-level` | 等保 (classified protection) level | `dengbao-3` |
| `--tool-calling` | tool calling is reliable on this config | — |
| `--breakages` | known breakages | `long-context-drift` |
| `-o` | output path (default: sidecar next to the .gguf) | `out.ggufbill.json` |

> Flags may appear before or after the file path (`emit <file> --flags` or `emit --flags <file>`); use the equals form for negatives (`--gpu-layers=-1`).

<h2><img src="https://api.iconify.design/tabler:photo.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> Demo</h2>

![demo](assets/demo.gif)

Build from source → emit a Qwen3-27B Q4_K_M manifest → `jq` the sidecar. The full tape is `docs/demo.tape`; CI re-renders the gif on demand.

<h2><img src="https://api.iconify.design/tabler:code.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> Manifest format</h2>

A `.ggufbill.json` is an SBOM-style deployment artifact, not a wrapper over llama.cpp CLI flags. The full schema is [`schemas/manifest.v0.1.json`](./schemas/manifest.v0.1.json) (reflected from the Go structs via `invopop/jsonschema`).

| block | fields | meaning |
|---|---|---|
| `model` | name, arch, quant, params_b, file, sha256 | model identity read from the GGUF header |
| `runtime` | engine, engine_build, context_length, kv_cache_dtype, gpu_layers, extra_flags | the operator-validated llama.cpp config |
| `hardware` | min_vram_gb, target_gpus, gpu_count, notes | target GPU hardware |
| `quirks` | tool_calling, known_breakages | known behavior of this quant/config |
| `audit` | emitted_at, emitted_by, operator, procurement_ref, 等保_level | 等保 / procurement audit provenance |

`procurement_ref` and `等保_level` are the wedge the paid tier reads over the same format — the CLI is never paywalled; the 等保 audit dashboard is the paid layer.

<h2><img src="https://api.iconify.design/tabler:credit-card.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> Pricing</h2>

The CLI and manifest format are free MIT forever; commercialization runs through a 等保 audit dashboard that reads the same `.ggufbill.json` — no paywall on the CLI.

| tier | price | fit |
|---|---|---|
| **OSS CLI** | free (MIT) | solo / small team, local emit + inspect, data stays on-prem |
| **Hosted 等保 audit dashboard** | ¥3,000/box/year or ¥30,000/year team plan (≤5 auditors, unlimited boxes) | 等保 audit firms — aggregate a client's GPU-fleet manifests, generate 测评报告 addenda, with RBAC so junior auditors can't edit a manifest post-audit |
| **On-prem** | ¥80,000/year + support | state-affiliated / defense clients that refuse auditor-hosted |

- Data residency: hosted on Alibaba Cloud / Huawei Cloud + WeChat Pay / bank transfer; on-prem is fully air-gapped.
- First paid customer: 等保 audit firms — every engagement already collects client deployment data and writes 测评报告 by hand; a manifest addendum that saves ≥1 person-day/engagement closes a team plan within 30 days pilot-to-paid.
- v0.1 ships no hosted tier; the free CLI and format come first. If a 等保 firm confirms the manifest satisfies an audit/procurement clause within 60 days, the hosted layer opens.

<h2><img src="https://api.iconify.design/tabler:map-2.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> Roadmap</h2>

- [x] **m1 — emit manifest**: `ggufbill emit` reads the GGUF header (arch, quant, params, sha256) and writes a valid `.ggufbill.json` sidecar. **(current build)**
- [ ] **m2 — inspect/reproduce**: `ggufbill inspect` schema-validates the sidecar, re-verifies the GGUF sha256, and prints the exact `llama-server` invocation + a human-readable deployment summary.
- [ ] **m3 — 等保 audit report**: `ggufbill audit` emits a 等保 audit view (markdown + HTML) with per-box quant/context/hardware/operator/procurement_ref/等保_level.
- [ ] Future: auto-emit from Unsloth/HuggingFace release pipelines; org manifest registry + RBAC (team plan v0.2+); manifest signing / Sigstore provenance; multi-engine (vLLM, Ollama API, llama-server).

<h2><img src="https://api.iconify.design/tabler:license.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> License</h2>

MIT, see [LICENSE](./LICENSE). File bugs or open PRs in [Issues](https://github.com/SuperMarioYL/ggufbill/issues) — especially to convert your hand-written llama.cpp config into a canonical manifest under `examples/`.

<p align="center"><sub><a href="./LICENSE">MIT</a> © 2026 SuperMarioYL</sub></p>
