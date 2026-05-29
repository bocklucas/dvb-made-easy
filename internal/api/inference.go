package api

import (
	"encoding/json"
	"net/http"

	"github.com/bocklucas/dvb-made-easy/internal/compose"
)

type inferRequest struct {
	ComposeContent string `json:"compose_content"`
}

func (s *Server) handleInfer(w http.ResponseWriter, r *http.Request) {
	var req inferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.ComposeContent == "" {
		http.Error(w, `{"error":"compose_content is required"}`, http.StatusBadRequest)
		return
	}

	result, err := compose.Infer(req.ComposeContent)
	if err != nil {
		http.Error(w, `{"error":"failed to parse compose file"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
