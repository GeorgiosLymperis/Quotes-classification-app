package main

import (
	"context"
	"flag"
	"fmt"
	"sync"
	"time"
	"strings"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	"github.com/GeorgiosLymperis/quote_app/internal/scraper"
)

func main() {
	site := flag.String("site", "azquotes", "site to scrape (azquotes|goodreads|famousquotes)")
	start := flag.Int("start", 1, "start page")
	end := flag.Int("end", 1, "end page")
	out := flag.String("out", "data.jsonl", "output file")
	topic := flag.String("topic", "", "single topic to scrape (e.g., life)")
	topics := flag.String("topics", "", "comma-separated topics (e.g., life,love,success)")
	workers := flag.Int("workers", 8, "max concurrency")
	flag.Parse()

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

	// Client με retries + backoff
	client := scraper.NewClient(
		30*time.Second,
		"Mozilla/5.0 (QuoteApp)",
		5,              // retries
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
    if err := sem.Acquire(ctx, 1); err != nil { return err }

    g.Go(func() error {
        defer sem.Release(1)

        recs, err := s.ScrapePageCtx(ctx, url)
        if err != nil {
            fmt.Println("error:", err, "url:", url)
            return nil // keep going
        }
        fmt.Println( "done:", url, "records:", len(recs))

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
    if err := sem.Acquire(ctx, 1); err != nil { return err }

    g.Go(func() error {
        defer sem.Release(1)

        recs, err := s.ScrapePageCtx(ctx, url)
        if err != nil {
            fmt.Println("error:", err, "url:", url)
            return nil // keep going
        }
        fmt.Println( "done:", url, "records:", len(recs))

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
    if err := sem.Acquire(ctx, 1); err != nil { return err }

    g.Go(func() error {
        defer sem.Release(1)

        recs, err := s.ScrapePageCtx(ctx, url)
        if err != nil {
            fmt.Println("error:", err, "url:", url)
            return nil // keep going
        }
        fmt.Println( "done:", url, "records:", len(recs))

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