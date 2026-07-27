// Package goatcounter talks to GoatCounter's export API. It is the only part
// of quinto that touches the network — everything else reads the local file.
//
// The behaviour encoded here was verified against the live API on 2026-07-25,
// because the published docs turned out to be wrong or silent on four points:
//
//   - format must be sent explicitly; posting {} yields `unknown format: ""`
//   - start_from_hit_id is the CSV cursor; JSON only offers start_from_day
//   - exports are rate limited to roughly one per hour, separate from the
//     documented 4 req/s
//   - that hourly export budget is scoped to the account, not the site: an
//     API token is valid for every site on the account (its own settings page
//     says so), and confirmed by observation on 2026-07-27 — exporting a
//     second, never-before-exported site immediately after the first still
//     returned 429, with the same rate-limit-reset countdown the first site's
//     response carried. The window is a fixed deadline anchored to that first
//     export, not a rolling one: two 429s 321s apart reported reset values of
//     1551s and 1230s, a difference of exactly the elapsed time.
//   - the 429 body's "try again in ..." duration is mirrored in response
//     headers on every export call, 2xx or not: x-rate-limit-limit,
//     x-rate-limit-remaining, x-rate-limit-reset, and (429 only) retry-after
//   - authenticated requests intermittently return 404, then succeed on retry
package goatcounter

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

const (
	// maxAttempts covers the transient 404s and 5xx responses observed on
	// otherwise valid requests. Only ever applied to idempotent reads.
	maxAttempts = 4
	retryDelay  = 700 * time.Millisecond
)

type Client struct {
	Site  string // full host, e.g. "example.goatcounter.com"
	Token string
	HTTP  *http.Client
}

func New(site, token string) *Client {
	return &Client{
		Site:  site,
		Token: token,
		HTTP:  &http.Client{Timeout: 60 * time.Second},
	}
}

// Export is a GoatCounter export job. FinishedAt is nil until it is ready.
type Export struct {
	ID         int64      `json:"id"`
	SiteID     int64      `json:"site_id"`
	Format     string     `json:"format"`
	NumRows    int64      `json:"num_rows"`
	LastHitID  int64      `json:"last_hit_id"`
	CreatedAt  *time.Time `json:"created_at"`
	FinishedAt *time.Time `json:"finished_at"`
	Error      string     `json:"error"`
}

// RateLimitError reports the hourly export budget being spent. This is a
// normal operating state for quinto, not a failure — callers should tell the
// user when the next sync is possible rather than treating it as an error.
//
// The budget is per account, not per site (see the package comment) — a
// RateLimitError from one site means every site on the account is blocked
// until RetryAfter elapses, not just this one.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("export rate limited; next sync possible in %s", e.RetryAfter.Round(time.Minute))
}

// APIError is any other non-2xx response.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("goatcounter API: HTTP %d: %s", e.StatusCode, e.Message)
}

// "rate limited exceeded; try again in 59m44.782961283s"
var retryAfterRe = regexp.MustCompile(`try again in ([0-9hms.]+)`)

func (c *Client) do(ctx context.Context, method, path string, body any, retry bool) ([]byte, *http.Response, error) {
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return nil, nil, fmt.Errorf("encoding request: %w", err)
		}
	}

	var lastErr error
	attempts := 1
	if retry {
		attempts = maxAttempts
	}

	for attempt := 1; attempt <= attempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method,
			"https://"+c.Site+"/api/v0"+path, bytes.NewReader(payload))
		if err != nil {
			return nil, nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.Token)

		res, err := c.HTTP.Do(req)
		if err != nil {
			lastErr = err
			if attempt < attempts {
				time.Sleep(retryDelay)
				continue
			}
			return nil, nil, err
		}

		// Retry the transient 404s and server errors seen against the live
		// API. A 404 here is not reliable evidence of a wrong URL.
		if retry && attempt < attempts && (res.StatusCode == http.StatusNotFound || res.StatusCode >= 500) {
			res.Body.Close()
			time.Sleep(retryDelay)
			continue
		}

		raw, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("reading response: %w", err)
		}

		if res.StatusCode == http.StatusTooManyRequests {
			return nil, res, &RateLimitError{RetryAfter: parseRetryAfter(raw)}
		}
		if res.StatusCode < 200 || res.StatusCode > 299 {
			return nil, res, &APIError{StatusCode: res.StatusCode, Message: apiMessage(raw)}
		}
		return raw, res, nil
	}
	return nil, nil, lastErr
}

