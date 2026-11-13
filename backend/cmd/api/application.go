package main

import (
	"errors"
	"os"
	"strings"

	"specuse/backend/internal/account"
	"specuse/backend/internal/build"
	"specuse/backend/internal/catalog"
	"specuse/backend/internal/supabase"
)

type application struct {
	catalog  catalog.Repository
	builds   build.Repository
	members  *account.Service
	database *supabase.Repository
}

func loadApplication() (application, error) {
	mode := os.Getenv("CATALOG_BACKEND")
	if mode == "" {
		mode = "memory"
	}
	if mode == "memory" {
		if os.Getenv("SUPABASE_URL") != "" {
			return application{}, errors.New("set CATALOG_BACKEND=supabase explicitly when configuring Supabase")
		}
		repo, err := loadCatalog(os.Getenv("APP_ENV"), os.Getenv("LOCAL_CATALOG_FILE"))
		return application{catalog: repo, builds: repo}, err
	}
	if mode != "supabase" {
		return application{}, errors.New("CATALOG_BACKEND must be memory or supabase")
	}
	if os.Getenv("LOCAL_CATALOG_FILE") != "" {
		return application{}, errors.New("Supabase mode cannot mix a local catalog fixture")
	}
	base := strings.TrimSuffix(os.Getenv("SUPABASE_URL"), "/")
	client, err := supabase.New(base, os.Getenv("SUPABASE_PUBLISHABLE_KEY"), os.Getenv("SUPABASE_SECRET_KEY"), os.Getenv("APP_ENV") == "local")
	if err != nil {
		return application{}, errors.New("invalid or missing Supabase configuration; check backend environment variables")
	}
	issuer := os.Getenv("SUPABASE_JWT_ISSUER")
	if issuer == "" {
		issuer = base + "/auth/v1"
	}
	audience := os.Getenv("SUPABASE_JWT_AUDIENCE")
	if audience == "" {
		audience = "authenticated"
	}
	repo := supabase.NewRepository(client)
	return application{catalog: repo, builds: repo, members: account.New(client, repo, issuer, audience), database: repo}, nil
}
