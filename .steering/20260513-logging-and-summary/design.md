# design.md

## 実装アプローチ

### 1. 通知時のログ追加

`internal/service/notify_service.go` の `Notify()` メソッド内、`dispatcher.send()` 呼び出し直前に1行追加する。

```go
log.Printf("notifying: [%s] %s - %s", category, v.ChannelName, v.Title)
```

### 2. タイミング計測とチャンネル別統計

`internal/controller/job_controller.go` を変更する。

#### 新規構造体

```go
// チャンネルごとの通知結果
type channelStat struct {
    ch       model.ChannelDTO
    newCount int   // ListNewVideos で返った件数
    notified int   // 通知成功件数
    failed   int   // 通知失敗件数
    fetchErr bool  // フェッチ自体がエラーだったか
}
```

#### RunOnce の変更

```
jobStart   := time.Now()
  Phase 1（フェッチ） → fetchElapsed
  Phase 2（通知）    → notifyElapsed
jobElapsed := time.Since(jobStart)
```

Phase 2 では `[]channelStat` を並行ゴルーチンが各自の index に書き込む（異なる index なのでロック不要）。

#### `jobController` フィールド追加

```go
ytRepo         *repository.YouTubeAPIRepository  // Metrics() 取得用（nil 許容）
summaryWebhook string                            // Discord サマリー webhook URL
```

`NewJobController` のシグネチャを以下に変更：

```go
func NewJobController(
    chRepo repository.ChannelRepository,
    fs     service.FeedService,
    ns     service.NotifyService,
    fetchSleep     time.Duration,
    ytRepo         *repository.YouTubeAPIRepository,
    summaryWebhook string,
) JobController
```

### 3. 最終サマリーログ

`printSummary()` ヘルパーを `jobController` のメソッドとして追加。

```
=== Job Summary ===
time: total=3m12s fetch=2m10s notify=1m02s
channels: total=50 with_new=12 errors=0
videos: new=45 notified=43 failed=2
  [camera_en] Sony Alpha: new=3 notified=3 failed=0
  [tech_jp] テクノロジーch: new=2 notified=2 failed=0
  ...
```

時間フォーマット用のヘルパー：

```go
func fmtDuration(d time.Duration) string  // "3m12s" / "45s" / "1h2m" 形式
```

### 4. Discord サマリー通知

`sendDiscordSummary()` メソッドを `jobController` に追加。

既存の `notifier.DiscordNotifier` を直接インスタンス化して `Send()` を呼ぶ（`notifySvc` は category ベースのため利用しない）。

Embed の `description` に以下を格納：

```
⏱ **実行時間**: 3m12s (フェッチ: 2m10s | 通知: 1m02s)

📺 **チャンネル**: 50件 (新着あり: 12件)
📼 **動画**: 新着45件 | 通知43件 | 失敗2件

**フィード**: RSS=45 API=5 (フォールバック: RSS→API=1 API→RSS=2 飽和=1)
**YouTube API**: リクエスト=5 クォータ=5       ← 使用時のみ
**リトライ**: 1件 (3回)                         ← 発生時のみ

**チャンネル別 (新着あり)**
• [camera_en] Sony Alpha: 3件通知
• [tech_jp] テクノロジーch: 2件通知
```

4000 文字を超えそうになったらチャンネル一覧を途中で打ち切り `• ...(以下省略)` を付ける。

`NotificationContent.URL` が空のときに Discord エラーにならないよう、`notifier.go` の embed 構築も修正する：

```go
// 変更前
embed["url"] = c.URL

// 変更後
if c.URL != "" {
    embed["url"] = c.URL
}
```

### 5. 設定追加

**`config/app.yaml`**

```yaml
summary_webhook_env: "DISCORD_WEBHOOK_SUMMARY"
```

**`config/config.go`**

`AppConfig` に `SummaryWebhookEnv string` フィールドを追加し、`applyTopLevel` でパース。

**`cmd/job/main.go`**

```go
summaryWebhook := ""
if cfg.SummaryWebhookEnv != "" {
    summaryWebhook = webhookSecrets[cfg.SummaryWebhookEnv]
    // 未設定はエラーにしない（サマリーはオプション機能）
}
job := controller.NewJobController(chRepo, feedSvc, notifySvc, fetchSleep, ytRepo, summaryWebhook)
```

また、YouTube API メトリクスのログ出力を `main.go` から削除し、`RunOnce` 内の `printSummary` / `sendDiscordSummary` に統合する。

---

## 変更するコンポーネント

| ファイル | 変更内容 |
|---|---|
| `internal/service/notify_service.go` | `Notify()` に通知ログ1行追加 |
| `internal/notifier/notifier.go` | 空 URL 時に `url` キーを省略 |
| `internal/controller/job_controller.go` | タイミング計測・チャンネル統計・サマリー出力・Discord 通知 |
| `internal/controller/job_controller_test.go` | `NewJobController` 呼び出しを新シグネチャに更新 |
| `config/config.go` | `SummaryWebhookEnv` フィールド追加 |
| `config/app.yaml` | `summary_webhook_env` キー追加 |
| `cmd/job/main.go` | summary webhook 取得・引き渡し、YouTube メトリクスログ削除 |

---

## 影響範囲

- **`docs/` への影響なし**：今回の変更はログ・運用改善であり、基本設計は変わらない。
- **インターフェース変更**：`NewJobController` のシグネチャが変わるため、テストの呼び出し箇所を更新する。
- **後方互換**：`summaryWebhook=""` かつ `ytRepo=nil` を渡せば既存動作と同一。