func parseRetryAfter(body []byte) time.Duration {
	if m := retryAfterRe.FindSubmatch(body); m != nil {
		if d, err := time.ParseDuration(string(m[1])); err == nil {
			return d
		}
	}
	return time.Hour // documented budget; a safe assumption when unparseable
}

func apiMessage(body []byte) string {
	var e struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &e) == nil && e.Error != "" {
		return e.Error
	}
	if len(body) > 200 {
		body = body[:200]
	}
	return string(body)
}

// CreateExport starts a CSV export. When startFromHitID is non-nil only hits
// with a greater ID are included — GoatCounter's own cursor, which is why
// quinto stores no watermark of its own.
//
// The transient 404s this API returns can land here too, and observation shows
// **the export is still created when they do** — the request succeeds server
// side and the response fails. That is the worst possible shape: the hourly
// budget is spent, the export's ID is lost with the response, and the caller
// sees a bare "not found".
//
// So a 404 is retried exactly once. If the first attempt really did create an
// export, the retry comes back 429 and the caller can report an honest wait
// instead of an error. If it did not, the retry simply succeeds. Other failures
// are not retried: for those a duplicate creation is plausible, and silently
// spending two hours of budget is worse than failing loudly.
func (c *Client) CreateExport(ctx context.Context, startFromHitID *int64) (*Export, error) {
	body := map[string]any{"format": "csv"}
	if startFromHitID != nil {
		body["start_from_hit_id"] = *startFromHitID
	}

	raw, _, err := c.do(ctx, http.MethodPost, "/export", body, false)

	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
		time.Sleep(retryDelay)
		raw, _, err = c.do(ctx, http.MethodPost, "/export", body, false)
	}
	if err != nil {
		return nil, err
	}
	var ex Export
	if err := json.Unmarshal(raw, &ex); err != nil {
		return nil, fmt.Errorf("decoding export: %w", err)
	}
	return &ex, nil
}

// GetExport reports on an export job.
func (c *Client) GetExport(ctx context.Context, id int64) (*Export, error) {
	raw, _, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/export/%d", id), nil, true)
	if err != nil {
		return nil, err
	}
	var ex Export
	if err := json.Unmarshal(raw, &ex); err != nil {
		return nil, fmt.Errorf("decoding export: %w", err)
	}
	return &ex, nil
}

// WaitForExport polls until the job finishes or ctx is done.
func (c *Client) WaitForExport(ctx context.Context, id int64, poll time.Duration) (*Export, error) {
	for {
		ex, err := c.GetExport(ctx, id)
		if err != nil {
			return nil, err
		}
		if ex.Error != "" {
			return ex, fmt.Errorf("export %d failed: %s", id, ex.Error)
		}
		if ex.FinishedAt != nil {
			return ex, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(poll):
		}
	}
}

// DownloadExport fetches a finished export and gunzips it. GoatCounter serves
// these as application/gzip; an export with no data is a valid gzip stream
// containing zero bytes, which is "no new hits", not an error.
func (c *Client) DownloadExport(ctx context.Context, id int64) ([]byte, error) {
	raw, _, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/export/%d/download", id), nil, true)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}

	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		// Some transports decompress transparently; fall back to the raw body.
		return raw, nil
	}
	defer zr.Close()

	out, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("decompressing export %d: %w", id, err)
	}
	return out, nil
}
