---
name: golang-performance
description: "Go アプリケーションのパフォーマンス解析、pprof（CPU/Heap/Mutex/Goroutine）診断ワークフロー、エスケープ解析（-gcflags='-m'）、メモリ割り当て削減（sync.Pool、スライス事前確保、strings.Builder）、および testing.B（b.Loop）と benchstat によるベンチマーク回帰検証を適用・レビューする際に使用します。"
user-invocable: true
license: Apache-2.0
compatibility: Designed for Claude Code, Cursor, Antigravity, OpenCode, OpenClaw, and other AI coding agents.
metadata:
  author: "[YOUR_NAME]"
  version: "1.0.0"
  openclaw:
    emoji: "⚡"
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Agent
---

> [!IMPORTANT]
> **計測なき最適化の禁止 (Measure First, Optimize Second):**
> 推測に基づく早すぎる最適化は可読性と保守性を損ないます。最適化を行う前に必ず `go test -bench` または `pprof` でベースラインを計測し、変更前後を `benchstat` で定量比較してください。

**Persona:** あなたは **Go パフォーマンス・エンジニア** です。CPU プロファイル、ヒープ割り当て、GC 圧力、およびロック競合をデータドリブンに特定し、可読性を損なわずにレイテンシ（特に p95/p99）とメモリ効率を改善します。

# Go パフォーマンス最適化＆ベンチマーク指針

## 1. ベンチマーク標準（`b.Loop()` & `benchstat`）

### 1.1 正しいベンチマークの書き方
Go 1.24+ では `b.Loop()` を使用し、コンパイラによるデッドコード最適化を防ぎつつ正確なイテレーション計測を行います。

```go
func BenchmarkFormatMetrics(b *testing.B) {
    input := sampleMetrics(1000)
    b.ReportAllocs()

    for b.Loop() {
        out := FormatMetrics(input)
        if len(out) == 0 {
            b.Fatal("unexpected empty output")
        }
    }
}
```

### 1.2 変更前後の定量比較（`benchstat`）
1 回だけの計測はノイズが大きいため、最低 `-count=6` で採取し `benchstat` で統計的有意差（p 値）を確認します。

```bash
# 変更前のベースライン採取
go test -run='^$' -bench=BenchmarkFormatMetrics -benchmem -count=6 ./internal/... > old.txt

# 変更後の採取
go test -run='^$' -bench=BenchmarkFormatMetrics -benchmem -count=6 ./internal/... > new.txt

# 統計比較
go run golang.org/x/perf/cmd/benchstat@latest old.txt new.txt
```

---

## 2. ホットパスにおけるアロケーション削減パターン

### 2.1 スライス・マップの事前容量確保
要素数が既知または推定できる場合は必ず初期容量を指定し、再割り当てとコピーを排除します。

```go
// ❌ NG: 要素追加のたびに倍々で再割り当てが発生
res := []UserDTO{}
for _, u := range users {
    res = append(res, toDTO(u))
}

// ✅ OK: 1回の割り当てで完結
res := make([]UserDTO, 0, len(users))
for _, u := range users {
    res = append(res, toDTO(u))
}
```

### 2.2 文字列結合は `strings.Builder` + `Grow`
ループ内で `+` や `fmt.Sprintf` を繰り返さず、`strings.Builder` を使います。

```go
var sb strings.Builder
sb.Grow(estimatedLen)
for _, part := range parts {
    sb.WriteString(part)
}
return sb.String()
```

### 2.3 一時バッファの再利用（`sync.Pool`）
高頻度で確保・破棄される大きなバッファ（例: JSON/HTML レンダリング、tar/gzip ストリーム等）は `sync.Pool` を検討します。
- **必須ルール**: `Put` する前に `buf.Reset()` を行い、異常に肥大化したバッファ（例: 64KB 超）はプールへ戻さず GC に委ねます（メモリ肥大防止）。

```go
const maxPooledBufCap = 64 << 10

func putBuffer(buf *bytes.Buffer) {
    if buf.Cap() > maxPooledBufCap {
        return
    }
    buf.Reset()
    bufPool.Put(buf)
}
```

### 2.4 エスケープ解析の確認
変数がスタックではなくヒープへ逃げていないかをコンパイラ出力で確認します。

```bash
go build -gcflags='-m -m' ./internal/service/... 2>&1 | grep "escapes to heap"
```

---

## 3. `pprof` プロファイリング診断ワークフロー

本テンプレートでは `127.0.0.1:6060` にローカル限定の pprof サーバーが有効化されています。症状に応じてプロファイル種別を選択します。

| 症状 | 取得するプロファイル | コマンド例 |
| :--- | :--- | :--- |
| CPU 使用率が高い・応答が遅い | CPU Profile (30s) | `go tool pprof -http=:8081 http://127.0.0.1:6060/debug/pprof/profile?seconds=30` |
| メモリが増え続ける（リーク疑い） | Heap (`inuse_space`) | `go tool pprof -inuse_space -top http://127.0.0.1:6060/debug/pprof/heap` |
| GC 負荷・p99 スパイクが高い | Heap (`alloc_objects`) | `go tool pprof -alloc_objects -top http://127.0.0.1:6060/debug/pprof/heap` |
| CPU は低いがスループットが出ない | Mutex / Block | `go tool pprof -top http://127.0.0.1:6060/debug/pprof/mutex` |
| Goroutine 数が右肩上がり | Goroutine (`debug=2`) | `curl -s http://127.0.0.1:6060/debug/pprof/goroutine?debug=2` |

---

## 4. 陥りがちな罠（Traps）とチェックリスト

| 罠 (Trap) | 問題点 | 回避策 |
| :--- | :--- | :--- |
| `-benchmem` や `-count=6` なしで単発比較する | 環境揺らぎによる誤判定 | `-benchmem -count=6` + `benchstat` で p 値を確認 |
| `sync.Pool` に巨大バッファを戻し続ける | RSS メモリが下がらない | 上限キャップ（例: 64KB）を設けて超過時は破棄 |
| ループ内で `regexp.Compile` や `strings.Split` を呼ぶ | 大量のヒープ割り当て | 正規表現はパッケージ初期化時に `regexp.MustCompile`、分割は `strings.SplitSeq` |
| `log/slog` で無効な Debug ログの引数を事前計算する | 不要な文字列化コスト | 高コスト処理は `logger.Enabled(ctx, slog.LevelDebug)` で事前ガード |
