package handler

import (
	"encoding/json"
	"net/http"

	"github.com/magomedcoder/repo/pkg/i18n"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, r *http.Request, status int, message string) {
	writeJSON(w, status, map[string]string{"error": i18n.T(r.Context(), message)})
}
