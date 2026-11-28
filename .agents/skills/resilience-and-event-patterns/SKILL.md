---
name: resilience-and-event-patterns
description: "分散・モジュラーモノリス環境における高信頼実装パターン（Transactional Outbox、Idempotency-Key によるべき等 API、ジッター付き指数バックオフリトライ、サーキットブレイカー、バルクヘッド、Context タイムアウトバジェット伝播、縮退運転）を Go で設計・実装する際に使用します。"
user-invocable: true
license: Apache-2.0
compatibility: Designed for Claude Code, Cursor, Antigravity, OpenCode, OpenClaw, and other AI coding agents.
metadata:
  author: "[YOUR_NAME]"
  version: "1.0.0"
  openclaw:
    emoji: "🔄"
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Agent
---

> [!IMPORTANT]
> **障害は必ず起きる前提での設計 (Design for Failure):**
> 外部 API 呼び出し、DB 書き込み、モジュール間イベント連携は必ず「タイムアウト」「重複実行」「部分障害」が起きる前提で実装します。特に DB トランザクション内での外部通信は厳禁とし、Outbox パターンで一貫性を担保してください。

**Persona:** あなたは **レジリエンス＆イベント駆動アーキテクト** です。`software-architecture` で規定された自己修復・疎結合の要件を、Go + PostgreSQL/SQLite 上の堅牢な実装パターンへ落とし込みます。

# レジリエンス & イベント駆動実装指針

## 1. Transactional Outbox パターン（二重書き込み問題の解決）

DB 更新と外部イベント発行（または別モジュールへの非同期通知）を別々に行うと、片方だけ失敗してデータ不整合（Dual-Write Problem）が発生します。

### 正しいフロー
1. **同一 DB トランザクション内**で、業務テーブルの更新と `outbox_events` テーブルへの INSERT を行う（外部通信は行わない）。
2. バックグラウンドリレーワーカーが `outbox_events` から未送信イベントを取得し、宛先へ配信して送信済みマークを付ける。
3. 受信側（コンシューマ）は **At-Least-Once 配信**を前提とし、`event_id` による重複排除（べき等処理）を実装する。

```text
[Service Tx] ──(1 Tx)──▶ 業務テーブル UPDATE + outbox_events INSERT
                                                      │
[Outbox Worker] ──(Polling / LISTEN)──▶ 未送信イベント取得 ──▶ 宛先へ配信（リトライ付）
```

---

## 2. べき等性（Idempotency-Key）の設計

ネットワーク再送やクライアントリトライによって、POST 系更新 API（課金、登録、バックアップ作成、リストア等）が二重実行されないよう、更新系エンドポイントでは `Idempotency-Key` ヘッダのサポートを標準とします。

### 実装ルール
- **保存キー**: `(idempotency_key, endpoint, request_hash)` をユニーク制約付きテーブル（または TTL 付き Valkey）に保存。
- **パラメータ不一致検知**: 同じ `Idempotency-Key` で異なるリクエストボディが送られた場合は `409 Conflict` または `422 Unprocessable Entity` を返す。
- **処理中競合**: 同一キーが現在処理中の場合は `409 Conflict`（または `Retry-After` 付きレスポンス）を返す。
- **保持期間**: 最低 24 時間保持し、期限超過レコードは Retention Cleaner で自動パージする。

---

## 3. リトライ（指数バックオフ + ジッター）とリトライ対象分類

すべてのエラーを無条件にリトライしてはなりません（リトライストームの原因）。

| エラー分類 | リトライ可否 | 代表例 |
| :--- | :---: | :--- |
| **一時的障害（Transient）** | **可** | HTTP `429`, `502`, `503`, `504`、ネットワーク一時断、PostgreSQL Serialization Failure (`40001`)、SQLite `SQLITE_BUSY` |
| **恒久的障害（Permanent）** | **不可** | HTTP `400`, `401`, `403`, `404`, `409`, `422`、バリデーションエラー、`context.Canceled` |

### Full Jitter 付き指数バックオフ実装例
```go
func nextBackoff(base, maxDelay time.Duration, attempt int) time.Duration {
    delay := base << min(attempt, 10)
    if delay > maxDelay || delay <= 0 {
        delay = maxDelay
    }
    // Full Jitter: [0, delay) の一様乱数でリトライ集中（Thundering Herd）を分散
    return time.Duration(rand.Int64N(int64(delay)))
}
```
- 待機時は必ず `select { case <-timer.C: case <-ctx.Done(): return ctx.Err() }` とし、`context` キャンセル時に即時中断する。

---

## 4. サーキットブレイカー・バルクヘッド・タイムアウトバジェット

1. **タイムアウトバジェット伝播**:
   - 外部呼び出しごとに固定値タイムアウトを無造作に付けず、上流 `ctx.Deadline()` の残余時間を尊重する（残余時間が下限未満なら呼び出さずに即 Fail-Fast）。
   - `http.Client` はデフォルト（タイムアウト無制限）を禁止し、必ず `Timeout` および `http.Transport` の `DialContext` / `TLSHandshakeTimeout` / `ResponseHeaderTimeout` を明示設定する。
2. **サーキットブレイカー（Closed → Open → Half-Open）**:
   - 連続失敗や一定ウィンドウ内のエラー率超過で回路を Open にし、下流障害の連鎖伝播を遮断する。
   - Open 中は下流を叩かずに即エラー返却、または**縮退運転（Graceful Degradation: キャッシュ応答・読み取り専用応答）**へフォールバックする。
3. **バルクヘッド（隔壁）**:
   - 重要 API とバッチ処理・外部通知で DB コネクションプールや同時実行セマフォを分離し、低優先処理の滞留が高優先 API を道連れにしない構造にする。

---

## 5. 陥りがちな罠（Traps）

| 罠 (Trap) | 障害シナリオ | 回避策 |
| :--- | :--- | :--- |
| DB トランザクション内で外部 HTTP 呼び出しを実行 | 外部遅延で DB ロック・コネクション枯渇 | Transactional Outbox で DB コミット後に非同期送信 |
| ジッターなしの固定間隔リトライ | 障害復旧直後に全クライアントが一斉再送し再ダウン | Full Jitter + 上限回数 + `ctx.Done()` 監視を必須化 |
| `400 Bad Request` や `401` までリトライする | 無意味な負荷増幅とログ汚染 | 一時的エラー判定関数 `isRetryable(err)` を明示定義 |
| `http.DefaultClient` をそのまま使う | タイムアウト無制限で Goroutine が永久ハング | タイムアウト・接続上限を設定した専用 `*http.Client` を DI で注入 |
