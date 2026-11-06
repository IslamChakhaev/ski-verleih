package store

import (
	"fmt"
	"time"
)

// Seed fills the store with fictional demo data: 20 skis, 10 customers,
// 3 tariffs and 3 active rentals.
func (s *Store) Seed(now time.Time) error {
	tariffs := []struct {
		name  string
		cents int64
	}{
		{"Kinder", 300},
		{"Standard", 500},
		{"Premium", 800},
	}
	for _, tariff := range tariffs {
		if _, err := s.AddTariff(tariff.name, tariff.cents); err != nil {
			return fmt.Errorf("seed tariff %q: %w", tariff.name, err)
		}
	}

	models := []string{"Pistenfuchs", "Bergblitz", "Schneefeder", "Gipfelstürmer", "Talwind"}
	lengths := []int{140, 150, 160, 170}
	for _, model := range models {
		for _, length := range lengths {
			name := fmt.Sprintf("%s %d cm", model, length)
			if _, err := s.AddSki(name); err != nil {
				return fmt.Errorf("seed ski %q: %w", name, err)
			}
		}
	}

	customers := [][3]string{
		{"Max Mustermann", "Personalausweis", "T00000001"},
		{"Erika Musterfrau", "Personalausweis", "T00000002"},
		{"Lena Beispiel", "Reisepass", "C00000003"},
		{"Jonas Probe", "Führerschein", "F00000004"},
		{"Mia Testmann", "Personalausweis", "T00000005"},
		{"Paul Demo", "Reisepass", "C00000006"},
		{"Sofia Muster", "Führerschein", "F00000007"},
		{"Felix Fiktiv", "Personalausweis", "T00000008"},
		{"Clara Erfunden", "Reisepass", "C00000009"},
		{"Noah Platzhalter", "Personalausweis", "T00000010"},
	}
	for _, customer := range customers {
		if _, err := s.AddCustomer(customer[0], customer[1], customer[2]); err != nil {
			return fmt.Errorf("seed customer %q: %w", customer[0], err)
		}
	}

	start := now.Truncate(time.Hour)
	rentals := []struct {
		customerID, skiID, tariffID int
		startedHoursAgo, duration   int
	}{
		{1, 3, 2, 2, 4},
		{2, 8, 3, 1, 3},
		{3, 13, 1, 3, 6},
	}
	for _, rental := range rentals {
		rentalStart := start.Add(-time.Duration(rental.startedHoursAgo) * time.Hour)
		if _, err := s.CreateRental(rental.customerID, rental.skiID, rental.tariffID, rentalStart, rental.duration); err != nil {
			return fmt.Errorf("seed rental of ski %d: %w", rental.skiID, err)
		}
	}
	return nil
}
