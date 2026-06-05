package rss_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"koran-ai-backend/internal/crawler/rss"
)

// rssXML is a minimal RSS 2.0 fixture with one article item.
const rssXML = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Kompas Tech</title>
    <link>https://tekno.kompas.com</link>
    <description>Berita Teknologi</description>
    <item>
      <title>Go 1.25 Dirilis dengan Fitur Baru</title>
      <link>https://tekno.kompas.com/read/2026/06/06/go-125</link>
      <description>Go 1.25 membawa banyak improvement.</description>
      <author>Budi Santoso</author>
      <pubDate>Sat, 06 Jun 2026 00:00:00 +0700</pubDate>
      <enclosure url="https://tekno.kompas.com/img/go125.jpg" type="image/jpeg" length="12345"/>
    </item>
    <item>
      <title>  </title>
      <link></link>
      <description>Item without title or URL should be skipped.</description>
    </item>
  </channel>
</rss>`

// atomXML is a minimal Atom 1.0 fixture.
const atomXML = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>CNN Indonesia</title>
  <link href="https://www.cnnindonesia.com"/>
  <entry>
    <title>Ekonomi Indonesia Tumbuh 5%</title>
    <link href="https://www.cnnindonesia.com/ekonomi/2026-06-06/ekonomi-tumbuh"/>
    <content type="html">Pertumbuhan ekonomi Indonesia mencapai 5 persen.</content>
    <updated>2026-06-06T10:00:00Z</updated>
    <author><name>Andi Wijaya</name></author>
  </entry>
</feed>`

func serveFeed(t *testing.T, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
}

func TestParser_ParseRSS(t *testing.T) {
	srv := serveFeed(t, rssXML)
	defer srv.Close()

	p := rss.NewParser()
	items, err := p.Parse(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// The blank item must be skipped
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	got := items[0]

	if got.Title != "Go 1.25 Dirilis dengan Fitur Baru" {
		t.Errorf("unexpected title: %q", got.Title)
	}
	if got.URL != "https://tekno.kompas.com/read/2026/06/06/go-125" {
		t.Errorf("unexpected url: %q", got.URL)
	}
	if got.Author != "Budi Santoso" {
		t.Errorf("unexpected author: %q", got.Author)
	}
	if got.Content == "" {
		t.Error("content must not be empty")
	}
	if got.ImageURL != "https://tekno.kompas.com/img/go125.jpg" {
		t.Errorf("unexpected image_url: %q", got.ImageURL)
	}
	if got.PublishedAt == nil {
		t.Error("published_at must not be nil")
	}
}

func TestParser_ParseAtom(t *testing.T) {
	srv := serveFeed(t, atomXML)
	defer srv.Close()

	p := rss.NewParser()
	items, err := p.Parse(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	got := items[0]
	if got.Title != "Ekonomi Indonesia Tumbuh 5%" {
		t.Errorf("unexpected title: %q", got.Title)
	}
	if got.Content == "" {
		t.Error("content must not be empty for atom entry with <content>")
	}
	if got.Author != "Andi Wijaya" {
		t.Errorf("unexpected author: %q", got.Author)
	}
}

func TestParser_ParseAtom_UpdatedPreferredOverPublished(t *testing.T) {
	// Atom feed where both updated and published are present; updated wins.
	feed := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <title>Test</title>
    <link href="https://example.com/test"/>
    <published>2026-01-01T00:00:00Z</published>
    <updated>2026-06-06T12:00:00Z</updated>
    <content type="text">Content here</content>
  </entry>
</feed>`
	srv := serveFeed(t, feed)
	defer srv.Close()

	p := rss.NewParser()
	items, err := p.Parse(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least one item")
	}

	expected := time.Date(2026, 6, 6, 12, 0, 0, 0, time.UTC)
	if items[0].PublishedAt == nil || !items[0].PublishedAt.Equal(expected) {
		t.Errorf("expected updated time %v, got %v", expected, items[0].PublishedAt)
	}
}

func TestParser_InvalidURL(t *testing.T) {
	p := rss.NewParser()
	_, err := p.Parse(context.Background(), "http://127.0.0.1:0/nonexistent")
	if err == nil {
		t.Fatal("expected error for unreachable URL, got nil")
	}
}

func TestParser_ImageExtractionPriority(t *testing.T) {
	mediaXML := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:media="http://search.yahoo.com/mrss/">
  <channel>
    <title>Media Test Feed</title>
    <link>https://example.com</link>
    <item>
      <title>Article 1</title>
      <link>https://example.com/1</link>
      <pubDate>Sat, 06 Jun 2026 00:00:00 +0000</pubDate>
      <media:content url="https://example.com/media-content.jpg" medium="image"/>
      <media:thumbnail url="https://example.com/media-thumbnail.jpg"/>
      <enclosure url="https://example.com/enclosure.jpg" type="image/jpeg"/>
    </item>
    <item>
      <title>Article 2</title>
      <link>https://example.com/2</link>
      <pubDate>Sat, 06 Jun 2026 00:00:00 +0000</pubDate>
      <media:thumbnail url="https://example.com/media-thumbnail-only.jpg"/>
      <enclosure url="https://example.com/enclosure2.jpg" type="image/jpeg"/>
    </item>
  </channel>
</rss>`

	srv := serveFeed(t, mediaXML)
	defer srv.Close()

	p := rss.NewParser()
	items, err := p.Parse(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// First item: media:content preferred over media:thumbnail and enclosure
	if items[0].ImageURL != "https://example.com/media-content.jpg" {
		t.Errorf("expected image url 'https://example.com/media-content.jpg', got %q", items[0].ImageURL)
	}

	// Second item: media:thumbnail preferred over enclosure
	if items[1].ImageURL != "https://example.com/media-thumbnail-only.jpg" {
		t.Errorf("expected image url 'https://example.com/media-thumbnail-only.jpg', got %q", items[1].ImageURL)
	}
}

