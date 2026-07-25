package goatcounter

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestClient points a Client at a test server. The Site field carries a
// host, so the handler is reached by overriding the transport.
func newTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	c := New(strings.TrimPrefix(srv.URL, "http://"), "test-token")
	c.HTTP = srv.Client()
	c.HTTP.Transport = rewriteToHTTP{srv.Client().Transport}
	return c
}

// rewriteToHTTP downgrades the https:// the client builds to the test
// server's http://, without weakening the production code path.
type rewriteToHTTP struct{ base http.RoundTripper }

func (r rewriteToHTTP) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	base := r.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

func TestCreateExportSendsFormatAndCursor(t *testing.T) {
	var gotBody map[string]any
	var gotAuth string

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"id":11266,"site_id":1,"format":"csv"}`))
	})

	cursor := int64(4321)
	ex, err := c.CreateExport(context.Background(), &cursor)
	if err != nil {
		t.Fatalf("CreateExport: %v", err)
	}
	if ex.ID != 11266 {
		t.Errorf("id = %d, want 11266", ex.ID)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	// The live API rejects a missing format with `unknown format: ""`.
	if gotBody["format"] != "csv" {
		t.Errorf("format = %v, want csv", gotBody["format"])
	}
	if gotBody["start_from_hit_id"] != float64(4321) {
		t.Errorf("start_from_hit_id = %v, want 4321", gotBody["start_from_hit_id"])
	}
}

func TestCreateExportOmitsCursorOnFirstSync(t *testing.T) {
	var gotBody map[string]any
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"id":1}`))
	})

	if _, err := c.CreateExport(context.Background(), nil); err != nil {
		t.Fatalf("CreateExport: %v", err)
	}
	if _, present := gotBody["start_from_hit_id"]; present {
		t.Error("start_from_hit_id must be absent when there is no cursor, not zero")
	}
}

// A 429 is a normal state — the caller needs the retry window, not an error
// string to show a user.
func TestRateLimitIsTypedAndCarriesRetryWindow(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "rate limited exceeded; try again in 59m44.782961283s"}`))
	})

	_, err := c.CreateExport(context.Background(), nil)
	var rl *RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("error = %v, want *RateLimitError", err)
	}
	want, _ := time.ParseDuration("59m44.782961283s")
	if rl.RetryAfter != want {
		t.Errorf("RetryAfter = %v, want %v", rl.RetryAfter, want)
	}
}

func TestRateLimitFallsBackToAnHourWhenUnparseable(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"slow down"}`))
	})

	_, err := c.CreateExport(context.Background(), nil)
	var rl *RateLimitError
	if !errors.As(err, &rl) || rl.RetryAfter != time.Hour {
		t.Fatalf("err = %v, want 1h fallback", err)
	}
}

// The live API returns 404 on valid authenticated requests, then succeeds
// immediately after. Reads must survive that.
func TestTransient404IsRetried(t *testing.T) {
	var calls int
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not found"}`))
			return
		}
		w.Write([]byte(`{"id":7,"num_rows":2,"finished_at":"2026-07-25T10:43:17Z"}`))
	})

	ex, err := c.GetExport(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetExport: %v", err)
	}
	if ex.NumRows != 2 {
		t.Errorf("num_rows = %d, want 2", ex.NumRows)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3 (two 404s then success)", calls)
	}
}

// Creating an export is not idempotent and costs an hour of budget, so a
// failure must surface rather than being retried.
func TestCreateExportIsNotRetried(t *testing.T) {
	var calls int
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	})

	if _, err := c.CreateExport(context.Background(), nil); err == nil {
		t.Fatal("expected an error")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 — creating an export must never be retried", calls)
	}
}

func TestWaitForExportPollsUntilFinished(t *testing.T) {
	var calls int
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.Write([]byte(`{"id":7,"finished_at":null}`))
			return
		}
		w.Write([]byte(`{"id":7,"num_rows":5,"last_hit_id":99,"finished_at":"2026-07-25T10:43:17Z"}`))
	})

	ex, err := c.WaitForExport(context.Background(), 7, time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForExport: %v", err)
	}
	if ex.LastHitID != 99 || ex.NumRows != 5 {
		t.Errorf("got last_hit_id=%d num_rows=%d, want 99/5", ex.LastHitID, ex.NumRows)
	}
}

func TestWaitForExportSurfacesJobError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":7,"error":"something broke"}`))
	})

	if _, err := c.WaitForExport(context.Background(), 7, time.Millisecond); err == nil {
		t.Fatal("expected the job error to surface")
	}
}

func TestDownloadExportGunzips(t *testing.T) {
	const payload = "2,Path,Title\n/,Home\n"

	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	zw.Write([]byte(payload))
	zw.Close()

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.Write(gz.Bytes())
	})

	got, err := c.DownloadExport(context.Background(), 7)
	if err != nil {
		t.Fatalf("DownloadExport: %v", err)
	}
	if string(got) != payload {
		t.Errorf("got %q, want %q", got, payload)
	}
}

// An export with no new hits is a valid, empty gzip — "nothing new", not a
// failure. Observed live: 23 bytes, zero lines, no header.
func TestDownloadEmptyExportIsNotAnError(t *testing.T) {
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	zw.Close()

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.Write(gz.Bytes())
	})

	got, err := c.DownloadExport(context.Background(), 7)
	if err != nil {
		t.Fatalf("DownloadExport: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d bytes, want 0", len(got))
	}
}
