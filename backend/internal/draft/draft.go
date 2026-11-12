// Package draft stores member build requirements, not prices or validation results.
package draft

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"specuse/backend/internal/account"
	"specuse/backend/internal/catalog"
)

var ErrPublicDisabled = errors.New("public builds are not enabled")

type ItemInput struct {
	ProductID     string `json:"product_id"`
	Quantity      int64  `json:"quantity"`
	OwnedQuantity int64  `json:"owned_quantity"`
}
type RAMInput struct {
	RAMModelID      string `json:"ram_model_id"`
	RequiredModules int64  `json:"required_modules"`
	OwnedModules    int64  `json:"owned_modules"`
}

// Nil fields are omitted updates; pointers to empty arrays explicitly clear them.
type Change struct {
	Name        *string      `json:"name,omitempty"`
	Description *string      `json:"description,omitempty"`
	Visibility  *string      `json:"visibility,omitempty"`
	Items       *[]ItemInput `json:"items,omitempty"`
	RAMItems    *[]RAMInput  `json:"ram_items,omitempty"`
}
type Summary struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Visibility  string    `json:"visibility"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type ProductSummary struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	IsActive bool   `json:"is_active"`
}
type RAMModelSummary struct {
	RAMType string `json:"ram_type"`
}
type Item struct {
	ItemInput
	ProductSummary ProductSummary `json:"product_summary"`
}
type RAMItem struct {
	RAMInput
	RAMModelSummary    RAMModelSummary `json:"ram_model_summary"`
	ActivePackageCount int64           `json:"-"`
}
type AvailabilityIssue struct {
	Code       string  `json:"code"`
	Message    string  `json:"message"`
	ProductID  *string `json:"product_id"`
	RAMModelID *string `json:"ram_model_id"`
}
type Availability struct {
	ReadyForValidation bool                `json:"ready_for_validation"`
	Issues             []AvailabilityIssue `json:"issues"`
}
type Build struct {
	Summary
	Items        []Item       `json:"items"`
	RAMItems     []RAMItem    `json:"ram_items"`
	Availability Availability `json:"availability"`
}
type Filter struct{ Page, PageSize int }
type Page struct {
	Items    []Summary `json:"items"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
	Total    int64     `json:"total"`
}
type Repository interface {
	ListDrafts(context.Context, string, Filter) (Page, error)
	GetDraft(context.Context, string, string) (Build, error)
	CreateDraft(context.Context, string, Change) (Build, error)
	PatchDraft(context.Context, string, string, Change) (Build, error)
	DeleteDraft(context.Context, string, string) error
}
type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }

func actorID(actor account.Identity) (string, error) {
	if !catalog.ValidUUID(actor.ID) || actor.Profile.ID != actor.ID || (actor.Profile.Role != "member" && actor.Profile.Role != "admin") {
		return "", account.ErrForbidden
	}
	return strings.ToLower(actor.ID), nil
}
func cleanText(value string, min, max int) (string, error) {
	value = strings.TrimSpace(value)
	if !utf8.ValidString(value) || strings.ContainsRune(value, 0) || utf8.RuneCountInString(value) < min || utf8.RuneCountInString(value) > max {
		return "", account.ErrInput
	}
	return value, nil
}
func normalize(c Change, create bool) (Change, error) {
	if create {
		if c.Name == nil || c.Items == nil || c.RAMItems == nil {
			return c, account.ErrInput
		}
		if c.Description == nil {
			v := ""
			c.Description = &v
		}
		if c.Visibility == nil {
			v := "private"
			c.Visibility = &v
		}
	} else if c.Name == nil && c.Description == nil && c.Visibility == nil && c.Items == nil && c.RAMItems == nil {
		return c, account.ErrInput
	}
	if c.Name != nil {
		v, e := cleanText(*c.Name, 1, 100)
		if e != nil {
			return c, e
		}
		c.Name = &v
	}
	if c.Description != nil {
		v := *c.Description
		if !utf8.ValidString(v) || strings.ContainsRune(v, 0) || utf8.RuneCountInString(v) > 2000 {
			return c, account.ErrInput
		}
		c.Description = &v
	}
	if c.Visibility != nil {
		if *c.Visibility == "public" {
			return c, ErrPublicDisabled
		}
		if *c.Visibility != "private" {
			return c, account.ErrInput
		}
	}
	if c.Items != nil {
		rows := make([]ItemInput, 0, len(*c.Items))
		seen := map[string]bool{}
		for _, item := range *c.Items {
			item.ProductID = strings.ToLower(item.ProductID)
			if !catalog.ValidUUID(item.ProductID) || item.Quantity < 1 || item.OwnedQuantity < 0 || item.OwnedQuantity > item.Quantity || seen[item.ProductID] {
				return c, account.ErrInput
			}
			seen[item.ProductID] = true
			rows = append(rows, item)
		}
		c.Items = &rows
	}
	if c.RAMItems != nil {
		rows := make([]RAMInput, 0, len(*c.RAMItems))
		seen := map[string]bool{}
		for _, item := range *c.RAMItems {
			item.RAMModelID = strings.ToLower(item.RAMModelID)
			if !catalog.ValidUUID(item.RAMModelID) || item.RequiredModules < 1 || item.OwnedModules < 0 || item.OwnedModules > item.RequiredModules || seen[item.RAMModelID] {
				return c, account.ErrInput
			}
			seen[item.RAMModelID] = true
			rows = append(rows, item)
		}
		c.RAMItems = &rows
	}
	return c, nil
}

