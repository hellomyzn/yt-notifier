# yt-notifier

YouTubeのRSSを巡回し、カテゴリごとに Discord / Slack へ新着動画を通知するジョブ（Go 1.24+）。

## 特長
- 100〜数百チャンネル対応 / 6時間ごと実行（GitHub Actions）
- DB不使用、CSVで軽量運用（Git未管理）
- カテゴリ別に Discord/Slack を切替可能
- Controller / Service / Repository のクリーン構成

## ディレクトリ
src/config/          # app.yaml（カテゴリ→出力先／ENV名、レート、フィルタ）, webhooks.env（Webhook実体、Git未管理）
src/cmd/job/         # エントリポイント（RunOnceジョブ）
src/internal/        # controller, service, repository, notifier, model, config, util
src/src/csv/         # CSV置き場（Git未管理）
docs/                # 設計ドキュメント

## 前提
- Go 1.24+
- Webhook（Discord/Slack）のURLは src/config/webhooks.env に記載（Git未管理）

## セットアップ（ローカル）
```bash
cd src
go mod tidy
mkdir -p src/csv
# Webhook設定ファイルを作成
cp config/webhooks.env.example config/webhooks.env
# channels.csv / notified.csv を配置（Gitに含めない）
go run ./cmd/job
```

## GitHub Actions（6時間ごと）

- ワークフロー：.github/workflows/youtube-notify.yml
- webhooks.env のキー例：DISCORD_WEBHOOK_TRAVEL, SLACK_WEBHOOK_NEWS, DISCORD_WEBHOOK_TECH

## CSV スキーマ
```channels.csv
channel_id,category,name,enabled
UCxxxxxx1,travel,Backpacking Asia,true
UCyyyyyy2,news,World News Digest,true
```

```notified.csv
video_id,channel_id,published_at,notified_at
```
