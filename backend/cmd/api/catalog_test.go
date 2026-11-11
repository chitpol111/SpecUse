package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"specuse/backend/internal/catalog"
)

func TestLoadCatalogRequiresLocalOptIn(t *testing.T) {
	for _, appEnv := range []string{"", "production", "staging", "LOCAL"} {
		t.Run("reject_"+appEnv, func(t *testing.T) {
			// This path deliberately does not exist: the environment gate must
			// reject the request before even attempting to open a fixture.
			_, err := loadCatalog(appEnv, "does-not-exist.json")
			if err == nil || !strings.Contains(err.Error(), "requires APP_ENV=local") {
				t.Fatalf("expected local-only gate, got %v", err)
			}
		})
	}
	for _, appEnv := range []string{"", "production", "local"} {
		t.Run("empty_"+appEnv, func(t *testing.T) {
			repo, err := loadCatalog(appEnv, "")
			if err != nil {
				t.Fatal(err)
			}
			items, total, err := repo.ListActive(context.Background(), catalog.Filter{Page: 1, PageSize: 20})
			if err != nil || len(items) != 0 || total != 0 {
				t.Fatalf("default catalog must be empty: %v %d %v", items, total, err)
			}
		})
	}
}

func TestLoadLocalFixture(t *testing.T) {
	repo, err := loadCatalog("local", "../../testdata/catalog.local.json")
	if err != nil {
		t.Fatal(err)
	}
	items, total, err := repo.ListActive(context.Background(), catalog.Filter{Page: 1, PageSize: 20})
	if err != nil || len(items) != 4 || total != 4 {
		t.Fatalf("items=%d total=%d err=%v", len(items), total, err)
	}
	item, err := repo.FindActiveByID(context.Background(), "abcdefab-0000-0000-0000-000000000001")
	if err != nil || item.PriceTHB != 25000 || !strings.Contains(item.Name, "LOCAL TEST") {
		t.Fatalf("item=%+v err=%v", item, err)
	}
	_, err = repo.FindActiveByID(context.Background(), "abcdefab-0000-0000-0000-000000000002")
	if !errors.Is(err, catalog.ErrNotFound) {
		t.Fatalf("inactive fixture leaked: %v", err)
	}
}

func TestLoadCatalogRejectsInvalidFixtures(t *testing.T) {
	valid := `[{"id":"abcdefab-0000-0000-0000-000000000001","name":"Test","category":"cpu","brand":"Example","price_thb":25000,"is_active":true}]`
	for _, tt := range []struct {
		name string
		body string
	}{
		{"empty file", ""},
		{"null", "null"},
		{"object", "{}"},
		{"trailing document", valid + "[]"},
		{"trailing junk", valid + "oops"},
		{"unknown field", strings.Replace(valid, `"name":`, `"socket":"AM5","name":`, 1)},
		{"decimal price", strings.Replace(valid, "25000", "25000.50", 1)},
		{"negative price", strings.Replace(valid, "25000", "-1", 1)},
		{"invalid UUID", strings.Replace(valid, "abcdefab-0000-0000-0000-000000000001", "bad-id", 1)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "catalog.json")
			if err := os.WriteFile(path, []byte(tt.body), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := loadCatalog("local", path); err == nil {
				t.Fatal("expected fixture error")
			}
		})
	}
	if _, err := loadCatalog("local", filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("missing fixture should fail startup")
	}
}
