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

type AZQuoteScraper struct {
	Client *Client
	BaseURL string
}

func NewAZQuoteScraper(c *Client) *AZQuoteScraper {
	return &AZQuoteScraper{
		Client: c,
		BaseURL: "https://www.azquotes.com",
	}
}

func (s *AZQuoteScraper) ScrapePageCtx(ctx context.Context, url string) ([]QuoteRecord, error) {
	body, err := s.Client.GetWithRetry(ctx, url)
	if err != nil {return nil, err}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {return nil, err}

	var out []QuoteRecord
	now := time.Now().UTC()

	doc.Find("div.wrap-block").Each(func(_ int, sel *goquery.Selection) {
		quote := strings.TrimSpace(sel.Find("a.title").Text())
		authorTxt := strings.TrimSpace(sel.Find("div.author").Text())
		if authorTxt == "" {authorTxt = "Unknown"}
		author := &authorTxt
	
	// get tags
	var tags []string
	sel.Find("div.mytags a").Each(func(_ int, a *goquery.Selection){
		text := strings.TrimSpace(a.Text())
		if text != "" {tags = append(tags, text)}
	})

	// get likes
	likes := 0
	txt := strings.TrimSpace(sel.Find("div.share-icons a.heart24, div.share-icons a.heart24-off").Text())
	if fields := strings.Fields(txt); len(fields) > 0 {
		if n, err := strconv.Atoi(fields[0]); err == nil {
			likes = n
		}
	}

	sourceURL := url
	if href, ok := sel.Find("a.title").Attr("href"); ok && href != "" {
		if strings.HasPrefix(href, "http"){
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

func (s *AZQuoteScraper) ScrapePage(url string) ([]QuoteRecord, error) {
	return s.ScrapePageCtx(context.Background(), url)
}