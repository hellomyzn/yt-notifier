## 📘 基本設計（docs/README.md）


**File: `docs/README.md`**


```markdown
# 基本設計 — yt-notifier


## 1. 概要
YouTube RSS を巡回し、新着動画をカテゴリごとに Discord / Slack へ通知する。GitHub Actions で6時間ごとに実行。


## 2. 機能一覧
- F1: チャンネル一覧の読込（CSV）
- F2: RSS取得（YouTubeチャンネル）
- F3: 差分検知（既通知CSVに存在しない videoId の抽出）
- F4: 通知（Discord/Slack）
- F5: 既通知の記録（CSV に追記）
- F6: 失敗時のリトライとサマリログ


## 3. システム構成
- 実行基盤：GitHub Actions（cron: 6h）
- アプリ：Go 1.24+ 単一バイナリ
- 設定：`src/config/app.yaml`
- データ：`src/src/csv/`（Git未管理）


## 4. データ設計（CSV）
### channels.csv
- `channel_id` (string)
- `category` (string)
- `name` (string, optional)
- `enabled` (bool)


### notified.csv
- `video_id` (string, pk)
- `channel_id` (string)
- `published_at` (RFC3339)
- `notified_at` (RFC3339)


## 5. 外部連携
- YouTube RSS: `https://www.youtube.com/feeds/videos.xml?channel_id={id}`
- Discord Webhook（Embed） / Slack Webhook（Blocks/Mrkdwn）


## 6. エラーハンドリング
- RSS/POST は最大3回リトライ（指数バックオフ）
- 失敗件数は最後にサマリ出力


## 7. レート制限
- 取得間隔：`fetch_sleep_ms`
- 投稿間隔：`post_sleep_ms`


## 8. セキュリティ
- Webhook は Git未管理の `src/config/webhooks.env` から取得
- `config/app.yaml` は Webhook キー名のみ保持


## 9. ログ仕様
- JSONライク：`{"level":"info","msg":"notified","category":"news","video_id":"..."}`


## 10. 拡張ポイント
- Shorts/Live/Premiere の判定強化
- `@handle` → `channel_id` 事前解決スクリプト
