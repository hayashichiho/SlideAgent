# SlideAgent: CLI(MCP stdio) から Web アプリへの段階的進化ロードマップ

## 1. 目的と設計方針
- 目的: 現在の CLI ベース MVP を、最終的に Web アプリ（Frontend + API + Worker）へ段階的に拡張する。
- 維持する方針: MCP は「実行層（ツール層）」として残し、Web 側では Worker が MCP Client としてツールを呼ぶ。
- 非機能要件（Web化で追加）:
  - ジョブ化（非同期実行）
  - 保存先（永続化）
  - ダウンロード
  - プレビュー

## 2. 現状整理（2026-02-14 時点）
- `client` (Go) と `mcp-server` (Go) が stdio で JSON-RPC/MCP 通信できる。
- `tools/list` と `tools/call(make_pptx)` の配線は存在。
- `pptx-engine` は Node + PptxGenJS を利用。
- LLM は Gemini API（`GEMINI_API_KEY` 環境変数）。
- `make_diagram` / `render_png` は未実装（またはスタブ）。

## 3. ターゲットアーキテクチャ
```text
[Web Frontend]
   |
   | HTTPS
   v
[API Service (Go)] ---- [DB: jobs, artifacts, logs]
   |                              |
   | enqueue                      | metadata
   v                              v
[Queue/Job Store] ----------> [Object Storage (S3/MinIO/local)]
   |
   | dequeue
   v
[Worker (Go)] --(MCP Client over stdio)--> [MCP Server (Go)] --> [Node pptx-engine]
                                             |\
                                             | \--> make_diagram
                                             \----> render_png
```

## 4. ディレクトリ構成案
```text
SlideAgent/
  docs/
    slideagent-web-roadmap.md
  mcp-server/                # MCP実行層（ツール定義と実行）
  pptx-engine/               # Node/PptxGenJS実装
  services/
    api/                     # REST API（ジョブ受付・状態参照・署名URL発行）
    worker/                  # 非同期実行（MCP Client）
    frontend/                # Web UI（Next.js等）
  libs/
    mcpclient-go/            # Worker用のMCP client共通化
    jobmodel-go/             # Job/Artifactの型・状態遷移
  deploy/
    docker-compose.yml
    k8s/
  scripts/
  output/                    # ローカル開発の生成物
```

## 5. サービス責務分離
- Frontend:
  - ユーザー入力（ピッチ文、オプション）
  - ジョブ作成リクエスト
  - 進捗/状態表示
  - プレビュー表示（画像/PDF変換結果）
  - ダウンロード導線
- API Service:
  - 認証・認可（最低限トークン）
  - `POST /jobs` でジョブ受付
  - `GET /jobs/:id` で状態返却
  - `GET /jobs/:id/artifacts` で成果物一覧
  - ダウンロード用署名URL発行
- Worker:
  - キューからジョブ取得
  - Gemini 呼び出しで `slides_json` 生成
  - MCP Client として `make_pptx` / `make_diagram` / `render_png` 実行
  - 生成物アップロード、DB更新、リトライ処理
- MCP Server:
  - ツール実行の抽象化・標準化
  - Nodeエンジン呼び出しの責務を保持
- Node pptx-engine:
  - レンダリング実装専任（PPTX/図版生成）

## 6. フェーズ別ロードマップ

## Phase 0: CLI 安定化（基盤固め）
### ゴール
- CLI で「入力 -> slides_json -> make_pptx -> ファイル生成」まで再現性高く成功する。

### タスク
- `make_pptx` 実装を完成（必須引数検証、エラーメッセージ整備）。
- 出力ディレクトリを `output/` に統一。
- `client` の MCP 起動方式を一本化（内包起動 or 外部起動のどちらか）。
- Geminiレスポンスの堅牢化（JSON以外返却時の再試行/補正）。
- 最低限の構造化ログ（request_id/job_id相当）導入。

### 成果物
- CLI デモ手順（コマンド + 成功判定）
- `make setup`, `make server`, `make client` の安定実行
- エラーケース表（API key欠如、JSON不正、Node失敗）

### 検証方法
- 手動E2E: 5回連続実行で成功率 >= 90%
- 失敗時に原因が1画面で判別できること
- 生成された `.pptx` が実際に開けること

## Phase 1: 非同期ジョブ基盤（API + Worker の最小化）
### ゴール
- Web化前段として、CLI 直実行をやめ、ジョブキュー経由で非同期実行できる。

### タスク
- `services/api` を作成:
  - `POST /jobs`（入力受領）
  - `GET /jobs/:id`（状態: queued/running/succeeded/failed）
