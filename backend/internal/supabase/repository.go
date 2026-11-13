package supabase

import (
	"context"
	"encoding/json"
	"strings"

	"specuse/backend/internal/account"
	"specuse/backend/internal/build"
	"specuse/backend/internal/catalog"
	"specuse/backend/internal/repository"
)

type Repository struct{ client *Client }

func NewRepository(c *Client) *Repository { return &Repository{client: c} }
func (r *Repository) LoadSnapshot(ctx context.Context) (build.Snapshot, error) {
	snapshot, _, err := r.LoadSnapshotForWrite(ctx)
	return snapshot, err
}
func (r *Repository) LoadSnapshotForWrite(ctx context.Context) (build.Snapshot, map[string]json.RawMessage, error) {
	var data struct {
		Products  []json.RawMessage `json:"products"`
		RAMModels []struct {
			ID      string `json:"id"`
			RAMType string `json:"ram_type"`
		} `json:"ram_models"`
	}
	if err := r.client.RPC(ctx, "backend_catalog_snapshot", map[string]any{}, &data); err != nil {
		return build.Snapshot{}, nil, err
	}
	if data.Products == nil || data.RAMModels == nil {
		return build.Snapshot{}, nil, account.ErrUnavailable
	}
	snapshot := build.Snapshot{Products: []build.Component{}, RAMModels: []build.RAMModel{}}
	expected := make(map[string]json.RawMessage, len(data.Products))
	for _, raw := range data.Products {
		var p catalog.ProductDetails
		if err := json.Unmarshal(raw, &p); err != nil {
			return build.Snapshot{}, nil, account.ErrUnavailable
		}
		expected[strings.ToLower(p.ID)] = raw
		specs := catalog.Specs{}
		if p.Specs != nil {
			specs = *p.Specs
		}
		snapshot.Products = append(snapshot.Products, build.Component{Product: p.Product, RAMModelID: p.RAMModelID, Specs: build.Specs{CPUSocket: specs.CPUSocket, RAMType: specs.RAMType, ModulesPerPackage: specs.ModulesPerPackage, GPU: specs.GPU, Storage: specs.Storage, Motherboard: specs.Motherboard}})
	}
	for _, m := range data.RAMModels {
		snapshot.RAMModels = append(snapshot.RAMModels, build.RAMModel{ID: m.ID, RAMType: m.RAMType})
	}
	// Validate trusted storage before use; never accept synthetic evidence in
	// the database adapter or silently substitute a local fixture on failure.
	mem, err := repository.NewBuildMemory(snapshot)
	if err != nil {
		return build.Snapshot{}, nil, account.ErrUnavailable
	}
	snapshot, err = mem.LoadSnapshot(ctx)
	if err != nil {
		return build.Snapshot{}, nil, err
	}
	return snapshot, expected, nil
}
func (r *Repository) memory(ctx context.Context) (*repository.BuildMemory, error) {
	snapshot, err := r.LoadSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	return repository.NewBuildMemory(snapshot)
}
func (r *Repository) ListActive(ctx context.Context, f catalog.Filter) ([]catalog.ProductDetails, int, error) {
	m, err := r.memory(ctx)
	if err != nil {
		return nil, 0, err
	}
	return m.ListActive(ctx, f)
}
func (r *Repository) FindActiveByID(ctx context.Context, id string) (catalog.ProductDetails, error) {
	m, err := r.memory(ctx)
	if err != nil {
		return catalog.ProductDetails{}, err
	}
	return m.FindActiveByID(ctx, id)
}
func (r *Repository) GetProfile(ctx context.Context, id string) (account.Profile, error) {
	var profile account.Profile
	err := r.client.RPC(ctx, "backend_profile_get", map[string]string{"p_user_id": strings.ToLower(id)}, &profile)
	return profile, err
}
func (r *Repository) UpdateProfile(ctx context.Context, id, name string) (account.Profile, error) {
	var profile account.Profile
	err := r.client.RPC(ctx, "backend_profile_update", map[string]string{"p_user_id": strings.ToLower(id), "p_display_name": name}, &profile)
	return profile, err
}
