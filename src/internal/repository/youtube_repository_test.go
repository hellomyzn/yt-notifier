package repository

import (
	"testing"
	"time"
)

// ---- uploadsPlaylistID ----

func TestUploadsPlaylistID(t *testing.T) {
	tests := []struct {
		channelID string
		want      string
	}{
		{"UCabcdef1234567890", "UUabcdef1234567890"},
		{"UCxxxxxxxxxxxxxxxxxxx", "UUxxxxxxxxxxxxxxxxxxx"},
		{"", ""},
		{"  ", ""},
		{"notUCprefix", "notUCprefix"},
		{"UC", "UC"},
	}
	for _, tt := range tests {
		got := uploadsPlaylistID(tt.channelID)
		if got != tt.want {
			t.Errorf("uploadsPlaylistID(%q) = %q; want %q", tt.channelID, got, tt.want)
		}
	}
}

// ---- clamp ----

func TestClamp(t *testing.T) {
	tests := []struct {
		v, min, max, want int
	}{
		{5, 1, 10, 5},
		{0, 1, 10, 1},
		{15, 1, 10, 10},
		{50, 50, 50, 50},
	}
	for _, tt := range tests {
		got := clamp(tt.v, tt.min, tt.max)
		if got != tt.want {
			t.Errorf("clamp(%d,%d,%d) = %d; want %d", tt.v, tt.min, tt.max, got, tt.want)
		}
	}
}

// ---- firstTime ----

func TestFirstTime(t *testing.T) {
	fixed := "2024-06-01T12:00:00Z"
	parsed, _ := time.Parse(time.RFC3339, fixed)

	tests := []struct {
		name   string
		values []string
		want   time.Time
	}{
		{"最初の有効な値を返す", []string{fixed, "2024-07-01T00:00:00Z"}, parsed},
		{"空文字をスキップして次を使う", []string{"", fixed}, parsed},
		{"全て空のとき time.Now() に近い値", []string{"", ""}, time.Time{}}, // ゼロ値チェックなし
		{"不正フォーマットをスキップ", []string{"not-a-time", fixed}, parsed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firstTime(tt.values...)
			if !tt.want.IsZero() && !got.Equal(tt.want) {
				t.Errorf("firstTime() = %v; want %v", got, tt.want)
			}
		})
	}
}

// ---- YouTubeAPIMetrics ----

func TestYouTubeAPIMetrics_IncrementAndSnapshot(t *testing.T) {
	m := &YouTubeAPIMetrics{}
	m.IncrementRequests()
	m.IncrementRequests()

	snap := m.Snapshot()
	if snap.Requests != 2 {
		t.Errorf("Requests = %d; want 2", snap.Requests)
	}
	if snap.QuotaUnits != 2 {
		t.Errorf("QuotaUnits = %d; want 2", snap.QuotaUnits)
	}
}

func TestNewYouTubeAPIRepository_NilWhenNoKey(t *testing.T) {
	if got := NewYouTubeAPIRepository(""); got != nil {
		t.Error("want nil for empty API key")
	}
}

func TestNewYouTubeAPIRepository_NotNilWithKey(t *testing.T) {
	if got := NewYouTubeAPIRepository("somekey"); got == nil {
		t.Error("want non-nil for non-empty API key")
	}
}

// ---- parseRetryAfter ----

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"60", 60 * time.Second},
		{"0", 0},
		{"", 0},
		{"abc", 0},
	}
	for _, tt := range tests {
		got := parseRetryAfter(tt.input)
		if got != tt.want {
			t.Errorf("parseRetryAfter(%q) = %v; want %v", tt.input, got, tt.want)
		}
	}
}
