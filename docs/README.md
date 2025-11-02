# 📌 要件定義サマリ

## 目的
- 100〜数百の YouTube チャンネルを 6時間ごとに巡回し、新着動画をカテゴリ別に Discord / Slack へ通知する。

## 運用モード
- GitHub Actions のみ（6時間ごと／cron: "0 */6 * * *"）。常駐コンテナは範囲外。

## 技術要件
- 言語：Go 1.24+
- アーキテクチャ：Controller / Service / Repository（インターフェース駆動）。必要に応じて DTO 使用。
- ストレージ：DB 不使用。通知済み管理・チャンネル定義は CSV（リポジトリ外・Git未管理）。
- 出力：Discord / Slack の Webhook 両対応（カテゴリ単位で切り替え）。
- フィード：YouTube RSS（APIキー不要）。
- カテゴリ：travel, news, など任意。カテゴリ→出力先（Discord/Slack）および Webhook キー名を config/app.yaml で定義。
- 実行制御：レート制限のため、取得間隔・投稿間隔を設定値で制御。

## 非機能要件
- 無料運用（GitHub Actions の無料枠想定、短時間ジョブ）。
- 冪等性：同一 videoId の重複通知防止（notified.csv で管理）。
- 可観測性：標準出力に JSON ライクなログ（成功/失敗件数、経過時間）。
- セキュリティ：Webhook URL は src/config/webhooks.env（Git未管理）に保存し、config/app.yaml はキー名のみを保持。

## 除外範囲（当面実施しない）
- データベース永続化。
- 秒〜分単位のリアルタイム通知。
- YouTube Data API 連携（統計情報など）。
