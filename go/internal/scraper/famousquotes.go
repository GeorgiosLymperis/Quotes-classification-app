package scraper

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type FamousQuotesScraper struct {
	Client  *Client
	BaseURL string
}

func NewFamousQuotesScraper(c *Client) *FamousQuotesScraper {
	return &FamousQuotesScraper{
		Client:  c,
		BaseURL: "http://www.famousquotesandauthors.com",
	}
}

func (s *FamousQuotesScraper) ScrapePageCtx(ctx context.Context, pageURL string) ([]QuoteRecord, error) {
	body, err := s.Client.GetWithRetry(ctx, pageURL)
	if err != nil { return nil, err }

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil { return nil, err }

	// narrow to the main cell with quotes
	cell := doc.Find(`td[style="padding-left:16px; padding-right:16px;"][valign="top"]`).First()
	if cell.Length() == 0 {
		// try a bit more permissive (site sometimes varies spaces)
		cell = doc.Find(`td[valign="top"]`).Has(`div[style*="font-size:12px"]`).First()
	}

	// tag (topic) header
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

	// quotes
	qs := make([]string, 0, 64)
	cell.Find(`div[style="font-size:12px;font-family:Arial;"]`).Each(func(_ int, q *goquery.Selection) {
		txt := strings.TrimSpace(q.Text())
		if txt != "" {
			qs = append(qs, normalizeSpaces(txt))
		}
	})

	// authors (same count as quotes in your Python logic)
	as := make([]string, 0, len(qs))
	cell.Find(`div[style="padding-top:2px;"] a`).Each(func(_ int, a *goquery.Selection) {
		name := strings.TrimSpace(a.Text())
		if name == "" {
			name = "Unknown"
		}
		as = append(as, name)
	})

	// Pair up (min length safeguard)
	n := min(len(qs), len(as))
	now := time.Now().UTC()
	out := make([]QuoteRecord, 0, n)

	for i := 0; i < n; i++ {
		q := qs[i]
		a := as[i]
		author := &a

		h := sha1.Sum([]byte(strings.ToLower(q) + "|" + strings.ToLower(a) + "|famousquotes"))
		id := hex.EncodeToString(h[:])

		out = append(out, QuoteRecord{
			ID:        id,
			Quote:     q,
			Author:    author,
			Tags:      withIf(tag), // one topic tag per page
			Likes:     0,
			Source:    "famousquotes",
			SourceURL: pageURL,
			ScrapedAt: now,
		})
	}
	return out, nil
}

func normalizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func withIf(tag string) []string {
	if tag == "" { return nil }
	return []string{tag}
}