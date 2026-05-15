# Requirements: news_jp Webhook 分割

## 背景・課題

最新のjobで35分かかっている。内訳：

```
time: total=35m14s  fetch=2m28s  notify=32m45s
```

notify が全体の93%を占めており、原因は以下の2点：

1. **Discord レート制限**: `post_sleep_ms: 900` により送信ペースが60件/分を超え、Discordの上限（30件/分/Webhook）に引っかかり、6〜7分の待機が4回発生している
2. **news_jp チャンネルの集中**: news_jp の13チャンネルが1つのWebhookに直列化され、合計308件の通知が詰まっている（全体466件の66%）

## 解決策

news_jp の Webhook を1つから5つに分割し、並列送信できる母数を増やす。

Discordのレート制限は Webhook URL 単位にかかるため、Webhook を増やすことで同時スループットが向上する（1×30件/分 → 5×30件/分）。

## ユーザーストーリー

- As a user, jobの実行時間を35分から5〜7分程度に短縮したい
- As a user, news_jpの全通知を欠損なく受け取りたい

## 受け入れ条件

- [ ] news_jp チャンネルが5つの Webhook に分散される
- [ ] 各 Webhook のスループットが Discord レート制限（30件/分）を超えない
- [ ] コード変更なし（設定ファイルと CSV の変更のみ）
- [ ] notify 時間が10分以内になる

## 変更スコープ

| ファイル | 変更内容 |
|---|---|
| `channels.csv`（本番: `footprints/rss/`） | news_jp チャンネルのカテゴリを `news_jp_1`〜`news_jp_5` に振り分け |
| `src/config/app.yaml` | `news_jp_1`〜`news_jp_5` の `category_to_env` エントリを追加 |
| `config/webhooks.env` | 新規 Webhook URL 5件を追加（ユーザーがDiscordで発行） |
| `src/config/app.yaml` | `post_sleep_ms` を 900 → 2000 に変更（レート制限防止） |

## 制約事項

- コードの変更は行わない
- `news_en`（The Wall Street Journal）は対象外
- 既存の `news_jp` カテゴリは廃止し、`news_jp_1`〜`news_jp_5` に完全移行
