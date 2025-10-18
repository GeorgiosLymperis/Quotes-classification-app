package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

type classifyRequest struct {
	Text         string             `json:"text"`
	Probs        map[string]float64 `json:"probs"`
	ModelVersion string             `json:"model_version"`
}

// main starts a tiny HTTP gateway that proxies /classify requests
// to an upstream ML service (ML_URL) and serves a minimal HTML UI.
// Configuration:
//   - GATEWAY_ADDRESS (default ":8080")
//   - ML_URL (default "http://localhost:8000")
func main() {
	address := env("GATEWAY_ADDRESS", ":8080")
	mlURL := env("ML_URL", "http://localhost:8000")
	mux := http.NewServeMux()

	// Health probe endpoint.
	mux.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.Write([]byte(`{"status":"ok"}`))
	})

	// /classify proxies JSON to the upstream ML service.
	mux.HandleFunc("/classify", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		defer request.Body.Close()
		var in classifyRequest
		if err := json.NewDecoder(request.Body).Decode(&in); err != nil {
			http.Error(writer, "bad request", http.StatusBadRequest)
			return
		}

		body, _ := json.Marshal(in)
		responseBody, status, err := postJSON(mlURL+"/classify", body, 5*time.Second)
		if err != nil {
			// NOTE: returns 400 on upstream failure; consider 502 Bad Gateway.
			http.Error(writer, "ml service unavailable: "+err.Error(), http.StatusBadRequest)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(status)
		writer.Write(responseBody)
	})

	// Root serves a tiny interactive HTML UI.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(makeHTML()))
	})

	server := &http.Server{
		Addr:              address,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	fmt.Println("gateway listening on", address, "-> ML:", mlURL)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

// makeHTML returns the embedded single-file UI for manual testing.
func makeHTML() string {
	return `<!doctype html>
<html>
<head>
<meta charset="utf-8" />
<title>Quote Classifier</title>
<meta name="viewport" content="width=device-width, initial-scale=1" />
<style>
  body { font-family: system-ui, sans-serif; max-width: 720px; margin: 40px auto; padding: 0 16px; }
  textarea { width: 100%; min-height: 130px; font-size: 16px; padding: 12px; }
  button { padding: 10px 14px; margin-top: 8px; font-size: 15px; cursor: pointer; }
  .row { display: flex; gap: 12px; align-items: center; }
  .label { font-weight: 600; }
  /* probs styles */
  #probs { margin-top: 8px; max-width: 460px; }
  #probs .row { display: block; }
  #probs .barwrap { background:#eee; border-radius:4px; overflow:hidden; height:12px; }
  #probs .legend { font-weight:600; margin-bottom:2px; }
  #probs .pct { float:right; font-size:13px; color:#444; }
</style>
</head>
<body>
<h1>Quote Classifier</h1>
<p>Give a quote and click "Classify".</p>

<textarea id="txt" placeholder="e.g., The only source of knowledge is experience."></textarea>
<div class="row">
  <button id="btn">Classify</button>
  <span id="status"></span>
</div>

<h3>Result</h3>
<div class="row"><span class="label">Top label:</span> <span id="label">—</span></div>
<h4>Probabilities</h4>
<div id="probs"></div>

<script>
const btn = document.getElementById('btn');
const txt = document.getElementById('txt');
const labelEl = document.getElementById('label');
const statusEl = document.getElementById('status');
const probsDiv = document.getElementById('probs');

btn.addEventListener('click', async () => {
  const text = txt.value.trim();
  if (!text) { alert('Give me a quote first.'); return; }
  statusEl.textContent = '⏳ classifying...';
  probsDiv.innerHTML = '';
  try {
    const res = await fetch('/classify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ text })
    });
    if (!res.ok) throw new Error('HTTP ' + res.status);
    const data = await res.json();

    // set top label
    labelEl.textContent = data.label || '—';

    // render colorful bars
    const probs = data.probs || {};
    const entries = Object.entries(probs); // [["Inspirational", 0.8271], ...]
    probsDiv.innerHTML = '';
    entries.forEach(([label, p]) => {
      const percent = (p * 100).toFixed(1);
      const hue = Math.round(120 * p);           // 0=red .. 120=green
      const color = 'hsl(' + hue + ',70%,50%)';  // vivid, readable

      const row = document.createElement('div');
      row.className = 'row';
      row.style.margin = '6px 0';

      row.innerHTML =
        '<div class="legend">' + label +
        '<span class="pct">' + percent + '%</span></div>' +
        '<div class="barwrap"><div class="bar" ' +
        'style="width:' + percent + '%; height:100%; background:' + color + '; transition:width 0.6s ease;"></div></div>';

      probsDiv.appendChild(row);
    });

    statusEl.textContent = '✅';
  } catch (e) {
    statusEl.textContent = '❌';
    alert('Error: ' + e.message);
  }
});
</script>
</body>
</html>`
}

// env retrieves an environment variable or returns a default value.
func env(key, defaultKey string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultKey
}

// postJSON posts a JSON body to the given URL with a short timeout,
// returning the response body, status code, and any network error.
func postJSON(url string, body []byte, timeout time.Duration) ([]byte, int, error) {
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext:         (&net.Dialer{Timeout: 3 * time.Second}).DialContext,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}
	request, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()

	b, _ := io.ReadAll(response.Body)
	return b, response.StatusCode, nil
}
