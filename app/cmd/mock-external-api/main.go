package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

type customer struct {
	ID           string  `json:"id"`
	Status       string  `json:"status"`
	RiskSegment  string  `json:"riskSegment"`
	CreditLimit  float64 `json:"creditLimit"`
	SourceSystem string  `json:"sourceSystem"`
}

func main() {
	http.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	})
	http.HandleFunc("GET /customers/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.PathValue("id"))
		profile := customer{
			ID:           id,
			Status:       "ACTIVE",
			RiskSegment:  "LOW",
			CreditLimit:  2500,
			SourceSystem: "mock-external-api",
		}
		switch strings.ToLower(id) {
		case "cust-blocked":
			profile.Status = "BLOCKED"
			profile.RiskSegment = "HIGH"
			profile.CreditLimit = 0
		case "cust-resume":
			profile.RiskSegment = "MEDIUM"
			profile.CreditLimit = 900
		}
		writeJSON(w, profile)
	})
	log.Println("mock external API listening on :8091")
	log.Fatal(http.ListenAndServe(":8091", nil))
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}
