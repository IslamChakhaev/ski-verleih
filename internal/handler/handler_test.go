package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ski-verleih/internal/domain"
	"ski-verleih/internal/handler"
	"ski-verleih/internal/store"
)

const rentalBody = `{"customerId":1,"skiId":1,"tariffId":1,"start":"2026-01-15T09:00:00+01:00","durationHours":3}`

// api sends requests to the real routes with an in-memory store.
type api struct {
	t      *testing.T
	routes http.Handler
}

func newAPI(t *testing.T) api {
	t.Helper()
	return api{t: t, routes: handler.New(store.New()).Routes()}
}

// newSeededAPI adds one ski, one customer and one tariff through the API.
func newSeededAPI(t *testing.T) api {
	t.Helper()
	a := newAPI(t)
	a.expect(http.MethodPost, "/api/skis", `{"name":"Carver 160"}`, http.StatusCreated)
	a.expect(http.MethodPost, "/api/customers", `{"name":"Max Mustermann","idType":"Reisepass","idNumber":"C0000001"}`, http.StatusCreated)
	a.expect(http.MethodPost, "/api/tariffs", `{"name":"Standard","hourlyPriceCents":500}`, http.StatusCreated)
	return a
}

func (a api) do(method, path, body string) *httptest.ResponseRecorder {
	a.t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Host = "127.0.0.1:8080"
	recorder := httptest.NewRecorder()
	a.routes.ServeHTTP(recorder, request)
	return recorder
}

func (a api) expect(method, path, body string, status int) *httptest.ResponseRecorder {
	a.t.Helper()
	response := a.do(method, path, body)
	if response.Code != status {
		a.t.Fatalf("%s %s: status %d, want %d, body %s", method, path, response.Code, status, response.Body)
	}
	return response
}

func decode[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatalf("decode %q: %v", response.Body, err)
	}
	return value
}

func (a api) assertStats(want domain.Stats) {
	a.t.Helper()
	got := decode[domain.Stats](a.t, a.expect(http.MethodGet, "/api/stats", "", http.StatusOK))
	if got != want {
		a.t.Errorf("stats = %+v, want %+v", got, want)
	}
}

func TestRentalAndReturnFlowKeepsDashboardCorrect(t *testing.T) {
	a := newAPI(t)
	a.assertStats(domain.Stats{})

	a.expect(http.MethodPost, "/api/skis", `{"name":"Carver 160"}`, http.StatusCreated)
	a.expect(http.MethodPost, "/api/skis", `{"name":"Allround 170"}`, http.StatusCreated)
	a.expect(http.MethodPost, "/api/customers", `{"name":"Max Mustermann","idType":"Reisepass","idNumber":"C0000001"}`, http.StatusCreated)
	a.expect(http.MethodPost, "/api/tariffs", `{"name":"Standard","hourlyPriceCents":500}`, http.StatusCreated)
	a.assertStats(domain.Stats{AvailableSkis: 2, TotalSkis: 2, RegisteredCustomers: 1})

	created := decode[domain.RentalDetails](t, a.expect(http.MethodPost, "/api/rentals", rentalBody, http.StatusCreated))
	if created.TotalPriceCents != 1500 || created.SkiName != "Carver 160" || !created.Active {
		t.Errorf("wrong rental: %+v", created)
	}
	if got := created.Start.Format("2006-01-02T15:04Z07:00"); got != "2026-01-15T08:00Z" {
		t.Errorf("start = %s, want normalized UTC time", got)
	}
	a.assertStats(domain.Stats{AvailableSkis: 1, RentedSkis: 1, TotalSkis: 2, RegisteredCustomers: 1})

	skis := decode[[]domain.Ski](t, a.expect(http.MethodGet, "/api/skis", "", http.StatusOK))
	if skis[0].Available || !skis[1].Available {
		t.Errorf("wrong availability: %+v", skis)
	}

	returned := decode[domain.RentalDetails](t, a.expect(http.MethodPost, "/api/rentals/1/return", "", http.StatusOK))
	if returned.Active || returned.ReturnedAt == nil {
		t.Errorf("rental should be returned: %+v", returned)
	}
	a.assertStats(domain.Stats{AvailableSkis: 2, TotalSkis: 2, RegisteredCustomers: 1})

	// The ski can be rented again, and the history keeps both rentals.
	a.expect(http.MethodPost, "/api/rentals", rentalBody, http.StatusCreated)
	rentals := decode[[]domain.RentalDetails](t, a.expect(http.MethodGet, "/api/rentals", "", http.StatusOK))
	if len(rentals) != 2 || rentals[0].Active || !rentals[1].Active {
		t.Errorf("wrong rental history: %+v", rentals)
	}
}

