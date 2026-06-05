package usecase

import (
	"context"
	"time"

	productiondom "narratives/internal/domain/production"
)

type ModelQuantityCommand struct {
	ModelID  string `json:"modelId"`
	Quantity int    `json:"quantity"`
}

type CreateProductionCommand struct {
	ProductBlueprintID string                 `json:"productBlueprintId"`
	AssigneeID         string                 `json:"assigneeId"`
	Models             []ModelQuantityCommand `json:"models"`

	// Status は廃止。Printed(boolean) に統一。
	Printed   *bool      `json:"printed,omitempty"`
	PrintedAt *time.Time `json:"printedAt,omitempty"`

	CreatedBy *string `json:"createdBy,omitempty"`
}

type UpdateProductionCommand struct {
	ID         string                 `json:"id"`
	AssigneeID string                 `json:"assigneeId"`
	Models     []ModelQuantityCommand `json:"models"`

	// Status は廃止。Printed(boolean) に統一。
	Printed   *bool      `json:"printed,omitempty"`
	PrintedAt *time.Time `json:"printedAt,omitempty"`
	PrintedBy *string    `json:"printedBy,omitempty"`

	UpdatedBy *string `json:"updatedBy,omitempty"`
}

type ProductionRepo interface {
	productiondom.RepositoryPort
}

type ProductionUsecase struct {
	repo ProductionRepo
	now  func() time.Time
}

func NewProductionUsecase(
	repo ProductionRepo,
) *ProductionUsecase {
	return &ProductionUsecase{
		repo: repo,
		now:  time.Now,
	}
}

func (u *ProductionUsecase) Create(
	ctx context.Context,
	cmd CreateProductionCommand,
) (productiondom.Production, error) {
	now := u.now().UTC()

	printed := false
	if cmd.Printed != nil {
		printed = *cmd.Printed
	}

	var printedAt *time.Time
	if cmd.PrintedAt != nil && !cmd.PrintedAt.IsZero() {
		t := cmd.PrintedAt.UTC()
		printedAt = &t
		printed = true
	}

	if printed && printedAt == nil {
		t := now
		printedAt = &t
	}

	if !printed {
		printedAt = nil
	}

	p, err := productiondom.NewForCreate(
		cmd.ProductBlueprintID,
		cmd.AssigneeID,
		modelQuantityCommandsToDomain(cmd.Models),
		printed,
		printedAt,
		cmd.CreatedBy,
		now,
	)
	if err != nil {
		return productiondom.Production{}, err
	}

	in := productiondom.CreateProductionInput{
		ProductBlueprintID: p.ProductBlueprintID,
		AssigneeID:         p.AssigneeID,
		Models:             p.Models,
		Printed:            &p.Printed,
		PrintedAt:          p.PrintedAt,
		CreatedBy:          p.CreatedBy,
		CreatedAt:          &p.CreatedAt,
	}

	created, err := u.repo.Create(ctx, in)
	if err != nil {
		return productiondom.Production{}, err
	}
	if created == nil {
		return productiondom.Production{}, productiondom.ErrNotFound
	}

	if err := created.Validate(); err != nil {
		return productiondom.Production{}, err
	}

	return *created, nil
}

func (u *ProductionUsecase) Update(
	ctx context.Context,
	cmd UpdateProductionCommand,
) (productiondom.Production, error) {
	currentPtr, err := u.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		return productiondom.Production{}, err
	}
	if currentPtr == nil {
		return productiondom.Production{}, productiondom.ErrNotFound
	}

	current := *currentPtr

	if err := current.ApplyUpdate(
		cmd.AssigneeID,
		modelQuantityCommandsToDomain(cmd.Models),
		cmd.Printed,
		cmd.PrintedAt,
		cmd.PrintedBy,
		cmd.UpdatedBy,
		u.now().UTC(),
	); err != nil {
		return productiondom.Production{}, err
	}

	updated, err := u.repo.Update(ctx, current)
	if err != nil {
		return productiondom.Production{}, err
	}
	if updated == nil {
		return productiondom.Production{}, productiondom.ErrNotFound
	}

	if err := updated.Validate(); err != nil {
		return productiondom.Production{}, err
	}

	return *updated, nil
}

func (u *ProductionUsecase) Delete(ctx context.Context, id string) error {
	if id == "" {
		return productiondom.ErrInvalidID
	}
	return u.repo.Delete(ctx, id)
}

func modelQuantityCommandsToDomain(
	models []ModelQuantityCommand,
) []productiondom.ModelQuantity {
	out := make([]productiondom.ModelQuantity, 0, len(models))
	for _, m := range models {
		out = append(out, productiondom.ModelQuantity{
			ModelID:  m.ModelID,
			Quantity: m.Quantity,
		})
	}
	return out
}
