// Package domain contains the data models and business rules of the ski
// rental. It knows nothing about HTTP or data storage.
package domain

import (
	"errors"
	"fmt"
	"strings"
)

// Error messages are German on purpose: they are shown to users unchanged.
var (
	ErrInvalidInput    = errors.New("Ungültige Eingabe")
	ErrNotFound        = errors.New("Nicht gefunden")
	ErrSkiUnavailable  = errors.New("Der Ski ist bereits ausgeliehen")
	ErrAlreadyReturned = errors.New("Die Ausleihe wurde bereits zurückgegeben")
)

// InvalidInput returns an error wrapping ErrInvalidInput with a specific message.
func InvalidInput(message string) error {
	return fmt.Errorf("%w: %s", ErrInvalidInput, message)
}

// required trims value and returns an error if nothing remains.
func required(value, label string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", InvalidInput(label + " fehlt")
	}
	return value, nil
}
