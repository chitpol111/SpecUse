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
	var products []catalog.ProductDetails
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
		specs := catalog.Specs{}
		if product.Specs != nil {
			specs = *product.Specs
		}
		snapshot.Products = append(snapshot.Products, build.Component{
			Product: product.Product, RAMModelID: product.RAMModelID,
			Specs: build.Specs{CPUSocket: specs.CPUSocket, RAMType: specs.RAMType, ModulesPerPackage: specs.ModulesPerPackage, GPU: specs.GPU, Storage: specs.Storage, Motherboard: specs.Motherboard},
		})
		modelID := strings.ToLower(product.RAMModelID)
		if product.Category == "ram" && !models[modelID] {
			snapshot.RAMModels = append(snapshot.RAMModels, build.RAMModel{ID: modelID, RAMType: specs.RAMType})
			models[modelID] = true
		}
	}
	repo, err := repository.NewLocalBuildMemory(snapshot)
	if err != nil {
		return nil, fmt.Errorf("validate local catalog fixture: %w", err)
	}
	return repo, nil
}
