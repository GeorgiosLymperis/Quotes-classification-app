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

const (
	grBlockSel      = "div.quote.mediumText"
	grQuoteTextSel  = "div.quoteText"
	grAuthorSel     = "span.authorOrTitle"
	grLikesLinkSel  = "div.quoteFooter a.smallText"
	grTagLinkSel    = "div.quoteFooter a[href*='/quotes/tag/']"
)

type GoodReadsScraper struct {
	Client *Client
	BaseURL string
}

func NewAGoodReadsScraper(c *Client) *GoodReadsScraper {
	return &GoodReadsScraper{
		Client: c,
		BaseURL: "https://www.goodreads.com/quotes",
	}
}

func (s *GoodReadsScraper) ScrapePageCtx(ctx context.Context, url string) ([]QuoteRecord, error) {
	body, err := s.Client.GetWithRetry(ctx, url)
	if err != nil { return nil, err }

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil { return nil, err }

	now := time.Now().UTC()
	out := make([]QuoteRecord, 0, 25)

	doc.Find(grBlockSel).Each(func(_ int, blk *goquery.Selection) {
		// author
		authorTxt := strings.TrimSpace(blk.Find(grAuthorSel).Text())
		if authorTxt == "" { authorTxt = "Unknown" }
		author := &authorTxt

		// quote: take quoteText and strip author & trailing parts
		qSel := blk.Find(grQuoteTextSel).First().Clone()
		// remove the author node and other known non-quote spans
		qSel.Find(grAuthorSel).Remove()
		qSel.Find("a.smallText").Remove()
		qSel.Find("br").Remove()
		q := strings.TrimSpace(qSel.Text())

		// tags: only tag links
		var tags []string
		blk.Find(grTagLinkSel).Each(func(_ int, a *goquery.Selection) {
			t := strings.TrimSpace(a.Text())
			if t != "" {
				tags = append(tags, t)
			}
		})

		// likes: in smallText link ("123 likes")
		likes := 0
		likesTxt := strings.TrimSpace(blk.Find(grLikesLinkSel).First().Text())
		if likesTxt != "" {
			fields := strings.Fields(likesTxt) // ["123", "likes"]
			if len(fields) > 0 {
				if n, err := strconv.Atoi(strings.ReplaceAll(fields[0], ",", "")); err == nil {
					likes = n
				}
			}
		}

		// ID: sha1(quote|author|goodreads)
		if q == "" { return }
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

func (s *GoodReadsScraper) ScrapePage(url string) ([]QuoteRecord, error) {
	return s.ScrapePageCtx(context.Background(), url)
}