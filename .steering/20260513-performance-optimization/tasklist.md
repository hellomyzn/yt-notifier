# タスクリスト — GitHub Actions 実行時間の削減

## タスク

- [x] `CachedNotifiedRepository` の実装（notified_repository.go）
- [x] `main.go` で `CachedNotifiedRepository` を使用するよう変更
- [x] `job_controller.go` を Phase 1（5並列フェッチ）に変更
- [x] `job_controller.go` を Phase 2（チャンネルごと並列通知）に変更
- [x] `.github/workflows/notify.yml` に Go キャッシュ有効化・バイナリ化を追加
- [x] `docs/` を CLAUDE.md の定義に合わせて再構成
- [x] `.steering/` ディレクトリの作成・本ドキュメントの整備

## 完了条件

- [x] コードが `go build ./...` でビルドエラーなし
- [x] `docs/functional-design.md` にシーケンス図（Mermaid）が記載されている
- [ ] GitHub Actions で実際に 10 分以内に完了することを確認
