package api_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestMetricsEndpoint(t *testing.T) {
	server := newTestServer(newFakeStore())

	// Generate one request so the counter has something to show.
	doRequest(server, http.MethodGet, "/health", "", nil)

	rec := doRequest(server, http.MethodGet, "/metrics", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	body := rec.Body.String()
	for _, metric := range []string{
		"mb_http_requests_total",
		"mb_http_request_duration_seconds",
		"go_goroutines",
	} {
		if !strings.Contains(body, metric) {
			t.Errorf("metrics output missing %q", metric)
		}
	}

	// The route label must be the PATTERN, not a raw path — cardinality guard.
	if !strings.Contains(body, `route="/health"`) {
		t.Errorf("expected route label for /health, body sample: %.300s", body)
	}
}
