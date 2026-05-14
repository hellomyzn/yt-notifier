# Tasklist: GitHub Actions go.mod エラー修正

## タスク一覧

| # | タスク | 状態 |
|---|---|---|
| 1 | `notify.yml` の「Run notifier job」ステップに `working-directory: src` を追加 | [x] |
| 2 | `go test ./...` でテストが通ることを確認 | [x] |

## 完了条件

- `notify.yml` の「Run notifier job」ステップに `working-directory: src` が設定されている
- テストがすべて通る
