package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGRQuote(t *testing.T) {
	html := readHTML(t, filepath.Join("testdata", "goodreads_html.html"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(html)
	}))
	t.Cleanup(server.Close)

	client := NewClient(
		3*time.Second,
		"TestAgent/1.0",
		5,
		5*time.Millisecond,
		10*time.Millisecond,
	)

	s := NewAGoodReadsScraper(client)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel() // releases resources if slowOperation completes before timeout elapses
	records, err := s.ScrapePageCtx(ctx, server.URL)
	if err != nil {
		t.Fatalf("Scrape Page error: %v", err)
	}
	// --- record #1 checks
	r1 := records[0]

	// authorOrTitle should become Author and be removed from quote text
	if r1.Author == nil || *r1.Author != "Albert Einstein" {
		t.Errorf("author[0] = %v, want Albert Einstein", r1.Author)
	}

	wantQ1 := "“Two things are infinite: the universe and human stupidity; and I'm not sure about the universe.”"
	if r1.Quote != strings.TrimSpace(wantQ1) {
		t.Errorf("quote[0] = %q, want %q", r1.Quote, wantQ1)
	}

	// likes: "1,234 likes" -> 1234
	if r1.Likes != 1234 {
		t.Errorf("likes[0] = %d, want 1234", r1.Likes)
	}

	// tags: science, humor (order preserved by fixture)
	if len(r1.Tags) != 2 || r1.Tags[0] != "science" || r1.Tags[1] != "humor" {
		t.Errorf("tags[0] = %#v, want [science humor]", r1.Tags)
	}

	// source is hardcoded to "goodreads" and SourceURL is the page URL we passed in
	if r1.Source != "goodreads" {
		t.Errorf("source[0] = %q, want goodreads", r1.Source)
	}
	if r1.SourceURL != server.URL {
		t.Errorf("sourceURL[0] = %q, want %q", r1.SourceURL, server.URL)
	}

	if r1.ID == "" || r1.ScrapedAt.IsZero() {
		t.Errorf("id or scrapedAt not set for record #1")
	}

	// --- record #2 checks
	r2 := records[1]
	if r2.Author == nil || *r2.Author != "Frank Zappa" {
		t.Errorf("author[1] = %v, want Frank Zappa", r2.Author)
	}
	if r2.Quote != "“So many books, so little time.”" {
		t.Errorf("quote[1] = %q, want %q", r2.Quote, "“So many books, so little time.”")
	}
	if len(r2.Tags) != 0 {
		t.Errorf("tags[1] = %#v, want empty", r2.Tags)
	}
	if r2.Likes != 0 {
		t.Errorf("likes[1] = %d, want 0", r2.Likes)
	}
	if r2.Source != "goodreads" || r2.SourceURL != server.URL {
		t.Errorf("source/sourceURL mismatch: %q / %q", r2.Source, r2.SourceURL)
	}
}
