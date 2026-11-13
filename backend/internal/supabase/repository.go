package supabase

import (
	"context"
	"strings"

	"specuse/backend/internal/account"
	"specuse/backend/internal/build"
	"specuse/backend/internal/catalog"
	"specuse/backend/internal/repository"
)

type Repository struct{ client *Client }

func NewRepository(c *Client) *Repository { return &Repository{client: c} }
func (r *Repository) LoadSnapshot(ctx context.Context) (build.Snapshot, error) {
	var data struct {
		Products  []catalog.ProductDetails `json:"products"`
		RAMModels []struct {
			ID      string `json:"id"`
			RAMType string `json:"ram_type"`
		} `json:"ram_models"`
	}
	if err := r.client.RPC(ctx, "backend_catalog_snapshot", map[string]any{}, &data); err != nil {
		return build.Snapshot{}, err
	}
	snapshot := build.Snapshot{Products: []build.Component{}, RAMModels: []build.RAMModel{}}
	for _, p := range data.Products {
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
		return build.Snapshot{}, account.ErrUnavailable
	}
	return mem.LoadSnapshot(ctx)
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
