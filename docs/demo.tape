Output assets/demo.gif

Set Width 760
Set Height 360
Set FontSize 13
Set Padding 16
Set TypingSpeed 50ms

# ggufbill is a single static binary (built by CI / `go build -o ggufbill ./cmd/ggufbill`)
Type "./ggufbill version"
Enter
Sleep 1

# Emit a portable deployment manifest from a GGUF
Type "./ggufbill emit internal/gguf/testdata/qwen3-27b-q4_k_m.gguf --context 32768 --kv Q8_0 --vram 16 --gpu RTX-3060-12GB --operator ops-demo -o Qwen3-27B-Q4_K_M.ggufbill.json"
Enter
Sleep 2

# The sidecar is a portable, auditable JSON artifact
Type "jq .model, .runtime Qwen3-27B-Q4_K_M.ggufbill.json"
Enter
Sleep 2

# One file travels with the .gguf across boxes and rebuilds
Type "ls -lh Qwen3-27B-Q4_K_M.ggufbill.json"
Enter
Sleep 1

Type "rm -f Qwen3-27B-Q4_K_M.ggufbill.json"
Enter
Sleep 1
