package domain

// Ski is a single rentable item.
type Ski struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Available bool   `json:"available"`
}

// NewSki creates an available ski. The store assigns the ID.
func NewSki(name string) (Ski, error) {
	ski := Ski{Available: true}
	if err := ski.Rename(name); err != nil {
		return Ski{}, err
	}
	return ski, nil
}

// Rename changes the name. Availability can only change through a rental.
func (s *Ski) Rename(name string) error {
	name, err := required(name, "Bezeichnung")
	if err != nil {
		return err
	}
	s.Name = name
	return nil
}

// rent and release are unexported so that availability only changes together
// with a rental (see NewRental and Rental.Return).
func (s *Ski) rent() error {
	if !s.Available {
		return ErrSkiUnavailable
	}
	s.Available = false
	return nil
}

func (s *Ski) release() {
	s.Available = true
}
