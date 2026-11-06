// Ski-Verleih is a prototype of a ski rental with a web dashboard.
package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"ski-verleih/internal/handler"
	"ski-verleih/internal/store"
)

// Local only: the data contains ID documents and the API has no login.
const address = "127.0.0.1:8080"

// frontendFiles must be package-level because go:embed requires it.
//
//go:embed frontend
var frontendFiles embed.FS

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	s := store.New()
	if err := s.Seed(time.Now()); err != nil {
		return fmt.Errorf("load demo data: %w", err)
	}

	frontend, err := fs.Sub(frontendFiles, "frontend")
	if err != nil {
		return fmt.Errorf("open frontend files: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", handler.New(s).Routes())
	mux.Handle("/", http.FileServerFS(frontend))

	server := &http.Server{
		Addr:              address,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("Ski-Verleih läuft auf http://%s", address)
	return server.ListenAndServe()
}
