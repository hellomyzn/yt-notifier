# 機能設計書

## 機能ごとのアーキテクチャ

### レイヤー構成

```
cmd/job/main.go          依存解決・起動
internal/controller/     ジョブ全体のオーケストレーション
internal/service/        ビジネスロジック（フィード取得・通知）
internal/repository/     データアクセス（CSV・RSS・YouTube API）
internal/notifier/       Discord Webhook HTTP クライアント
internal/model/          DTO 定義
config/                  YAML・.env ファイルローダー
```

依存方向: `controller → service → repository`（逆依存禁止）

## コンポーネント設計

| コンポーネント | 役割 |
|---|---|
| `JobController` | Phase1（並列フェッチ）と Phase2（並列通知）を制御 |
| `FeedService` | RSS / YouTube API の切り替えと重複除外 |
| `NotifyService` | webhook ディスパッチ・レート制御・リトライ |
| `CachedNotifiedRepository` | 起動時に notified.csv を全件読込み、Has() を O(1) で提供 |
| `CSVChannelRepository` | channels.csv からチャンネル一覧を読込む |
| `RSSFeedRepository` | YouTube RSS XML をパースして VideoDTO を返す |
| `YouTubeAPIRepository` | playlistItems API でアップロード一覧を取得 |
| `DiscordNotifier` | Discord Embed Webhook POST・429 ハンドリング |

## データモデル定義

### channels.csv

| フィールド | 型 | 説明 |
|---|---|---|
| channel_id | string | YouTube チャンネル ID（UC...） |
| category | string | 通知カテゴリ（travel_jp, news_jp 等） |
| name | string | チャンネル名（任意） |
| enabled | bool | false の場合スキップ |
| fetch_limit | int | 15 以上で YouTube Data API を使用 |

### notified.csv

| フィールド | 型 | 説明 |
|---|---|---|
| video_id | string | YouTube 動画 ID（主キー相当） |
| channel_id | string | 動画のチャンネル ID |
| published_at | RFC3339 | 動画の公開日時 |
| notified_at | RFC3339 | 通知した日時 |

## シーケンス図

### 全体フロー（並列フェッチ・並列通知）

```mermaid
sequenceDiagram
autonumber
participant GH as GitHub Actions
participant Main as main.go
participant Cache as CachedNotifiedRepository
participant C as JobController
participant CH as ChannelRepository
participant FS as FeedService
participant FR as FeedRepository (RSS/API)
participant NS as NotifyService
participant WB as Webhook (Discord)

GH->>Main: 実行開始
Main->>Cache: NewCachedNotifiedRepository()
Cache->>Cache: notified.csv を全件読込 → map[videoID]bool
Cache-->>Main: 初期化完了
Main->>C: RunOnce()
C->>CH: ListEnabled()
CH-->>C: channels[]

Note over C,FR: Phase 1 — 5並列フェッチ（goroutine × fetchWorkers）
par goroutine 1
    C->>FS: ListNewVideos(ch1)
    FS->>FR: Fetch(ch1)
    FR-->>FS: videos[]
    FS->>Cache: Has(videoID) ×n  ※O(1)
    Cache-->>FS: true / false
    FS-->>C: newVideos[]
and goroutine 2
    C->>FS: ListNewVideos(ch2)
    FS->>FR: Fetch(ch2)
    FR-->>FS: videos[]
    FS->>Cache: Has(videoID) ×n  ※O(1)
    Cache-->>FS: true / false
    FS-->>C: newVideos[]
and goroutine N
    C->>FS: ListNewVideos(chN)
    FS-->>C: newVideos[]
end

Note over C,WB: Phase 2 — webhook 別並列通知（チャンネルごと goroutine）
par channel A → travel_jp webhook
    C->>NS: Notify(travel_jp, video)
    NS->>WB: POST webhook
    WB-->>NS: 2xx
    NS->>Cache: Append(videoID)
and channel B → news_jp webhook
    C->>NS: Notify(news_jp, video)
    NS->>WB: POST webhook
    WB-->>NS: 2xx
    NS->>Cache: Append(videoID)
end

C-->>Main: 完了ログ出力
Main-->>GH: 正常終了
```

### RSS / YouTube API 切り替えフロー

```mermaid
sequenceDiagram
autonumber
participant FS as FeedService
participant RSS as RSSFeedRepository
participant YT as YouTubeAPIRepository

alt fetch_limit >= 15 かつ API キーあり
    FS->>YT: FetchUploads(channelID, fetchLimit)
    alt API 成功
        YT-->>FS: videos[]
    else API 失敗（quota超過 or エラー）
        YT-->>FS: error
        FS->>RSS: Fetch(channelID)
        RSS-->>FS: videos[]（最大15件）
    end
else fetch_limit < 15
    FS->>RSS: Fetch(channelID)
    RSS-->>FS: videos[]
    alt 件数がぴったり15件（飽和の可能性）
        FS->>YT: FetchUploads(channelID, fetchLimit)
        alt API 成功
            YT-->>FS: videos[]（より多く取得）
        else API 失敗
            YT-->>FS: error
            Note over FS: RSS の結果をそのまま使用
        end
    end
end
```

### Discord レート制限ハンドリング

```mermaid
sequenceDiagram
autonumber
participant NS as NotifyService
participant D as webhookDispatcher
participant WB as Discord Webhook

NS->>D: send(content)
D->>D: nextAvailable まで待機（あれば）
D->>WB: POST /webhooks/...
alt 2xx 成功
    WB-->>D: 200 OK
    D->>D: nextAvailable = now + postSleepMS
    D-->>NS: ok
else 429 Too Many Requests
    WB-->>D: 429 + Retry-After ヘッダ or body.retry_after
    D->>D: time.Sleep(RetryAfter)
    D->>D: nextAvailable = now + RetryAfter
    D->>WB: POST（リトライ、attempts カウントなし）
    WB-->>D: 200 OK
    D-->>NS: ok（retries=1）
else その他エラー（5xx 等）
    WB-->>D: 5xx
    D->>D: time.Sleep(backoff)  ※2s→4s→8s→16s→30s 上限
    Note over D: attempts++ → maxRetries(5) で打ち切り
    D-->>NS: error
end
```

## ユースケース図

```mermaid
graph TD
    U[運用者] -->|channels.csv 編集| F1[チャンネル追加・削除]
    U -->|app.yaml 編集| F2[カテゴリ・webhook 設定変更]
    GH[GitHub Actions] -->|cron| F3[定期実行]
    F3 --> F4[RSS / YouTube API でフィード取得]
    F4 --> F5[新着動画の差分検知]
    F5 --> F6[Discord へ通知]
    F6 --> F7[notified.csv へ記録]
```
