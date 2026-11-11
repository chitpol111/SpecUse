package management

import (
	"context"
	"encoding/json"
	"errors"
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
func (r *repoStub) SaveProduct(_ context.Context, actor string, p catalog.ProductDetails, create bool) (catalog.ProductDetails, error) {
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
