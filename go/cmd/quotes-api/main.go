package main

import (
	"bytes"
	"context"
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

	client := newHTTPClient()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/classify", classifyHandler(client, mlURL))
	mux.HandleFunc("/generate", generateHandler(client, mlURL))
	mux.HandleFunc("/", rootHandler)

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

func newHTTPClient() *http.Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 3 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	return &http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func classifyHandler(client *http.Client, mlURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()

		var in classifyRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		body, err := json.Marshal(in)
		if err != nil {
			http.Error(w, "failed to encode request", http.StatusInternalServerError)
			return
		}

		respBody, status, err := postJSON(r.Context(), client, mlURL+"/classify", body)
		if err != nil {
			http.Error(w, "ml service unavailable: "+err.Error(), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(respBody)
	}
}

func generateHandler(client *http.Client, mlURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}

		respBody, status, err := postJSON(r.Context(), client, mlURL+"/generate", body)
		if err != nil {
			http.Error(w, "ml service unavailable: "+err.Error(), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(respBody)
	}
}

func rootHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(makeHTML()))
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
  body {
    font-family: system-ui, sans-serif;
    max-width: 720px;
    margin: 40px auto;
    padding: 0 16px;
  }
  textarea {
    width: 100%;
    min-height: 130px;
    font-size: 16px;
    padding: 12px;
  }
  button {
    padding: 10px 14px;
    margin-top: 8px;
    font-size: 15px;
    cursor: pointer;
  }
  select {
    padding: 8px;
    font-size: 14px;
  }
  .row {
    display: flex;
    gap: 12px;
    align-items: center;
    margin-top: 8px;
  }
  .label {
    font-weight: 600;
  }
  #probs {
    margin-top: 12px;
    max-width: 460px;
  }
  .barwrap {
    background: #eee;
    border-radius: 4px;
    overflow: hidden;
    height: 12px;
  }
  .legend {
    font-weight: 600;
    margin-bottom: 2px;
  }
  .pct {
    float: right;
    font-size: 13px;
    color: #444;
  }
</style>
</head>

<body>
<h1>Quote Classifier</h1>
<p>Enter text to classify, or generate a quote by topic.</p>

<textarea id="txt" placeholder="e.g., The only source of knowledge is experience."></textarea>

<div class="row">
  <button id="btn-classify">Classify</button>

  <select id="topic">
    <option value="Inspirational">Inspirational</option>
    <option value="Philosophical">Philosophical</option>
    <option value="Love">Love</option>
    <option value="Romantic">Romantic</option>
    <option value="Humor">Humor</option>
  </select>

  <button id="btn-generate">Generate</button>
  <span id="status"></span>
</div>

<h3>Result</h3>
<div class="row">
  <span class="label">Top label:</span>
  <span id="label">—</span>
</div>

<h4>Probabilities</h4>
<div id="probs"></div>

<script>
const txt = document.getElementById('txt');
const classifyBtn = document.getElementById('btn-classify');
const generateBtn = document.getElementById('btn-generate');
const topicSel = document.getElementById('topic');
const labelEl = document.getElementById('label');
const statusEl = document.getElementById('status');
const probsDiv = document.getElementById('probs');

/* -------------------------
   CLASSIFY
------------------------- */
classifyBtn.addEventListener('click', async () => {
  const text = txt.value.trim();
  if (!text) {
    alert('Please enter text to classify.');
    return;
  }

  statusEl.textContent = 'classifying...';
  probsDiv.innerHTML = '';
  labelEl.textContent = '—';

  try {
    const res = await fetch('/classify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ text })
    });

    if (!res.ok) {
      throw new Error('HTTP ' + res.status);
    }

    const data = await res.json();

    labelEl.textContent = data.label || '—';

    const probs = data.probs || {};
    probsDiv.innerHTML = '';

    Object.entries(probs).forEach(([label, p]) => {
      const percent = Math.round(p * 1000) / 10;

      const row = document.createElement('div');
      row.style.margin = '6px 0';

      row.innerHTML =
        '<div class="legend">' + label +
        '<span class="pct">' + percent + '%</span></div>' +
        '<div class="barwrap">' +
        '<div style="width:' + percent + '%; height:100%; background:#4c8bf5;"></div>' +
        '</div>';

      probsDiv.appendChild(row);
    });

    statusEl.textContent = 'done';
  } catch (err) {
    statusEl.textContent = 'error';
    alert('Classification failed: ' + err.message);
  }
});

/* -------------------------
   GENERATE
------------------------- */
generateBtn.addEventListener('click', async () => {
  const topic = topicSel.value;

  statusEl.textContent = 'generating...';
  probsDiv.innerHTML = '';
  labelEl.textContent = '—';

  try {
    const res = await fetch('/generate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        topic: topic,
        min_confidence: 0.6
      })
    });

    if (!res.ok) {
      throw new Error('HTTP ' + res.status);
    }

    const data = await res.json();

    txt.value = data.quote || '';
    labelEl.textContent = topic;
    statusEl.textContent =
      data.confidence !== undefined
        ? 'confidence ' + Math.round(data.confidence * 1000) / 10 + '%'
        : 'done';

  } catch (err) {
    statusEl.textContent = 'error';
    alert('Generation failed: ' + err.message);
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
func postJSON(
	ctx context.Context,
	client *http.Client,
	url string,
	body []byte,
) ([]byte, int, error) {

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	return b, resp.StatusCode, nil
}