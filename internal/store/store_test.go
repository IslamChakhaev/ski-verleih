package store

import (
	"errors"
	"sync"
	"testing"
	"time"

	"ski-verleih/internal/domain"
)

var testStart = time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)

// fixture contains one ski, one customer and one tariff, all with ID 1.
func fixture(t *testing.T) *Store {
	t.Helper()
	s := New()
	if _, err := s.AddSki("Carver 160"); err != nil {
		t.Fatalf("AddSki: %v", err)
	}
	if _, err := s.AddCustomer("Max Mustermann", "Reisepass", "C0000001"); err != nil {
		t.Fatalf("AddCustomer: %v", err)
	}
	if _, err := s.AddTariff("Standard", 500); err != nil {
		t.Fatalf("AddTariff: %v", err)
	}
	return s
}

func assertStats(t *testing.T, s *Store, want domain.Stats) {
	t.Helper()
	if got := s.Stats(); got != want {
		t.Errorf("Stats() = %+v, want %+v", got, want)
	}
}

func TestAddAssignsUniqueSortedIDs(t *testing.T) {
	s := New()
	for _, name := range []string{"A", "B", "C"} {
		if _, err := s.AddSki(name); err != nil {
			t.Fatalf("AddSki: %v", err)
		}
	}
	skis := s.Skis()
	if len(skis) != 3 {
		t.Fatalf("got %d skis, want 3", len(skis))
	}
	for i, ski := range skis {
		if ski.ID != i+1 {
			t.Errorf("skis[%d].ID = %d, want %d", i, ski.ID, i+1)
		}
	}
}

func TestListsAreEmptyNotNil(t *testing.T) {
	s := New()
	if s.Skis() == nil || s.Customers() == nil || s.Tariffs() == nil || s.Rentals() == nil {
		t.Error("empty lists must not be nil")
	}
}

func TestReturnedSlicesAreCopies(t *testing.T) {
	s := fixture(t)
	s.Skis()[0].Name = "changed"
	if got := s.Skis()[0].Name; got != "Carver 160" {
		t.Errorf("stored name changed to %q", got)
	}
}

func TestAddRejectsInvalidInputWithoutConsumingData(t *testing.T) {
	s := New()
	if _, err := s.AddSki(""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("AddSki: got %v, want ErrInvalidInput", err)
	}
	if _, err := s.AddCustomer("", "", ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("AddCustomer: got %v, want ErrInvalidInput", err)
	}
	if _, err := s.AddTariff("Standard", -1); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("AddTariff: got %v, want ErrInvalidInput", err)
	}
	assertStats(t, s, domain.Stats{})
}

func TestUpdateSkiKeepsAvailability(t *testing.T) {
	s := fixture(t)
	if _, err := s.CreateRental(1, 1, 1, testStart, 2); err != nil {
		t.Fatalf("CreateRental: %v", err)
	}

	ski, err := s.UpdateSki(1, "Carver 170")
	if err != nil {
		t.Fatalf("UpdateSki: %v", err)
	}
	if ski.Name != "Carver 170" || ski.Available {
		t.Errorf("got %+v, want new name and still rented", ski)
	}
	if got := s.Rentals()[0].SkiName; got != "Carver 170" {
		t.Errorf("rental shows ski name %q", got)
	}
}

func TestUpdateCustomer(t *testing.T) {
	s := fixture(t)
	customer, err := s.UpdateCustomer(1, "Erika Musterfrau", "Personalausweis", "T0000002")
	if err != nil {
		t.Fatalf("UpdateCustomer: %v", err)
	}
	if customer.ID != 1 || customer.Name != "Erika Musterfrau" || customer.IDNumber != "T0000002" {
		t.Errorf("got %+v", customer)
	}
}

func TestUpdateErrors(t *testing.T) {
	s := fixture(t)
	tests := []struct {
		name string
		call func() error
		want error
	}{
		{"unknown ski", func() error { _, err := s.UpdateSki(99, "X"); return err }, domain.ErrNotFound},
		{"invalid ski", func() error { _, err := s.UpdateSki(1, ""); return err }, domain.ErrInvalidInput},
		{"unknown customer", func() error { _, err := s.UpdateCustomer(99, "X", "Y", "Z"); return err }, domain.ErrNotFound},
		{"invalid customer", func() error { _, err := s.UpdateCustomer(1, "", "Y", "Z"); return err }, domain.ErrInvalidInput},
		{"unknown tariff", func() error { _, err := s.UpdateTariff(99, "X", 1); return err }, domain.ErrNotFound},
		{"invalid tariff", func() error { _, err := s.UpdateTariff(1, "X", -1); return err }, domain.ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, tt.want) {
				t.Errorf("got %v, want %v", err, tt.want)
			}
		})
	}
	if tariff, _ := s.Tariff(1); tariff.HourlyPriceCents != 500 {
		t.Errorf("failed update changed price to %d", tariff.HourlyPriceCents)
	}
}

func TestCreateRentalUpdatesAvailabilityAndStats(t *testing.T) {
	s := fixture(t)

	rental, err := s.CreateRental(1, 1, 1, testStart, 3)
	if err != nil {
		t.Fatalf("CreateRental: %v", err)
	}
	if rental.ID != 1 || rental.CustomerName != "Max Mustermann" || rental.SkiName != "Carver 160" {
		t.Errorf("wrong details: %+v", rental)
	}
	if rental.TotalPriceCents != 1500 || !rental.Active {
		t.Errorf("wrong price or status: %+v", rental)
	}
	if s.Skis()[0].Available {
		t.Error("ski should be rented")
	}
	assertStats(t, s, domain.Stats{RentedSkis: 1, TotalSkis: 1, RegisteredCustomers: 1})
}

