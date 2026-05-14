# Requirements: GitHub Actions go.mod エラー修正

## 変更・追加する機能の説明

GitHub Actions の `notify` ワークフローで `/tmp/yt-notifier` バイナリ実行時に以下のエラーが発生している問題を修正する。

```
2026/05/14 08:45:16 go.mod not found from /
Error: Process completed with exit code 1.
```

## 原因

`src/cmd/job/main.go` の `repoRoot()` 関数は、現在の作業ディレクトリから親へと遡りながら `go.mod` を探す。

「Run notifier job」ステップには `working-directory` が指定されていないため、バイナリはリポジトリルート（`/home/runner/work/yt-notifier/yt-notifier`）から実行される。`go.mod` は `src/go.mod` に存在するが、リポジトリルートや親ディレクトリには存在しないためエラーになる。

「Build notifier」ステップには `working-directory: src` が設定されており、同様に「Run notifier job」ステップにも `working-directory: src` を設定することで、`repoRoot()` が `src/go.mod` を発見できるようになる。

## ユーザーストーリー

GitHub Actions の定期実行ワークフローが正常に完了し、YouTube の新着動画通知が Discord に届くようにしたい。

## 受け入れ条件

- `notify` ワークフローの「Run notifier job」ステップがエラーなく完了する
- `repoRoot()` が `src/` を正しく返し、設定ファイル・CSV ファイルのパスが正しく解決される

## 制約事項

- 既存の `src/src/csv/` パス構造（「Sync CSV」ステップが `src/src/csv/` を前提としている）を維持する
- バイナリのビルド方法・成果物パスは変更しない
