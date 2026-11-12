package draft

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"specuse/backend/internal/account"
)

const owner = "11111111-1111-4111-8111-111111111111"
const other = "22222222-2222-4222-8222-222222222222"
const productA = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
const productB = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
const modelA = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
const modelB = "dddddddd-dddd-4ddd-8ddd-dddddddddddd"

func identity(id, role string) account.Identity {
	return account.Identity{ID: id, Profile: account.Profile{ID: id, Role: role}}
}
func ptr[T any](v T) *T { return &v }

type memory struct {
	values           map[string]Build
	owners           map[string]string
	writes, sequence int
}

func newMemory() *memory { return &memory{values: map[string]Build{}, owners: map[string]string{}} }
func clone(b Build) Build {
	b.Items = append([]Item{}, b.Items...)
	b.RAMItems = append([]RAMItem{}, b.RAMItems...)
	return b
}
func (m *memory) ListDrafts(_ context.Context, actor string, f Filter) (Page, error) {
	rows := []Summary{}
	for id, b := range m.values {
		if m.owners[id] == actor {
			rows = append(rows, b.Summary)
		}
	}
	return Page{Items: rows, Page: f.Page, PageSize: f.PageSize, Total: int64(len(rows))}, nil
}
func (m *memory) GetDraft(_ context.Context, actor, id string) (Build, error) {
	if m.owners[id] != actor {
		return Build{}, account.ErrNotFound
	}
	return clone(m.values[id]), nil
}
func (m *memory) apply(b Build, c Change) Build {
	if c.Name != nil {
		b.Name = *c.Name
	}
	if c.Description != nil {
		b.Description = *c.Description
	}
	if c.Visibility != nil {
		b.Visibility = *c.Visibility
	}
	if c.Items != nil {
		b.Items = []Item{}
		for _, row := range *c.Items {
			b.Items = append(b.Items, Item{ItemInput: row, ProductSummary: ProductSummary{Name: "[TEST] component", Category: "gpu", IsActive: row.ProductID != productA}})
		}
	}
	if c.RAMItems != nil {
		b.RAMItems = []RAMItem{}
		for _, row := range *c.RAMItems {
			b.RAMItems = append(b.RAMItems, RAMItem{RAMInput: row, RAMModelSummary: RAMModelSummary{RAMType: "DDR4"}, ActivePackageCount: 0})
		}
	}
	b.UpdatedAt = time.Now().UTC()
	return b
}
func (m *memory) CreateDraft(_ context.Context, actor string, c Change) (Build, error) {
	m.sequence++
	id := fmt.Sprintf("00000000-0000-4000-8000-%012d", m.sequence)
	b := m.apply(Build{Summary: Summary{ID: id, CreatedAt: time.Now().UTC()}}, c)
	m.values[id] = clone(b)
	m.owners[id] = actor
	m.writes++
	return clone(b), nil
}
func (m *memory) PatchDraft(ctx context.Context, actor, id string, c Change) (Build, error) {
	b, e := m.GetDraft(ctx, actor, id)
	if e != nil {
		return b, e
	}
	b = m.apply(b, c)
	m.values[id] = clone(b)
	m.writes++
	return b, nil
}
func (m *memory) DeleteDraft(_ context.Context, actor, id string) error {
	if m.owners[id] != actor {
		return account.ErrNotFound
	}
	delete(m.values, id)
	delete(m.owners, id)
	m.writes++
	return nil
}
func populated() Change {
	return Change{Name: ptr(" Draft "), Items: ptr([]ItemInput{{ProductID: productA, Quantity: 3, OwnedQuantity: 2}, {ProductID: productB, Quantity: 5, OwnedQuantity: 4}}), RAMItems: ptr([]RAMInput{{RAMModelID: modelA, RequiredModules: 4, OwnedModules: 2}, {RAMModelID: modelB, RequiredModules: 6, OwnedModules: 6}})}
}

