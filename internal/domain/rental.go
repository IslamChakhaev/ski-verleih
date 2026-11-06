package domain

import "time"

// maxYear is the last year that time.Time can write as JSON.
const maxYear = 9999

// Rental is a ski rented by a customer. The tariff name and prices are copied
// when the rental is created, so later tariff changes do not affect it.
type Rental struct {
	ID               int        `json:"id"`
	CustomerID       int        `json:"customerId"`
	SkiID            int        `json:"skiId"`
	Start            time.Time  `json:"start"`
	DurationHours    int        `json:"durationHours"`
	TariffName       string     `json:"tariffName"`
	HourlyPriceCents int64      `json:"hourlyPriceCents"`
	TotalPriceCents  int64      `json:"totalPriceCents"`
	ReturnedAt       *time.Time `json:"returnedAt"`
}

// RentalDetails is a rental with the display data derived from it.
type RentalDetails struct {
	Rental
	CustomerName string    `json:"customerName"`
	SkiName      string    `json:"skiName"`
	End          time.Time `json:"end"`
	Active       bool      `json:"active"`
}

// ValidateRentalSelection reports missing customer, ski or tariff selections,
// so they are not mistaken for unknown IDs.
func ValidateRentalSelection(customerID, skiID, tariffID int) error {
	if customerID <= 0 {
		return InvalidInput("Person fehlt")
	}
	if skiID <= 0 {
		return InvalidInput("Ski fehlt")
	}
	if tariffID <= 0 {
		return InvalidInput("Tarif fehlt")
	}
	return nil
}

// NewRental creates a rental and marks the ski as rented. If it returns an
// error, the ski stays unchanged. The store assigns the ID.
func NewRental(customer Customer, ski *Ski, tariff Tariff, start time.Time, durationHours int) (Rental, error) {
	if start.IsZero() {
		return Rental{}, InvalidInput("Verleihbeginn fehlt")
	}
	total, err := tariff.Price(durationHours)
	if err != nil {
		return Rental{}, err
	}
	rental := Rental{
		CustomerID:       customer.ID,
		SkiID:            ski.ID,
		Start:            start.UTC(),
		DurationHours:    durationHours,
		TariffName:       tariff.Name,
		HourlyPriceCents: tariff.HourlyPriceCents,
		TotalPriceCents:  total,
	}
	if rental.Start.Year() < 0 || rental.End().Year() > maxYear {
		return Rental{}, InvalidInput("Der Verleihzeitraum liegt außerhalb des gültigen Bereichs")
	}
	if err := ski.rent(); err != nil {
		return Rental{}, err
	}
	return rental, nil
}

// IsActive reports whether the ski has not been returned yet.
func (r Rental) IsActive() bool {
	return r.ReturnedAt == nil
}

// End returns the planned end of the rental.
func (r Rental) End() time.Time {
	return r.Start.Add(time.Duration(r.DurationHours) * time.Hour)
}

// Return ends the rental and makes the ski available again.
func (r *Rental) Return(ski *Ski, at time.Time) error {
	if !r.IsActive() {
		return ErrAlreadyReturned
	}
	if at.IsZero() {
		return InvalidInput("Rückgabezeitpunkt fehlt")
	}
	returnedAt := at.UTC()
	r.ReturnedAt = &returnedAt
	ski.release()
	return nil
}

// Details combines the rental with the names of its customer and ski.
func (r Rental) Details(customer Customer, ski Ski) RentalDetails {
	return RentalDetails{
		Rental:       r,
		CustomerName: customer.Name,
		SkiName:      ski.Name,
		End:          r.End(),
		Active:       r.IsActive(),
	}
}
