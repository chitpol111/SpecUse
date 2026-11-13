package supabase

import (
	"context"
	"specuse/backend/internal/account"
)

func (r *Repository) Ping(ctx context.Context) error {
	var ready bool
	if err := r.client.RPC(ctx, "backend_health", map[string]any{}, &ready); err != nil {
		return err
	}
	if !ready {
		return account.ErrUnavailable
	}
	return nil
}
