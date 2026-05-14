# Design: GitHub Actions go.mod エラー修正

## 実装アプローチ

`.github/workflows/notify.yml` の「Run notifier job」ステップに `working-directory: src` を追加する。

### 変更前

```yaml
- name: Run notifier job
  env:
    TZ: Asia/Tokyo
  run: /tmp/yt-notifier
```

### 変更後

```yaml
- name: Run notifier job
  working-directory: src
  env:
    TZ: Asia/Tokyo
  run: /tmp/yt-notifier
```

## 変更するコンポーネント

| ファイル | 変更内容 |
|---|---|
| `.github/workflows/notify.yml` | 「Run notifier job」ステップに `working-directory: src` を追加 |

## パス解決の確認

`working-directory: src` で実行した場合の `repoRoot()` の動作：

1. `os.Getwd()` が `src/` を返す
2. `src/go.mod` が存在 → `src/` をルートとして返す

その後のパス解決：

| 用途 | コード | 解決パス（リポジトリルート基準）|
|---|---|---|
| 設定ファイル | `filepath.Join(root, "config", "app.yaml")` | `src/config/app.yaml` |
| Webhook env | `filepath.Join(root, "config", "webhooks.env")` | `src/config/webhooks.env` |
| CSV ディレクトリ | `filepath.Join(root, "src", "csv")` | `src/src/csv/` |

「Sync CSV from footprints」ステップが `src/src/csv/` を対象にしているため、CSV パスも整合する。

## 影響範囲の分析

- **影響あり:** `.github/workflows/notify.yml`（1行追加）
- **影響なし:** アプリケーションコード、設定ファイル、CSV、永続的ドキュメント
- **`docs/` への更新:** 不要（基本設計の変更なし）
