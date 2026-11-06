package domain

import (
	"errors"
	"math"
	"testing"
	"time"
)

var testStart = time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC)

func newTestRental(t *testing.T, ski *Ski) Rental {
	t.Helper()
	tariff := Tariff{Name: "Standard", HourlyPriceCents: 500}
	rental, err := NewRental(Customer{ID: 1}, ski, tariff, testStart, 3)
	if err != nil {
		t.Fatalf("NewRental: %v", err)
	}
	return rental
}

func TestNewSki(t *testing.T) {
	ski, err := NewSki("  Carver 160  ")
	if err != nil {
		t.Fatalf("NewSki: %v", err)
	}
	if ski.Name != "Carver 160" || !ski.Available {
		t.Errorf("got %+v, want trimmed name and available ski", ski)
	}
	if _, err := NewSki("   "); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("blank name: got %v, want ErrInvalidInput", err)
	}
}

func TestSkiRenameKeepsNameOnError(t *testing.T) {
	ski := Ski{Name: "Carver"}
	if err := ski.Rename(""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("got %v, want ErrInvalidInput", err)
	}
	if ski.Name != "Carver" {
		t.Errorf("name changed to %q", ski.Name)
	}
}

func TestNewCustomerRequiresAllFields(t *testing.T) {
	tests := []struct {
		name, customerName, idType, idNumber string
		wantErr                              bool
	}{
		{"valid", "Max Mustermann", "Reisepass", "C0000001", false},
		{"missing name", " ", "Reisepass", "C0000001", true},
		{"missing type", "Max Mustermann", "", "C0000001", true},
		{"missing number", "Max Mustermann", "Reisepass", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCustomer(tt.customerName, tt.idType, tt.idNumber)
			if tt.wantErr != errors.Is(err, ErrInvalidInput) {
				t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCustomerUpdateKeepsDataOnError(t *testing.T) {
	customer := Customer{Name: "Max", IDType: "Reisepass", IDNumber: "C1"}
	if err := customer.Update("Erika", "Reisepass", ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("got %v, want ErrInvalidInput", err)
	}
	if customer.Name != "Max" {
		t.Errorf("name changed to %q", customer.Name)
	}
}

func TestNewTariffValidation(t *testing.T) {
	tests := []struct {
		name       string
		tariffName string
		cents      int64
		wantErr    bool
	}{
		{"valid", "Standard", 500, false},
		{"free of charge", "Aktion", 0, false},
		{"negative price", "Standard", -1, true},
		{"missing name", "", 500, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTariff(tt.tariffName, tt.cents)
			if tt.wantErr != errors.Is(err, ErrInvalidInput) {
				t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTariffPrice(t *testing.T) {
	tests := []struct {
		name    string
		cents   int64
		hours   int
		want    int64
		wantErr bool
	}{
		{"one hour", 500, 1, 500, false},
		{"several hours", 500, 3, 1500, false},
		{"long rental", 350, 1000, 350000, false},
		{"free tariff", 0, 5, 0, false},
		{"zero hours", 500, 0, 0, true},
		{"negative hours", 500, -2, 0, true},
		{"duration overflow", 0, math.MaxInt, 0, true},
		{"total overflow", math.MaxInt64, 2, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Tariff{HourlyPriceCents: tt.cents}.Price(tt.hours)
			if tt.wantErr != errors.Is(err, ErrInvalidInput) {
				t.Fatalf("got error %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNewRentalCopiesPriceAndRentsSki(t *testing.T) {
	ski := Ski{ID: 7, Available: true}
	rental := newTestRental(t, &ski)

	if ski.Available {
		t.Error("ski should be rented")
	}
	if rental.SkiID != 7 || rental.CustomerID != 1 {
		t.Errorf("wrong references: %+v", rental)
	}
	if rental.TariffName != "Standard" || rental.HourlyPriceCents != 500 || rental.TotalPriceCents != 1500 {
		t.Errorf("wrong price data: %+v", rental)
	}
	if want := testStart.Add(3 * time.Hour); !rental.End().Equal(want) {
		t.Errorf("End() = %v, want %v", rental.End(), want)
	}
	if !rental.IsActive() {
		t.Error("new rental should be active")
	}
}

func TestNewRentalRejectsInvalidInput(t *testing.T) {
	tariff := Tariff{HourlyPriceCents: 500}
	tests := []struct {
		name  string
		start time.Time
		hours int
	}{
		{"missing start", time.Time{}, 3},
		{"zero duration", testStart, 0},
		{"negative duration", testStart, -1},
		{"end not representable", time.Date(9999, 12, 31, 23, 0, 0, 0, time.UTC), 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ski := Ski{Available: true}
			_, err := NewRental(Customer{}, &ski, tariff, tt.start, tt.hours)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("got %v, want ErrInvalidInput", err)
			}
			if !ski.Available {
				t.Error("ski must stay available after a failed rental")
			}
		})
	}
}

func TestNewRentalRejectsRentedSki(t *testing.T) {
	ski := Ski{Available: false}
	_, err := NewRental(Customer{}, &ski, Tariff{}, testStart, 1)
	if !errors.Is(err, ErrSkiUnavailable) {
		t.Errorf("got %v, want ErrSkiUnavailable", err)
	}
}

func TestRentalReturn(t *testing.T) {
	ski := Ski{Available: true}
	rental := newTestRental(t, &ski)
	returnedAt := testStart.Add(2 * time.Hour)

	if err := rental.Return(&ski, returnedAt); err != nil {
		t.Fatalf("Return: %v", err)
	}
	if !ski.Available {
		t.Error("ski should be available after return")
	}
	if rental.IsActive() || !rental.ReturnedAt.Equal(returnedAt) {
		t.Errorf("wrong return state: %+v", rental)
	}

	ski.Available = false // must not be released by a second return
	if err := rental.Return(&ski, returnedAt); !errors.Is(err, ErrAlreadyReturned) {
		t.Errorf("second return: got %v, want ErrAlreadyReturned", err)
	}
	if ski.Available {
		t.Error("second return must not change the ski")
	}
}

func TestRentalReturnRequiresTime(t *testing.T) {
	ski := Ski{Available: true}
	rental := newTestRental(t, &ski)
	if err := rental.Return(&ski, time.Time{}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("got %v, want ErrInvalidInput", err)
	}
	if !rental.IsActive() || ski.Available {
		t.Error("failed return must not change rental or ski")
	}
}

func TestValidateRentalSelection(t *testing.T) {
	tests := []struct {
		name                        string
		customerID, skiID, tariffID int
		wantErr                     bool
	}{
		{"complete", 1, 2, 3, false},
		{"no customer", 0, 2, 3, true},
		{"no ski", 1, 0, 3, true},
		{"no tariff", 1, 2, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRentalSelection(tt.customerID, tt.skiID, tt.tariffID)
			if tt.wantErr != errors.Is(err, ErrInvalidInput) {
				t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRentalDetails(t *testing.T) {
	ski := Ski{Name: "Carver", Available: true}
	rental := newTestRental(t, &ski)

	details := rental.Details(Customer{Name: "Max"}, ski)

	if details.CustomerName != "Max" || details.SkiName != "Carver" {
		t.Errorf("wrong names: %+v", details)
	}
	if !details.Active || !details.End.Equal(rental.End()) {
		t.Errorf("wrong derived data: %+v", details)
	}
}
