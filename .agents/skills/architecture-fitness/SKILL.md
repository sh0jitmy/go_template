---
name: architecture-fitness
description: "モジュラーモノリスおよびレイヤードアーキテクチャの依存境界を静的解析（golangci-lint depguard）で強制し、OpenAPI スキーマの後方互換性（oasdiff 等）や循環依存・禁止パッケージ混入を CI で自動検証するアーキテクチャ適応度関数（Fitness Functions）の設計・運用に適用します。"
user-invocable: true
license: Apache-2.0
compatibility: Designed for Claude Code, Cursor, Antigravity, OpenCode, OpenClaw, and other AI coding agents.
metadata:
  author: "[YOUR_NAME]"
  version: "1.0.0"
  openclaw:
    emoji: "🏛️"
    requires:
      bins:
        - go
        - golangci-lint
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(make:*) Agent
---

> [!IMPORTANT]
> **ドキュメントで終わらせないアーキテクチャ:**
> `software-architecture` で定義したレイヤー境界やモジュラーモノリスの規約は、人間のレビュー頼みにせず **Fitness Function（自動検証ルール）** として `.golangci.yml` や CI に実装し、違反コードのマージを機械的に阻止します。

**Persona:** あなたは **アーキテクチャ・ガバナンス＆適応度関数（Fitness Functions）エンジニア** です。時間の経過によるアーキテクチャの腐敗（Architectural Drift）を、静的依存ルールと契約互換性チェックによって自動的に防ぎます。

# アーキテクチャ適応度関数（Architecture Fitness）指針

## 1. レイヤーおよびモジュール境界の依存ルール

本リポジトリでは以下の依存方向を原則とし、逆方向・飛び越えの import を禁止します。

```
cmd/{app,web}  (エントリポイント・DI配線)
    │
    ▼
internal/web   (HTTP/HTMX トランスポート・プレゼンテーション層)
    │
    ├───────────────┐
    ▼               ▼
internal/service  internal/database  (ビジネスロジック / 永続化・バックアップ層)
    │               │
    ▼               ▼
internal/domain   ent (ORM)          (純粋ドメイン層: 外部I/O・フレームワーク非依存)
```

| レイヤー / パッケージ | 禁止される依存先 | 理由 |
| :--- | :--- | :--- |
| 全パッケージ（生成コード除く） | `io/ioutil`, `github.com/pkg/errors`, `math/rand` | 非推奨 API 排除、`errors`/`math/rand/v2`/`crypto/rand` への統一 |
| `internal/service` | `github.com/gin-gonic/gin`, `net/http`, `internal/web`, `internal/database` | ビジネスロジックを HTTP フレームワーク・DB 起動詳細から分離 |
| `internal/database` | `github.com/gin-gonic/gin`, `net/http`, `internal/web`, `internal/service` | 永続化層から上位層への逆依存・循環参照を防止 |
| `internal/domain`（拡張時） | `github.com/gin-gonic/gin`, `net/http`, `entgo.io/ent`, `database/sql` | 純粋ドメインモデルをインフラ・ORM から完全隔離 |
| `internal/version` | 任意の外部・内部パッケージ | 循環参照ゼロの純粋メタデータパッケージを維持 |

---

## 2. `golangci-lint` (`depguard`) による静的強制

`.golangci.yml`（v2 形式）に `depguard` を有効化し、`make lint` 実行時に境界違反を即座にエラー化します。

```yaml
linters:
  enable:
    - depguard
  settings:
    depguard:
      rules:
        global:
          files:
            - "$all"
            - "!**/ent/**"
            - "!**/ogen/**"
          deny:
            - pkg: "io/ioutil"
              desc: "Use os or io package instead (deprecated since Go 1.16)"
            - pkg: "github.com/pkg/errors"
              desc: "Use standard errors package and fmt.Errorf with %w"
        service-layer:
          files:
            - "**/internal/service/*.go"
            - "!$test"
          deny:
            - pkg: "github.com/gin-gonic/gin"
              desc: "Service layer must not depend on HTTP framework (gin)"
            - pkg: "net/http"
              desc: "Service layer must not depend on net/http transport"
            - pkg: "github.com/shjtmy/go_sh0jitmy_template/internal/web"
              desc: "Service layer must not depend on web layer"
            - pkg: "github.com/shjtmy/go_sh0jitmy_template/internal/database"
              desc: "Service layer must not depend on database setup package"
```

### 新ドメイン（モジュラーモノリス）追加時のルール追加手順
新しいドメインパッケージ `internal/<domain>/` を追加した際は、以下をセットで実施します：
1. 公開 API（インターフェース・DTO）と内部実装（`internal/<domain>/internal/` または非公開型）を分離する。
2. `.golangci.yml` の `depguard.rules` に他ドメインの内部パッケージ参照禁止ルールを追加する。
3. `go list -f '{{.ImportPath}} -> {{.Imports}}' ./internal/...` で依存グラフに意図しないエッジがないか確認する。

---

## 3. OpenAPI 契約の後方互換性チェック（Breaking Change Detection）

公開 API（`api/openapi.yaml`）を変更する PR では、既存クライアントを破壊する変更（必須フィールド追加、フィールド削除、型変更、ステータスコード削除等）がないかを自動検証します。

```bash
# main ブランチの OpenAPI 定義との破壊的変更チェック（oasdiff）
go run github.com/oasdiff/oasdiff@latest breaking \
  <(git show origin/main:api/openapi.yaml) api/openapi.yaml
```

- 破壊的変更が必要な場合は、必ずパスバージョニング（例: `/v1/...` → `/v2/...`）と ADR（`docs/adr/`）を併せて作成します。

---

## 4. 定期監査チェックコマンド

```bash
# 1. レイヤー依存グラフの可視化確認
go list -f '{{.ImportPath}} -> {{.Imports}}' ./cmd/... ./internal/...

# 2. depguard を含む全静的解析の実行
make lint

# 3. 公開スキーマと生成コードの完全一致確認
make generate && git diff --exit-code
```

---

## 5. 陥りがちな罠（Traps）

| 罠 (Trap) | 起きる問題 | 回避策 |
| :--- | :--- | :--- |
| `depguard` に自動生成ディレクトリ（`ent/`, `ogen/`）を含めてしまう | 生成コード側の import で lint が失敗 | `files` に `!**/ent/**`, `!**/ogen/**` を明示する |
| テンプレート複製後にモジュールパスを変えた際、`depguard` の旧モジュール名を更新し忘れる | 内部レイヤー禁止ルールが効かなくなる | `go.mod` のモジュール名変更と同時に `.golangci.yml` の `depguard` パスも更新する |
| テストファイルで一時的に禁則 import を入れて `//nolint:depguard` を乱発する | 境界ルールが形骸化する | テスト用ヘルパーは `*_test.go` の別パッケージ（`package foo_test`）または `cmd/` 側 E2E に分離する |
