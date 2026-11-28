---
name: docs-sync
description: "コード・設定・API・スキーマ・Makefile・CI・スキルの実装変更に合わせて、README.md、docs/architecture.md、AGENTS.md、CLAUDE.md、REQUIREMENTS.md などのドキュメントを常に最新状態へ自動同期・ドリフト検知する際に使用します。"
user-invocable: true
license: Apache-2.0
compatibility: Designed for Claude Code, Cursor, Antigravity, OpenCode, OpenClaw, and other AI coding agents.
metadata:
  author: "[YOUR_NAME]"
  version: "1.0.0"
  openclaw:
    emoji: "📚"
    requires:
      bins:
        - git
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(git:*) Bash(make:*) Agent
---

> [!IMPORTANT]
> **実装とドキュメントの常時同期原則 (Code-as-Truth / Zero Doc Drift):**
> 実装・設定・依存関係・スキルを変更したタスクでは、**コード修正だけで完了としてはなりません**。
> タスク完了前に必ず `git diff --name-only`（または `git status -s`）を確認し、下記の「実装→ドキュメント対応表」に該当するドキュメントの乖離（ドリフト）を検出・最新化してから作業を完了してください。

**Persona:** あなたは **ドキュメンテーション同期＆ドリフト監査エンジニア (Docs Sync Specialist)** です。実装（Code-as-Truth）とドキュメントの乖離を機械的・網羅的に検知し、常に「ドキュメントを読めば現在の実装を 100% 正確に理解・再現できる状態」を維持します。

# 実装連動ドキュメント常時同期指針 (`docs-sync`)

## 1. 実装変更 → 同期対象ドキュメント対応表（Trigger Matrix）

変更したファイルパスに応じて、対応するドキュメントを必ず点検・更新します。

| 変更された実装・設定ファイル | 必須チェック・更新先ドキュメント | 主な同期ポイント |
| :--- | :--- | :--- |
| `go.mod`（Go バージョン・主要依存） | `REQUIREMENTS.md`, `docs/deployment_guide_and_templates.md`, `.github/ISSUE_TEMPLATE/bug_report.yml` | サポート Go バージョン表記（最新 & 直前マイナー）、プレースホルダー |
| `Makefile`（ターゲット追加・変更・削除） | `README.md`, `AGENTS.md`, `CLAUDE.md` | 開発コマンド一覧表・コマンド説明の完全一致 |
| `cmd/**`, `internal/**`（構成・責務・新パッケージ） | `docs/architecture.md`, `.golangci.yml`（`depguard`） | コンポーネント構成図（Mermaid）、パッケージ責務、依存境界ルール |
| `api/openapi.yaml`（エンドポイント・認証・仕様） | `README.md`, `docs/architecture.md`, `docs/manual.md`（存在時） | API パス、認証方式、クイックスタートの検証 URL |
| `ent/schema/**`, `internal/database/**` | `docs/architecture.md`, `README.md`, `docker-compose.yml` | DB スキーマ・バックアップ/リストア・保持期間パージの挙動 |
| `internal/web/**`（UI・HTMX・SSG 変更） | `README.md`, `docs/architecture.md`, `docs/images/*` | スクリーンショット、UI 機能説明、SSG 出力手順（`documentation-governance` 併用） |
| `.golangci.yml`, `.github/workflows/**`, `.goreleaser.yaml` | `README.md`, `REQUIREMENTS.md`, `AGENTS.md` | CI/Lint/セキュリティ/リリースフローの説明、品質ゲート基準 |
| `.claude/skills/**`, `.agents/skills/**` | `README.md`, `REQUIREMENTS.md`（R-3.7）, `AGENTS.md`, `CLAUDE.md` | スキル総数、主要スキル一覧、`.claude/skills/` ⇔ `.agents/skills/` の同期（`make install-agents`） |
| `deploy/**`, `docker-compose.yml`, `terraform/**` | `docs/architecture.md`, `README.md` | 実ファイルパス、ポート番号、環境変数、Secrets 一覧 |
| 破壊的・不可逆な設計変更（One-Way Door） | `docs/adr/XXXX-<topic>.md` | 新規 ADR の作成、既存 ADR のステータス更新（`Superseded`） |

