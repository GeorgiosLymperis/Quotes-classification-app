package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestFQuote(t *testing.T) {
	html := readHTML(t, filepath.Join("testdata", "famousquotes_html.html"))
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

	s := NewFamousQuotesScraper(client)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel() // releases resources if slowOperation completes before timeout elapses
	records, err := s.ScrapePageCtx(ctx, server.URL)
	if err != nil {
		t.Fatalf("Scrape Page error: %v", err)
	}
	// --- record #1 checks
	r1 := records[0]
	wantQ1 := "Life is really simple, but we insist on making it complicated."
	if r1.Quote != wantQ1 {
		t.Errorf("quote[0] = %q, want %q", r1.Quote, wantQ1)
	}
	if r1.Author == nil || *r1.Author != "Confucius" {
		t.Errorf("author[0] = %v, want Confucius", r1.Author)
	}
	// tag comes from page header ("Life")
	if len(r1.Tags) != 1 || r1.Tags[0] != "Life" {
		t.Errorf("tags[0] = %#v, want [Life]", r1.Tags)
	}
	if r1.Source != "famousquotes" {
		t.Errorf("source[0] = %q, want famousquotes", r1.Source)
	}
	if r1.SourceURL != server.URL {
		t.Errorf("sourceURL[0] = %q, want %q", r1.SourceURL, server.URL)
	}
	if r1.Likes != 0 {
		t.Errorf("likes[0] = %d, want 0", r1.Likes)
	}
	if r1.ID == "" || r1.ScrapedAt.IsZero() {
		t.Errorf("id/scrapedAt missing on record #1")
	}

	// --- record #2 checks
	r2 := records[1]
	wantQ2 := "The purpose of our lives is to be happy."
	if r2.Quote != wantQ2 {
		t.Errorf("quote[1] = %q, want %q", r2.Quote, wantQ2)
	}
	if r2.Author == nil || *r2.Author != "Dalai Lama" {
		t.Errorf("author[1] = %v, want Dalai Lama", r2.Author)
	}
	if len(r2.Tags) != 1 || r2.Tags[0] != "Life" {
		t.Errorf("tags[1] = %#v, want [Life]", r2.Tags)
	}
	if r2.Source != "famousquotes" || r2.SourceURL != server.URL {
		t.Errorf("source/sourceURL mismatch: %q / %q", r2.Source, r2.SourceURL)
	}
}