func TestErrorResponses(t *testing.T) {
	tests := []struct {
		name         string
		method, path string
		body         string
		status       int
		message      string
	}{
		{"ski without name", "POST", "/api/skis", `{"name":"  "}`, 400, "Bezeichnung fehlt"},
		{"malformed JSON", "POST", "/api/skis", `{`, 400, "ungültige oder unvollständige Daten"},
		{"empty body", "POST", "/api/skis", ``, 400, "ungültige oder unvollständige Daten"},
		{"customer without ID number", "POST", "/api/customers", `{"name":"Max","idType":"Reisepass"}`, 400, "Ausweisnummer fehlt"},
		{"negative price", "POST", "/api/tariffs", `{"name":"X","hourlyPriceCents":-1}`, 400, "nicht negativ"},
		{"fractional cents", "POST", "/api/tariffs", `{"name":"X","hourlyPriceCents":1.5}`, 400, "ungültige oder unvollständige Daten"},
		{"update to negative price", "PUT", "/api/tariffs/1", `{"name":"X","hourlyPriceCents":-1}`, 400, "nicht negativ"},
		{"tariff without price", "POST", "/api/tariffs", `{"name":"X"}`, 400, "Stundenpreis fehlt"},
		{"update without price", "PUT", "/api/tariffs/1", `{"name":"X"}`, 400, "Stundenpreis fehlt"},
		{"non-numeric ID", "PUT", "/api/skis/abc", `{"name":"X"}`, 400, "ID ist ungültig"},
		{"unknown ski update", "PUT", "/api/skis/99", `{"name":"X"}`, 404, "Ski mit ID 99"},
		{"unknown customer update", "PUT", "/api/customers/99", `{"name":"X","idType":"Y","idNumber":"Z"}`, 404, "Person mit ID 99"},
		{"unknown tariff update", "PUT", "/api/tariffs/99", `{"name":"X","hourlyPriceCents":1}`, 404, "Tarif mit ID 99"},
		{"rental without selection", "POST", "/api/rentals", `{"start":"2026-01-15T09:00:00Z","durationHours":1}`, 400, "Person fehlt"},
		{"rental without start", "POST", "/api/rentals", `{"customerId":1,"skiId":1,"tariffId":1,"durationHours":1}`, 400, "Verleihbeginn fehlt"},
		{"rental with invalid start", "POST", "/api/rentals", `{"customerId":1,"skiId":1,"tariffId":1,"start":"morgen","durationHours":1}`, 400, "ungültige oder unvollständige Daten"},
		{"rental with zero duration", "POST", "/api/rentals", `{"customerId":1,"skiId":1,"tariffId":1,"start":"2026-01-15T09:00:00Z","durationHours":0}`, 400, "mindestens 1 Stunde"},
		{"rental with overflowing duration", "POST", "/api/rentals", `{"customerId":1,"skiId":1,"tariffId":1,"start":"2026-01-15T09:00:00Z","durationHours":9000000000000}`, 400, "zu groß"},
		{"rental ending after year 9999", "POST", "/api/rentals", `{"customerId":1,"skiId":1,"tariffId":1,"start":"9999-12-31T23:00:00Z","durationHours":2}`, 400, "gültigen Bereichs"},
		{"rental with unknown customer", "POST", "/api/rentals", `{"customerId":99,"skiId":1,"tariffId":1,"start":"2026-01-15T09:00:00Z","durationHours":1}`, 404, "Person mit ID 99"},
		{"rental with unknown ski", "POST", "/api/rentals", `{"customerId":1,"skiId":99,"tariffId":1,"start":"2026-01-15T09:00:00Z","durationHours":1}`, 404, "Ski mit ID 99"},
		{"rental with unknown tariff", "POST", "/api/rentals", `{"customerId":1,"skiId":1,"tariffId":99,"start":"2026-01-15T09:00:00Z","durationHours":1}`, 404, "Tarif mit ID 99"},
		{"return of unknown rental", "POST", "/api/rentals/99/return", ``, 404, "Ausleihe mit ID 99"},
		{"quote without hours", "GET", "/api/price-quote?tariffId=1", ``, 400, "Verleihdauer fehlt"},
		{"quote without tariff", "GET", "/api/price-quote?hours=1", ``, 400, "Tarif fehlt"},
		{"quote with zero hours", "GET", "/api/price-quote?tariffId=1&hours=0", ``, 400, "mindestens 1 Stunde"},
		{"quote with unknown tariff", "GET", "/api/price-quote?tariffId=99&hours=1", ``, 404, "Tarif mit ID 99"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newSeededAPI(t)
			response := a.expect(tt.method, tt.path, tt.body, tt.status)
			body := decode[map[string]string](t, response)
			if !strings.Contains(body["error"], tt.message) {
				t.Errorf("error = %q, want it to contain %q", body["error"], tt.message)
			}
			if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
				t.Errorf("Content-Type = %q", got)
			}
			a.assertStats(domain.Stats{AvailableSkis: 1, TotalSkis: 1, RegisteredCustomers: 1})
		})
	}
}