// EvaluateAvailability checks catalog readiness only. It does not calculate a
// price or claim compatibility/completeness; the existing validate API does that.
func EvaluateAvailability(b Build) Build {
	if b.Items == nil {
		b.Items = []Item{}
	}
	if b.RAMItems == nil {
		b.RAMItems = []RAMItem{}
	}
	b.Availability = Availability{ReadyForValidation: true, Issues: []AvailabilityIssue{}}
	for _, item := range b.Items {
		if !item.ProductSummary.IsActive {
			id := item.ProductID
			b.Availability.Issues = append(b.Availability.Issues, AvailabilityIssue{Code: "PRODUCT_INACTIVE", Message: "สินค้าถูกปิดใช้งาน กรุณาเปลี่ยนหรือนำรายการออกก่อนตรวจราคา", ProductID: &id})
		}
	}
	for _, item := range b.RAMItems {
		if item.RequiredModules > item.OwnedModules && item.ActivePackageCount == 0 {
			id := item.RAMModelID
			b.Availability.Issues = append(b.Availability.Issues, AvailabilityIssue{Code: "RAM_PACKAGES_UNAVAILABLE", Message: "ไม่มีแพ็ก RAM ที่ active สำหรับจำนวนที่ต้องซื้อเพิ่ม", RAMModelID: &id})
		}
	}
	b.Availability.ReadyForValidation = len(b.Availability.Issues) == 0
	return b
}
func (s *Service) List(ctx context.Context, actor account.Identity, f Filter) (Page, error) {
	id, e := actorID(actor)
	if e != nil {
		return Page{}, e
	}
	if catalog.ValidateFilter(catalog.Filter{Page: f.Page, PageSize: f.PageSize}) != nil {
		return Page{}, account.ErrInput
	}
	p, e := s.repo.ListDrafts(ctx, id, f)
	if p.Items == nil {
		p.Items = []Summary{}
	}
	return p, e
}
func (s *Service) Get(ctx context.Context, actor account.Identity, id string) (Build, error) {
	owner, e := actorID(actor)
	if e != nil {
		return Build{}, e
	}
	if !catalog.ValidUUID(id) {
		return Build{}, account.ErrInput
	}
	b, e := s.repo.GetDraft(ctx, owner, strings.ToLower(id))
	if e != nil {
		return Build{}, e
	}
	return EvaluateAvailability(b), nil
}
func (s *Service) Create(ctx context.Context, actor account.Identity, c Change) (Build, error) {
	owner, e := actorID(actor)
	if e != nil {
		return Build{}, e
	}
	c, e = normalize(c, true)
	if e != nil {
		return Build{}, e
	}
	b, e := s.repo.CreateDraft(ctx, owner, c)
	if e != nil {
		return Build{}, e
	}
	return EvaluateAvailability(b), nil
}
func (s *Service) Patch(ctx context.Context, actor account.Identity, id string, c Change) (Build, error) {
	owner, e := actorID(actor)
	if e != nil {
		return Build{}, e
	}
	if !catalog.ValidUUID(id) {
		return Build{}, account.ErrInput
	}
	id = strings.ToLower(id)
	// Ownership is checked before revealing validation details for this build.
	if _, e = s.repo.GetDraft(ctx, owner, id); e != nil {
		return Build{}, e
	}
	c, e = normalize(c, false)
	if e != nil {
		return Build{}, e
	}
	b, e := s.repo.PatchDraft(ctx, owner, id, c)
	if e != nil {
		return Build{}, e
	}
	return EvaluateAvailability(b), nil
}
func (s *Service) Delete(ctx context.Context, actor account.Identity, id string) error {
	owner, e := actorID(actor)
	if e != nil {
		return e
	}
	if !catalog.ValidUUID(id) {
		return account.ErrInput
	}
	return s.repo.DeleteDraft(ctx, owner, strings.ToLower(id))
}
