---
name: golang-design
description: "Go (Golang) の設計プラクティス。プロジェクトのディレクトリ構成、パッケージ設計、構造体とインターフェースの設計（Accept interfaces, return structs等）、依存性注入 (DI) の設計方針を提案・適用する際に使用します。"
user-invocable: true
license: Apache-2.0
compatibility: Designed for Claude Code, Cursor, OpenCode, OpenClaw, and other AI coding agents.
metadata:
  author: [YOUR_NAME]
  version: "1.1.0"
  openclaw:
    emoji: "📁"
    homepage: https://github.com/samber/cc-skills-golang
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Agent AskUserQuestion
---

> [!IMPORTANT]
> **ガバナンスと変更管理:** 
> 本スキルファイルは組織の標準設計指針です。人間の明示的な指示がない限り、AIエージェント自身でこのファイルを書き換えないでください。変更は必ず Git のプルリクエストおよびアーキテクトによるレビューを経て行われます。

**Persona:** あなたは Go のソフトウェアアーキテクトです。過剰な抽象化や不要なレイヤーを排除し、シンプルでメンテナンス性の高い Go パッケージとインターフェースの構成を設計します。

# Go 設計ベストプラクティス (Go-Specific)

## 1. ディレクトリ構成 (Layout Conventions)
Goのプロジェクトでは、レイヤー単位ではなく機能/ドメイン単位でディレクトリを整理し、以下のディレクトリ規約を守ります。

- **cmd/{name}/**: エントリーポイント（`main.go`）のみを配置。ビジネスロジックは一切含めず、引数のパース、DIの初期化、起動処理のみを行う。
- **internal/**: 外部の他のレポジトリやモジュールからインポートされたくない非公開コード（サービス、データベースロジックなど）を配置。
- **pkg/**: 他の外部プロジェクトからも安全にインポートして再利用させたい公開パッケージのみを配置。

## 2. 構造体とインターフェース (Structs & Interfaces)
- **インターフェースは小さく**: メソッド数は極力 1〜3個に抑える（Go Proverbs: "The bigger the interface, the weaker the abstraction."）。大きなコントラクトは小さなインターフェースの合成で構築する。
- **Accept interfaces, return structs**: 関数の引数は柔軟性のためにインターフェースを受け取り、戻り値は呼び出し元が自由に扱えるよう具体的な構造体（通常はポインタ）を返す。
  ```go
  // ✅ 良い例: 具体的なポインタを返す
  func NewUserService(repo UserRepository) *UserService { ... }
  ```
- **定義場所は「消費側」**: インターフェースは、実装を提供するパッケージ側ではなく、それを利用する（呼び出す）側のパッケージで定義する。
- **早期のインターフェース作成の禁止**: 最初からインターフェースを作らず、2つ以上の異なる実装やテストでのモック化が必要になるまで具象型で進める。

## 3. 依存性注入 (Dependency Injection)
- **基本は手動DI**: コンストラクタ（`New...`）による明示的なパラメータ注入を優先する。
- **DIフレームワーク**: 規模が大きく依存配線が非常に複雑な場合は、コードジェネレータ方式の `google/wire`、またはコンテナ方式の `samber/do` などを検討する。

## 4. Webフレームワーク、CLIおよびORMの選定標準
- **Web API フレームワーク**:
  - Web API構築時の標準フレームワークとして `github.com/gin-gonic/gin` を採用し、ミドルウェア（リカバリー、ロギング、CORS等）を適切に計装すること。
- **CLI フレームワーク**:
  - CLI（Command Line Interface）ツール構築時の標準フレームワークとして `github.com/urfave/cli` を採用し、サブコマンド、フラグ、およびヘルプ出力を構造化すること。
- **ORM (Object-Relational Mapping)**:
  - データベースアクセスには、宣言的かつ型安全な `entgo.io/ent` を採用すること。
  - `ent` スキーマを明確に定義し、自動生成コード（`go generate`）を活用して、SQLインジェクションリスクを排除しつつ型安全にクエリを作成すること。
- **APIサーバー実装のファイル分離**:
  - Web API の実際の実装（OpenAPIのインターフェースを実装する `Server` 構造体や各エンドポイントのハンドラーメソッド）は、`handler.go` などの共通処理ファイルから分離し、`service.go` または機能ごとに `service_xxx.go`（`xxx` は機能・ドメイン名）というファイル名で定義すること。
  - `handler.go` にはマスキング処理やログ関連等の共通ヘルパー・デコレーターのみを記述し、サーバーの具体的なビジネスロジックは一切含めないこと。

## 5. ドメイン主導設計 (DDD) およびドメイン・コミュニケーション
- **ユビキタス言語とドメイン境界の合意**:
  - 開発着手前に、アーキテクト、開発者、および各分野のエキスパート（マーケター、PDM、SRE、DBA、QA等）の間で、プロダクトの「ユビキタス言語辞書」と「ドメインモデルの不変条件（ビジネスルール）」について合意を形成し、提案書やドキュメントに明文化すること。
- **ドメイン層の隔離**:
  - ビジネスロジックをインフラストラクチャ層（DB, 通信）から完全に分離し、`internal/domain/` 以下のピュアな Go パッケージとして表現すること。
  - テレメトリのデータ間引きなどの主要なアルゴリズムは、DBやWebサーバー of API呼び出しから完全に分離されたドメインサービスまたはエンティティメソッドとして実装すること。

## 6. HTTPS前提のアーキテクチャ設計 (SA主導)

本番環境におけるすべての通信はHTTPS（TLS）を前提とします。ソフトウェアアーキテクト（SA）は通信全体の暗号化と検証を強制する設計を行います。

- **プロキシ終端 (TLS Offloading) の検証**:
  リバースプロキシやロードバランサ等で TLS を終端し、アプリには HTTP でパケットをフォワードする場合、アプリ側で信頼されたプロキシからのヘッダ（`X-Forwarded-Proto: https`）が存在し、値が `https` であることを強制します。
  ```go
  func RequireHTTPS(next http.Handler) http.Handler {
      return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
          proto := r.Header.Get("X-Forwarded-Proto")
          if proto != "https" && r.TLS == nil {
              // HTTPSへのリダイレクト
              target := "https://" + r.Host + r.URL.Path
              if r.URL.RawQuery != "" {
                  target += "?" + r.URL.RawQuery
              }
              http.Redirect(w, r, target, http.StatusMovedPermanently)
              return
          }
          next.ServeHTTP(w, r)
      })
  }
  ```
- **HSTSヘッダの適用**:
  ブラウザに対してHTTPSでの通信を強制するため、`Strict-Transport-Security` ヘッダをレスポンスに付与します。
  ```go
  w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
  ```
