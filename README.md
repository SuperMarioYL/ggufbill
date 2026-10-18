[English](README.en.md) | **简体中文**

<picture>
  <source media="(max-width: 640px) and (prefers-color-scheme: dark)" srcset="assets/presentation/hero-mobile-dark.svg">
  <source media="(max-width: 640px)" srcset="assets/presentation/hero-mobile-light.svg">
  <source media="(prefers-color-scheme: dark)" srcset="assets/presentation/hero-dark.svg">
  <img src="assets/presentation/hero-light.svg" width="960" alt="ggufbill — Keep deployment notes beside the model file.">
</picture>

**ggufbill 读取 GGUF 元数据与整文件摘要，把操作者填写的运行参数、硬件条件和部署说明保存为 JSON 旁挂清单。**

`Go 1.24+ · Python 3 demo` · [MIT](LICENSE) · [GitHub](https://github.com/SuperMarioYL/ggufbill) · [网站](https://ggufbill.lei6393.com)

## 为什么需要它

只保留模型文件，往往会丢掉当时使用的上下文长度、KV 类型和 engine build。把这些字段放在相邻清单中，交接和复查时就有明确记录。清单记录的是输入与声明，不会自动证明某个部署能运行或符合审计要求。

<picture>
  <source media="(max-width: 640px) and (prefers-color-scheme: dark)" srcset="assets/presentation/process-mobile-dark.svg">
  <source media="(max-width: 640px)" srcset="assets/presentation/process-mobile-light.svg">
  <source media="(prefers-color-scheme: dark)" srcset="assets/presentation/process-dark.svg">
  <img src="assets/presentation/process-light.svg" width="960" alt="Emit one fixture manifest">
</picture>

## 架构

GGUF reader 提取名称、架构、量化和参数元数据并计算 SHA-256；emit 收集运行/硬件/操作者字段；manifest.Validate 检查必需值后写 JSON。inspect 复核清单、重验文件哈希并合成准确的 llama-server 命令；audit 把多份清单汇总为等保审计视图。

<picture>
  <source media="(max-width: 640px) and (prefers-color-scheme: dark)" srcset="assets/presentation/architecture-mobile-dark.svg">
  <source media="(max-width: 640px)" srcset="assets/presentation/architecture-mobile-light.svg">
  <source media="(prefers-color-scheme: dark)" srcset="assets/presentation/architecture-dark.svg">
  <img src="assets/presentation/architecture-light.svg" width="960" alt="File identity meets operator context">
</picture>

## 安装

需要 Go 1.24+；完整自清理示例使用 Python 3。首次构建可能下载 Go 模块，示例不下载模型。

```bash
git clone https://github.com/SuperMarioYL/ggufbill.git
cd ggufbill
go build -o ggufbill ./cmd/ggufbill
```

## 快速开始

```bash
python3 examples/presentation_demo.py
```

脚本使用仓库的 318 字节 metadata-only GGUF，写出临时清单并回读字段。结果确认完整文件 SHA-256 一致、context_length=4096、kv=f16、operator=demo-operator。Qwen3-27B 是该构造文件的元数据名称，并非加载了 27B 权重。

## 用法

```bash
./ggufbill emit internal/gguf/testdata/qwen3-27b-q4_k_m.gguf --context 4096 --kv f16 --operator demo-operator --non-interactive --output demo.ggufbill.json
./ggufbill inspect demo.ggufbill.json        # 重验哈希并打印准确的 llama-server 命令
./ggufbill audit *.ggufbill.json > 等保-部署审计.md   # 逐盒等保审计视图(markdown;另有 --format html)
./ggufbill version
```

emit 可以交互式询问字段；脚本建议显式给 --context、--operator 和 --non-interactive。--gpu-layers=-1 表示把该配置值记入清单，不是实际执行 GPU offload。

## 能力与集成

| JSON 区块 | 来源 |
|---|---|
| model | GGUF 元数据、文件名与 SHA-256 |
| runtime | engine build、context、KV、GPU layers、extra flags |
| hardware | 操作者声明的 VRAM、GPU 与备注 |
| quirks | 操作者声明的工具调用/已知问题 |
| audit | 生成时间、操作者、采购引用与级别标签 |

格式定义见 [manifest.v0.1.json](schemas/manifest.v0.1.json)。

<picture>
  <source media="(max-width: 640px) and (prefers-color-scheme: dark)" srcset="assets/presentation/integrations-mobile-dark.svg">
  <source media="(max-width: 640px)" srcset="assets/presentation/integrations-mobile-light.svg">
  <source media="(prefers-color-scheme: dark)" srcset="assets/presentation/integrations-dark.svg">
  <img src="assets/presentation/integrations-light.svg" width="960" alt="Current input and output format">
</picture>

## 配置与边界

常用参数包括 --context、--kv、--gpu、--gpu-count、--vram、--engine-build、--operator、--procurement-ref、--dengbao-level 和 --output。GPU 名称可逗号分隔或重复输入；engine build 未填写时为 unknown。

等保级别、硬件容量和 tool_calling 都是声明字段，程序不核验它们。文件哈希用于关联字节身份，不是签名或来源认证。inspect 复核清单结构，在模型文件与清单同目录时重验整文件哈希（不一致则失败；模型缺失则报告 unverified），并打印准确的 llama-server 命令；audit 把多份清单汇总为 markdown 或 HTML 报告。两者记录的是声明与指纹，不构成合规性结论。

## 运行记录

真实 emit 和 JSON 回读（演示录制于 v0.1.0，v0.3.0 的命令与输出保持一致）。硬件、工具调用可靠性和监管符合性均未测试，临时文件结束后清理。

[输入、命令和完整输出](docs/demo-results.json)

[保留的历史终端录屏](assets/demo.gif) · [录制脚本](docs/demo.tape)。本轮示例以以上可重放记录为准。

## 路线图

- [x] GGUF 读取、整文件摘要和清单生成。
- [x] 交互/非交互输入与机器可读格式。
- [x] inspect 的文件复核和命令合成。
- [x] 多清单 audit 报告。
- [ ] 更多引擎、签名与组织级管理。

不提供已上线的收费审计看板。

## 开发与许可证

```bash
go test ./...
```

Go 格式定义在 internal/manifest/manifest.go，GGUF 读取器在 internal/gguf/reader.go。

[MIT](LICENSE) · [Issues](https://github.com/SuperMarioYL/ggufbill/issues)
