package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func readHTML(t *testing.T, path string) []byte {
	t.Helper()
	html, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Error at reading html: %v", err)
	}
	return html
}

func TestAZQuote(t *testing.T) {
	html := readHTML(t, filepath.Join("testdata", "azquote_html.html"))
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

	s := NewAZQuoteScraper(client)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel() // releases resources if slowOperation completes before timeout elapses
	records, err := s.ScrapePageCtx(ctx, server.URL)
	if err != nil {
		t.Fatalf("Scrape Page error: %v", err)
	}
	// --- check record #1
	r1 := records[0]
	if got, want := r1.Quote, "Be yourself; everyone else is already taken."; got != want {
		t.Errorf("quote[0] = %q, want %q", got, want)
	}
	if r1.Author == nil || *r1.Author != "Oscar Wilde" {
		t.Errorf("author[0] = %v, want Oscar Wilde", r1.Author)
	}
	if r1.Source != "azquotes" {
		t.Errorf("source[0] = %q, want azquotes", r1.Source)
	}
	// relative href should be resolved using BaseURL (inside scraper code)
	if !strings.Contains(r1.SourceURL, "/quote/123") {
		t.Errorf("sourceURL[0] = %q, want path /quote/123", r1.SourceURL)
	}
	// two tags
	if len(r1.Tags) != 2 || r1.Tags[0] != "inspirational" || r1.Tags[1] != "self" {
		t.Errorf("tags[0] = %#v, want [inspirational self]", r1.Tags)
	}
	// likes parsed from "123 likes"
	if r1.Likes != 123 {
		t.Errorf("likes[0] = %d, want 123", r1.Likes)
	}
	if r1.ID == "" {
		t.Errorf("id[0] empty")
	}

	// --- check record #2
	r2 := records[1]
	if got, want := r2.Quote, "Simplicity is the ultimate sophistication."; got != want {
		t.Errorf("quote[1] = %q, want %q", got, want)
	}
	if r2.Author == nil || *r2.Author != "Leonardo da Vinci" {
		t.Errorf("author[1] = %v, want Leonardo da Vinci", r2.Author)
	}
	// absolute link should be preserved
	if r2.SourceURL != "https://www.azquotes.com/quote/456" {
		t.Errorf("sourceURL[1] = %q, want https://www.azquotes.com/quote/456", r2.SourceURL)
	}
	// no tags, 0 likes
	if len(r2.Tags) != 0 {
		t.Errorf("tags[1] = %#v, want empty", r2.Tags)
	}
	if r2.Likes != 0 {
		t.Errorf("likes[1] = %d, want 0", r2.Likes)
	}
	if r2.ID == "" {
		t.Errorf("id[1] empty")
	}

	// sanity: timestamps are set (non-zero)
	if r1.ScrapedAt.IsZero() || r2.ScrapedAt.IsZero() {
		t.Errorf("ScrapedAt not set")
	}
}
