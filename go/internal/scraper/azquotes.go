package scraper

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// AZQuoteScraper is a scraper for azquotes.com
type AZQuoteScraper struct {
	Client  *Client // HTTP client
	BaseURL string  // Base URL
}

// NewAZQuoteScraper returns a new AZQuoteScraper
//
// Example:
//
// client := NewClient()
//
// scraper := NewAZQuoteScraper(client)
func NewAZQuoteScraper(c *Client) *AZQuoteScraper {
	return &AZQuoteScraper{
		Client:  c,
		BaseURL: "https://www.azquotes.com",
	}
}

// ScrapePageCtx scrapes a given AZQuotes page and extracts all quotes found
// on that page into a slice of QuoteRecord structs.
//
// It performs the following steps:
//  1. Fetches the HTML document using the provided context.
//  2. Parses the HTML using goquery.
//  3. Extracts each quote block (div.wrap-block) containing:
//     - Quote text (a.title)
//     - Author name (div.author), defaults to "Unknown" if missing
//     - Tags (div.mytags a)
//     - Like count (from div.share-icons a.heart24)
//  4. Builds a unique SHA-1 ID from the quote, author, and source.
//  5. Returns a slice of valid QuoteRecords.
//
// Parameters:
//   - ctx: Context for cancellation or timeout.
//   - url: AZQuotes page URL to scrape.
//
// Returns:
//   - []QuoteRecord: A list of extracted quotes.
//   - error: Non-nil if the HTTP request or HTML parsing fails.
//
// Example:
//
//	ctx := context.Background()
//	quotes, err := scraper.ScrapePageCtx(ctx, "https://www.azquotes.com/quotes/topics/inspiring.html")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, q := range quotes {
//	    fmt.Printf("%s — %s\n", q.Quote, *q.Author)
//	}
func (s *AZQuoteScraper) ScrapePageCtx(ctx context.Context, url string) ([]QuoteRecord, error) {
	// Check if the URL is valid
	body, err := s.Client.GetWithRetry(ctx, url)
	if err != nil {
		return nil, err
	}

	// Parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	var out []QuoteRecord
	now := time.Now().UTC()

	// Find quote block
	doc.Find("div.wrap-block").Each(func(_ int, sel *goquery.Selection) {
		quote := strings.TrimSpace(sel.Find("a.title").Text())
		authorTxt := strings.TrimSpace(sel.Find("div.author").Text())
		if authorTxt == "" {
			authorTxt = "Unknown"
		}
		author := &authorTxt

		// get tags
		var tags []string
		sel.Find("div.mytags a").Each(func(_ int, a *goquery.Selection) {
			text := strings.TrimSpace(a.Text())
			if text != "" {
				tags = append(tags, text)
			}
		})

		// get likes
		likes := 0
		txt := strings.TrimSpace(sel.Find("div.share-icons a.heart24, div.share-icons a.heart24-off").Text())
		if fields := strings.Fields(txt); len(fields) > 0 {
			if n, err := strconv.Atoi(fields[0]); err == nil {
				likes = n
			}
		}

		// get source
		sourceURL := url
		if href, ok := sel.Find("a.title").Attr("href"); ok && href != "" {
			if strings.HasPrefix(href, "http") {
				sourceURL = href
			} else {
				sourceURL = s.BaseURL + href
			}
		}

		// id = sha1(quote|author|source)
		h := sha1.Sum([]byte(strings.ToLower(quote) + "|" + strings.ToLower(*author) + "|azquotes"))
		id := hex.EncodeToString(h[:])

		rec := QuoteRecord{
			ID:        id,
			Quote:     quote,
			Author:    author,
			Tags:      tags,
			Likes:     likes,
			Source:    "azquotes",
			SourceURL: sourceURL,
			ScrapedAt: now,
		}
		if rec.Quote != "" {
			out = append(out, rec)
		}
	})

	return out, nil
}

// ScrapePage is a convenience wrapper around ScrapePageCtx that uses
// context.Background() as the context.
//
// Example:
//
//	quotes, err := scraper.ScrapePage("https://www.azquotes.com/quotes/topics/inspiring.html")
func (s *AZQuoteScraper) ScrapePage(url string) ([]QuoteRecord, error) {
	return s.ScrapePageCtx(context.Background(), url)
}
