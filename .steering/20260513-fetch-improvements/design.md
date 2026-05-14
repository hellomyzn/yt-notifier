# design.md

## 改善 1: 不要な API エスカレーション削減

### 変更ファイル
`internal/service/feed_service.go`

### 現状のロジック
```
RSS 15件取得
→ len(videos) >= 15 なら API エスカレーション
```

### 変更後のロジック
```
RSS 15件取得
→ len(videos) >= 15 の場合:
    通知済みチェックを先に実施
    新着動画を特定する
    → 新着数 == 取得数（= 全件未通知）なら API エスカレーション
    → 新着数 < 取得数（= 通知済みが1件以上ある）なら API 不要
```

### 実装詳細

`ListNewVideos` 内の処理順序を変更する。

**現状:**
1. RSS 取得
2. 飽和判定 → API エスカレーション
3. 通知済みフィルタ（dedup）

**変更後:**
1. RSS 取得
2. 通知済みフィルタ（dedup）← 先に実施
3. 飽和 かつ 全件新着 → API エスカレーション
4. API 取得後も通知済みフィルタを再適用

```go
// 飽和エスカレーション条件の変更
// 変更前
if !useYouTube && s.ytRepo != nil && len(videos) >= rssMaxWindow {

// 変更後
if !useYouTube && s.ytRepo != nil && len(videos) >= rssMaxWindow && len(out) == len(videos) {
    // out = dedup後の新着、videos = RSS取得全件
    // 全件新着 = 連続性がない = API で追加取得が必要
```

API エスカレーション後は、API 結果に対して再度 dedup を行い `out` を更新する。

---

## 改善 2: フェッチログのチャンネル名表示

### 変更ファイル
`internal/service/feed_service.go`

### 変更箇所
`log.Printf` の `ch.ChannelID` を `ch.Name` に置換（全4箇所）。

```go
// 変更前
log.Printf("youtube api quota exceeded for channel=%s, ...", ch.ChannelID)
// 変更後
log.Printf("youtube api quota exceeded for channel=%s, ...", ch.Name)
```

---

## 改善 3: フェッチ後の新着動画数ログ出力

### 変更ファイル
`internal/service/feed_service.go`

### 実装詳細

`ListNewVideos` の return 直前に追加。新着0件はノイズになるため出力しない。

```go
if len(out) > 0 {
    log.Printf("fetched: [%s] %s - 新着%d件", ch.Category, ch.Name, len(out))
}
return out, nil
```

---

## 改善 4: フェッチ失敗チャンネルの警告 + リンク

### 変更ファイル
`internal/controller/job_controller.go`

### YouTube チャンネルリンク

```
https://www.youtube.com/channel/{channelID}
```

### printSummary の変更

```go
// 変更前
log.Printf("  [%s] %s: fetch error", s.ch.Category, s.ch.Name)

// 変更後
log.Printf("  [%s] %s: fetch error - 削除または非公開になった可能性があります (%s)",
    s.ch.Category, s.ch.Name,
    "https://www.youtube.com/channel/"+s.ch.ChannelID)
```

### sendDiscordSummary の変更

```go
// 変更前
line = fmt.Sprintf("• [%s] %s: ⚠ フェッチエラー\n", s.ch.Category, s.ch.Name)

// 変更後
line = fmt.Sprintf("• [%s] [%s](https://www.youtube.com/channel/%s): ⚠ 削除または非公開になった可能性があります\n",
    s.ch.Category, s.ch.Name, s.ch.ChannelID)
```

Discord の Markdown リンク記法 `[テキスト](URL)` を使用する。

---

## 変更するコンポーネント

| ファイル | 変更内容 |
|---|---|
| `internal/service/feed_service.go` | 改善1（エスカレーション条件）+ 改善2（ログ名前）+ 改善3（新着数ログ） |
| `internal/controller/job_controller.go` | 改善4（警告文 + リンク） |
| `internal/service/feed_service_test.go` | 改善1に対応するテスト追加・更新 |

---

## docs/ への影響

なし。アルゴリズムの最適化とログ改善であり、基本設計は変わらない。
