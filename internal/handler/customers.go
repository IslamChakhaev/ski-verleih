package handler

import "net/http"

type customerRequest struct {
	Name     string `json:"name"`
	IDType   string `json:"idType"`
	IDNumber string `json:"idNumber"`
}

func (h *Handler) listCustomers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.Customers())
}

func (h *Handler) createCustomer(w http.ResponseWriter, r *http.Request) {
	var req customerRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	customer, err := h.store.AddCustomer(req.Name, req.IDType, req.IDNumber)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, customer)
}

func (h *Handler) updateCustomer(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var req customerRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	customer, err := h.store.UpdateCustomer(id, req.Name, req.IDType, req.IDNumber)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, customer)
}
