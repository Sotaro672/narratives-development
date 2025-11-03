// backend\internal\application\usecase\discount_usecase.go
package usecase

import (
	"context"
	"strings"
	"time"

	discountdom "narratives/internal/domain/discount"
)

// DiscountRepo defines the minimal persistence port needed by DiscountUsecase.
type DiscountRepo interface {
	GetByID(ctx context.Context, id string) (discountdom.Discount, error)
	Exists(ctx context.Context, id string) (bool, error)
	Create(ctx context.Context, d discountdom.Discount) (discountdom.Discount, error)
	Save(ctx context.Context, d discountdom.Discount) (discountdom.Discount, error)
	Delete(ctx context.Context, id string) error
}

// DiscountUsecase orchestrates discount operations.
type DiscountUsecase struct {
	repo DiscountRepo
	now  func() time.Time
}

func NewDiscountUsecase(repo DiscountRepo) *DiscountUsecase {
	return &DiscountUsecase{
		repo: repo,
		now:  time.Now,
	}
}

// =======================
// Queries
// =======================

func (u *DiscountUsecase) GetByID(ctx context.Context, id string) (discountdom.Discount, error) {
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *DiscountUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// =======================
// Commands
// =======================

type CreateDiscountInput struct {
	ID           string
	ListID       string
	Items        []discountdom.DiscountItem
	Description  *string
	DiscountedBy string
	UpdatedBy    string
	// Optional: set custom clock (for tests)
	CreatedAt time.Time // if zero, now() will be used for both discountedAt/updatedAt
}

func (u *DiscountUsecase) Create(ctx context.Context, in CreateDiscountInput) (discountdom.Discount, error) {
	now := u.now().UTC()
	discAt := now
	updAt := now
	if !in.CreatedAt.IsZero() {
		discAt = in.CreatedAt.UTC()
		updAt = in.CreatedAt.UTC()
	}
	return u.repo.Create(ctx, must(discountdom.New(
		strings.TrimSpace(in.ID),
		strings.TrimSpace(in.ListID),
		in.Items,
		trimPtr(in.Description),
		strings.TrimSpace(in.DiscountedBy),
		discAt,
		updAt,
		strings.TrimSpace(in.UpdatedBy),
	)))
}

type UpdateDiscountInput struct {
	ID string

	// Items operations (compose in order)
	ReplaceItems *[]discountdom.DiscountItem
	SetItem      *struct {
		ModelNumber string
		Percent     int
	}
	RemoveModelNumber *string

	// Meta
	ListID       *string
	DiscountedBy *string
	Description  *string
	DiscountedAt *time.Time

	// Updated meta
	UpdatedBy string     // required (non-empty)
	UpdatedAt *time.Time // optional; defaults to now
}

func (u *DiscountUsecase) Update(ctx context.Context, in UpdateDiscountInput) (discountdom.Discount, error) {
	d, err := u.repo.GetByID(ctx, strings.TrimSpace(in.ID))
	if err != nil {
		return discountdom.Discount{}, err
	}

	// Items
	if in.ReplaceItems != nil {
		if err := d.ReplaceItems(*in.ReplaceItems); err != nil {
			return discountdom.Discount{}, err
		}
	}
	if in.SetItem != nil {
		if err := d.SetItem(strings.TrimSpace(in.SetItem.ModelNumber), in.SetItem.Percent); err != nil {
			return discountdom.Discount{}, err
		}
	}
	if in.RemoveModelNumber != nil {
		mn := strings.TrimSpace(*in.RemoveModelNumber)
		if mn != "" {
			d.RemoveItem(mn)
		}
	}

	// Meta
	if in.ListID != nil || in.DiscountedBy != nil {
		listID := d.ListID
		discBy := d.DiscountedBy
		if in.ListID != nil {
			listID = strings.TrimSpace(*in.ListID)
		}
		if in.DiscountedBy != nil {
			discBy = strings.TrimSpace(*in.DiscountedBy)
		}
		if err := d.UpdateMeta(listID, discBy); err != nil {
			return discountdom.Discount{}, err
		}
	}
	if in.Description != nil {
		if err := d.UpdateDescription(trimPtr(in.Description)); err != nil {
			return discountdom.Discount{}, err
		}
	}
	if in.DiscountedAt != nil {
		if err := d.SetDiscountedAt(in.DiscountedAt.UTC()); err != nil {
			return discountdom.Discount{}, err
		}
	}

	// Updated meta
	updAt := u.now().UTC()
	if in.UpdatedAt != nil && !in.UpdatedAt.IsZero() {
		updAt = in.UpdatedAt.UTC()
	}
	if err := d.SetUpdated(updAt, strings.TrimSpace(in.UpdatedBy)); err != nil {
		return discountdom.Discount{}, err
	}

	return u.repo.Save(ctx, d)
}

func (u *DiscountUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}

// =======================
// small helper
// =======================

func must(d discountdom.Discount, err error) discountdom.Discount {
	if err != nil {
		// propagate; caller already returns (Discount, error)
		panic(err)
	}
	return d
}
