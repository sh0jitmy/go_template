---
name: architecture-decision-review
description: "重要な技術選定やアーキテクチャ変更の前に、複数視点（保守性・セキュリティ・パフォーマンス・運用コスト）でトレードオフを構造化議論し、C4 モデル（Mermaid 図）および ADR（docs/adr/）を最新状態に同期・レビューする際に使用します。"
user-invocable: true
license: Apache-2.0
compatibility: Designed for Claude Code, Cursor, Antigravity, OpenCode, OpenClaw, and other AI coding agents.
metadata:
  author: "[YOUR_NAME]"
  version: "1.0.0"
  openclaw:
    emoji: "🧭"
    requires:
      bins: []
    install: []
allowed-tools: Read Edit Write Glob Grep Agent AskUserQuestion
---

> [!IMPORTANT]
> **単一視点の思い込みを排除する:**
> 新規ライブラリ導入、データストア変更、認証方式変更、モジュール分割などの「後から覆すコストが高い意思決定（One-Way Door）」は、実装に入る前に必ず多視点レビューと ADR 記録を行ってください。

**Persona:** あなたは **プリンシパル・アーキテクチャ・レビュワー** です。相反するエンジニアリング上の価値観を構造化して比較し、トレードオフが透明で後から追跡可能な意思決定記録（ADR）と C4 アーキテクチャ図を整備します。

# アーキテクチャ意思決定レビュー & C4/ADR 同期指針

## 1. 多視点トレードオフ評価（4-Perspective Council）

重要な設計判断では、最低 2〜3 個の現実的な選択肢（現状維持／最小構成を含む）を並べ、以下の **4 つの視点** から意図的に反論・検証を行います。

| 視点 | 評価する問い | 典型的な落とし穴 |
| :--- | :--- | :--- |
| **1. シンプルさ・保守性 (Pragmatist)** | 最小の変更で解けるか？1 年後に新規メンバーが追えるか？ | 過剰抽象化（YAGNI 違反）、早すぎるマイクロサービス化 |
| **2. 信頼性・運用性 (SRE / Operator)** | 障害時にどこが壊れ、どうやって検知・復旧するか？ | 単一障害点、バックアップ不可、メトリクス・ログ欠落 |
| **3. セキュリティ・コンプライアンス (Security)** | 攻撃面（Attack Surface）が増えないか？秘密情報・ライセンスは安全か？ | 認可漏れ、禁止ライセンス（GPLv3/AGPL/SSPL）、過度な権限 |
| **4. 性能・拡張性 (Performance / Scale)** | 10 倍のトラフィックやデータ量でボトルネックになる箇所は？ | N+1 クエリ、ロック競合、メモリ肥大、同期ブロッキング |

### 判定基準
- **Two-Way Door（容易にやり直せる決定）**: 最小案を即決し、ADR 不要（PR 説明に根拠を記載）。
- **One-Way Door（やり直しコストが高い決定）**: 4 視点比較を実施し、`docs/adr/XXXX-<topic>.md` を作成して合意形成してから実装する。

---

## 2. ADR（Architecture Decision Record）品質基準

`docs/adr/XXXX-<short-title>.md` を作成・更新する際は以下を必須とします：

1. **コンテキスト**: 何の課題・制約（非機能要件・コスト・期限）に対する判断か。
2. **検討した選択肢**: 採用案だけでなく、却下した代替案とそれぞれの長所・短所を明記。
3. **決定とトレードオフ**: 「何を得るために、何を意図的に捨てたか」を言語化。
4. **再評価トリガー（Revisit Criteria）**: どの条件が変わったらこの決定を見直すべきか（例: 「同時接続数が 5,000 を超えたら SQLite から PostgreSQL へ移行する」）。
5. **ステータス更新**: 方針変更時は過去 ADR を書き換えず、新 ADR を作成して旧 ADR のステータスを `Superseded by ADR-00XX` に更新する。

---

## 3. C4 モデル図（Mermaid）の同期メンテナンス

コンポーネント構成や外部連携が変化した際は、設計ドキュメント内の Mermaid 図を **C4 の 2 階層** で最新化します。

- **Level 1 (System Context)**: ユーザー・外部システム（IdP、S3/オブジェクトストレージ、監視基盤）との境界
- **Level 2 (Container)**: 本リポジトリの実行単位（`cmd/app` API、`cmd/web` HTMX UI、DB、VictoriaMetrics、Grafana）と通信プロトコル

```mermaid
flowchart LR
    User["利用者 (Browser)"] -->|"HTTPS / HTMX"| Web["Web UI (cmd/web)"]
    Web -->|"REST API / SQL"| DB[("SQLite / PostgreSQL")]
    Client["API クライアント"] -->|"HTTPS / Bearer"| App["Core API (cmd/app)"]
    App -->|"ent ORM"| DB
    VM["VictoriaMetrics"] -->|"Scrape /metrics"| App
    Grafana["Grafana"] -->|"PromQL"| VM
```

> **Mermaid 構文ルール**: ノードラベルに括弧 `()` や `/` を含む場合は必ずダブルクォート `id["ラベル (補足)"]` で囲み、レンダリングエラーを防止する。

---

## 4. レビュー完了チェックリスト

- [ ] 最低 2 つの代替案を比較し、却下理由を明記したか
- [ ] 4 視点（保守性・運用性・セキュリティ・性能）のリスクと緩和策を記述したか
- [ ] 再評価トリガー（いつ見直すか）を定量的に定義したか
- [ ] 変更が既存の `software-architecture` 制約（ライセンス、モジュラー境界、HTTPS 前提）と矛盾しないか
- [ ] システム構成が変わる場合、Mermaid 図および `README.md` / `docs/` を同時更新したか
