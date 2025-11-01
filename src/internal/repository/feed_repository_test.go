package repository

import (
	"strings"
	"testing"
)

const sampleFeed = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:yt="http://www.youtube.com/xml/schemas/2015">
  <title>Sample Channel</title>
  <entry>
    <id>yt:video:VIDEO123</id>
    <yt:videoId>VIDEO123</yt:videoId>
    <title>First Video</title>
    <link rel="alternate" href="https://www.youtube.com/watch?v=VIDEO123"/>
    <published>2024-01-01T12:34:56+00:00</published>
  </entry>
  <entry>
    <id>yt:video:VIDEO456</id>
    <title>Second Video</title>
    <link rel="alternate" href="https://www.youtube.com/watch?v=VIDEO456"/>
    <published>2024-01-02T12:34:56+00:00</published>
  </entry>
</feed>`

func TestDecodeFeed(t *testing.T) {
	feed, err := decodeFeed(strings.NewReader(sampleFeed))
	if err != nil {
		t.Fatalf("decodeFeed error: %v", err)
	}
	if got, want := len(feed.Entries), 2; got != want {
		t.Fatalf("unexpected entry count %d, want %d", got, want)
	}
	if feed.Entries[0].ID != "VIDEO123" {
		t.Fatalf("expected video id from yt:videoId, got %q", feed.Entries[0].ID)
	}
	if feed.Entries[1].ID != "yt:video:VIDEO456" {
		t.Fatalf("expected fallback to entry id, got %q", feed.Entries[1].ID)
	}
}
