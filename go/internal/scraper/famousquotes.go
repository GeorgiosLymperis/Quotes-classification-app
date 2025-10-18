package scraper

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// FamousQuotesScraper scrapes quotes and author names from
// the website https://www.famousquotesandauthors.com.
//
// Each page typically represents a topic (e.g., "Life Quotes")
// and contains a list of quotes and corresponding authors.
//
// The scraper extracts quotes, their authors, and an optional topic tag
// from a given page, producing structured QuoteRecord entries.
type FamousQuotesScraper struct {
	Client  *Client // HTTP client used to fetch pages (with retry and backoff).
	BaseURL string  // Base URL for famousquotesandauthors.com.
}

// NewFamousQuotesScraper creates and returns a new FamousQuotesScraper
// configured with the provided HTTP Client.
//
// Example:
//
//	client := scraper.NewClient(10*time.Second, "MyScraperBot/1.0", 3, 500*time.Millisecond, 5*time.Second)
//	fq := scraper.NewFamousQuotesScraper(client)
func NewFamousQuotesScraper(c *Client) *FamousQuotesScraper {
	return &FamousQuotesScraper{
		Client:  c,
		BaseURL: "http://www.famousquotesandauthors.com",
	}
}

// ScrapePageCtx scrapes a single page of FamousQuotesAndAuthors using the provided context.
//
// It performs the following steps:
//  1. Fetches the HTML document with the Client (using retry and backoff).
//  2. Locates the main content cell (`td[valign="top"]`) that contains quotes.
//  3. Extracts the topic or tag (page header).
//  4. Parses all quotes and corresponding authors (one per line).
//  5. Generates a unique SHA-1 ID for each record using (quote|author|source).
//  6. Returns a slice of QuoteRecord objects.
//
// Each page usually includes one topic header (used as the tag for all quotes).
// Missing authors default to "Unknown".
//
// Parameters:
//   - ctx: Context for cancellation or timeout.
//   - pageURL: Full URL of the page to scrape.
//
// Returns:
//   - []QuoteRecord: Parsed quote entries.
//   - error: Non-nil if network or parsing fails.
//
// Example:
//
//	ctx := context.Background()
//	quotes, err := fq.ScrapePageCtx(ctx, "http://www.famousquotesandauthors.com/topics/inspirational_quotes.html")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, q := range quotes {
//	    fmt.Printf("%s — %s (tag: %v)\n", q.Quote, *q.Author, q.Tags)
//	}
func (s *FamousQuotesScraper) ScrapePageCtx(ctx context.Context, pageURL string) ([]QuoteRecord, error) {
	body, err := s.Client.GetWithRetry(ctx, pageURL)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	// Locate main content cell containing the quotes.
	cell := doc.Find(`td[style="padding-left:16px; padding-right:16px;"][valign="top"]`).First()
	if cell.Length() == 0 {
		// Fallback selector (the site varies its HTML slightly).
		cell = doc.Find(`td[valign="top"]`).Has(`div[style*="font-size:12px"]`).First()
	}

	// Extract the topic (page-level tag).
	tag := ""
	cell.Find(`div[style*="font-size:19px"][style*="Times New Roman"]`).EachWithBreak(func(_ int, hdr *goquery.Selection) bool {
		t := strings.TrimSpace(hdr.Text())
		if idx := strings.Index(t, " Quote"); idx > 0 {
			tag = strings.TrimSpace(t[:idx])
		} else {
			tag = t
		}
		return false
	})

	// Extract quotes.
	qs := make([]string, 0, 64)
	cell.Find(`div[style="font-size:12px;font-family:Arial;"]`).Each(func(_ int, q *goquery.Selection) {
		txt := strings.TrimSpace(q.Text())
		if txt != "" {
			qs = append(qs, normalizeSpaces(txt))
		}
	})

	// Extract authors (usually one per quote).
	as := make([]string, 0, len(qs))
	cell.Find(`div[style="padding-top:2px;"] a`).Each(func(_ int, a *goquery.Selection) {
		name := strings.TrimSpace(a.Text())
		if name == "" {
			name = "Unknown"
		}
		as = append(as, name)
	})

	// Pair quotes and authors (truncate to shortest slice).
	n := min(len(qs), len(as))
	now := time.Now().UTC()
	out := make([]QuoteRecord, 0, n)

	for i := 0; i < n; i++ {
		q := qs[i]
		a := as[i]
		author := &a

		// Generate deterministic ID = sha1(quote|author|famousquotes).
		h := sha1.Sum([]byte(strings.ToLower(q) + "|" + strings.ToLower(a) + "|famousquotes"))
		id := hex.EncodeToString(h[:])

		out = append(out, QuoteRecord{
			ID:        id,
			Quote:     q,
			Author:    author,
			Tags:      withIf(tag), // Single topic tag per page.
			Likes:     0,
			Source:    "famousquotes",
			SourceURL: pageURL,
			ScrapedAt: now,
		})
	}

	return out, nil
}

// normalizeSpaces collapses all runs of whitespace into single spaces.
// Useful for cleaning up inconsistent HTML spacing in quotes.
func normalizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// withIf returns a single-element slice containing tag if non-empty,
// or nil if tag is empty. Used for adding a page-level topic tag.
func withIf(tag string) []string {
	if tag == "" {
		return nil
	}
	return []string{tag}
}
