package repository

import (
	"os"
	"path/filepath"
	"testing"
)

func writeChannelCSV(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "channels.csv")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCSVChannelRepository_ListEnabled(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantLen int
	}{
		{
			name:    "ヘッダ行をスキップ",
			content: "channel_id,category,name,enabled,fetch_limit\nUC1,travel_jp,Ch1,true,10\n",
			wantLen: 1,
		},
		{
			name:    "enabled=false は除外",
			content: "UC1,travel_jp,Ch1,true,10\nUC2,news_jp,Ch2,false,10\n",
			wantLen: 1,
		},
		{
			name:    "全て有効",
			content: "UC1,travel_jp,Ch1,true,10\nUC2,tech_en,Ch2,true,15\n",
			wantLen: 2,
		},
		{
			name:    "空ファイル",
			content: "",
			wantLen: 0,
		},
		{
			name:    "フィールド数が足りない行はスキップ",
			content: "UC1,travel\nUC2,news_jp,Ch2,true,10\n",
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeChannelCSV(t, tt.content)
			repo := &CSVChannelRepository{Path: path}
			got, err := repo.ListEnabled()
			if err != nil {
				t.Fatalf("ListEnabled() error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Errorf("len(result) = %d; want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestCSVChannelRepository_ListEnabled_FieldValues(t *testing.T) {
	content := "UC1,Travel_JP,My Channel,true,50\n"
	path := writeChannelCSV(t, content)
	repo := &CSVChannelRepository{Path: path}

	got, err := repo.ListEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 result, got %d", len(got))
	}

	ch := got[0]
	tests := []struct {
		field string
		got   any
		want  any
	}{
		{"ChannelID", ch.ChannelID, "UC1"},
		{"Category（小文字化）", ch.Category, "travel_jp"},
		{"Name", ch.Name, "My Channel"},
		{"Enabled", ch.Enabled, true},
		{"FetchLimit", ch.FetchLimit, 50},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v; want %v", tt.field, tt.got, tt.want)
		}
	}
}

func TestCSVChannelRepository_FileNotExist(t *testing.T) {
	repo := &CSVChannelRepository{Path: "/nonexistent/path/channels.csv"}
	_, err := repo.ListEnabled()
	if err == nil {
		t.Error("want error for missing file, got nil")
	}
}
