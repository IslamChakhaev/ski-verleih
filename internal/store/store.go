// Package store keeps all data in memory and is safe for concurrent use.
// Methods return copies, so callers cannot change stored data directly.
package store

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"ski-verleih/internal/domain"
)

// Store holds all skis, customers, tariffs and rentals.
type Store struct {
	mu sync.RWMutex

	skis      map[int]domain.Ski
	customers map[int]domain.Customer
	tariffs   map[int]domain.Tariff
	rentals   map[int]domain.Rental

	nextSkiID      int
	nextCustomerID int
	nextTariffID   int
	nextRentalID   int
}

// New returns an empty store.
func New() *Store {
	return &Store{
		skis:           make(map[int]domain.Ski),
		customers:      make(map[int]domain.Customer),
		tariffs:        make(map[int]domain.Tariff),
		rentals:        make(map[int]domain.Rental),
		nextSkiID:      1,
		nextCustomerID: 1,
		nextTariffID:   1,
		nextRentalID:   1,
	}
}

func notFound(label string, id int) error {
	return fmt.Errorf("%w: %s mit ID %d", domain.ErrNotFound, label, id)
}

// sortedValues returns the map values ordered by ID, because map iteration
// order is random. The result is never nil, so it encodes as a JSON array.
func sortedValues[T any](items map[int]T) []T {
	ids := make([]int, 0, len(items))
	for id := range items {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	values := make([]T, 0, len(ids))
	for _, id := range ids {
		values = append(values, items[id])
	}
	return values
}

// Skis returns all skis ordered by ID.
func (s *Store) Skis() []domain.Ski {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortedValues(s.skis)
}

// AddSki creates a new ski.
func (s *Store) AddSki(name string) (domain.Ski, error) {
	ski, err := domain.NewSki(name)
	if err != nil {
		return domain.Ski{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	ski.ID = s.nextSkiID
	s.nextSkiID++
	s.skis[ski.ID] = ski
	return ski, nil
}

// UpdateSki changes the name of a ski.
func (s *Store) UpdateSki(id int, name string) (domain.Ski, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ski, ok := s.skis[id]
	if !ok {
		return domain.Ski{}, notFound("Ski", id)
	}
	if err := ski.Rename(name); err != nil {
		return domain.Ski{}, err
	}
	s.skis[id] = ski
	return ski, nil
}

// Customers returns all customers ordered by ID.
func (s *Store) Customers() []domain.Customer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortedValues(s.customers)
}

// AddCustomer registers a new customer.
func (s *Store) AddCustomer(name, idType, idNumber string) (domain.Customer, error) {
	customer, err := domain.NewCustomer(name, idType, idNumber)
	if err != nil {
		return domain.Customer{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	customer.ID = s.nextCustomerID
	s.nextCustomerID++
	s.customers[customer.ID] = customer
	return customer, nil
}

// UpdateCustomer replaces the data of a customer.
func (s *Store) UpdateCustomer(id int, name, idType, idNumber string) (domain.Customer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	customer, ok := s.customers[id]
	if !ok {
		return domain.Customer{}, notFound("Person", id)
	}
	if err := customer.Update(name, idType, idNumber); err != nil {
		return domain.Customer{}, err
	}
	s.customers[id] = customer
	return customer, nil
}

// Tariffs returns all tariffs ordered by ID.
func (s *Store) Tariffs() []domain.Tariff {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortedValues(s.tariffs)
}

// Tariff returns a single tariff.
func (s *Store) Tariff(id int) (domain.Tariff, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tariff, ok := s.tariffs[id]
	if !ok {
		return domain.Tariff{}, notFound("Tarif", id)
	}
	return tariff, nil
}

// AddTariff creates a new tariff.
func (s *Store) AddTariff(name string, hourlyPriceCents int64) (domain.Tariff, error) {
	tariff, err := domain.NewTariff(name, hourlyPriceCents)
	if err != nil {
		return domain.Tariff{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	tariff.ID = s.nextTariffID
	s.nextTariffID++
	s.tariffs[tariff.ID] = tariff
	return tariff, nil
}

// UpdateTariff changes a tariff. Existing rentals keep their copied prices.
func (s *Store) UpdateTariff(id int, name string, hourlyPriceCents int64) (domain.Tariff, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tariff, ok := s.tariffs[id]
	if !ok {
		return domain.Tariff{}, notFound("Tarif", id)
	}
	if err := tariff.Update(name, hourlyPriceCents); err != nil {
		return domain.Tariff{}, err
	}
	s.tariffs[id] = tariff
	return tariff, nil
}

// Rentals returns all rentals with their display data, ordered by ID.
func (s *Store) Rentals() []domain.RentalDetails {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rentals := sortedValues(s.rentals)
	details := make([]domain.RentalDetails, 0, len(rentals))
	for _, rental := range rentals {
		details = append(details, s.details(rental))
	}
	return details
}

// details must be called with the lock held. Skis and customers cannot be
// deleted, so the lookups always succeed.
func (s *Store) details(rental domain.Rental) domain.RentalDetails {
	return rental.Details(s.customers[rental.CustomerID], s.skis[rental.SkiID])
}

// CreateRental rents a ski. All checks and changes happen under one lock, so a
// ski can never be rented twice at the same time.
func (s *Store) CreateRental(customerID, skiID, tariffID int, start time.Time, durationHours int) (domain.RentalDetails, error) {
	if err := domain.ValidateRentalSelection(customerID, skiID, tariffID); err != nil {
		return domain.RentalDetails{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	customer, ok := s.customers[customerID]
	if !ok {
		return domain.RentalDetails{}, notFound("Person", customerID)
	}
	ski, ok := s.skis[skiID]
	if !ok {
		return domain.RentalDetails{}, notFound("Ski", skiID)
	}
	tariff, ok := s.tariffs[tariffID]
	if !ok {
		return domain.RentalDetails{}, notFound("Tarif", tariffID)
	}

	rental, err := domain.NewRental(customer, &ski, tariff, start, durationHours)
	if err != nil {
		return domain.RentalDetails{}, err
	}
	rental.ID = s.nextRentalID
	s.nextRentalID++
	s.rentals[rental.ID] = rental
	s.skis[skiID] = ski
	return s.details(rental), nil
}

// ReturnRental ends a rental and makes its ski available in one step.
func (s *Store) ReturnRental(id int, at time.Time) (domain.RentalDetails, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rental, ok := s.rentals[id]
	if !ok {
		return domain.RentalDetails{}, notFound("Ausleihe", id)
	}
	ski := s.skis[rental.SkiID]
	if err := rental.Return(&ski, at); err != nil {
		return domain.RentalDetails{}, err
	}
	s.rentals[id] = rental
	s.skis[ski.ID] = ski
	return s.details(rental), nil
}

// Stats counts the current data under one lock, so the figures always match.
func (s *Store) Stats() domain.Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := domain.Stats{
		TotalSkis:           len(s.skis),
		RegisteredCustomers: len(s.customers),
	}
	for _, ski := range s.skis {
		if ski.Available {
			stats.AvailableSkis++
		} else {
			stats.RentedSkis++
		}
	}
	return stats
}
