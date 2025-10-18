package scraper

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
)

// WriteJSONL writes a slice of QuoteRecord structs to a JSONL file.
//
// Parameters:
//   - path: The path to the JSONL file.
//   - records: The slice of QuoteRecord structs to write.
//
// Returns:
//   - error: Non-nil if there was an error writing the file.
//
// Example:
//
//	quotes, err := scraper.ScrapePage("https://www.azquotes.com/quotes/topics/inspiring.html")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if err := scraper.WriteJSONL("quotes.jsonl", quotes); err != nil {
//	    log.Fatal(err)
//	}
func WriteJSONL(path string, records []QuoteRecord) error {
	if len(records) == 0 {
		return nil
	}

	// Create the parent directory
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	defer f.Close()

	writer := bufio.NewWriter(f)
	enc := json.NewEncoder(writer)
	for _, record := range records {
		if err := enc.Encode(record); err != nil {
			return err
		}

	}
	return writer.Flush()
}
