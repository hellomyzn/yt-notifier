package repository

import (
	"encoding/csv"
	"os"
	"sync"
	"time"
)

type NotifiedRepository interface {
	Has(videoID string) (bool, error)
	Append(videoID, channelID string, publishedAt, notifiedAt time.Time) error
}

// CachedNotifiedRepository wraps CSVNotifiedRepository with an in-memory map.
// Has() is O(1) and safe for concurrent reads; Append() is serialized via a mutex.
type CachedNotifiedRepository struct {
	inner *CSVNotifiedRepository
	mu    sync.RWMutex
	seen  map[string]bool
}

func NewCachedNotifiedRepository(path string) (*CachedNotifiedRepository, error) {
	inner := &CSVNotifiedRepository{Path: path}
	seen, err := inner.loadAll()
	if err != nil {
		return nil, err
	}
	return &CachedNotifiedRepository{inner: inner, seen: seen}, nil
}

func (r *CachedNotifiedRepository) Has(videoID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.seen[videoID], nil
}

func (r *CachedNotifiedRepository) Append(videoID, channelID string, publishedAt, notifiedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.inner.Append(videoID, channelID, publishedAt, notifiedAt); err != nil {
		return err
	}
	r.seen[videoID] = true
	return nil
}

type CSVNotifiedRepository struct{ Path string }

func (r *CSVNotifiedRepository) loadAll() (map[string]bool, error) {
	f, err := os.Open(r.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]bool{}, nil
		}
		return nil, err
	}
	defer f.Close()

	cr := csv.NewReader(f)
	cr.FieldsPerRecord = -1
	rows, err := cr.ReadAll()
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(rows))
	for _, row := range rows {
		if len(row) > 0 {
			seen[row[0]] = true
		}
	}
	return seen, nil
}

func (r *CSVNotifiedRepository) Has(videoID string) (bool, error) {
	seen, err := r.loadAll()
	if err != nil {
		return false, err
	}
	return seen[videoID], nil
}

func (r *CSVNotifiedRepository) Append(videoID, channelID string, publishedAt, notifiedAt time.Time) error {
	if err := ensureFile(r.Path); err != nil {
		return err
	}
	f, err := os.OpenFile(r.Path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	rec := []string{
		videoID,
		channelID,
		publishedAt.Format(time.RFC3339),
		notifiedAt.Format(time.RFC3339),
	}
	return w.Write(rec)
}

func ensureFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		f, e := os.Create(path)
		if e != nil {
			return e
		}
		f.Close()
	}
	return nil
}