func TestDoubleRentalIsConflict(t *testing.T) {
	a := newSeededAPI(t)
	a.expect(http.MethodPost, "/api/rentals", rentalBody, http.StatusCreated)

	response := a.expect(http.MethodPost, "/api/rentals", rentalBody, http.StatusConflict)

	if body := decode[map[string]string](t, response); body["error"] != domain.ErrSkiUnavailable.Error() {
		t.Errorf("error = %q", body["error"])
	}
	a.assertStats(domain.Stats{RentedSkis: 1, TotalSkis: 1, RegisteredCustomers: 1})
}

// Registering a person during a rental uses two separate requests, so neither
// one may undo or trigger the other.
func TestCustomerRegistrationAndRentalStaySeparate(t *testing.T) {
	a := newSeededAPI(t)
	a.expect(http.MethodPost, "/api/rentals", rentalBody, http.StatusCreated)

	a.expect(http.MethodPost, "/api/customers", `{"name":" ","idType":"Reisepass","idNumber":"C0000002"}`, http.StatusBadRequest)
	a.assertStats(domain.Stats{RentedSkis: 1, TotalSkis: 1, RegisteredCustomers: 1})

	created := decode[domain.Customer](t, a.expect(http.MethodPost, "/api/customers", `{"name":"Erika Beispiel","idType":"Reisepass","idNumber":"C0000002"}`, http.StatusCreated))
	if created.ID != 2 || created.Name != "Erika Beispiel" || created.IDNumber != "C0000002" {
		t.Errorf("wrong customer: %+v", created)
	}
	a.assertStats(domain.Stats{RentedSkis: 1, TotalSkis: 1, RegisteredCustomers: 2})

	secondRental := `{"customerId":2,"skiId":1,"tariffId":1,"start":"2026-01-15T09:00:00+01:00","durationHours":3}`
	a.expect(http.MethodPost, "/api/rentals", secondRental, http.StatusConflict)

	customers := decode[[]domain.Customer](t, a.expect(http.MethodGet, "/api/customers", "", http.StatusOK))
	if len(customers) != 2 || customers[1] != created {
		t.Errorf("failed rental changed customers: %+v", customers)
	}
	a.assertStats(domain.Stats{RentedSkis: 1, TotalSkis: 1, RegisteredCustomers: 2})
}

func TestDoubleReturnIsConflict(t *testing.T) {
	a := newSeededAPI(t)
	a.expect(http.MethodPost, "/api/rentals", rentalBody, http.StatusCreated)
	a.expect(http.MethodPost, "/api/rentals/1/return", "", http.StatusOK)

	response := a.expect(http.MethodPost, "/api/rentals/1/return", "", http.StatusConflict)

	if body := decode[map[string]string](t, response); body["error"] != domain.ErrAlreadyReturned.Error() {
		t.Errorf("error = %q", body["error"])
	}
}

func TestTariffUpdateKeepsExistingRentalPrice(t *testing.T) {
	a := newSeededAPI(t)
	a.expect(http.MethodPost, "/api/rentals", rentalBody, http.StatusCreated)

	updated := decode[domain.Tariff](t, a.expect(http.MethodPut, "/api/tariffs/1", `{"name":"Hochsaison","hourlyPriceCents":900}`, http.StatusOK))
	if updated.Name != "Hochsaison" || updated.HourlyPriceCents != 900 {
		t.Errorf("tariff not updated: %+v", updated)
	}

	rental := decode[[]domain.RentalDetails](t, a.expect(http.MethodGet, "/api/rentals", "", http.StatusOK))[0]
	if rental.TariffName != "Standard" || rental.HourlyPriceCents != 500 || rental.TotalPriceCents != 1500 {
		t.Errorf("stored rental changed: %+v", rental)
	}

	// New rentals use the new price.
	a.expect(http.MethodPost, "/api/rentals/1/return", "", http.StatusOK)
	next := decode[domain.RentalDetails](t, a.expect(http.MethodPost, "/api/rentals", rentalBody, http.StatusCreated))
	if next.TariffName != "Hochsaison" || next.TotalPriceCents != 2700 {
		t.Errorf("new rental uses wrong price: %+v", next)
	}
}

