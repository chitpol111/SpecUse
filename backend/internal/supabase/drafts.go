package supabase

import (
	"context"
	"specuse/backend/internal/account"
	"specuse/backend/internal/catalog"
	"specuse/backend/internal/draft"
)

type draftRow struct {
	draft.Summary
	Items    []draft.Item `json:"items"`
	RAMItems []struct {
		draft.RAMItem
		ActivePackages *int64 `json:"active_package_count"`
	} `json:"ram_items"`
}

func (row draftRow) build() (draft.Build, error) {
	if !catalog.ValidUUID(row.ID) || row.Visibility != "private" || row.Items == nil || row.RAMItems == nil || row.Name == "" || row.CreatedAt.IsZero() || row.UpdatedAt.IsZero() {
		return draft.Build{}, account.ErrUnavailable
	}
	seen := map[string]bool{}
	for _, item := range row.Items {
		if !catalog.ValidUUID(item.ProductID) || item.Quantity < 1 || item.OwnedQuantity < 0 || item.OwnedQuantity > item.Quantity || item.ProductSummary.Name == "" || item.ProductSummary.Category == "" || item.ProductSummary.Category == "ram" || seen[item.ProductID] {
			return draft.Build{}, account.ErrUnavailable
		}
		seen[item.ProductID] = true
	}
	row.CreatedAt = row.CreatedAt.UTC()
	row.UpdatedAt = row.UpdatedAt.UTC()
	result := draft.Build{Summary: row.Summary, Items: row.Items, RAMItems: []draft.RAMItem{}}
	seen = map[string]bool{}
	for _, r := range row.RAMItems {
		if r.ActivePackages == nil || *r.ActivePackages < 0 || !catalog.ValidUUID(r.RAMModelID) || r.RequiredModules < 1 || r.OwnedModules < 0 || r.OwnedModules > r.RequiredModules || r.RAMModelSummary.RAMType == "" || seen[r.RAMModelID] {
			return draft.Build{}, account.ErrUnavailable
		}
		seen[r.RAMModelID] = true
		item := r.RAMItem
		item.ActivePackageCount = *r.ActivePackages
		result.RAMItems = append(result.RAMItems, item)
	}
	return result, nil
}
func (r *Repository) ListDrafts(ctx context.Context, actor string, f draft.Filter) (draft.Page, error) {
	var result draft.Page
	err := r.client.RPC(ctx, "backend_member_build_list", map[string]any{"p_actor": actor, "p_page": f.Page, "p_page_size": f.PageSize}, &result)
	if err == nil && (result.Items == nil || result.Page != f.Page || result.PageSize != f.PageSize || result.Total < 0) {
		err = account.ErrUnavailable
	}
	return result, err
}
func (r *Repository) GetDraft(ctx context.Context, actor, id string) (draft.Build, error) {
	var row draftRow
	if err := r.client.RPC(ctx, "backend_member_build_get", map[string]string{"p_actor": actor, "p_id": id}, &row); err != nil {
		return draft.Build{}, err
	}
	b, err := row.build()
	if err == nil && b.ID != id {
		err = account.ErrUnavailable
	}
	return b, err
}
func (r *Repository) CreateDraft(ctx context.Context, actor string, c draft.Change) (draft.Build, error) {
	var row draftRow
	if err := r.client.RPC(ctx, "backend_member_build_create", map[string]any{"p_actor": actor, "p_build": c}, &row); err != nil {
		return draft.Build{}, err
	}
	return row.build()
}
func (r *Repository) PatchDraft(ctx context.Context, actor, id string, c draft.Change) (draft.Build, error) {
	var row draftRow
	if err := r.client.RPC(ctx, "backend_member_build_patch", map[string]any{"p_actor": actor, "p_id": id, "p_patch": c}, &row); err != nil {
		return draft.Build{}, err
	}
	b, err := row.build()
	if err == nil && b.ID != id {
		err = account.ErrUnavailable
	}
	return b, err
}
func (r *Repository) DeleteDraft(ctx context.Context, actor, id string) error {
	return r.client.RPC(ctx, "backend_member_build_delete", map[string]string{"p_actor": actor, "p_id": id}, nil)
}
