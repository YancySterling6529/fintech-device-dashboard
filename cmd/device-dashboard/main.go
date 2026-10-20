package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/infrai-examples/fintech-device-dashboard/internal/dashboard"
	api "github.com/infrai-examples/fintech-device-dashboard/internal/infrai"
)

func main() {
	client, err := api.NewClient(os.Getenv("INFRAI_BASE_URL"), os.Getenv("INFRAI_API_KEY"), &http.Client{Timeout: 10 * time.Second})
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.CreateChannel(ctx, dashboard.DeviceChannel); err != nil {
		log.Fatalf("create dashboard channel: %v", err)
	}

	pipeline := dashboard.NewPipeline(client)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /payments", paymentHandler(pipeline))
	mux.HandleFunc("GET /dashboard/metrics", metricsHandler(client))

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("device dashboard listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func paymentHandler(pipeline *dashboard.Pipeline) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var event dashboard.PaymentEvent
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&event); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payment event"})
			return
		}
		notification, err := pipeline.Process(r.Context(), event)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, notification)
	}
}

func metricsHandler(client *api.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		allowed := map[string]bool{
			"name": true, "agg": true, "tags": true, "window": true, "step": true,
			"since": true, "until": true, "from": true, "to": true,
		}
		for field := range params {
			if !allowed[field] {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported metrics field: " + field})
				return
			}
		}
		if params.Get("name") == "" || params.Get("agg") == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and agg are required"})
			return
		}
		var result json.RawMessage
		if err := client.QueryMetrics(r.Context(), params, &result); err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func writeServiceError(w http.ResponseWriter, err error) {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
		writeJSON(w, apiErr.StatusCode, map[string]string{"error": apiErr.Message, "code": apiErr.Code})
		return
	}
	writeJSON(w, http.StatusBadGateway, map[string]string{"error": "upstream request failed"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}
