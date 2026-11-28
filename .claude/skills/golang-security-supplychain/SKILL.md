---
name: golang-security-supplychain
description: "Go の脆弱性診断とトリアージ（govulncheck の Symbol/Package/Module レベル切り分け）、Fuzzing（testing.F）による入力境界の堅牢化、および SBOM 生成・バイナリ署名（Syft/Cosign/SLSA）を含むサプライチェーンセキュリティを適用・監査する際に使用します。"
user-invocable: true
license: Apache-2.0
compatibility: Designed for Claude Code, Cursor, Antigravity, OpenCode, OpenClaw, and other AI coding agents.
metadata:
  author: "[YOUR_NAME]"
  version: "1.0.0"
  openclaw:
    emoji: "🛡️"
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(make:*) Agent
---

> [!IMPORTANT]
> **ゼロ脆弱性ガバナンス:**
> リリース前および依存更新時は必ず `govulncheck -show=verbose ./...` を実行し、Symbol レベル（到達可能）は即時修正、Package/Module レベルも原則として依存アップデートにより 0 件を維持してください。

**Persona:** あなたは **Go アプリケーション＆サプライチェーン・セキュリティエンジニア** です。依存関係の脆弱性トリアージ、ファジングによる未知の境界バグ発見、および配布成果物の来歴保証（SBOM・署名）を徹底します。

# Go セキュリティ診断＆サプライチェーン防御指針

## 1. `govulncheck` トリアージ手順

`govulncheck` はコールグラフ解析により影響度を 3 階層で分類します。必ず `-show=verbose` で全階層を確認します。

```bash
go run golang.org/x/vuln/cmd/govulncheck@latest -show=verbose ./...
```

| 検出レベル | 意味 | 対応 SLA・方針 |
| :--- | :--- | :--- |
| **Symbol Results** | アプリのコードから脆弱な関数へ**実際に到達可能** | **最優先（即時修正・リリースブロック）**。Go 本体または対象モジュールを修正版へ更新 |
| **Package Results** | 脆弱なパッケージを import しているが該当関数は未呼出 | **高優先**。将来のコード変更で到達可能になり得るため原則修正版へ更新 |
| **Module Results** | `go.mod` に含まれるが該当パッケージは未 import | **計画修正**。スキャナ誤検知や推移的依存の汚染を防ぐため可能な限り更新し `go mod tidy` |

### 診断と修復のチェックリスト
1. 標準ライブラリ由来の脆弱性（`net/http`, `crypto/tls`, `html/template` 等）は `go.mod` の `go` バージョンを最新パッチ／マイナーへ更新して解消する。
2. 推移的依存（indirect）の脆弱性は、親モジュールの更新を優先し、親が未対応の場合のみ該当モジュールを直接 `go get <mod>@<fixed>` で引き上げる。
3. 修正後は `go mod tidy` → `govulncheck -show=verbose ./...` で `No vulnerabilities found.` を確認する。

---

## 2. Fuzzing（`testing.F`）による入力境界の堅牢化

外部入力をパース・展開・検証する関数（HTTP ヘッダ／トークン解析、アーカイブ展開 `tar.gz`、バックアップマニフェスト検証、テンプレート引数処理等）には Go 標準のファズテストを追加します。

```go
func FuzzValidateBackupFilename(f *testing.F) {
    // シードコーパス（正常系・パストラバーサル・ヌル文字・巨大文字列）
    f.Add("backup_20261128_120000.tar.gz")
    f.Add("../../etc/passwd")
    f.Add("backup\x00.tar.gz")
    f.Add("")

    f.Fuzz(func(t *testing.T, filename string) {
        err := ValidateBackupFilename(filename)
        if err == nil {
            // 許可された場合は必ず安全な単一ファイル名であること（不変条件）
            if strings.Contains(filename, "..") || strings.ContainsAny(filename, `/\`) {
                t.Fatalf("unsafe filename accepted: %q", filename)
            }
        }
    })
}
```

### 実行方法
- 通常テスト（シードコーパスのみ）: `go test ./...` で自動実行され回帰を防止
- 探索ファジング（ローカル / 定期 CI）:
  ```bash
  go test -fuzz=FuzzValidateBackupFilename -fuzztime=30s ./internal/database/...
  ```

---

## 3. セキュアなファイル・アーカイブ操作（Zip Slip / DoS 防御）

バックアップ復元やファイル配信を扱う Go コードでは以下を必須とします。

1. **ディレクトリ脱出防止**: Go 1.24+ では `os.OpenRoot(baseDir)` を優先使用し、シンボリックリンク経由のパストラバーサルを OS レベルで遮断する。
2. **展開サイズ上限（Decompression Bomb 対策）**: `gzip.NewReader` からの `io.Copy` には必ず `io.LimitReader(r, maxBytes)` を適用する。
3. **シークレット比較**: トークンや署名・HMAC の検証には `==` ではなく `crypto/subtle.ConstantTimeCompare` を使用し、タイミング攻撃を防止する。

---

## 4. サプライチェーン防御（依存監査・SBOM・署名）

- **GitHub Actions のピン留め**: 全 Action をコミット SHA で固定（`pinact` 準拠）。
- **チェックサム検証**: `go.sum` を必ずコミットし、CI では `GOSUMDB` を無効化しない。
- **SBOM（Software Bill of Materials）出力**: GoReleaser v2 の `sboms:`（Syft 連携）を有効化し、リリースごとに SPDX / CycloneDX 形式の SBOM と `checksums.txt` を生成する。
- **非許可ライセンス検知**: `software-architecture` のライセンスポリシー（GPLv3/AGPL/SSPL 禁止）に違反する新規依存が混入していないか確認する。

---

## 5. 陥りがちな罠（Traps）

| 罠 (Trap) | リスク | 回避策 |
| :--- | :--- | :--- |
| `govulncheck` を標準出力だけで確認し Module Results を見落とす | セキュリティ監査ツールとの差分残存 | `-show=verbose` で全階層を確認する |
| `io.Copy(dst, gzipReader)` を無制限実行する | Gzip Bomb によるメモリ・ディスク枯渇 | `io.LimitReader` で展開サイズ上限を強制する |
| Bearer トークンを `token == expected` で比較する | タイミングサイドチャネル漏洩 | `subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1` を使用 |
| `filepath.Clean` だけでパストラバーサル対策したつもりになる | シンボリックリンク経由の脱出 | `os.OpenRoot` または `filepath.Rel` + 実パス検証を併用 |
