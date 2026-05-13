# 技術仕様書

## テクノロジースタック

| 区分 | 技術 |
|---|---|
| 言語 | Go 1.24+ |
| 外部依存 | なし（stdlib のみ） |
| 実行基盤 | GitHub Actions（ubuntu-latest） |
| データ永続化 | CSV ファイル（別リポジトリ "footprints" で管理） |
| フィード取得 | YouTube RSS / YouTube Data API v3 |
| 通知先 | Discord Webhook（Embed 形式） |

## 開発ツールと手法

- **ローカル実行**: Docker Compose（`make up` / `make exec`）
- **ビルド**: `cd src && go build ./cmd/job`
- **テスト**: `cd src && go test ./...`
- **CI/CD**: GitHub Actions（`.github/workflows/notify.yml`）

## アーキテクチャ方針

- **インターフェース駆動 DI**: 全コンポーネントをインターフェース経由で注入し、差し替えを容易にする
- **外部依存ゼロ**: stdlib のみで実装し、依存管理コストを排除する
- **2フェーズ実行**:
  - Phase 1（並列フェッチ）: goroutine × `fetchWorkers`（デフォルト 5）で RSS / API を並列取得
  - Phase 2（並列通知）: チャンネルごとに goroutine を起動し、異なる webhook への通知を並列化。同一 webhook 内は `webhookDispatcher.mu` で直列化
- **インメモリキャッシュ**: 起動時に `notified.csv` を全件読込み `map[string]bool` に保持。`Has()` を O(1) に保つ

## 技術的制約と要件

| 制約 | 内容 |
|---|---|
| GitHub Actions 無料枠 | ubuntu-latest: 2,000 分/月。1 回の実行を 10 分以内に抑える |
| YouTube RSS 上限 | 1 チャンネルあたり最新 15 件のみ返却 |
| YouTube Data API クォータ | 10,000 ユニット/日。`list` は 1 ユニット/回 |
| Discord Webhook レート制限 | 30 リクエスト/60 秒/webhook URL。429 は `Retry-After` を遵守 |
| 秘匿情報 | Webhook URL・API キーは Git 未管理ファイル（`.env` 形式）で管理 |

## パフォーマンス要件

- 547 チャンネルのフェッチを **5 分以内**に完了（5並列 × 1.2s スリープ ≈ 2.2 分）
- `Has()` は O(1)（インメモリ map 参照）
- 通知フェーズは最も多い webhook の動画数 × `postSleepMS` で決まる