---

## 2. タスク完了前の自動ドリフト監査フロー

実装作業が一通り終わり、テスト・Lint が通ったら、必ず以下の 4 ステップを実行します。

### Step 1: 変更ファイルの抽出
```bash
git status -s && git diff --name-only
```
抽出されたパスを第 1 章の Trigger Matrix と照合し、更新候補ドキュメントをリストアップする。

### Step 2: 数値・バージョン・パス・コマンドの機械照合
実装の正本（Code-as-Truth）とドキュメント記述にズレがないか確認する：
- **Go バージョン**: `grep '^go ' go.mod` の値と `REQUIREMENTS.md` / Issue テンプレートの整合性
- **スキル総数と同期**:
  ```bash
  echo "claude=$(ls -1 .claude/skills | wc -l | tr -d ' ') agents=$(ls -1 .agents/skills | wc -l | tr -d ' ')"
  ```
  `.claude/skills/` と `.agents/skills/` の件数が一致し、`README.md` / `REQUIREMENTS.md` の数値と合致していること
- **Makefile ターゲット**: `Makefile` の `.PHONY` / ヘルプ出力と `README.md` / `AGENTS.md` / `CLAUDE.md` のコマンド名が実在すること
- **実ファイルパス**: ドキュメント内に記載された `deploy/...`、`scripts/...`、`internal/...` などのパスが実際にリポジトリ上に存在すること

### Step 3: 差分のみを最小・正確に反映
- **既存正本を更新する**: 新しい Markdown ファイルを無闇に増やさず、既存の正本（`README.md`, `docs/architecture.md`, `AGENTS.md`, `CLAUDE.md`, `REQUIREMENTS.md`）を更新する。
- **歴史的証跡は改ざんしない**: 過去の監査スナップショット（`docs/audit_report.md`, `docs/inspection_report.md`）や承認済み ADR の過去の決定本文は書き換えない。方針変更時は新規 ADR を追加する。
- **推測で書かない**: 必ず実コード・設定ファイルを読んで確認した事実だけを記載する。
- **Mermaid 図の構文安全性を保つ**: ノードラベル内の特殊文字（括弧 `()` や `/`）はダブルクォート `id["Label (Detail)"]` で囲み、HTML タグ（`<br/>`, `<code>`）は使わない。

### Step 4: 検証コマンドの実行
```bash
# スキルを変更した場合は同期＆検証
make install-agents && make check

# 要件チェックリストの適合率を再計算（100% 維持）
make self-eval
```

---

## 3. 既存スキルとの役割分担

| スキル | 担当領域 |
| :--- | :--- |
| **`docs-sync`（本スキル）** | **あらゆる実装・設定・スキル変更**に伴う技術ドキュメント全般のドリフト検知と常時同期 |
| `documentation-governance` | ユーザー体験（UX・画面操作・設定体験）が変化した際の操作マニュアル・FAQ の体験整合性レビュー |
| `architecture-decision-review` | 重要な設計変更前の多視点トレードオフ議論、ADR 起票、C4 図の設計レビュー |

---

## 4. 陥りがちな罠（Traps）

| 罠 (Trap) | 発生する問題 | 回避策 |
| :--- | :--- | :--- |
| `.claude/skills/` だけ更新して `.agents/skills/` を忘れる | Claude と Antigravity でスキル構成が乖離 | 必ず `make install-agents` を実行して件数一致を確認 |
| スキル数や Go バージョンを 1 箇所だけ直して他を見落とす | `README.md` と `REQUIREMENTS.md` で数値が矛盾 | `grep` でリポジトリ全体の旧表記を検索してから一括同期 |
| 存在しないパスやコマンド名を記憶だけで記載する | 初見開発者が手順通り実行できず失敗 | 実ファイル・`Makefile` を確認してから記載（Code-as-Truth） |
| 過去の `docs/audit_report.md` 等を現在状態へ上書きしてしまう | 監査証跡としての改ざん・履歴喪失 | 過去レポートは不変とし、現行ドキュメントと新 ADR を更新 |
