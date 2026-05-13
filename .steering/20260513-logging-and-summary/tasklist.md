# tasklist.md

## タスク一覧

### T1. `internal/notifier/notifier.go` — 空URL時に `url` キー省略
- [x] `embed["url"] = c.URL` を `if c.URL != "" { embed["url"] = c.URL }` に変更

---

### T2. `internal/service/notify_service.go` — 通知時のログ追加
- [x] `import` に `"log"` を追加
- [x] `dispatcher.send(content)` 呼び出し直前に `log.Printf("notifying: [%s] %s - %s", ...)` を追加

---

### T3. `config/config.go` — `SummaryWebhookEnv` フィールド追加
- [x] `AppConfig` に `SummaryWebhookEnv string` を追加
- [x] `applyTopLevel` に `case "summary_webhook_env": cfg.SummaryWebhookEnv = value` を追加

---

### T4. `config/app.yaml` — `summary_webhook_env` キー追加
- [x] ファイル末尾付近に `summary_webhook_env: "DISCORD_WEBHOOK_SUMMARY"` を追加

---

### T5. `internal/controller/job_controller.go` — 全面改修
- [x] `channelStat` 構造体を追加
- [x] `jobController` に `ytRepo *repository.YouTubeAPIRepository` と `summaryWebhook string` フィールドを追加
- [x] `NewJobController` シグネチャを更新（`ytRepo`, `summaryWebhook` 引数追加）
- [x] `RunOnce` にタイミング計測（`jobStart`, `fetchStart`, `notifyStart`）を追加
- [x] Phase 2 を `[]channelStat` でチャンネル別統計を収集するよう変更
- [x] YouTube API メトリクスログを `RunOnce` 内に移動（`main.go` から削除）
- [x] `printSummary()` メソッドを追加
- [x] `sendDiscordSummary()` メソッドを追加
- [x] `aggregateStats()` ヘルパーを追加
- [x] `fmtDuration()` ヘルパーを追加

---

### T6. `internal/controller/job_controller_test.go` — 新シグネチャに対応
- [x] `NewJobController(chRepo, feedSvc, notifySvc, 0)` → `NewJobController(chRepo, feedSvc, notifySvc, 0, nil, "")` に全箇所更新

---

### T7. `cmd/job/main.go` — summary webhook 取得・引き渡し
- [x] `cfg.SummaryWebhookEnv` から `summaryWebhook` を取得（未設定はスキップ、エラーにしない）
- [x] `NewJobController` 呼び出しを新シグネチャに更新（`ytRepo`, `summaryWebhook` を渡す）
- [x] YouTube API メトリクスログ（`ytRepo.Metrics()` の `log.Printf`）を削除

---

### T8. 動作確認
- [x] `make test` が通ること（Green 確認） ✅ 全パッケージ pass
- [ ] `make test-v` で各テストの詳細が確認できること

---

## テスト実行環境

- **Go はローカルに未インストール**。`go` コマンドは直接実行不可。
- テストは Docker コンテナ内で実行する。`make test` / `make test-v` を使う。
- TDD サイクルは以下の手順で行う：
  1. **Red**: テストコードを書く（この時点ではコンパイルエラーまたは失敗が期待される）
  2. **Green**: 実装コードを書く
  3. `make test` で通ることを確認する
- `make test` はコンテナ起動・停止を含むため時間がかかる。Red / Green の各フェーズをまとめてから1回実行する。

---

## 完了条件

- 全タスクのチェックボックスが埋まっている
- `make test` がエラーなし
