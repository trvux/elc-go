package presentation

import "github.com/go-chi/chi/v5"

// RegisterRoutes mounts the AI chat module's routes. Public/anonymous — the
// public site's chat widget calls this without logging in — the rate
// limiter in AIHandler is the abuse defense, same pattern as
// internal/inquiry's public Create endpoint.
func RegisterRoutes(r chi.Router, h *AIHandler) {
	r.Post("/ai/chat", h.Chat)
}
