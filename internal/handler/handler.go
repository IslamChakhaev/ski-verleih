// Package handler exposes the store as an HTTP/JSON API. It only translates
// between HTTP and the store; business rules live in the domain package.
package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strconv"

	"ski-verleih/internal/domain"
	"ski-verleih/internal/store"
)

const maxBodyBytes = 1 << 20

// Handler holds the dependencies of the HTTP handlers.
type Handler struct {
	store *store.Store
}

// New creates a Handler that works on the given store.
func New(s *store.Store) *Handler {
	return &Handler{store: s}
}

// Routes returns the API routes. Patterns with methods need Go 1.22 or newer.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/stats", h.stats)

	mux.HandleFunc("GET /api/skis", h.listSkis)
	mux.HandleFunc("POST /api/skis", h.createSki)
	mux.HandleFunc("PUT /api/skis/{id}", h.updateSki)

	mux.HandleFunc("GET /api/customers", h.listCustomers)
	mux.HandleFunc("POST /api/customers", h.createCustomer)
	mux.HandleFunc("PUT /api/customers/{id}", h.updateCustomer)

	mux.HandleFunc("GET /api/tariffs", h.listTariffs)
	mux.HandleFunc("POST /api/tariffs", h.createTariff)
	mux.HandleFunc("PUT /api/tariffs/{id}", h.updateTariff)
	mux.HandleFunc("GET /api/price-quote", h.priceQuote)

	mux.HandleFunc("GET /api/rentals", h.listRentals)
	mux.HandleFunc("POST /api/rentals", h.createRental)
	mux.HandleFunc("POST /api/rentals/{id}/return", h.returnRental)

	return localOnly(mux)
}

// localOnly protects the API from other websites open in the same browser.
func localOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLocalRequest(r) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "Der Zugriff ist nur über diese Anwendung erlaubt"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isLocalRequest checks the Host header against DNS rebinding and the Origin
// header against requests sent by other websites. Browsers always send Origin
// on cross-site POST and PUT requests; tools like curl may omit it.
func isLocalRequest(r *http.Request) bool {
	hostname := r.Host
	if host, _, err := net.SplitHostPort(r.Host); err == nil {
		hostname = host
	}
	if hostname != "127.0.0.1" && hostname != "localhost" {
		return false
	}
	origin := r.Header.Get("Origin")
	return origin == "" || origin == "http://"+r.Host
}

// decodeJSON reads the request body into target. The technical cause is logged
// because the response only carries a German message for users.
func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		log.Printf("%s %s: invalid request body: %v", r.Method, r.URL.Path, err)
		return domain.InvalidInput("Die Anfrage enthält ungültige oder unvollständige Daten")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		log.Printf("write response: %v", err)
	}
}

// writeError maps domain errors to HTTP status codes. Unexpected errors are
// logged and hidden from the client.
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrSkiUnavailable), errors.Is(err, domain.ErrAlreadyReturned):
		status = http.StatusConflict
	}

	message := err.Error()
	if status == http.StatusInternalServerError {
		log.Printf("internal error: %v", err)
		message = "Interner Serverfehler"
	}
	writeJSON(w, status, map[string]string{"error": message})
}

// pathID reads the {id} part of the URL.
func pathID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		return 0, domain.InvalidInput("Die ID ist ungültig")
	}
	return id, nil
}

// queryInt reads a whole number from the URL query. The label names the value
// in the German error message.
func queryInt(r *http.Request, key, label string) (int, error) {
	value, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return 0, domain.InvalidInput(label + " fehlt oder ist keine ganze Zahl")
	}
	return value, nil
}
