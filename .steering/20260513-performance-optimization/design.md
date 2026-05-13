# 設計 — GitHub Actions 実行時間の削減

## 実装アプローチ

### 根本原因

1. `notifiedRepo.Has()` が呼ばれるたびに `notified.csv` を全件読込む（O(n)）
   - 547 チャンネル × 最大 15 動画 = 最大 8,200 回のファイル全読み込み
2. チャンネルを 1 件ずつ直列処理し、547 × 1.2s = 11 分がスリープで消費される
3. 通知フェーズも全チャンネル直列で、異なる webhook への通知が順番待ちになる

### 対策

| 対策 | 効果 |
|---|---|
| `CachedNotifiedRepository` 導入 | `Has()` を O(1) に改善、並列読み取り対応 |
| Phase 1: 5並列フェッチ | フェッチ時間を 1/5 に短縮（11分 → 2.2分） |
| Phase 2: チャンネルごと goroutine で通知 | 異なる webhook への通知を並列化 |
| GitHub Actions の Go キャッシュ有効化 | ビルド時間を短縮（cache: true） |

## 変更するコンポーネント

| ファイル | 変更内容 |
|---|---|
| `internal/repository/notified_repository.go` | `CachedNotifiedRepository` を追加 |
| `internal/controller/job_controller.go` | 並列フェッチ（Phase 1）と並列通知（Phase 2）に変更 |
| `cmd/job/main.go` | `CachedNotifiedRepository` を使用するよう変更 |
| `.github/workflows/notify.yml` | Go キャッシュ有効化・`go build` + 実行バイナリ化 |

## データ構造の変更

### CachedNotifiedRepository

```go
type CachedNotifiedRepository struct {
    inner *CSVNotifiedRepository
    mu    sync.RWMutex
    seen  map[string]bool  // 起動時に全件ロード
}
```

- `Has()`: RLock で並列読み取り対応、O(1)
- `Append()`: Lock で CSV 書き込みとキャッシュ更新を原子的に実施

## 影響範囲の分析

- **スレッドセーフ**: `Has()` は RWMutex、`Append()` は write lock で保護
- **同一 webhook への並列通知**: `webhookDispatcher.mu` が自動的に直列化するため安全
- **通知の順序**: Phase 2 でチャンネル間の順序は保証されないが、業務上問題なし
- **重複通知リスク**: 同一動画を複数チャンネルが返した場合、Phase 1 完了後に `Append()` が write lock を取るため重複しない
