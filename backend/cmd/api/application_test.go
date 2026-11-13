package main

import (
	"strings"
	"testing"
)

func cleanConfig(t *testing.T) {
	t.Helper()
	for _, k := range []string{"APP_ENV", "CATALOG_BACKEND", "SUPABASE_URL", "SUPABASE_PUBLISHABLE_KEY", "SUPABASE_SECRET_KEY", "SUPABASE_JWT_ISSUER", "SUPABASE_JWT_AUDIENCE", "LOCAL_CATALOG_FILE"} {
		t.Setenv(k, "")
	}
}
func TestApplicationConfigurationDoesNotSilentlyFallback(t *testing.T) {
	for _, tt := range []struct {
		name    string
		env     map[string]string
		wantErr bool
	}{
		{"default test memory", nil, false},
		{"production memory", map[string]string{"APP_ENV": "production"}, true},
		{"unknown mode", map[string]string{"CATALOG_BACKEND": "typo"}, true},
		{"incomplete Supabase", map[string]string{"CATALOG_BACKEND": "supabase"}, true},
		{"accidental memory", map[string]string{"SUPABASE_URL": "https://project.invalid"}, true},
		{"mixed fixture and DB", map[string]string{"CATALOG_BACKEND": "supabase", "LOCAL_CATALOG_FILE": "../../testdata/catalog.local.json"}, true},
		{"configured production", map[string]string{"APP_ENV": "production", "CATALOG_BACKEND": "supabase", "SUPABASE_URL": "https://project.invalid", "SUPABASE_PUBLISHABLE_KEY": "public-test", "SUPABASE_SECRET_KEY": "server-test"}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cleanConfig(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			app, err := loadApplication()
			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected config result %v", err)
			}
			if err != nil && strings.Contains(err.Error(), "server-test") {
				t.Fatal("secret leaked")
			}
			if !tt.wantErr && tt.env != nil && (app.database == nil || app.members == nil) {
				t.Fatal("Supabase not wired")
			}
		})
	}
}
