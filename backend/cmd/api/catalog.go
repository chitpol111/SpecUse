package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"specuse/backend/internal/catalog"
	"specuse/backend/internal/repository"
)

// loadCatalog requires two explicit settings before opening a local fixture.
// Without a path, every environment retains the default empty repository.
func loadCatalog(appEnv, path string) (*repository.Memory, error) {
	if path == "" {
		return repository.NewMemory(nil)
	}
	if appEnv != "local" {
		return nil, errors.New("LOCAL_CATALOG_FILE requires APP_ENV=local; refusing to load development data")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open local catalog fixture: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var products []catalog.Product
	if err := decoder.Decode(&products); err != nil {
		return nil, fmt.Errorf("decode local catalog fixture: %w", err)
	}
	if products == nil {
		return nil, errors.New("local catalog fixture must be a JSON array, not null")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, errors.New("local catalog fixture must contain exactly one JSON array")
	}
	repo, err := repository.NewMemory(products)
	if err != nil {
		return nil, fmt.Errorf("validate local catalog fixture: %w", err)
	}
	return repo, nil
}
