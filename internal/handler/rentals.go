package handler

import (
	"net/http"
	"time"
)

type rentalRequest struct {
	CustomerID    int       `json:"customerId"`
	SkiID         int       `json:"skiId"`
	TariffID      int       `json:"tariffId"`
	Start         time.Time `json:"start"`
	DurationHours int       `json:"durationHours"`
}

func (h *Handler) listRentals(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.Rentals())
}

func (h *Handler) createRental(w http.ResponseWriter, r *http.Request) {
	var req rentalRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	rental, err := h.store.CreateRental(req.CustomerID, req.SkiID, req.TariffID, req.Start, req.DurationHours)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, rental)
}

func (h *Handler) returnRental(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rental, err := h.store.ReturnRental(id, time.Now())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rental)
}
