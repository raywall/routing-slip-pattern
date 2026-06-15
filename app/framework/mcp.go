package framework

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/raywall/routing-slip-pattern/app/slip"
)

func newMCPHandler(workflow Workflow, store slip.StateStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var request struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
			Params struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			} `json:"params"`
		}
		if r.Method != http.MethodPost || json.NewDecoder(r.Body).Decode(&request) != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if request.Method == "tools/list" {
			mcpResult(w, request.ID, map[string]any{"tools": []map[string]any{
				{"name": "explain_workflow", "description": "Explica etapas e regras de negocio do workflow.", "inputSchema": mcpSchema()},
				{"name": "list_business_rules", "description": "Lista regras de negocio documentadas no workflow.", "inputSchema": mcpSchema()},
				{"name": "get_execution", "description": "Recupera uma execucao por message_id.", "inputSchema": mcpSchema()},
				{"name": "find_executions", "description": "Busca execucoes por correlation_id, trace_id, status ou workflow.", "inputSchema": mcpSchema()},
			}})
			return
		}
		if request.Method != "tools/call" {
			mcpError(w, request.ID, "method not found")
			return
		}
		switch request.Params.Name {
		case "explain_workflow":
			mcpResult(w, request.ID, workflow)
		case "list_business_rules":
			mcpResult(w, request.ID, workflow.BusinessRules)
		case "get_execution":
			snapshot, err := store.Load(r.Context(), fmt.Sprint(request.Params.Arguments["message_id"]))
			if err != nil {
				mcpError(w, request.ID, err.Error())
				return
			}
			mcpResult(w, request.ID, snapshot)
		case "find_executions":
			lister, ok := store.(slip.StateSnapshotLister)
			if !ok {
				mcpError(w, request.ID, "state store does not support listing")
				return
			}
			filter := slip.SnapshotFilter{
				CorrelationID: fmt.Sprint(request.Params.Arguments["correlation_id"]),
				TraceID:       fmt.Sprint(request.Params.Arguments["trace_id"]),
				Status:        fmt.Sprint(request.Params.Arguments["status"]),
				Workflow:      fmt.Sprint(request.Params.Arguments["workflow"]),
				Limit:         20,
			}
			snapshots, err := lister.List(r.Context(), filter)
			if err != nil {
				mcpError(w, request.ID, err.Error())
				return
			}
			mcpResult(w, request.ID, snapshots)
		default:
			mcpError(w, request.ID, "unknown tool")
		}
	})
}

func mcpSchema() map[string]any {
	return map[string]any{"type": "object", "additionalProperties": true}
}
func mcpResult(w http.ResponseWriter, id, result any) {
	_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}
func mcpError(w http.ResponseWriter, id any, message string) {
	_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": -32000, "message": message}})
}
