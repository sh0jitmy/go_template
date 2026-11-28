---
name: golang-modernize
description: "Go のバージョンアップグレード手順、go fix / modernize アナライザによる新イディオム適用（slices/maps、range over int/func、b.Loop、wg.Go、errors.AsType 等）、および Go 1.24+ の go.mod tool ディレクティブによるコード生成ツール管理（x/tools 不整合防止）を適用・レビューする際に使用します。"
user-invocable: true
license: Apache-2.0
compatibility: Designed for Claude Code, Cursor, Antigravity, OpenCode, OpenClaw, and other AI coding agents.
metadata:
  author: "[YOUR_NAME]"
  version: "1.0.0"
  openclaw:
    emoji: "🚀"
    requires:
      bins:
        - go
        - golangci-lint
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(make:*) Agent
---

> [!IMPORTANT]
> **ガバナンスと変更管理:**
> 本スキルファイルは組織の Go バージョン更新・モダナイゼーション標準指針です。Go のバージョンは常に `go.mod` を単一の信頼できる情報源（SSOT）とし、GitHub Actions（`go-version-file: 'go.mod'`）や `Dockerfile`（`GOTOOLCHAIN=auto`）と一貫性を維持してください。

**Persona:** あなたは **Go モダナイゼーション＆アップグレード・スペシャリスト** です。最新 Go ツールチェーンへの安全な移行、古いイディオムの自動刷新、および依存コード生成ツールの整合性維持を担当します。

# Go モダナイゼーション＆バージョンアップ指針

## 1. Go バージョンアップ標準手順（SSOT 遵守）

Go のバージョンを引き上げる際は、以下の順序で安全に移行します。

```bash
# 1. Go ツールチェーンと go.mod の更新（最新安定版へ）
go get go@<LATEST_VERSION>

# 2. 依存モジュールの更新と整合性整理
go get -u ./...
go mod tidy

# 3. 破壊的変更・自動移行チェック（go fix / modernize）
go vet ./...
go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -test ./...

# 4. コード生成・静的解析・脆弱性診断・テストの一括検証
make generate && make fmt && make lint && make vulncheck && make test && make sqlite-e2e
```

### バージョン同期チェック対象
- `go.mod`: `go X.Y.Z`（SSOT）
- `.github/workflows/*.yml`: `go-version-file: 'go.mod'` を使用し、バージョン直書きを禁止
- `Dockerfile`: `ENV GOTOOLCHAIN=auto` を維持
- `REQUIREMENTS.md` / `.github/ISSUE_TEMPLATE/bug_report.yml` / `docs/`: サポートバージョン表記を最新（最新マイナーおよび直前マイナー）へ同期

---

## 2. コード生成ツールのバージョン不整合対策（`//go:generate` & `tool`）

Go をメジャー/マイナーアップグレードした際、最も頻繁に起きる障害は **`go run pkg/cmd@vX.Y.Z` が内部で古い `golang.org/x/tools` を引き込み、新しい Go の型エクスポート形式を読めず失敗する現象**（例: `internal error: package "context" without types was imported`）です。

### 対策方針
1. **プロジェクト内モジュールと同じツール（例: `ent`）は `@version` を外す**:
   ```go
   // ❌ NG: ent@v0.14.6 側の古い go.mod（古い x/tools）が優先され、新しい Go で失敗しやすい
   //go:generate go run -mod=mod entgo.io/ent/cmd/ent@v0.14.6 generate ../../ent/schema

   // ✅ OK: プロジェクトの go.mod に解決済みの最新 golang.org/x/tools を利用する
   //go:generate go run -mod=mod entgo.io/ent/cmd/ent generate ../../ent/schema
   ```
2. **Go 1.24+ `tool` ディレクティブの活用（推奨）**:
   CLI ツール（`oapi-codegen`, `ent`, `govulncheck` 等）を `go get -tool <pkg>` で `go.mod` に登録し、`go tool <name>` で実行することで、キャッシュ共有とバージョン管理を `go.mod` に一元化できます。
   ```go
   // go.mod へ登録: go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
   //go:generate go tool oapi-codegen --config=../../api/oapi-codegen.cfg.yaml -o ../../ogen/api.gen.go ../../api/openapi.yaml
   ```

---

## 3. モダン Go イディオム（Go 1.22〜1.27+）

新規実装やリファクタリングでは、以下の標準ライブラリ・言語機能を優先採用します。

| 旧パターン | モダンパターン | 導入バージョン |
| :--- | :--- | :--- |
| `for i := 0; i < n; i++` | `for i := range n` | Go 1.22+ |
| `tt := tt` によるループ変数キャプチャ | 不要（ループ変数はイテレーション毎スコープ）※`paralleltest` linter 設定に準拠 | Go 1.22+ |
| 手書きのスライス探索・ソート・コピー | `slices.Contains` / `slices.SortFunc` / `slices.Clone` | Go 1.21+ |
| 手書きのマップ複製・キー抽出 | `maps.Clone` / `maps.Keys` / `slices.Collect(maps.Keys(m))` | Go 1.21〜1.23+ |
| `strings.Split` で一時スライス生成してループ | `for s := range strings.SplitSeq(str, ",")` | Go 1.24+ |
| `for i := 0; i < b.N; i++` | `for b.Loop()` | Go 1.24+ |
| `os.Open` + 独自パス検証のシンボリックリンク抜け | `os.OpenRoot(dir)` (`*os.Root`) によるディレクトリ隔離 | Go 1.24+ |
| `wg.Add(1); go func() { defer wg.Done(); ... }()` | `wg.Go(func() { ... })` | Go 1.25+ |
| `var target *MyErr; if errors.As(err, &target)` | `if target, ok := errors.AsType[*MyErr](err); ok` | Go 1.26+ |
| `math/rand`（非暗号用途の乱数） | `math/rand/v2`（シード不要・高速・安全な API） | Go 1.22+ |

---

## 4. 陥りがちな罠（Traps）とチェックリスト

| 罠 (Trap) | 影響 | 回避策 |
| :--- | :--- | :--- |
| `go.mod` だけ上げて `golangci-lint` が古い Go でビルドされている | 新構文で lint がクラッシュ | `golangci-lint --version` でビルド時の Go バージョンを確認し最新版へ更新 |
| `//go:generate` の `@vX.Y.Z` が古い `x/tools` を参照 | Go アップグレード後に `make generate` が失敗 | `go.mod` 解決（`@version` なし）または `go tool` に統一 |
| 自動生成コード（`ogen/`, `ent/`）をアップデート後に再生成し忘れる | CI の `Generate Code Diff Check` が失敗 | `make generate && git diff --exit-code` をローカルで確認 |
| `io/ioutil` や `interface{}` が残存 | 可読性低下・非推奨 API 警告 | `modernize` / `depguard` で自動検出・置換 |
