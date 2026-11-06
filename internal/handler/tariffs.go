package handler

import (
	"net/http"

	"ski-verleih/internal/domain"
)

// tariffRequest uses a pointer so that a missing price can be told apart from
// a free tariff with a price of 0.
type tariffRequest struct {
	Name             string `json:"name"`
	HourlyPriceCents *int64 `json:"hourlyPriceCents"`
}

type priceQuoteResponse struct {
	TariffName       string `json:"tariffName"`
	HourlyPriceCents int64  `json:"hourlyPriceCents"`
	DurationHours    int    `json:"durationHours"`
	TotalPriceCents  int64  `json:"totalPriceCents"`
}

// decodeTariff reads the tariff name and price from the request body.
func decodeTariff(w http.ResponseWriter, r *http.Request) (string, int64, error) {
	var req tariffRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return "", 0, err
	}
	if req.HourlyPriceCents == nil {
		return "", 0, domain.InvalidInput("Stundenpreis fehlt")
	}
	return req.Name, *req.HourlyPriceCents, nil
}

func (h *Handler) listTariffs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.Tariffs())
}

func (h *Handler) createTariff(w http.ResponseWriter, r *http.Request) {
	name, cents, err := decodeTariff(w, r)
	if err != nil {
		writeError(w, err)
		return
	}
	tariff, err := h.store.AddTariff(name, cents)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, tariff)
}

func (h *Handler) updateTariff(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	name, cents, err := decodeTariff(w, r)
	if err != nil {
		writeError(w, err)
		return
	}
	tariff, err := h.store.UpdateTariff(id, name, cents)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tariff)
}

// priceQuote previews a price without creating a rental, so the frontend does
// not have to duplicate the calculation.
func (h *Handler) priceQuote(w http.ResponseWriter, r *http.Request) {
	tariffID, err := queryInt(r, "tariffId", "Tarif")
	if err != nil {
		writeError(w, err)
		return
	}
	hours, err := queryInt(r, "hours", "Verleihdauer")
	if err != nil {
		writeError(w, err)
		return
	}
	tariff, err := h.store.Tariff(tariffID)
	if err != nil {
		writeError(w, err)
		return
	}
	total, err := tariff.Price(hours)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, priceQuoteResponse{
		TariffName:       tariff.Name,
		HourlyPriceCents: tariff.HourlyPriceCents,
		DurationHours:    hours,
		TotalPriceCents:  total,
	})
}
