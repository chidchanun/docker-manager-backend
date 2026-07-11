package handler

import (
	"docker-manager-backend/internal/audit"
	"docker-manager-backend/internal/response"
	"net/http"
	"strconv"
)

type AuditHandler struct{ store *audit.Store }

func NewAuditHandler(store *audit.Store) *AuditHandler { return &AuditHandler{store: store} }
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	count := int64(100)
	if value, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64); value > 0 && value <= 1000 {
		count = value
	}
	items, err := h.store.List(r.Context(), count)
	if err != nil {
		response.Error(w, 500, "Cannot read audit log")
		return
	}
	response.JSON(w, 200, map[string]any{"items": items})
}
