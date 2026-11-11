package supabase

import (
	"context"
	"specuse/backend/internal/catalog"
)

func (r *Repository) SaveProduct(ctx context.Context, actor string, p catalog.ProductDetails, create bool) (catalog.ProductDetails, error) {
	var saved catalog.ProductDetails
	err := r.client.RPC(ctx, "backend_product_save", map[string]any{"p_actor": actor, "p_product": p, "p_create": create}, &saved)
	return saved, err
}
func (r *Repository) DeactivateProduct(ctx context.Context, actor, id string) error {
	return r.client.RPC(ctx, "backend_product_deactivate", map[string]string{"p_actor": actor, "p_id": id}, nil)
}
