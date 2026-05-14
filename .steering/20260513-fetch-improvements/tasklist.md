# tasklist.md

## テスト実行環境
- Go はローカル未インストール。テストは `make test` / `make test-v`（Docker 経由）で実行する。
- Red → Green をまとめてから `make test` で確認する。

---

## タスク一覧

### T1. `internal/service/feed_service_test.go` — 改善1のテスト追加（Red）

- [x] **連続性ありのケース**: RSS 15件のうち一部が通知済み → API エスカレーションしないことを確認
- [x] **連続性なしのケース**: RSS 15件が全件未通知 → API エスカレーションすることを確認
- [x] 既存テストが新ロジックでも通ることを確認（修正が必要なら合わせて修正）

---

### T2. `internal/service/feed_service.go` — 改善1実装（Green）

- [x] `ListNewVideos` 内で dedup（通知済みフィルタ）を飽和判定の**前**に移動する
- [x] 飽和エスカレーション条件を `len(out) == len(videos)` （全件新着）に変更する
- [x] API エスカレーション後に再度 dedup を実施して `out` を更新する

---

### T3. `internal/service/feed_service.go` — 改善2: ログのチャンネル名表示

- [x] `log.Printf` の `ch.ChannelID` を `ch.Name` に変更（全4箇所）

---

### T4. `internal/service/feed_service.go` — 改善3: 新着動画数ログ

- [x] `return out, nil` の直前に `len(out) > 0` のときだけ新着数をログ出力する
  ```go
  if len(out) > 0 {
      log.Printf("fetched: [%s] %s - 新着%d件", ch.Category, ch.Name, len(out))
  }
  ```

---

### T5. `internal/controller/job_controller.go` — 改善4: 警告文 + リンク

- [x] `printSummary` のフェッチエラー行に警告文と YouTube チャンネルリンクを追加
- [x] `sendDiscordSummary` のフェッチエラー行を Discord Markdown リンク記法に変更
  - `[チャンネル名](https://www.youtube.com/channel/{channelID})`
  - 警告文「削除または非公開になった可能性があります」を追加

---

### T6. 動作確認

- [x] `make test` が通ること（全パッケージ Green） ✅

---

## 完了条件

- 全タスクのチェックボックスが埋まっている
- `make test` がエラーなし
- 改善1により API エスカレーション数が大幅に減少すること（次回実行で確認）