func TestDraftRoundTripPreservesEveryRequirementAndOmittedArrays(t *testing.T) {
	ctx := context.Background()
	repo := newMemory()
	s := New(repo)
	actor := identity(owner, "member")
	b, err := s.Create(ctx, actor, populated())
	if err != nil {
		t.Fatal(err)
	}
	if b.Name != "Draft" || b.Visibility != "private" || b.Description != "" || len(b.Items) != 2 || len(b.RAMItems) != 2 {
		t.Fatal("create defaults/data lost")
	}
	if b.Availability.ReadyForValidation || len(b.Availability.Issues) != 2 {
		t.Fatal("inactive and unavailable RAM were not preserved/reported")
	}
	first, second := b.Availability.Issues[0], b.Availability.Issues[1]
	if first.Code != "PRODUCT_INACTIVE" || first.ProductID == nil || first.RAMModelID != nil || second.Code != "RAM_PACKAGES_UNAVAILABLE" || second.ProductID != nil || second.RAMModelID == nil || *second.RAMModelID != modelA {
		t.Fatal("issue references inconsistent or fully owned RAM blocked")
	}
	beforeItems, beforeRAM := b.Items, b.RAMItems
	b, err = s.Patch(ctx, actor, b.ID, Change{Name: ptr("Renamed"), Description: ptr(" Notes\n")})
	if err != nil {
		t.Fatal(err)
	}
	if b.Description != " Notes\n" {
		t.Fatal("description formatting changed")
	}
	got, err := s.Get(ctx, actor, b.ID)
	if err != nil || !reflect.DeepEqual(got.Items, beforeItems) || !reflect.DeepEqual(got.RAMItems, beforeRAM) {
		t.Fatal("metadata PATCH dropped hidden selections or quantities")
	}
	// A client that can edit an item merges it into the complete arrays.
	change := populated()
	(*change.Items)[1].OwnedQuantity = 3
	b, err = s.Patch(ctx, actor, b.ID, Change{Items: change.Items, RAMItems: change.RAMItems})
	if err != nil || len(b.Items) != 2 || len(b.RAMItems) != 2 || b.Items[0].Quantity != 3 || b.Items[1].OwnedQuantity != 3 {
		t.Fatal("full-array replacement collapsed multiple SKUs/counts")
	}
	raw, _ := json.Marshal(b)
	for _, field := range []string{"active_package_count", "owner_id", "price_thb", "total_price_thb", "compatible"} {
		if strings.Contains(string(raw), `"`+field+`"`) {
			t.Fatal("internal/derived field leaked")
		}
	}
	if !strings.Contains(string(raw), `"ram_model_id":null`) {
		t.Fatal("missing fixed null issue field")
	}
	b, err = s.Patch(ctx, actor, b.ID, Change{Items: ptr([]ItemInput{})})
	if err != nil || len(b.Items) != 0 || len(b.RAMItems) != 2 {
		t.Fatal("explicit clear changed omitted RAM")
	}
	b, err = s.Patch(ctx, actor, b.ID, Change{RAMItems: ptr([]RAMInput{})})
	if err != nil || len(b.RAMItems) != 0 || !b.Availability.ReadyForValidation || len(b.Availability.Issues) != 0 {
		t.Fatal("empty private draft not supported")
	}
	if err = s.Delete(ctx, actor, b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Get(ctx, actor, b.ID); !errors.Is(err, account.ErrNotFound) {
		t.Fatal("deleted build still present")
	}
}
func TestDraftOwnershipIncludingAdminAndPublicWrites(t *testing.T) {
	ctx := context.Background()
	repo := newMemory()
	s := New(repo)
	b, err := s.Create(ctx, identity(owner, "member"), populated())
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"member", "admin"} {
		actor := identity(other, role)
		if _, e := s.Get(ctx, actor, b.ID); !errors.Is(e, account.ErrNotFound) {
			t.Fatal("foreign read")
		}
		if _, e := s.Patch(ctx, actor, b.ID, Change{Name: ptr("stolen")}); !errors.Is(e, account.ErrNotFound) {
			t.Fatal("foreign edit")
		}
		if e := s.Delete(ctx, actor, b.ID); !errors.Is(e, account.ErrNotFound) {
			t.Fatal("foreign delete")
		}
		page, e := s.List(ctx, actor, Filter{Page: 1, PageSize: 20})
		if e != nil || page.Total != 0 || len(page.Items) != 0 {
			t.Fatal("foreign list")
		}
	}
	c := populated()
	c.Visibility = ptr("public")
	writes := repo.writes
	if _, e := s.Create(ctx, identity(owner, "member"), c); !errors.Is(e, ErrPublicDisabled) {
		t.Fatal("public create")
	}
	if _, e := s.Patch(ctx, identity(owner, "member"), b.ID, Change{Name: ptr("must rollback"), Visibility: ptr("public")}); !errors.Is(e, ErrPublicDisabled) {
		t.Fatal("public patch")
	}
	if repo.writes != writes {
		t.Fatal("rejected public request mutated data")
	}
	if _, e := s.Create(ctx, identity(owner, "guest"), populated()); !errors.Is(e, account.ErrForbidden) {
		t.Fatal("untrusted identity accepted")
	}
}
func TestDraftInputValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Change)
	}{
		{"missing name", func(c *Change) { c.Name = nil }},
		{"missing items", func(c *Change) { c.Items = nil }},
		{"missing RAM", func(c *Change) { c.RAMItems = nil }},
		{"blank name", func(c *Change) { c.Name = ptr("  ") }},
		{"name length", func(c *Change) { c.Name = ptr(strings.Repeat("ก", 101)) }},
		{"description length", func(c *Change) { c.Description = ptr(strings.Repeat("ก", 2001)) }},
		{"NUL", func(c *Change) { c.Name = ptr("bad\x00name") }},
		{"visibility", func(c *Change) { c.Visibility = ptr("unlisted") }},
		{"UUID", func(c *Change) { (*c.Items)[0].ProductID = "bad" }},
		{"zero", func(c *Change) { (*c.Items)[0].Quantity = 0 }},
		{"negative owned", func(c *Change) { (*c.Items)[0].OwnedQuantity = -1 }},
		{"owned exceeds", func(c *Change) { (*c.Items)[0].OwnedQuantity = 4 }},
		{"duplicate case", func(c *Change) { (*c.Items)[1].ProductID = strings.ToUpper(productA) }},
		{"RAM UUID", func(c *Change) { (*c.RAMItems)[0].RAMModelID = "bad" }},
		{"RAM zero", func(c *Change) { (*c.RAMItems)[0].RequiredModules = 0 }},
		{"RAM owned", func(c *Change) { (*c.RAMItems)[0].OwnedModules = 5 }},
		{"RAM duplicate", func(c *Change) { (*c.RAMItems)[1].RAMModelID = strings.ToUpper(modelA) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMemory()
			c := populated()
			tc.change(&c)
			if _, e := New(repo).Create(context.Background(), identity(owner, "member"), c); !errors.Is(e, account.ErrInput) || repo.writes != 0 {
				t.Fatal("invalid draft persisted")
			}
		})
	}
	repo := newMemory()
	s := New(repo)
	b, _ := s.Create(context.Background(), identity(owner, "member"), populated())
	if _, e := s.Patch(context.Background(), identity(owner, "member"), b.ID, Change{}); !errors.Is(e, account.ErrInput) {
		t.Fatal("empty patch accepted")
	}
	if _, e := s.List(context.Background(), identity(owner, "member"), Filter{Page: 0, PageSize: 20}); !errors.Is(e, account.ErrInput) {
		t.Fatal("invalid page")
	}
}
