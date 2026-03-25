package proxy

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/llm-proxy/internal/backend"
	"github.com/llm-proxy/internal/router"
)

type Handler struct {
	router *router.Router
}

func NewHandler(r *router.Router) *Handler {
	return &Handler{router: r}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/health":
		h.handleHealth(w, r)
	case r.URL.Path == "/v1/chat/completions" && r.Method == "POST":
		h.handleChatCompletions(w, r)
	case r.URL.Path == "/v1/models" && r.Method == "GET":
		h.handleModels(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
	})
}

func (h *Handler) handleModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"object": "list",
		"data": []map[string]interface{}{
			{"id": "gpt-4o", "object": "model", "owned_by": "openai"},
			{"id": "gpt-4o-mini", "object": "model", "owned_by": "openai"},
			{"id": "claude-3-5-sonnet-20241022", "object": "model", "owned_by": "anthropic"},
			{"id": "claude-3-haiku-20240307", "object": "model", "owned_by": "anthropic"},
			{"id": "gemini-pro", "object": "model", "owned_by": "google"},
		},
	})
}

func (h *Handler) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	var req backend.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Model == "" {
		http.Error(w, "Model is required", http.StatusBadRequest)
		return
	}

	b, err := h.router.Resolve(req.Model)
	if err != nil {
		http.Error(w, "No backend available for model", http.StatusBadGateway)
		return
	}

	log.Printf("Routing model %s to backend %s", req.Model, b.Name())

	if req.Stream {
		h.handleStream(w, r, b, &req)
	} else {
		h.handleSync(w, r, b, &req)
	}
}

func (h *Handler) handleSync(w http.ResponseWriter, r *http.Request, b backend.Backend, req *backend.Request) {
	resp, err := b.ChatCompletion(r.Context(), req)
	if err != nil {
		log.Printf("Backend error: %v", err)
		http.Error(w, "Backend error: "+err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleStream(w http.ResponseWriter, r *http.Request, b backend.Backend, req *backend.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := b.ChatCompletionStream(r.Context(), req, w); err != nil {
		if !strings.Contains(err.Error(), "context canceled") {
			log.Printf("Stream error: %v", err)
		}
		return
	}
}