func TestCreateRentalErrors(t *testing.T) {
	tests := []struct {
		name                        string
		customerID, skiID, tariffID int
		hours                       int
		want                        error
	}{
		{"unknown customer", 99, 1, 1, 2, domain.ErrNotFound},
		{"unknown ski", 1, 99, 1, 2, domain.ErrNotFound},
		{"unknown tariff", 1, 1, 99, 2, domain.ErrNotFound},
		{"missing customer", 0, 1, 1, 2, domain.ErrInvalidInput},
		{"invalid duration", 1, 1, 1, 0, domain.ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := fixture(t)
			_, err := s.CreateRental(tt.customerID, tt.skiID, tt.tariffID, testStart, tt.hours)
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
			assertStats(t, s, domain.Stats{AvailableSkis: 1, TotalSkis: 1, RegisteredCustomers: 1})
			if len(s.Rentals()) != 0 {
				t.Error("failed rental must not be stored")
			}
		})
	}
}

func TestSkiCannotBeRentedTwice(t *testing.T) {
	s := fixture(t)
	if _, err := s.CreateRental(1, 1, 1, testStart, 2); err != nil {
		t.Fatalf("CreateRental: %v", err)
	}
	if _, err := s.CreateRental(1, 1, 1, testStart, 2); !errors.Is(err, domain.ErrSkiUnavailable) {
		t.Errorf("got %v, want ErrSkiUnavailable", err)
	}
	if got := len(s.Rentals()); got != 1 {
		t.Errorf("got %d rentals, want 1", got)
	}
}

func TestReturnRental(t *testing.T) {
	s := fixture(t)
	if _, err := s.CreateRental(1, 1, 1, testStart, 3); err != nil {
		t.Fatalf("CreateRental: %v", err)
	}

	rental, err := s.ReturnRental(1, testStart.Add(time.Hour))
	if err != nil {
		t.Fatalf("ReturnRental: %v", err)
	}
	if rental.Active || rental.ReturnedAt == nil {
		t.Errorf("rental should be returned: %+v", rental)
	}
	assertStats(t, s, domain.Stats{AvailableSkis: 1, TotalSkis: 1, RegisteredCustomers: 1})

	if _, err := s.ReturnRental(1, testStart); !errors.Is(err, domain.ErrAlreadyReturned) {
		t.Errorf("second return: got %v, want ErrAlreadyReturned", err)
	}
	if _, err := s.ReturnRental(99, testStart); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("unknown rental: got %v, want ErrNotFound", err)
	}
}

func TestSecondReturnDoesNotReleaseNewRental(t *testing.T) {
	s := fixture(t)
	if _, err := s.CreateRental(1, 1, 1, testStart, 1); err != nil {
		t.Fatalf("first rental: %v", err)
	}
	if _, err := s.ReturnRental(1, testStart); err != nil {
		t.Fatalf("return: %v", err)
	}
	if _, err := s.CreateRental(1, 1, 1, testStart, 1); err != nil {
		t.Fatalf("second rental: %v", err)
	}

	if _, err := s.ReturnRental(1, testStart); !errors.Is(err, domain.ErrAlreadyReturned) {
		t.Fatalf("got %v, want ErrAlreadyReturned", err)
	}
	if s.Skis()[0].Available {
		t.Error("ski of the second rental must stay rented")
	}
}

func TestTariffChangeKeepsRentalPrice(t *testing.T) {
	s := fixture(t)
	if _, err := s.CreateRental(1, 1, 1, testStart, 3); err != nil {
		t.Fatalf("CreateRental: %v", err)
	}

	if _, err := s.UpdateTariff(1, "Hochsaison", 900); err != nil {
		t.Fatalf("UpdateTariff: %v", err)
	}

	rental := s.Rentals()[0]
	if rental.TariffName != "Standard" || rental.HourlyPriceCents != 500 || rental.TotalPriceCents != 1500 {
		t.Errorf("stored rental changed: %+v", rental)
	}
	tariff, err := s.Tariff(1)
	if err != nil {
		t.Fatalf("Tariff: %v", err)
	}
	if tariff.Name != "Hochsaison" || tariff.HourlyPriceCents != 900 {
		t.Errorf("tariff not updated: %+v", tariff)
	}
}

func TestConcurrentRentalsOfSameSki(t *testing.T) {
	s := fixture(t)
	const attempts = 100

	errs := make(chan error, attempts)
	var wg sync.WaitGroup
	for range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.CreateRental(1, 1, 1, testStart, 1)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	successes := 0
	for err := range errs {
		switch {
		case err == nil:
			successes++
		case !errors.Is(err, domain.ErrSkiUnavailable):
			t.Errorf("unexpected error: %v", err)
		}
	}
	if successes != 1 {
		t.Errorf("%d rentals succeeded, want exactly 1", successes)
	}
	assertStats(t, s, domain.Stats{RentedSkis: 1, TotalSkis: 1, RegisteredCustomers: 1})
}

func TestSeed(t *testing.T) {
	s := New()
	if err := s.Seed(testStart); err != nil {
		t.Fatalf("Seed: %v", err)
	}
	assertStats(t, s, domain.Stats{AvailableSkis: 17, RentedSkis: 3, TotalSkis: 20, RegisteredCustomers: 10})
	if got := len(s.Tariffs()); got != 3 {
		t.Errorf("got %d tariffs, want 3", got)
	}
	if got := len(s.Rentals()); got != 3 {
		t.Errorf("got %d rentals, want 3", got)
	}
}
