# ユビキタス言語定義

## ドメイン用語

| 用語 | 定義 |
|---|---|
| チャンネル (Channel) | 通知対象の YouTube チャンネル。`channels.csv` の 1 行に対応 |
| カテゴリ (Category) | チャンネルのグループ。Discord への振り分け先を決める単位（例: `news_jp`, `tech_en`） |
| フィード (Feed) | YouTube RSS または YouTube Data API から取得した動画一覧 |
| 新着動画 (New Video) | フィードから取得した動画のうち、`notified.csv` に存在しないもの |
| 通知済み (Notified) | Discord への送信が成功し `notified.csv` に記録された動画 |
| 飽和 (Saturation) | RSS が上限の 15 件を返した状態。より多くの未取得動画が存在する可能性がある |
| フェッチ (Fetch) | チャンネルの動画一覧を RSS または YouTube API から取得する処理 |
| ディスパッチ (Dispatch) | Discord Webhook に対して HTTP POST を送る処理 |

## ビジネス用語

| 用語 | 定義 |
|---|---|
| fetch_limit | 1 チャンネルあたりの動画取得上限数。15 以上で YouTube Data API を使用 |
| fetch_sleep_ms | チャンネル間のフェッチ待機時間（ミリ秒）。YouTube のレート制限を避ける |
| post_sleep_ms | 通知間の待機時間（ミリ秒）。Discord のレート制限を避ける |
| Retry-After | Discord が 429 レスポンス時に返す待機秒数の指示 |

## 英語・日本語対応表

| 英語 | 日本語 |
|---|---|
| Channel | チャンネル |
| Category | カテゴリ |
| Feed | フィード |
| Video | 動画 |
| Notified | 通知済み |
| Webhook | ウェブフック |
| Saturation | 飽和 |
| Fetch | フェッチ（取得） |
| Dispatch | ディスパッチ（送信） |
| Rate Limit | レート制限 |

## コード上の命名規則

| コード名 | 意味 |
|---|---|
| `ChannelDTO` | channels.csv の 1 行を表すデータ転送オブジェクト |
| `VideoDTO` | RSS / API から取得した動画を表すデータ転送オブジェクト |
| `CachedNotifiedRepository` | notified.csv をインメモリキャッシュで高速化したリポジトリ実装 |
| `webhookDispatcher` | 1 つの Webhook URL に対するレート制御付き送信クライアント |
| `fetchWorkers` | Phase 1 の並列フェッチ数（デフォルト: 5） |
| `rssMaxWindow` | RSS が返す最大件数（固定値: 15） |