func TestLocalOnly(t *testing.T) {
	tests := []struct {
		name, host, origin string
		wantStatus         int
	}{
		{"direct address", "127.0.0.1:8080", "", http.StatusCreated},
		{"localhost", "localhost:8080", "", http.StatusCreated},
		{"same origin", "127.0.0.1:8080", "http://127.0.0.1:8080", http.StatusCreated},
		{"same origin via localhost", "localhost:8080", "http://localhost:8080", http.StatusCreated},
		{"foreign host", "evil.example", "", http.StatusForbidden},
		{"DNS rebinding", "evil.example:8080", "http://evil.example:8080", http.StatusForbidden},
		{"foreign website", "127.0.0.1:8080", "https://evil.example", http.StatusForbidden},
		{"other local port", "127.0.0.1:8080", "http://127.0.0.1:3000", http.StatusForbidden},
		{"localhost page calling 127.0.0.1", "127.0.0.1:8080", "http://localhost:8080", http.StatusForbidden},
		{"sandboxed page", "127.0.0.1:8080", "null", http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newAPI(t)
			request := httptest.NewRequest(http.MethodPost, "/api/skis", strings.NewReader(`{"name":"Carver"}`))
			request.Host = tt.host
			request.Header.Set("Content-Type", "text/plain")
			if tt.origin != "" {
				request.Header.Set("Origin", tt.origin)
			}
			response := httptest.NewRecorder()

			a.routes.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status %d, want %d, body %s", response.Code, tt.wantStatus, response.Body)
			}
			wantSkis := 0
			if tt.wantStatus == http.StatusCreated {
				wantSkis = 1
			}
			a.assertStats(domain.Stats{AvailableSkis: wantSkis, TotalSkis: wantSkis})
		})
	}
}

func TestForeignHostCannotReadData(t *testing.T) {
	a := newSeededAPI(t)
	request := httptest.NewRequest(http.MethodGet, "/api/customers", nil)
	request.Host = "evil.example"
	response := httptest.NewRecorder()

	a.routes.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status %d, want 403", response.Code)
	}
	if strings.Contains(response.Body.String(), "Mustermann") {
		t.Errorf("response leaks customer data: %s", response.Body)
	}
}

func TestFreeTariffIsAllowed(t *testing.T) {
	a := newAPI(t)

	created := decode[domain.Tariff](t, a.expect(http.MethodPost, "/api/tariffs", `{"name":"Aktion","hourlyPriceCents":0}`, http.StatusCreated))

	if created.HourlyPriceCents != 0 {
		t.Errorf("got price %d, want 0", created.HourlyPriceCents)
	}
}

func TestPriceQuote(t *testing.T) {
	a := newSeededAPI(t)

	response := a.expect(http.MethodGet, "/api/price-quote?tariffId=1&hours=4", "", http.StatusOK)

	quote := decode[map[string]any](t, response)
	if quote["tariffName"] != "Standard" || quote["hourlyPriceCents"] != 500.0 ||
		quote["durationHours"] != 4.0 || quote["totalPriceCents"] != 2000.0 {
		t.Errorf("wrong quote: %+v", quote)
	}
	if rentals := decode[[]domain.RentalDetails](t, a.expect(http.MethodGet, "/api/rentals", "", http.StatusOK)); len(rentals) != 0 {
		t.Errorf("quote must not create a rental: %+v", rentals)
	}
}

func TestEditSkiAndCustomer(t *testing.T) {
	a := newSeededAPI(t)
	a.expect(http.MethodPost, "/api/rentals", rentalBody, http.StatusCreated)

	ski := decode[domain.Ski](t, a.expect(http.MethodPut, "/api/skis/1", `{"name":"Carver 170","available":true}`, http.StatusOK))
	if ski.Name != "Carver 170" || ski.Available {
		t.Errorf("edit must change the name but not the availability: %+v", ski)
	}

	customer := decode[domain.Customer](t, a.expect(http.MethodPut, "/api/customers/1", `{"name":"Erika Musterfrau","idType":"Personalausweis","idNumber":"T0000002"}`, http.StatusOK))
	if customer.Name != "Erika Musterfrau" || customer.IDType != "Personalausweis" || customer.IDNumber != "T0000002" {
		t.Errorf("customer not updated: %+v", customer)
	}

	rental := decode[[]domain.RentalDetails](t, a.expect(http.MethodGet, "/api/rentals", "", http.StatusOK))[0]
	if rental.SkiName != "Carver 170" || rental.CustomerName != "Erika Musterfrau" {
		t.Errorf("rental shows old names: %+v", rental)
	}
	a.assertStats(domain.Stats{RentedSkis: 1, TotalSkis: 1, RegisteredCustomers: 1})
}

func TestEmptyListsAreJSONArrays(t *testing.T) {
	a := newAPI(t)
	for _, path := range []string{"/api/skis", "/api/customers", "/api/tariffs", "/api/rentals"} {
		response := a.expect(http.MethodGet, path, "", http.StatusOK)
		if got := strings.TrimSpace(response.Body.String()); got != "[]" {
			t.Errorf("%s = %s, want []", path, got)
		}
	}
}
