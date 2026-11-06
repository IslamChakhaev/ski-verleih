package handler

import "net/http"

type skiRequest struct {
	Name string `json:"name"`
}

func (h *Handler) listSkis(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.Skis())
}

func (h *Handler) createSki(w http.ResponseWriter, r *http.Request) {
	var req skiRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	ski, err := h.store.AddSki(req.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ski)
}

func (h *Handler) updateSki(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var req skiRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	ski, err := h.store.UpdateSki(id, req.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ski)
}
