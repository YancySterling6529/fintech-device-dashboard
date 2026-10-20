package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	api "github.com/infrai-examples/fintech-device-dashboard/internal/infrai"
)

func TestMetricsHandlerForwardsContractFields(t *testing.T) {
	var upstreamQuery string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":{"points":[]}}`))
	}))
	defer upstream.Close()

	client, err := api.NewClient(upstream.URL, "test-key", upstream.Client())
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/dashboard/metrics?name=device_payment_decisions&agg=count&window=1h", nil)
	res := httptest.NewRecorder()
	metricsHandler(client).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	if upstreamQuery != "agg=count&name=device_payment_decisions&window=1h" {
		t.Fatalf("upstream query = %q", upstreamQuery)
	}
}

func TestMetricsHandlerRejectsInvalidShape(t *testing.T) {
	client, err := api.NewClient("http://example.test", "test-key", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{
		"/dashboard/metrics?query=device_payment_decisions",
		"/dashboard/metrics?name=device_payment_decisions",
	} {
		res := httptest.NewRecorder()
		metricsHandler(client).ServeHTTP(res, httptest.NewRequest(http.MethodGet, target, nil))
		if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "error") {
			t.Fatalf("target %q: status = %d, body = %s", target, res.Code, res.Body.String())
		}
	}
}
