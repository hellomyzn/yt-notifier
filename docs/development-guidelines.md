# 開発ガイドライン

## コーディング規約

- `go fmt` / `go vet` を必ず実行してからコミットする
- パブリック関数には Godoc コメントを付与する
- `golangci-lint` を推奨（CI への組込みも検討）

## 命名規則

| 対象 | 規則 | 例 |
|---|---|---|
| インターフェース | `XxxRepository` / `XxxService` | `FeedRepository`, `NotifyService` |
| 実装 | プレフィックスで区別 | `CSVNotifiedRepository`, `RSSFeedRepository` |
| パッケージ | 用途で分割 | `controller`, `service`, `repository`, `notifier`, `model`, `config` |

## 依存方向

```
controller → service → repository
                ↓
            notifier
```

- 逆依存禁止
- `notifier` は `service` からのみ利用し、`repository` と交差しない
- 全コンポーネントはインターフェース経由で注入（DI）

## スタイリング規約

- Go 標準の `gofmt` スタイルに従う
- 外部依存ライブラリは原則追加しない（stdlib で実現できる場合）

## テスト規約（TDD）

**実装前にテストを書く。** Red → Green → Refactor のサイクルで進める。

### ファイル配置

テストファイルは実装ファイルと同じディレクトリに置く。

```
internal/service/
├── feed_service.go
└── feed_service_test.go
internal/repository/
├── notified_repository.go
└── notified_repository_test.go
```

### モックの作り方

インターフェース単位でモックを定義し、依存先を差し替える。テスト用モックはテストファイル内に書く。

```go
type mockNotifiedRepo struct {
    seen map[string]bool
    appended []string
}
func (m *mockNotifiedRepo) Has(id string) (bool, error) { return m.seen[id], nil }
func (m *mockNotifiedRepo) Append(id, chID string, pub, notAt time.Time) error {
    m.appended = append(m.appended, id)
    return nil
}
```

### テーブル駆動テスト

複数ケースはテーブル駆動で書く（Go のイディオム）。

```go
func TestListNewVideos(t *testing.T) {
    tests := []struct {
        name  string
        seen  map[string]bool
        wantN int
    }{
        {"全て未通知", map[string]bool{}, 2},
        {"全て通知済み", map[string]bool{"v1": true, "v2": true}, 0},
        {"一部通知済み", map[string]bool{"v1": true}, 1},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ...
        })
    }
}
```

### テスト対象の優先順位

| レイヤー | テスト方針 |
|---|---|
| `service` | モックリポジトリを使ったユニットテストを優先 |
| `repository` | 純粋関数（`normalizeVideoID` 等）は必ずテストを書く。ファイル I/O は `t.TempDir()` を使用 |
| `controller` | モックサービスを使ってフェーズ制御をテスト |
| `notifier` | `httptest.NewServer` でモック HTTP サーバを立てる |

### 実行

```bash
make test                                          # internal/ 全体
make test-v                                        # 詳細出力
cd src && go test -v -run TestXxx ./internal/...   # 特定テストのみ
```

`make test` がパスすることを確認してからコミットする。

### カバレッジ

`cmd/job/` は DI 配線のみのためカバレッジ対象外。`internal/` を対象にする。

```bash
make coverage       # ターミナルに関数別カバレッジを表示
make coverage-html  # src/test/coverage/coverage.html を生成して確認
```

| フェーズ | `internal/` 全体の目標 |
|---|---|
| テスト導入初期 | 計測・レポートのみ（閾値なし） |
| テストが揃ってきたら | 60% 以上 |
| 安定期 | 80% 以上 |

カバレッジレポート（`src/test/coverage/`）は Git 管理対象外にする。

## Git 規約

- 小さな粒度でコミット・PR を作る
- 設計変更は `docs/` を先に更新してからコードを変更する
- コミットメッセージ: `<type>: <summary>`（例: `fix: prevent duplicate notification`, `update: parallel fetch`）
- 秘匿情報（Webhook URL・API キー）は絶対にコミットしない

## セキュリティ

- API キー・Webhook URL のハードコード禁止。必ず `.env` ファイル経由で渡す
- 外部 API（RSS・YouTube・Discord）のレスポンスは適切にバリデーションする
- 依存ライブラリを追加する際は `go list -m -json all | nancy` 等で脆弱性を確認する

## CSV 運用

- `src/src/csv/` は `.gitignore` 対象。別リポジトリ "footprints" で管理する
- スキーマ変更時は `docs/functional-design.md` のデータモデル定義も更新する
- `notified.csv` が肥大化した場合は古いレコードを削除するローテーション運用を検討する
