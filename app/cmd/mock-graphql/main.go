package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

func main() {
	externalBaseURL := env("EXTERNAL_API_URL", "http://localhost:8091")
	client := &http.Client{Timeout: 2 * time.Second}

	http.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	})
	http.HandleFunc("POST /graphql", func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		customerID := fmt.Sprintf("%v", req.Variables["customerID"])
		customerID = strings.TrimSpace(customerID)
		if customerID == "" {
			writeJSON(w, map[string]any{"errors": []map[string]string{{"message": "customerID is required"}}})
			return
		}
		resp, err := client.Get(externalBaseURL + "/customers/" + customerID)
		if err != nil {
			writeJSON(w, map[string]any{"errors": []map[string]string{{"message": err.Error()}}})
			return
		}
		defer resp.Body.Close()
		var customer map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&customer); err != nil {
			writeJSON(w, map[string]any{"errors": []map[string]string{{"message": err.Error()}}})
			return
		}
		writeJSON(w, map[string]any{
			"data": map[string]any{
				"customer": customer,
			},
		})
	})

	log.Println("mock GraphQL connector listening on :8090")
	log.Fatal(http.ListenAndServe(":8090", nil))
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
