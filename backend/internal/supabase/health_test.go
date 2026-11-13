package supabase

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"specuse/backend/internal/account"
)

func TestDatabaseReadinessAndInvalidSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/v1/rpc/backend_health" {
			_, _ = w.Write([]byte("true"))
			return
		}
		_, _ = w.Write([]byte("null"))
	}))
	defer server.Close()
	c, _ := New(server.URL, "public", "server", true)
	repo := NewRepository(c)
	if err := repo.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.LoadSnapshot(context.Background()); !errors.Is(err, account.ErrUnavailable) {
		t.Fatal("null snapshot became empty catalog")
	}
}
