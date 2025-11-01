package repository

import (
	"encoding/csv"
	"os"
	"time"
)

type NotifiedRepository interface {
	Has(videoID string) (bool, error)
	Append(videoID, channelID string, publishedAt, notifiedAt time.Time) error
}

type CSVNotifiedRepository struct{ Path string }

func (r *CSVNotifiedRepository) Has(videoID string) (bool, error) {
	f, err := os.Open(r.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer f.Close()

	cr := csv.NewReader(f)
	cr.FieldsPerRecord = -1
	rows, err := cr.ReadAll()
	if err != nil {
		return false, err
	}
	for i, row := range rows {
		if i == 0 {
			// ヘッダ考慮せずシンプルに走査（ヘッダ行があっても video_id と一致することはほぼない）
		}
		if len(row) > 0 && row[0] == videoID {
			return true, nil
		}
	}
	return false, nil
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
