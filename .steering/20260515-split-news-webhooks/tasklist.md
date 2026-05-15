# Tasklist: news_jp Webhook 分割

## タスク一覧

### [x] 1. Discord で Webhook URL を5つ発行する（ユーザー作業）

Discord サーバーで以下の5チャンネル分の Webhook を発行する。

| チャンネル用途 | 環境変数名 |
|---|---|
| ANNnewsCH 専用 | `DISCORD_WEBHOOK_NEWS_JP_1` |
| TBS NEWS DIG 専用 | `DISCORD_WEBHOOK_NEWS_JP_2` |
| FNN + テレ東BIZ | `DISCORD_WEBHOOK_NEWS_JP_3` |
| oricon + ウェザー + ABEMA + NewsPicks | `DISCORD_WEBHOOK_NEWS_JP_4` |
| PIVOT + BBC + その他 | `DISCORD_WEBHOOK_NEWS_JP_5` |

### [x] 2. `config/webhooks.env` に Webhook URL を追記する（ユーザー作業）

タスク1で発行した URL を `config/webhooks.env` に追記する。
完了後、既存の `DISCORD_WEBHOOK_NEWS_JP` は削除する。

### [ ] 3. `channels.csv` のカテゴリを更新する

`/Users/myzn/projects/footprints/rss/channels.csv` の対象チャンネルの
カテゴリ列を `news_jp` → `news_jp_1`〜`news_jp_5` に変更する。

### [ ] 4. `app.yaml` に新カテゴリの Webhook マッピングを追加する

`src/config/app.yaml` の `category_to_env` セクションに
`news_jp_1`〜`news_jp_5` のエントリを追加し、
既存の `news_jp` エントリを削除する。

### [ ] 5. `app.yaml` の `post_sleep_ms` を変更する

`rate_limit.post_sleep_ms` を `900` → `2000` に変更する。

### [ ] 6. 動作確認

コンテナ内で job を実行し、以下を確認する：
- 各 news_jp_1〜5 のチャンネルに通知が届く
- notify 時間が10分以内に収まる
- レート制限エラー（429）が発生しない

## 実装順序

タスク1・2はユーザー作業のため先に完了させる。
タスク3・4・5はファイル変更のみで順不同に実施可能。
タスク6は1〜5 完了後に実施する。
