// backend\internal\application\usecase\production_usecase.go
package usecase

import (
	"context"
	"errors"
	"time"

	resolver "narratives/internal/application/resolver"
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
	Status             string                 `json:"status,omitempty"`
}

type UpdateProductionCommand struct {
	ID         string                 `json:"id"`
	AssigneeID string                 `json:"assigneeId"`
	Models     []ModelQuantityCommand `json:"models"`
	Status     string                 `json:"status,omitempty"`
}

type ProductionModelRowDTO struct {
	ModelID      string `json:"modelId"`
	ModelNumber  string `json:"modelNumber"`
	Size         string `json:"size"`
	Color        string `json:"color"`
	RGB          *int   `json:"rgb,omitempty"`
	DisplayOrder int    `json:"displayOrder,omitempty"`
	Quantity     int    `json:"quantity"`
}

type ProductionDetailDTO struct {
	ID                 string `json:"id"`
	ProductBlueprintID string `json:"productBlueprintId"`

	BrandID   string `json:"brandId"`
	BrandName string `json:"brandName"`

	AssigneeID   string `json:"assigneeId"`
	AssigneeName string `json:"assigneeName"`

	Status string `json:"status"`

	Models        []ProductionModelRowDTO `json:"models"`
	TotalQuantity int                     `json:"totalQuantity"`

	PrintedAt     *time.Time `json:"printedAt,omitempty"`
	PrintedBy     *string    `json:"printedBy,omitempty"`
	PrintedByName string     `json:"printedByName,omitempty"`

	CreatedBy     *string   `json:"createdBy,omitempty"`
	CreatedByName string    `json:"createdByName,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`

	UpdatedBy     *string    `json:"updatedBy,omitempty"`
	UpdatedByName string     `json:"updatedByName,omitempty"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
}

type ProductionListItemDTO struct {
	productiondom.Production

	TotalQuantity int `json:"totalQuantity"`

	ProductName   string `json:"productName,omitempty"`
	BrandName     string `json:"brandName,omitempty"`
	AssigneeName  string `json:"assigneeName,omitempty"`
	CreatedByName string `json:"createdByName,omitempty"`
	UpdatedByName string `json:"updatedByName,omitempty"`
	PrintedByName string `json:"printedByName,omitempty"`
}

type ProductionRepo interface {
	productiondom.RepositoryPort
}

type ProductBlueprintService interface {
	GetBrandIDByID(ctx context.Context, blueprintID string) (string, error)
	ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)
}

type ProductionListQuery interface {
	ListProductionsByCurrentCompany(ctx context.Context) ([]productiondom.Production, error)
	ListProductionsWithAssigneeName(ctx context.Context) ([]ProductionListItemDTO, error)
}

type ProductionUsecase struct {
	repo ProductionRepo

	pbSvc ProductBlueprintService

	nameResolver *resolver.NameResolver

	listQuery ProductionListQuery

	now func() time.Time
}

func NewProductionUsecase(
	repo ProductionRepo,
	pbSvc ProductBlueprintService,
	nameResolver *resolver.NameResolver,
) *ProductionUsecase {
	return &ProductionUsecase{
		repo:         repo,
		pbSvc:        pbSvc,
		nameResolver: nameResolver,
		now:          time.Now,
	}
}

func (u *ProductionUsecase) SetListQuery(q ProductionListQuery) {
	if u == nil {
		return
	}
	u.listQuery = q
}

func (u *ProductionUsecase) Create(ctx context.Context, p productiondom.Production) (productiondom.Production, error) {
	if p.CreatedAt.IsZero() {
		p.CreatedAt = u.now().UTC()
	}

	in := productiondom.CreateProductionInput{
		ProductBlueprintID: p.ProductBlueprintID,
		AssigneeID:         p.AssigneeID,
		Models:             p.Models,
		PrintedAt:          p.PrintedAt,
		Printed:            &p.Printed,
		CreatedBy:          p.CreatedBy,
	}

	if !p.CreatedAt.IsZero() {
		t := p.CreatedAt
		in.CreatedAt = &t
	}

	created, err := u.repo.Create(ctx, in)
	if err != nil {
		return productiondom.Production{}, err
	}
	if created == nil {
		return productiondom.Production{}, productiondom.ErrNotFound
	}
	return *created, nil
}

func (u *ProductionUsecase) Save(ctx context.Context, p productiondom.Production) (productiondom.Production, error) {
	if p.CreatedAt.IsZero() {
		p.CreatedAt = u.now().UTC()
	}

	saved, err := u.repo.Save(ctx, p)
	if err != nil {
		return productiondom.Production{}, err
	}
	if saved == nil {
		return productiondom.Production{}, productiondom.ErrNotFound
	}
	return *saved, nil
}

func (u *ProductionUsecase) Update(
	ctx context.Context,
	id string,
	patch productiondom.Production,
) (productiondom.Production, error) {
	if id == "" {
		return productiondom.Production{}, productiondom.ErrInvalidID
	}

	currentPtr, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return productiondom.Production{}, err
	}
	if currentPtr == nil {
		return productiondom.Production{}, productiondom.ErrNotFound
	}

	current := *currentPtr

	if patch.AssigneeID != "" {
		current.AssigneeID = patch.AssigneeID
	}

	if len(patch.Models) > 0 {
		current.Models = patch.Models
	}

	current.Printed = patch.Printed

	if patch.PrintedAt != nil {
		t := patch.PrintedAt.UTC()
		current.PrintedAt = &t
		current.Printed = true
	}

	if patch.PrintedBy != nil {
		v := *patch.PrintedBy
		if v == "" {
			current.PrintedBy = nil
		} else {
			vCopy := v
			current.PrintedBy = &vCopy
			current.Printed = true
		}
	}

	if !current.Printed {
		current.PrintedAt = nil
		current.PrintedBy = nil
	}

	current.UpdatedAt = u.now().UTC()

	saved, err := u.repo.Save(ctx, current)
	if err != nil {
		return productiondom.Production{}, err
	}
	if saved == nil {
		return productiondom.Production{}, productiondom.ErrNotFound
	}

	return *saved, nil
}

func (u *ProductionUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, id)
}

func (u *ProductionUsecase) GetByID(ctx context.Context, id string) (productiondom.Production, error) {
	p, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return productiondom.Production{}, err
	}
	if p == nil {
		return productiondom.Production{}, productiondom.ErrNotFound
	}
	return *p, nil
}

func (u *ProductionUsecase) Exists(ctx context.Context, id string) (bool, error) {
	_, err := u.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, productiondom.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (u *ProductionUsecase) List(ctx context.Context) ([]productiondom.Production, error) {
	if u.listQuery == nil {
		return nil, errors.New("internal: ProductionUsecase.listQuery is not configured")
	}
	return u.listQuery.ListProductionsByCurrentCompany(ctx)
}

func (u *ProductionUsecase) ListWithAssigneeName(ctx context.Context) ([]ProductionListItemDTO, error) {
	if u.listQuery == nil {
		return nil, errors.New("internal: ProductionUsecase.listQuery is not configured")
	}
	return u.listQuery.ListProductionsWithAssigneeName(ctx)
}
