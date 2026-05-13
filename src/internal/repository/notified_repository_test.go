package repository

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// ---- helpers ----

func writeNotifiedCSV(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "notified.csv")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

var (
	testPub      = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	testNotified = time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC)
)

// ---- NewCachedNotifiedRepository ----

func TestNewCachedNotifiedRepository_FileNotExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notified.csv")
	repo, err := NewCachedNotifiedRepository(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := repo.Has("any")
	if err != nil {
		t.Fatalf("Has() error: %v", err)
	}
	if got {
		t.Error("Has() = true; want false for empty repo")
	}
}

func TestNewCachedNotifiedRepository_LoadsExistingIDs(t *testing.T) {
	csv := "video1,ch1,2024-01-01T00:00:00Z,2024-01-01T00:01:00Z\nvideo2,ch2,2024-01-01T00:00:00Z,2024-01-01T00:01:00Z\n"
	path := writeNotifiedCSV(t, csv)

	repo, err := NewCachedNotifiedRepository(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		id   string
		want bool
	}{
		{"video1", true},
		{"video2", true},
		{"video3", false},
	}
	for _, tt := range tests {
		got, err := repo.Has(tt.id)
		if err != nil {
			t.Errorf("Has(%q) error: %v", tt.id, err)
		}
		if got != tt.want {
			t.Errorf("Has(%q) = %v; want %v", tt.id, got, tt.want)
		}
	}
}

// ---- Has ----

func TestCachedNotifiedRepository_Has(t *testing.T) {
	path := writeNotifiedCSV(t, "v1,ch1,2024-01-01T00:00:00Z,2024-01-01T00:01:00Z\n")
	repo, err := NewCachedNotifiedRepository(path)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		id   string
		want bool
	}{
		{"v1", true},
		{"v2", false},
		{"", false},
	}
	for _, tt := range tests {
		got, err := repo.Has(tt.id)
		if err != nil {
			t.Errorf("Has(%q) error: %v", tt.id, err)
		}
		if got != tt.want {
			t.Errorf("Has(%q) = %v; want %v", tt.id, got, tt.want)
		}
	}
}

// ---- Append ----

func TestCachedNotifiedRepository_Append_UpdatesCacheAndFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notified.csv")
	repo, err := NewCachedNotifiedRepository(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := repo.Append("v1", "ch1", testPub, testNotified); err != nil {
		t.Fatalf("Append() error: %v", err)
	}

	// キャッシュに反映されている
	got, _ := repo.Has("v1")
	if !got {
		t.Error("Has() after Append = false; want true")
	}

	// ファイルにも書き込まれている（新しいリポジトリで読み込んで確認）
	repo2, err := NewCachedNotifiedRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	got2, _ := repo2.Has("v1")
	if !got2 {
		t.Error("Has() on reloaded repo = false; want true")
	}
}

func TestCachedNotifiedRepository_Append_NoDuplicateNotification(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notified.csv")
	repo, _ := NewCachedNotifiedRepository(path)

	repo.Append("v1", "ch1", testPub, testNotified)
	repo.Append("v1", "ch1", testPub, testNotified)

	// Has が true なので重複通知を防げる
	got, _ := repo.Has("v1")
	if !got {
		t.Error("Has() = false; want true")
	}
}

// ---- 並列安全性 ----

func TestCachedNotifiedRepository_ConcurrentHas(t *testing.T) {
	path := writeNotifiedCSV(t, "v1,ch1,2024-01-01T00:00:00Z,2024-01-01T00:01:00Z\n")
	repo, _ := NewCachedNotifiedRepository(path)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			repo.Has("v1")
		}()
	}
	wg.Wait()
}
