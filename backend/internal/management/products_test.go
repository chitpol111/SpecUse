package management

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"specuse/backend/internal/account"
	"specuse/backend/internal/build"
	"specuse/backend/internal/catalog"
)

type repoStub struct {
	snapshot      build.Snapshot
	calls, writes int
	actor         string
}

func (r *repoStub) LoadSnapshot(context.Context) (build.Snapshot, error) {
	r.calls++
	return r.snapshot, nil
}
func (r *repoStub) LoadSnapshotForWrite(ctx context.Context) (build.Snapshot, map[string]json.RawMessage, error) {
	snapshot, err := r.LoadSnapshot(ctx)
	expected := make(map[string]json.RawMessage)
	for _, c := range snapshot.Products {
		expected[c.Product.ID], _ = json.Marshal(c.Product)
	}
	return snapshot, expected, err
}
func (r *repoStub) SaveProduct(_ context.Context, actor string, p catalog.ProductDetails, create bool, expected json.RawMessage) (catalog.ProductDetails, error) {
	r.writes++
	r.actor = actor
	return p, nil
}
func (r *repoStub) DeactivateProduct(_ context.Context, actor, id string) error {
	r.writes++
	r.actor = actor
	return nil
}
func ptr[T any](v T) *T { return &v }

const actorID = "11111111-1111-4111-8111-111111111111"
const productID = "22222222-2222-4222-8222-222222222222"

func TestAdminCRUDAndInactiveVisibility(t *testing.T) {
	repo := &repoStub{snapshot: build.Snapshot{Products: []build.Component{{Product: catalog.Product{ID: productID, Name: "Demo", Category: "case", Brand: "Demo", PriceTHB: 100, IsActive: false}}}}}
	svc := New(repo)
	admin := account.Identity{ID: actorID, Profile: account.Profile{Role: "admin"}}
	page, err := svc.List(context.Background(), admin, catalog.Filter{Page: 1, PageSize: 20})
	if err != nil || page.Total != 1 || page.Items[0].IsActive || repo.snapshot.Products[0].Product.IsActive {
		t.Fatal("inactive list/source mutation")
	}
	created, err := svc.Save(context.Background(), admin, "", Change{Name: ptr("Case"), Category: ptr("case"), Brand: ptr("Demo"), Price: ptr(int64(200)), Active: ptr(true)}, true)
	if err != nil || !catalog.ValidUUID(created.ID) || created.ID == productID || repo.actor != actorID || created.PriceTHB != 200 {
		t.Fatalf("create %+v %v", created, err)
	}
	patched, err := svc.Save(context.Background(), admin, productID, Change{Price: ptr(int64(300))}, false)
	if err != nil || patched.PriceTHB != 300 || patched.IsActive {
		t.Fatal("sparse patch changed other data")
	}
	if err := svc.Delete(context.Background(), admin, productID); err != nil || repo.writes != 3 {
		t.Fatal("delete not passed to repository")
	}
}
func TestMemberCannotReadOrMutateAdminData(t *testing.T) {
	repo := &repoStub{}
	svc := New(repo)
	member := account.Identity{ID: actorID, Profile: account.Profile{Role: "member"}}
	if _, err := svc.List(context.Background(), member, catalog.Filter{Page: 1, PageSize: 20}); !errors.Is(err, account.ErrForbidden) {
		t.Fatal("member list allowed")
	}
	if _, err := svc.Save(context.Background(), member, "", Change{}, true); !errors.Is(err, account.ErrForbidden) {
		t.Fatal("member write allowed")
	}
	if err := svc.Delete(context.Background(), member, productID); !errors.Is(err, account.ErrForbidden) {
		t.Fatal("member delete allowed")
	}
	if repo.calls != 0 || repo.writes != 0 {
		t.Fatal("unauthorized request reached repository")
	}
}
func TestAdminInputAndSyntheticEvidenceRejected(t *testing.T) {
	admin := account.Identity{ID: actorID, Profile: account.Profile{Role: "admin"}}
	repo := &repoStub{}
	svc := New(repo)
	for _, change := range []Change{
		{}, {Name: ptr(""), Category: ptr("case"), Brand: ptr("Demo"), Price: ptr(int64(100)), Active: ptr(true)},
		{Name: ptr("Case"), Category: ptr("case"), Brand: ptr("Demo"), Price: ptr(int64(-1)), Active: ptr(true)},
		{Name: ptr("CPU"), Category: ptr("cpu"), Brand: ptr("Demo"), Price: ptr(int64(100)), Active: ptr(true)},
		{Name: ptr("RAM"), Category: ptr("ram"), Brand: ptr("Demo"), Price: ptr(int64(100)), Active: ptr(true), RAMModelID: ptr(productID), Specs: json.RawMessage(`{"ram_type":"DDR5","modules_per_package":1}`)},
		{Name: ptr("GPU"), Category: ptr("gpu"), Brand: ptr("Demo"), Price: ptr(int64(100)), Active: ptr(true), Specs: json.RawMessage(`{"sources":[{"source_id":"synthetic-test","manufacturer":"SYNTHETIC","title":"unit","url":"https://fixture.invalid","section":"test","checked_at":"2026-11-11T00:00:00Z"}]}`)},
	} {
		if _, err := svc.Save(context.Background(), admin, "", change, true); !errors.Is(err, account.ErrInput) {
			t.Fatalf("invalid product accepted %v", err)
		}
	}
	if repo.writes != 0 {
		t.Fatal("invalid mutation reached DB")
	}
	if _, err := svc.Save(context.Background(), admin, productID, Change{Category: ptr("gpu")}, false); !errors.Is(err, account.ErrInput) {
		t.Fatal("category changed")
	}
}

