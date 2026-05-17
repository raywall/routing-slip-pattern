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

type item struct {
	ProductID string `json:"produto_id"`
	Quantity  int    `json:"quantidade"`
}

type order struct {
	OrderID         string  `json:"pedido_id"`
	CustomerID      string  `json:"cliente_id"`
	Status          string  `json:"status"`
	TotalAmount     float64 `json:"valor_total"`
	DeliveryAddress string  `json:"endereco_entrega"`
	Items           []item  `json:"itens"`
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
	http.HandleFunc("GET /pedidos/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.PathValue("id"))
		if id == "" {
			http.Error(w, "pedido id is required", http.StatusBadRequest)
			return
		}
		writeJSON(w, order{
			OrderID:         id,
			CustomerID:      "CLI-001",
			Status:          "PAGO",
			TotalAmount:     150,
			DeliveryAddress: "Rua das Flores, 123 - Sao Paulo/SP",
			Items: []item{
				{ProductID: "PROD-123", Quantity: 1},
				{ProductID: "PROD-456", Quantity: 2},
			},
		})
	})
	http.HandleFunc("POST /lambda/notas-fiscais", func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		writeJSON(w, map[string]any{
			"nota_fiscal_id": "NF-2026-0001",
			"chave_acesso":   "35260500000000000190550010000000011000000010",
			"status":         "EMITIDA",
			"pedido_id":      payload["pedido_id"],
			"valor_total":    payload["valor_total"],
		})
	})
	http.HandleFunc("POST /api/expedicao", func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		writeJSON(w, map[string]any{
			"expedicao_id":     "EXP-112233",
			"codigo_rastreio":  "BR987654321BR",
			"status":           "PREPARANDO_PACOTE",
			"transportadora":   "Correios",
			"pedido_id":        payload["pedido_id"],
			"nota_fiscal_id":   payload["nota_fiscal_id"],
			"endereco_entrega": payload["endereco_entrega"],
		})
	})
	http.HandleFunc("PUT /api/estoque/baixar", func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		writeJSON(w, map[string]any{
			"status":    "SUCESSO",
			"mensagem":  "Estoque atualizado com sucesso.",
			"pedido_id": payload["pedido_id"],
			"itens":     payload["itens"],
		})
	})
	log.Println("mock external API listening on :8091")
	log.Fatal(http.ListenAndServe(":8091", nil))
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}
