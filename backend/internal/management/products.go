package management

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"

	"specuse/backend/internal/account"
	"specuse/backend/internal/build"
	"specuse/backend/internal/catalog"
	"specuse/backend/internal/hardware"
	"specuse/backend/internal/repository"
)

type Repository interface {
	LoadSnapshot(context.Context) (build.Snapshot, error)
	LoadSnapshotForWrite(context.Context) (build.Snapshot, map[string]json.RawMessage, error)
	SaveProduct(context.Context, string, catalog.ProductDetails, bool, json.RawMessage) (catalog.ProductDetails, error)
	DeactivateProduct(context.Context, string, string) error
}
type Change struct {
	Name       *string         `json:"name"`
	Category   *string         `json:"category"`
	Brand      *string         `json:"brand"`
	Price      *int64          `json:"price_thb"`
	Active     *bool           `json:"is_active"`
	Specs      json.RawMessage `json:"specs"`
	RAMModelID *string         `json:"ram_model_id"`
}
type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }
func admin(id account.Identity) error {
	if id.Profile.Role != "admin" || !catalog.ValidUUID(id.ID) {
		return account.ErrForbidden
	}
	return nil
}
func views(snapshot build.Snapshot) ([]catalog.ProductDetails, error) {
	copy, err := hardware.Clone(snapshot)
	if err != nil {
		return nil, account.ErrUnavailable
	}
	active := map[string]bool{}
	for i := range copy.Products {
		p := &copy.Products[i].Product
		active[p.ID] = p.IsActive
		p.IsActive = true
	}
	repo, err := repository.NewBuildMemory(copy)
	if err != nil {
		return nil, account.ErrUnavailable
	}
	size := len(copy.Products)
	if size == 0 {
		size = 1
	}
	rows, _, err := repo.ListActive(context.Background(), catalog.Filter{Page: 1, PageSize: size})
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].IsActive = active[rows[i].ID]
	}
	return rows, nil
}
func (s *Service) List(ctx context.Context, id account.Identity, f catalog.Filter) (catalog.Page, error) {
	if err := admin(id); err != nil {
		return catalog.Page{}, err
	}
	if catalog.ValidateFilter(f) != nil {
		return catalog.Page{}, account.ErrInput
	}
	snapshot, err := s.repo.LoadSnapshot(ctx)
	if err != nil {
		return catalog.Page{}, err
	}
	rows, err := views(snapshot)
	if err != nil {
		return catalog.Page{}, err
	}
	filtered := []catalog.ProductDetails{}
	for _, p := range rows {
		if (strings.TrimSpace(f.Category) == "" || p.Category == strings.TrimSpace(f.Category)) && strings.Contains(strings.ToLower(p.Name), strings.ToLower(strings.TrimSpace(f.Search))) {
			filtered = append(filtered, p)
		}
	}
	total := len(filtered)
	offset := (f.Page - 1) * f.PageSize
	result := []catalog.ProductDetails{}
	if offset < total {
		end := offset + f.PageSize
		if end < offset || end > total {
			end = total
		}
		result = filtered[offset:end]
	}
	return catalog.Page{Items: result, Page: f.Page, PageSize: f.PageSize, Total: total}, nil
}
func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", account.ErrUnavailable
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
func (s *Service) Save(ctx context.Context, actor account.Identity, id string, c Change, create bool) (catalog.ProductDetails, error) {
	if err := admin(actor); err != nil {
		return catalog.ProductDetails{}, err
	}
	if create {
		if c.Name == nil || c.Category == nil || c.Brand == nil || c.Price == nil || c.Active == nil {
			return catalog.ProductDetails{}, account.ErrInput
		}
	} else {
		if !catalog.ValidUUID(id) || c.Category != nil {
			return catalog.ProductDetails{}, account.ErrInput
		}
		if c.Name == nil && c.Brand == nil && c.Price == nil && c.Active == nil && len(c.Specs) == 0 && c.RAMModelID == nil {
			return catalog.ProductDetails{}, account.ErrInput
		}
	}
	snapshot, expected, err := s.repo.LoadSnapshotForWrite(ctx)
	if err != nil {
		return catalog.ProductDetails{}, err
	}
	rows, err := views(snapshot)
	if err != nil {
		return catalog.ProductDetails{}, err
	}
	p := catalog.ProductDetails{}
	if create {
		id, err = newID()
		if err != nil {
			return p, err
		}
		p.ID = id
	} else {
		found := false
		for _, row := range rows {
			if row.ID == strings.ToLower(id) {
				p = row
				found = true
				break
			}
		}
		if !found {
			return p, account.ErrNotFound
		}
	}
	if c.Name != nil {
		p.Name = *c.Name
	}
	if c.Category != nil {
		p.Category = *c.Category
	}
	if c.Brand != nil {
		p.Brand = *c.Brand
	}
	if c.Price != nil {
		p.PriceTHB = *c.Price
	}
	if c.Active != nil {
		p.IsActive = *c.Active
	}
	if c.RAMModelID != nil {
		p.RAMModelID = *c.RAMModelID
	}
	encoded, _ := json.Marshal(p)
	var object map[string]json.RawMessage
	_ = json.Unmarshal(encoded, &object)
	if len(c.Specs) > 0 {
		object["specs"] = c.Specs
	}
	encoded, err = json.Marshal(object)
	if err != nil {
		return p, account.ErrInput
	}
	if json.Unmarshal(encoded, &p) != nil {
		return catalog.ProductDetails{}, account.ErrInput
	}
	specs := catalog.Specs{}
	if p.Specs != nil {
		specs = *p.Specs
	}
	component := build.Component{Product: p.Product, RAMModelID: p.RAMModelID, Specs: build.Specs{CPUSocket: specs.CPUSocket, RAMType: specs.RAMType, ModulesPerPackage: specs.ModulesPerPackage, GPU: specs.GPU, Storage: specs.Storage, Motherboard: specs.Motherboard}}
	replaced := false
	for i := range snapshot.Products {
		if snapshot.Products[i].Product.ID == p.ID {
			snapshot.Products[i] = component
			replaced = true
			break
		}
	}
	if !replaced {
		snapshot.Products = append(snapshot.Products, component)
	}
	if _, err := repository.NewBuildMemory(snapshot); err != nil {
		return catalog.ProductDetails{}, account.ErrInput
	}
	var original json.RawMessage
	if !create {
		original = expected[p.ID]
		if len(original) == 0 {
			return catalog.ProductDetails{}, account.ErrUnavailable
		}
	}
	return s.repo.SaveProduct(ctx, actor.ID, p, create, original)
}
func (s *Service) Delete(ctx context.Context, actor account.Identity, id string) error {
	if err := admin(actor); err != nil {
		return err
	}
	if !catalog.ValidUUID(id) {
		return account.ErrInput
	}
	return s.repo.DeactivateProduct(ctx, actor.ID, strings.ToLower(id))
}
