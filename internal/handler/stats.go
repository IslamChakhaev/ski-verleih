package handler

import "net/http"

func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.Stats())
}