type concurrentRepo struct {
	mu       sync.Mutex
	loaded   int
	bothRead chan struct{}
	product  catalog.ProductDetails
}

func (r *concurrentRepo) LoadSnapshot(context.Context) (build.Snapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return build.Snapshot{Products: []build.Component{{Product: r.product.Product}}}, nil
}
func (r *concurrentRepo) LoadSnapshotForWrite(context.Context) (build.Snapshot, map[string]json.RawMessage, error) {
	r.mu.Lock()
	snapshot := build.Snapshot{Products: []build.Component{{Product: r.product.Product}}}
	raw, err := json.Marshal(r.product)
	r.loaded++
	if r.loaded == 2 {
		close(r.bothRead)
	}
	r.mu.Unlock()
	if err != nil {
		return build.Snapshot{}, nil, err
	}
	<-r.bothRead
	return snapshot, map[string]json.RawMessage{productID: raw}, nil
}
func (r *concurrentRepo) SaveProduct(_ context.Context, _ string, p catalog.ProductDetails, _ bool, expected json.RawMessage) (catalog.ProductDetails, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, err := json.Marshal(r.product)
	if err != nil {
		return catalog.ProductDetails{}, err
	}
	if !bytes.Equal(current, expected) {
		return catalog.ProductDetails{}, account.ErrConflict
	}
	r.product = p
	return p, nil
}
func (r *concurrentRepo) DeactivateProduct(context.Context, string, string) error { return nil }

func TestConcurrentSparseEditsConflictInsteadOfLosingUpdates(t *testing.T) {
	repo := &concurrentRepo{
		bothRead: make(chan struct{}),
		product: catalog.ProductDetails{Product: catalog.Product{
			ID: productID, Name: "Old", Category: "case", Brand: "Demo", PriceTHB: 100, IsActive: true,
		}},
	}
	svc := New(repo)
	admin := account.Identity{ID: actorID, Profile: account.Profile{Role: "admin"}}
	errs := make(chan error, 2)
	go func() {
		_, err := svc.Save(context.Background(), admin, productID, Change{Name: ptr("New")}, false)
		errs <- err
	}()
	go func() {
		_, err := svc.Save(context.Background(), admin, productID, Change{Price: ptr(int64(200))}, false)
		errs <- err
	}()
	a, b := <-errs, <-errs
	if !((a == nil && errors.Is(b, account.ErrConflict)) || (b == nil && errors.Is(a, account.ErrConflict))) {
		t.Fatalf("want one success and one conflict, got %v and %v", a, b)
	}
	if (repo.product.Name == "New") == (repo.product.PriceTHB == 200) {
		t.Fatalf("expected exactly one change to persist: %+v", repo.product.Product)
	}
	var retry Change
	if repo.product.Name != "New" {
		retry.Name = ptr("New")
	} else {
		retry.Price = ptr(int64(200))
	}
	if _, err := svc.Save(context.Background(), admin, productID, retry, false); err != nil {
		t.Fatalf("retry after reloading: %v", err)
	}
	if repo.product.Name != "New" || repo.product.PriceTHB != 200 {
		t.Fatalf("retry lost a change: %+v", repo.product.Product)
	}
}
