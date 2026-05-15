# Design: news_jp Webhook 分割

## Webhook 振り分け方針

今日の実績新着数をベースに、各 Webhook の最大負荷が均等になるよう分割する。
高頻度チャンネル（ANNnewsCH, TBS NEWS DIG）はそれぞれ専用 Webhook を割り当てる。

## チャンネル振り分け

| カテゴリ | チャンネル名 | Channel ID | FetchLimit | 実績新着数/日 |
|---|---|---|---|---|
| **news_jp_1** | ANNnewsCH | UCGCZAYq5Xxojl_tSXcVJhiQ | 100 | 97 |
| **news_jp_2** | TBS NEWS DIG Powered by JNN | UC6AG81pAkf6Lbi_1VC5NmPA | 100 | 78 |
| **news_jp_3** | FNNプライムオンライン | UCoQBJMzcwmXrRSHBFAlTsIw | 100 | 44 |
| **news_jp_3** | テレ東BIZ | UCkKVQ_GNjd8FbAuT6xDcWgg | 100 | 30 |
| **news_jp_4** | oricon -Japanese entertainment news | UCbZvkG2uAgr6Oiva4FytscQ | 50 | 19 |
| **news_jp_4** | ウェザーニュース | UCNsidkYpIAQ4QaufptQBPHQ | 50 | 14 |
| **news_jp_4** | ABEMAニュース【公式】 | UCk5a240pQsTVT9CWPnTyIJw | 30 | 6 |
| **news_jp_4** | ABEMA Prime #アベプラ【公式】 | UCB1dgsqLiEp57oDAyNV_vww | 30 | 6 |
| **news_jp_4** | NewsPicks /ニューズピックス | UCfTnJmRQP79C4y_BMF_XrlA | 100 | 4 |
| **news_jp_5** | PIVOT 公式チャンネル | UC8yHePe_RgUBE-waRWy6olw | 50 | 5 |
| **news_jp_5** | BBC News Japan | UCCcey5CP5GDZeom987gqTdg | 10 | 2 |
| **news_jp_5** | PETAアジア | UCU3XsnaCQJ0ew_R2b6eVkdg | 10 | 2 |
| **news_jp_5** | TBS CROSS DIG with Bloomberg | UCeCmAYh1ylwIsgGrmqaklzg | 10 | 1 |
| **news_jp_5** | Newbee ~テクノロジーメディア~ | UCS0Nxf1j4Jtfx2i9qTksezg | 50 | 1 |
| **news_jp_5** | 851fmosaka | UC-BDgiEQMNQ5MRTAkbl6qlw | 10 | - |
| **news_jp_5** | ふぃるトピ! | UCxRdXnvaH3MsOzCc_deyeJQ | 10 | - |

## 期待される並列実行時間（post_sleep_ms=2000 の場合）

| Webhook | 最大件数/日 | 推定時間 |
|---|---|---|
| news_jp_1 | 97件 | 約194秒（3.2分） |
| news_jp_2 | 78件 | 約156秒（2.6分） |
| news_jp_3 | 74件 | 約148秒（2.5分） |
| news_jp_4 | 49件 | 約98秒（1.6分） |
| news_jp_5 | 11件以下 | 約22秒 |

→ 並列実行のため、ボトルネックは **news_jp_1 の約3.2分**

全体の job 時間（fetch 2.5分 + notify 約5分）≈ **約7分**（現在の35分から大幅短縮）

## 変更ファイルの詳細

### 1. `footprints/rss/channels.csv`

対象チャンネルのカテゴリ列を `news_jp` → `news_jp_1`〜`news_jp_5` に変更。

### 2. `src/config/app.yaml`

```yaml
# 追加
category_to_env:
  news_jp_1: "DISCORD_WEBHOOK_NEWS_JP_1"
  news_jp_2: "DISCORD_WEBHOOK_NEWS_JP_2"
  news_jp_3: "DISCORD_WEBHOOK_NEWS_JP_3"
  news_jp_4: "DISCORD_WEBHOOK_NEWS_JP_4"
  news_jp_5: "DISCORD_WEBHOOK_NEWS_JP_5"

# 変更
rate_limit:
  post_sleep_ms: 2000  # 900 → 2000
```

既存の `news_jp` エントリは削除（全チャンネルを移行済みのため）。

### 3. `config/webhooks.env`

Discord で5つの Webhook URL を発行後、以下を追記（ユーザー作業）：

```
DISCORD_WEBHOOK_NEWS_JP_1=https://discord.com/api/webhooks/...
DISCORD_WEBHOOK_NEWS_JP_2=https://discord.com/api/webhooks/...
DISCORD_WEBHOOK_NEWS_JP_3=https://discord.com/api/webhooks/...
DISCORD_WEBHOOK_NEWS_JP_4=https://discord.com/api/webhooks/...
DISCORD_WEBHOOK_NEWS_JP_5=https://discord.com/api/webhooks/...
```

## 影響範囲

- `news_en` は変更なし
- 他カテゴリは変更なし
- コードは変更なし
- 既存の `DISCORD_WEBHOOK_NEWS_JP` は webhooks.env から削除して良い（移行完了後）
