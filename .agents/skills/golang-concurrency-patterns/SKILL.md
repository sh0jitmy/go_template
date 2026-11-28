---
name: golang-concurrency-patterns
description: "Go の高度な並行制御パターン（Bounded Concurrency / セマフォ / ワーカープール、errgroup.SetLimit、Graceful Shutdown の終了順序設計、および Go 標準 testing/synctest による決定論的並行テスト）を適用・レビューする際に使用します。"
user-invocable: true
license: Apache-2.0
compatibility: Designed for Claude Code, Cursor, Antigravity, OpenCode, OpenClaw, and other AI coding agents.
metadata:
  author: "[YOUR_NAME]"
  version: "1.0.0"
  openclaw:
    emoji: "🔀"
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Agent
---

> [!IMPORTANT]
> **構造化並行性 (Structured Concurrency):**
> ライフサイクル（誰が起動し、いつ・どうやって終了を待つか）が定義されていない `go func()` は禁止です。すべてのバックグラウンド処理は `context.Context` と完了待機（`errgroup` / `sync.WaitGroup`）に紐付け、`goleak` でリーク 0 を保証してください。

**Persona:** あなたは **Go 並行処理＆信頼性アーキテクト** です。無制限な Goroutine 生成によるメモリ枯渇、シャットダウン時のリクエスト欠損、および `time.Sleep` に依存した Flaky な並行テストを根絶します。

# Go 高度並行制御 & 決定論的テスト指針

## 1. Bounded Concurrency（並行数の上限制御）

入力件数や外部リクエストに比例して無制限に Goroutine を起動してはなりません。必ず同時実行数に上限を設けます。

### 1.1 `errgroup.SetLimit`（標準推奨）
バッチ処理や複数 API の並列呼び出しには `golang.org/x/sync/errgroup` と `SetLimit` を使用します。

```go
func ProcessItems(ctx context.Context, items []Item) error {
    g, gctx := errgroup.WithContext(ctx)
    g.SetLimit(8) // 同時実行数を最大 8 に制限

    for _, item := range items {
        g.Go(func() error {
            return processOne(gctx, item)
        })
    }
    return g.Wait()
}
```

### 1.2 キャンセル対応セマフォ（チャネルによるスロット制御）
コンテキストキャンセルを即時反映しながら同時実行数を制限するパターンです。

```go
select {
case sem <- struct{}{}:
    defer func() { <-sem }()
case <-ctx.Done():
    return ctx.Err()
}
```

---

## 2. Graceful Shutdown の標準シーケンス

サーバー終了時（`SIGINT` / `SIGTERM`）は、データ欠損や中途半端なコネクション切断を防ぐため、必ず以下の**依存逆順**で停止します。

```
1. シグナル受信 (signal.NotifyContext)
   ↓
2. Readiness を NotReady に切替（ロードバランサからの新規流入を停止）
   ↓
3. HTTP サーバーの Shutdown(shutdownCtx)（処理中リクエストの完了を待機）
   ↓
4. バックグラウンドワーカー・スケジューラの停止待機 (wg.Wait / errgroup.Wait)
   ↓
5. テレメトリ（OTel Tracer/MeterProvider）の Shutdown（未送信スパン・メトリクスの Flush）
   ↓
6. データベース接続プール（ent / sql.DB）の Close
```

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

// ... サーバー起動後、終了シグナルを待機 ...
<-ctx.Done()

shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()

// 1. HTTP Drain -> 2. Background Workers -> 3. OTel Flush -> 4. DB Close
_ = srv.Shutdown(shutdownCtx)
_ = tracerProvider.Shutdown(shutdownCtx)
_ = dbClient.Close()
```

---

## 3. `testing/synctest` による決定論的並行テスト（Go 1.25+）

タイムアウト、リトライ、バックオフ、Ticker などの時間依存ロジックをテストする際、実時間の `time.Sleep` に依存すると**遅く・Flaky なテスト**になります。Go 標準の `testing/synctest` を使い、仮想クロック（bubble）内で瞬時かつ決定論的に検証します。

```go
func TestRetryWithBackoff_Deterministic(t *testing.T) {
    t.Parallel()

    synctest.Test(t, func(t *testing.T) {
        ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
        defer cancel()

        attempts := 0
        err := RetryWithBackoff(ctx, 1*time.Second, func() error {
            attempts++
            if attempts < 3 {
                return errors.New("temporary failure")
            }
            return nil
        })

        require.NoError(t, err)
        assert.Equal(t, 3, attempts)
    })
}
```

### `synctest` 利用上の注意
- `synctest.Test` 内の goroutine はすべて終了させてからコールバックを抜けること（未終了 goroutine は即座にテスト失敗として検出される）。
- ネットワーク I/O や OS ファイル I/O などバブル外のイベントは仮想時間の進行をブロックするため、テスト対象はインターフェース経由でインメモリ化すること。

---

## 4. `singleflight` によるキャッシュスタンプード防止

同一キーに対する同時アクセスが集中するキャッシュミス時（Thundering Herd）は、`golang.org/x/sync/singleflight` でバックエンド問い合わせを 1 回に集約します。
- 共有される戻り値は呼び出し元間で破壊されないよう、不変値または防衛的コピーを返すこと。

---

## 5. 陥りがちな罠（Traps）

| 罠 (Trap) | 発生する障害 | 回避策 |
| :--- | :--- | :--- |
| `for _, item := range items { go work(item) }` | 大量入力時に OOM・DB 接続枯渇 | `errgroup.SetLimit(n)` で並行数を上限管理 |
| HTTP `Shutdown` より先に DB を `Close` する | 処理中リクエストが DB エラーで失敗 | HTTP Drain → Worker 待機 → OTel Flush → DB Close の順を厳守 |
| テスト内で `time.Sleep(200 * time.Millisecond)` で待機 | CI で高負荷時に Flaky 化 | `testing/synctest` またはチャネル同期で決定論的に検証 |
| `defer cancel()` を呼ばずに `context.WithTimeout` を生成 | タイマーリソースのリーク | 必ず直後に `defer cancel()` を記述（`govet` の `lostcancel` でも検出） |
