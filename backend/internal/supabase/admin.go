package supabase

import (
	"context"
	"encoding/json"

	"specuse/backend/internal/catalog"
)

func (r *Repository) SaveProduct(ctx context.Context, actor string, p catalog.ProductDetails, create bool, expected json.RawMessage) (catalog.ProductDetails, error) {
	var saved catalog.ProductDetails
	err := r.client.RPC(ctx, "backend_product_save_checked", map[string]any{"p_actor": actor, "p_product": p, "p_create": create, "p_expected": expected}, &saved)
	return saved, err
}
func (r *Repository) DeactivateProduct(ctx context.Context, actor, id string) error {
	return r.client.RPC(ctx, "backend_product_deactivate", map[string]string{"p_actor": actor, "p_id": id}, nil)
}
