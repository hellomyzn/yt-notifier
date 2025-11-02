package repository

import (
	"encoding/csv"
	"os"
	"strconv"
	"strings"

	"github.com/hellomyzn/yt-notifier/internal/model"
)

type ChannelRepository interface {
	ListEnabled() ([]model.ChannelDTO, error)
}

type CSVChannelRepository struct{ Path string }

func (r *CSVChannelRepository) ListEnabled() ([]model.ChannelDTO, error) {
	f, err := os.Open(r.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cr := csv.NewReader(f)
	cr.FieldsPerRecord = -1

	rows, err := cr.ReadAll()
	if err != nil {
		return nil, err
	}
	var out []model.ChannelDTO
	for i, row := range rows {
		if i == 0 && len(row) > 0 && strings.Contains(strings.ToLower(row[0]), "channel") {
			continue // header skip
		}
		if len(row) < 4 {
			continue
		}
		enabled := strings.EqualFold(strings.TrimSpace(row[3]), "true")
		if !enabled {
			continue
		}
		var fetchLimit int
		if len(row) >= 5 {
			if v, err := strconv.Atoi(strings.TrimSpace(row[4])); err == nil {
				fetchLimit = v
			}
		}
		out = append(out, model.ChannelDTO{
			ChannelID:  strings.TrimSpace(row[0]),
			Category:   strings.ToLower(strings.TrimSpace(row[1])),
			Name:       strings.TrimSpace(row[2]),
			Enabled:    true,
			FetchLimit: fetchLimit,
		})
	}
	return out, nil
}