- `services/worker` を作成:
  - ジョブ取得 -> Gemini -> MCP `make_pptx` -> 保存
- 永続化層導入（初期は SQLite/Postgres どちらか）
- 保存先導入（初期はローカルFS、将来S3互換へ差し替え可能に）
- Job 状態遷移モデル定義（再試行回数、失敗理由、開始/終了時刻）

### 成果物
- API/Worker のローカル起動セット（docker-compose でも可）
- ジョブIDベースで生成完了を追跡可能
- 成果物パスをDBに保存

### 検証方法
- APIテスト: `POST /jobs` の入力バリデーション
- Worker統合テスト: 1ジョブ処理で `succeeded` になる
- リトライテスト: 一時失敗時に再実行される

## Phase 2: Web UI と成果物配信
### ゴール
- ユーザーがブラウザでジョブ投入、進捗確認、ダウンロード、プレビューできる。

### タスク
- `services/frontend` を実装:
  - ジョブ作成フォーム
  - ステータスポーリング/更新
  - 成果物一覧表示
- APIに成果物配信機能を追加:
  - `GET /jobs/:id/artifacts`
  - `GET /artifacts/:id/download-url`（署名URL）
- プレビュー対応:
  - `render_png` 実装、または worker 内で PPTX -> PNG/PDF 変換
  - サムネイル/ページ画像を保存
- 認証の最小導入（開発段階は固定ユーザーでも可）

### 成果物
- ブラウザ完結の最小UI
- PPTX ダウンロードリンク
- 1枚以上のプレビュー画像表示

### 検証方法
- E2E（Playwright等）: フォーム入力 -> 成功表示 -> DL可能
- 失敗系E2E: 不正入力・タイムアウト時のUI表示
- 生成物整合性: DBメタデータと実体が一致

## Phase 3: 本番運用対応（信頼性・スケール）
### ゴール
- 複数ユーザー運用に耐える、監視可能な Web サービスにする。

### タスク
- キュー基盤強化（Redis/RabbitMQ/SQS など）
- Worker 水平スケール（同時ジョブ数制御）
- オブジェクトストレージ本番化（S3/Cloud Storage）
- 観測性:
  - 構造化ログ
  - メトリクス（処理時間、失敗率）
  - トレース（API -> Worker -> MCP）
- セキュリティ:
  - APIキー管理（Secret Manager）
  - 認証認可（JWT/OAuth）
  - アップロード/ダウンロード権限
- 運用:
  - デプロイ手順（CI/CD）
  - マイグレーション手順
  - 障害時Runbook

### 成果物
- 本番相当の運用構成図
- SLO/SLA 指標（例: 95% のジョブが5分以内完了）
- 監視ダッシュボードとアラート

### 検証方法
- 負荷試験（同時ジョブ N 件）
- 障害注入（Gemini失敗、MCP失敗、ストレージ失敗）
- リカバリ試験（再実行・途中再開）

## 7. API/ジョブ設計の最小仕様（初版）
- Job status:
  - `queued`
  - `running`
  - `succeeded`
  - `failed`
- Job payload（例）:
```json
{
  "pitchText": "...",
  "options": {
    "language": "ja",
    "theme": "default"
  }
}
```
- Artifact metadata（例）:
```json
{
  "id": "art_...",
  "jobId": "job_...",
  "type": "pptx|png|pdf|json",
  "path": "s3://bucket/... or local path",
  "size": 12345,
  "createdAt": "..."
}
```

## 8. 実装順序（推奨）
1. Phase 0 完了（CLIの安定化）
2. Phase 1 で API/Worker を先に作る（UIより先）
3. Phase 2 で UI と配信機能を追加
4. Phase 3 で運用要件を満たす

## 9. 直近2週間の実行計画（サンプル）
- Week 1:
  - `make_pptx` 完了
  - 出力ディレクトリ統一
  - Job model（DBスキーマ）確定
  - API `POST /jobs`, `GET /jobs/:id` 実装
- Week 2:
  - Worker 実装（MCP client化）
  - 成果物保存とメタデータ登録
  - 最小Web画面（ジョブ投入 + 状態確認）

## 10. 受け入れ基準（Definition of Done）
- Web UI から投入したジョブが非同期で処理される。
- Worker が MCP Client として `make_pptx` を実行している。
- 生成成果物が永続保存され、ダウンロードできる。
- 少なくとも1種類のプレビューが表示できる。
- 失敗時にユーザーが原因を確認できるエラーメッセージが返る。
