package scraper

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
)

func WriteJSONL(path string, records []QuoteRecord) error {
	if len(records) == 0 {
		return nil
	}

	// Δημιουργεί φακέλους αν δεν υπάρχουν
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	
	f, err := os.Create(path)
	if err != nil {return err}

	defer f.Close()

	writer := bufio.NewWriter(f)
	enc := json.NewEncoder(writer)
	for _, record := range(records) {
		if err := enc.Encode(record); err != nil {return err}

	}
	return writer.Flush()
}