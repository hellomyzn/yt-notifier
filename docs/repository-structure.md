# リポジトリ構造定義書

## フォルダ・ファイル構成

```
yt-notifier/
├── .github/workflows/notify.yml   GitHub Actions ワークフロー
├── .steering/                     作業単位のステアリングドキュメント
├── docs/                          永続的ドキュメント
├── infra/docker/go/Dockerfile     ローカル開発用コンテナ
├── src/                           Go アプリケーション本体
│   ├── cmd/job/main.go            エントリポイント・DI
│   ├── config/                    設定ローダー（YAML・.env）
│   │   ├── app.yaml               メイン設定（カテゴリ・レート・フィルタ）
│   │   ├── webhooks.env           Discord Webhook URL（Git 未管理）
│   │   └── youtube.env            YouTube API キー（Git 未管理）
│   ├── internal/
│   │   ├── controller/            ジョブ制御（並列フェッチ・並列通知）
│   │   ├── service/               ビジネスロジック
│   │   ├── repository/            データアクセス（CSV・RSS・API）
│   │   ├── notifier/              Discord HTTP クライアント
│   │   └── model/                 DTO 定義
│   └── src/csv/                   実行時 CSV（Git 未管理）
│       ├── channels.csv           チャンネル定義
│       └── notified.csv           通知済み動画記録
├── docker-compose.yml
└── Makefile
```

## ディレクトリの役割

| ディレクトリ | 役割 |
|---|---|
| `src/cmd/job/` | DI の配線と起動のみ。ロジックを持たない |
| `src/config/` | YAML・.env のパース。外部 I/O の境界 |
| `src/internal/controller/` | フェッチと通知の並列制御。サービスを呼び出すだけ |
| `src/internal/service/` | フィード戦略（RSS/API 切替）・通知レート制御 |
| `src/internal/repository/` | ファイル・HTTP アクセスの実装。インターフェース経由で注入 |
| `src/internal/notifier/` | Discord HTTP の詳細。service からのみ利用 |
| `src/internal/model/` | `ChannelDTO`・`VideoDTO` の定義 |
| `docs/` | 永続的ドキュメント。設計方針が変わらない限り更新しない |
| `.steering/` | 作業ごとのステアリングドキュメント。完了後も履歴として保持 |

## ファイル配置ルール

- `src/config/webhooks.env` と `src/config/youtube.env` は `.gitignore` 対象（秘匿情報）
- `src/src/csv/*.csv` は `.gitignore` 対象（別リポジトリ "footprints" で管理）
- ドキュメントは `docs/` 直下に配置（サブディレクトリは作らない）
- シーケンス図は `docs/functional-design.md` 内に Mermaid で記載
