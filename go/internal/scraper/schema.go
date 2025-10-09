package scraper

import "time"

type QuoteRecord struct {
	ID string `json:"id"`
	Quote string `json:"quote"`
	Author *string `json:"author,omitempty"`
	Tags []string `json:"tags,omitempty"`
	Likes int `json:"likes"`
	Source string `json:"source"`
	SourceURL string `json:"source_url"`
	ScrapedAt time.Time `json:"scraped_at"`
	Lang string `json:"lang,omitempty"`
}