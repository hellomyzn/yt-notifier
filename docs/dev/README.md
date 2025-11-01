# 🧭 開発ルール（提案）

1. 命名とディレクトリ
- パッケージは用途で分割：controller, service, repository, notifier, model, config, util。
- インターフェース命名は XxxRepository, XxxService を基本、実装は CSVXxxRepository 等で表現。

2. 依存方向
- controller → service → repository。逆依存禁止。インターフェースで注入（DI）。
- notifier は service から利用し、repository と交差しない。

3. DTO / Entity
- 外部I/O（RSS, Webhook）や境界で使う構造体を model に DTO として定義。

4. 設定管理
- src/config/app.yaml にカテゴリ→出力先のマッピング、レート、フィルタ等を定義。
- Webhook の実値は 環境変数（Secrets）でのみ注入。リポジトリ内に秘匿情報を置かない。

5. CSV 運用
- src/src/csv/ 配下の CSV は Git管理対象外。.gitignore にパスを含める。
- スキーマ変更時は docs/README.md に反映。

6. ログ / エラー
- ログは INFO/ERROR とメタ（category, channel_id, video_id, elapsed_ms）を付与。
- 外部通信はリトライ（指数バックオフ最大3回）＋失敗件数をサマリ表示。

7. テスト方針
- Service はインターフェースをモックし ユニットテスト を優先（RSS擬似データ／Webhookモック）。
- リグレッション防止のため、normalizeVideoID はテストケースを充実。

8. コード規約
- golangci-lint 推奨。go fmt / go vet を CI に組込み。
- パブリック関数には Godoc コメントを付与。

9. PR 運用
- 小さな粒度で PR。設計更新は docs/* に先出し。

19. リリース / ローテーション
- notified.csv が肥大化した場合のローテ手順を docs/README.md に明記。
