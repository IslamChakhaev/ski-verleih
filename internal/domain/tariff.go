package domain

import (
	"math"
	"time"
)

// maxDurationHours is the longest duration a time.Duration can represent.
// It is a technical limit that prevents overflow, not a business rule.
const maxDurationHours = math.MaxInt64 / int64(time.Hour)

// Tariff defines an hourly price. Amounts are integer cents to avoid
// floating-point rounding errors.
type Tariff struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	HourlyPriceCents int64  `json:"hourlyPriceCents"`
}

// NewTariff creates a tariff. The store assigns the ID.
func NewTariff(name string, hourlyPriceCents int64) (Tariff, error) {
	var tariff Tariff
	if err := tariff.Update(name, hourlyPriceCents); err != nil {
		return Tariff{}, err
	}
	return tariff, nil
}

// Update replaces name and price. On error the tariff stays unchanged.
func (t *Tariff) Update(name string, hourlyPriceCents int64) error {
	name, err := required(name, "Tarifname")
	if err != nil {
		return err
	}
	if hourlyPriceCents < 0 {
		return InvalidInput("Der Stundenpreis darf nicht negativ sein")
	}
	t.Name, t.HourlyPriceCents = name, hourlyPriceCents
	return nil
}

// Price returns the total for the given number of hours in cents. It is the
// only place where prices are calculated.
func (t Tariff) Price(hours int) (int64, error) {
	if hours <= 0 {
		return 0, InvalidInput("Die Verleihdauer muss mindestens 1 Stunde betragen")
	}
	if int64(hours) > maxDurationHours {
		return 0, InvalidInput("Die Verleihdauer ist zu groß")
	}
	if t.HourlyPriceCents > 0 && int64(hours) > math.MaxInt64/t.HourlyPriceCents {
		return 0, InvalidInput("Der Gesamtpreis ist zu groß")
	}
	return int64(hours) * t.HourlyPriceCents, nil
}
