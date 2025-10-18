package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	"github.com/GeorgiosLymperis/Quotes-classification-app/internal/scraper"
)

// main is the entrypoint for the quotes scraping CLI.
// It supports scraping from azquotes, goodreads, and famousquotes concurrently,
// writing results to a JSONL file.
//
// Flags:
//
//	-site      : target site ("azquotes" | "goodreads" | "famousquotes")
//	-start     : start page index (for paginated sites)
//	-end       : end page index (for paginated sites)
//	-out       : output JSONL filepath
//	-topic     : single topic (e.g., "life")
//	-topics    : comma-separated topics (e.g., "life,love,success")
//	-workers   : maximum concurrent requests
func main() {
	site := flag.String("site", "azquotes", "site to scrape (azquotes|goodreads|famousquotes)")
	start := flag.Int("start", 1, "start page")
	end := flag.Int("end", 1, "end page")
	out := flag.String("out", "data.jsonl", "output file")
	topic := flag.String("topic", "", "single topic to scrape (e.g., life)")
	topics := flag.String("topics", "", "comma-separated topics (e.g., life,love,success)")
	workers := flag.Int("workers", 8, "max concurrency")
	flag.Parse()

	// Build topic list from flags.
	var topicList []string
	if *topics != "" {
		for _, t := range strings.Split(*topics, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				topicList = append(topicList, t)
			}
		}
	} else if *topic != "" {
		topicList = []string{strings.TrimSpace(*topic)}
	} else {
		fmt.Println("You must provide --topic or --topics")
		return
	}

	fmt.Printf("site=%s start=%d end=%d out=%s workers=%d\n",
		*site, *start, *end, *out, *workers)

	// HTTP client with retries + backoff.
	client := scraper.NewClient(
		30*time.Second,
		"Mozilla/5.0 (QuoteApp)",
		5,                    // retries
		500*time.Millisecond, // base backoff
		10*time.Second,       // max backoff
	)

	switch *site {
	case "azquotes":
		if err := runAZQuotesConcurrent(client, topicList, *start, *end, *out, *workers); err != nil {
			fmt.Println("run error:", err)
		}
	case "goodreads":
		if err := runGoodReadsConcurrent(client, topicList, *start, *end, *out, *workers); err != nil {
			fmt.Println("run error:", err)
		}
	case "famousquotes":
		if err := runFamousQuotesConcurrent(client, topicList, *out, *workers); err != nil {
			fmt.Println("run error:", err)
		}
	default:
		fmt.Println("unsupported site:", *site)
	}
}

// runAZQuotesConcurrent scrapes AZQuotes for the given topics across a page range,
// using a weighted semaphore for concurrency limiting, and writes the aggregated
// records to a JSONL file.
//
// It logs per-URL progress and continues on per-URL errors (best-effort scraping).
func runAZQuotesConcurrent(client *scraper.Client, topics []string, start, end int, out string, workers int) error {
	s := scraper.NewAZQuoteScraper(client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sem := semaphore.NewWeighted(int64(workers))
	g, ctx := errgroup.WithContext(ctx)

	var mu sync.Mutex
	all := make([]scraper.QuoteRecord, 0, (end-start+1)*25)
	urls := make([]string, 0, len(topics)*(end-start+1))
	for _, topic := range topics {
		for p := start; p <= end; p++ {
			url := fmt.Sprintf("https://www.azquotes.com/quotes/topics/%s.html?p=%d", topic, p)
			urls = append(urls, url)
		}
	}

	for _, url := range urls {
		// Rebind loop variable to avoid goroutine capture issue.
		u := url

		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}
		g.Go(func() error {
			defer sem.Release(1)

			recs, err := s.ScrapePageCtx(ctx, u)
			if err != nil {
				fmt.Println("error:", err, "url:", u)
				return nil // continue other URLs
			}
			fmt.Println("done:", u, "records:", len(recs))

			mu.Lock()
			all = append(all, recs...)
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	if err := scraper.WriteJSONL(out, all); err != nil {
		return err
	}
	fmt.Println("saved:", out, "total records:", len(all))
	return nil
}

// runGoodReadsConcurrent scrapes Goodreads tag pages for the given topics and page range,
// aggregating results and writing them to a JSONL file.
func runGoodReadsConcurrent(client *scraper.Client, topics []string, start, end int, out string, workers int) error {
	s := scraper.NewAGoodReadsScraper(client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sem := semaphore.NewWeighted(int64(workers))
	g, ctx := errgroup.WithContext(ctx)

	var mu sync.Mutex
	all := make([]scraper.QuoteRecord, 0, (end-start+1)*25)
	urls := make([]string, 0, len(topics)*(end-start+1))
	for _, topic := range topics {
		for p := start; p <= end; p++ {
			url := fmt.Sprintf("https://www.goodreads.com/quotes/tag/%s?page=%d", topic, p)
			urls = append(urls, url)
		}
	}

	for _, url := range urls {
		u := url // avoid goroutine capture

		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}
		g.Go(func() error {
			defer sem.Release(1)

			recs, err := s.ScrapePageCtx(ctx, u)
			if err != nil {
				fmt.Println("error:", err, "url:", u)
				return nil // continue other URLs
			}
			fmt.Println("done:", u, "records:", len(recs))

			mu.Lock()
			all = append(all, recs...)
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	if err := scraper.WriteJSONL(out, all); err != nil {
		return err
	}
	fmt.Println("saved:", out, "total records:", len(all))
	return nil
}

// runFamousQuotesConcurrent scrapes one page per topic from FamousQuotesAndAuthors,
// aggregates the results, and writes them to a JSONL file.
func runFamousQuotesConcurrent(client *scraper.Client, topics []string, out string, workers int) error {
	s := scraper.NewFamousQuotesScraper(client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sem := semaphore.NewWeighted(int64(workers))
	g, ctx := errgroup.WithContext(ctx)

	var mu sync.Mutex
	all := make([]scraper.QuoteRecord, 0, 4*25)
	urls := make([]string, 0, len(topics))
	for _, topic := range topics {
		url := fmt.Sprintf("http://www.famousquotesandauthors.com/topics/%s_quotes.html", topic)
		urls = append(urls, url)
	}

	for _, url := range urls {
		u := url // avoid goroutine capture

		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}
		g.Go(func() error {
			defer sem.Release(1)

			recs, err := s.ScrapePageCtx(ctx, u)
			if err != nil {
				fmt.Println("error:", err, "url:", u)
				return nil // continue other URLs
			}
			fmt.Println("done:", u, "records:", len(recs))

			mu.Lock()
			all = append(all, recs...)
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	if err := scraper.WriteJSONL(out, all); err != nil {
		return err
	}
	fmt.Println("saved:", out, "total records:", len(all))
	return nil
}
