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

// CSS selectors for Goodreads quote blocks and metadata extraction.
const (
	grBlockSel     = "div.quote.mediumText"
	grQuoteTextSel = "div.quoteText"
	grAuthorSel    = "span.authorOrTitle"
	grLikesLinkSel = "div.quoteFooter a.smallText"
	grTagLinkSel   = "div.quoteFooter a[href*='/quotes/tag/']"
)

// GoodReadsScraper scrapes quotes, authors, tags, and like counts
// from Goodreads quote listing pages (https://www.goodreads.com/quotes).
type GoodReadsScraper struct {
	Client  *Client // HTTP client with retry and backoff logic.
	BaseURL string  // Base URL for Goodreads quotes.
}

// NewAGoodReadsScraper creates and returns a new GoodReadsScraper using the provided Client.
//
// Example:
//
//	client := scraper.NewClient(10*time.Second, "MyScraperBot/1.0", 3, 500*time.Millisecond, 5*time.Second)
//	gr := scraper.NewAGoodReadsScraper(client)
func NewAGoodReadsScraper(c *Client) *GoodReadsScraper {
	return &GoodReadsScraper{
		Client:  c,
		BaseURL: "https://www.goodreads.com/quotes",
	}
}

// ScrapePageCtx scrapes a Goodreads quote page using the provided context.
//
// It performs the following steps:
//  1. Fetches the HTML via the Client’s GetWithRetry method.
//  2. Parses the HTML with goquery.
//  3. Extracts data from each quote block (`div.quote.mediumText`):
//     - Quote text (from `div.quoteText`, removing author and footer nodes)
//     - Author name (`span.authorOrTitle`, defaults to "Unknown" if missing)
//     - Tags (from `div.quoteFooter a[href*='/quotes/tag/']`)
//     - Like count (parsed from `div.quoteFooter a.smallText`, e.g. "123 likes")
//  4. Generates a unique SHA-1 ID from the quote, author, and source ("goodreads").
//  5. Returns all valid QuoteRecords.
//
// Parameters:
//   - ctx: Context for request cancellation or timeout.
//   - url: Goodreads page URL to scrape.
//
// Returns:
//   - []QuoteRecord: Slice of extracted quotes.
//   - error: Non-nil if fetching or parsing fails.
//
// Example:
//
//	ctx := context.Background()
//	quotes, err := gr.ScrapePageCtx(ctx, "https://www.goodreads.com/quotes/tag/inspiration")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, q := range quotes {
//	    fmt.Printf("%s — %s (%d likes)\n", q.Quote, *q.Author, q.Likes)
//	}
func (s *GoodReadsScraper) ScrapePageCtx(ctx context.Context, url string) ([]QuoteRecord, error) {
	body, err := s.Client.GetWithRetry(ctx, url)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	out := make([]QuoteRecord, 0, 25)

	doc.Find(grBlockSel).Each(func(_ int, blk *goquery.Selection) {
		// Extract author
		authorTxt := strings.TrimSpace(blk.Find(grAuthorSel).Text())
		if authorTxt == "" {
			authorTxt = "Unknown"
		}
		author := &authorTxt

		// Extract quote text (clean out author, "likes", <br>, etc.)
		qSel := blk.Find(grQuoteTextSel).First().Clone()
		qSel.Find(grAuthorSel).Remove()
		qSel.Find("a.smallText").Remove()
		qSel.Find("br").Remove()
		q := strings.TrimSpace(qSel.Text())

		// Extract tags
		var tags []string
		blk.Find(grTagLinkSel).Each(func(_ int, a *goquery.Selection) {
			t := strings.TrimSpace(a.Text())
			if t != "" {
				tags = append(tags, t)
			}
		})

		// Extract like count ("123 likes")
		likes := 0
		likesTxt := strings.TrimSpace(blk.Find(grLikesLinkSel).First().Text())
		if likesTxt != "" {
			fields := strings.Fields(likesTxt)
			if len(fields) > 0 {
				if n, err := strconv.Atoi(strings.ReplaceAll(fields[0], ",", "")); err == nil {
					likes = n
				}
			}
		}

		// Skip if no quote text
		if q == "" {
			return
		}

		// Generate unique ID = sha1(quote|author|goodreads)
		h := sha1.Sum([]byte(strings.ToLower(q) + "|" + strings.ToLower(*author) + "|goodreads"))
		id := hex.EncodeToString(h[:])

		rec := QuoteRecord{
			ID:        id,
			Quote:     q,
			Author:    author,
			Tags:      tags,
			Likes:     likes,
			Source:    "goodreads",
			SourceURL: url,
			ScrapedAt: now,
		}
		out = append(out, rec)
	})

	return out, nil
}

// ScrapePage is a convenience wrapper around ScrapePageCtx that uses
// context.Background() as the context.
//
// Example:
//
//	quotes, err := gr.ScrapePage("https://www.goodreads.com/quotes/tag/happiness")
func (s *GoodReadsScraper) ScrapePage(url string) ([]QuoteRecord, error) {
	return s.ScrapePageCtx(context.Background(), url)
}
