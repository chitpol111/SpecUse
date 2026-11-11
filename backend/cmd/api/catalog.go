package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"specuse/backend/internal/build"
	"specuse/backend/internal/catalog"
	"specuse/backend/internal/repository"
)

// loadCatalog requires two explicit settings before opening a local fixture.
// Without a path, every environment retains the default empty repository.
func loadCatalog(appEnv, path string) (*repository.BuildMemory, error) {
	if path == "" {
		return repository.NewBuildMemory(build.Snapshot{})
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
	// These fields are trusted local fixture metadata, not public product DTOs.
	var products []struct {
		catalog.Product
		Specs struct {
			CPUSocket         string `json:"cpu_socket"`
			RAMType           string `json:"ram_type"`
			ModulesPerPackage int64  `json:"modules_per_package"`
		} `json:"specs"`
		RAMModelID string `json:"ram_model_id"`
	}
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
	snapshot := build.Snapshot{Products: []build.Component{}, RAMModels: []build.RAMModel{}}
	models := map[string]bool{}
	for _, product := range products {
		snapshot.Products = append(snapshot.Products, build.Component{
			Product: product.Product, RAMModelID: product.RAMModelID,
			Specs: build.Specs{CPUSocket: product.Specs.CPUSocket, RAMType: product.Specs.RAMType, ModulesPerPackage: product.Specs.ModulesPerPackage},
		})
		modelID := strings.ToLower(product.RAMModelID)
		if product.Category == "ram" && !models[modelID] {
			snapshot.RAMModels = append(snapshot.RAMModels, build.RAMModel{ID: modelID, RAMType: product.Specs.RAMType})
			models[modelID] = true
		}
	}
	repo, err := repository.NewBuildMemory(snapshot)
	if err != nil {
		return nil, fmt.Errorf("validate local catalog fixture: %w", err)
	}
	return repo, nil
}
