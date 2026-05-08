// backend/internal/domain/brand/entity.go
package brand

import (
	"errors"
	"net/url"
	"time"
)

type Brand struct {
	ID                   string     `json:"id"`
	CompanyID            string     `json:"companyId"`
	Name                 string     `json:"name"`
	Description          string     `json:"description"`
	URL                  string     `json:"websiteUrl,omitempty"`
	BrandIcon            string     `json:"brandIcon,omitempty"`
	BrandBackgroundImage string     `json:"brandBackgroundImage,omitempty"`
	IsActive             bool       `json:"isActive"`
	ManagerID            *string    `json:"manager,omitempty"`
	WalletAddress        string     `json:"walletAddress"`
	CreatedAt            time.Time  `json:"createdAt"`
	CreatedBy            *string    `json:"createdBy,omitempty"`
	UpdatedAt            *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy            *string    `json:"updatedBy,omitempty"`
	DeletedAt            *time.Time `json:"deletedAt,omitempty"`
	DeletedBy            *string    `json:"deletedBy,omitempty"`
}

// Errors
var (
	ErrInvalidID            = errors.New("brand: invalid id")
	ErrInvalidCompanyID     = errors.New("brand: invalid companyId")
	ErrInvalidName          = errors.New("brand: invalid name")
	ErrInvalidDescription   = errors.New("brand: invalid description")
	ErrInvalidURL           = errors.New("brand: invalid url")
	ErrInvalidWalletAddress = errors.New("brand: invalid walletAddress")
	ErrInvalidCreatedAt     = errors.New("brand: invalid createdAt")
	ErrInvalidUpdatedAt     = errors.New("brand: invalid updatedAt")
)

// New constructs a Brand aligned to TS Brand.
func New(
	id, companyID, name, description, walletAddress string,
	websiteURL, brandIcon, brandBackgroundImage string,
	isActive bool,
	managerID, createdBy *string,
	createdAt time.Time,
) (Brand, error) {
	createdAt = createdAt.UTC()
	b := Brand{
		ID:                   id,
		CompanyID:            companyID,
		Name:                 name,
		Description:          description,
		URL:                  websiteURL,
		BrandIcon:            brandIcon,
		BrandBackgroundImage: brandBackgroundImage,
		IsActive:             isActive,
		ManagerID:            managerID,
		WalletAddress:        walletAddress,
		CreatedAt:            createdAt,
		CreatedBy:            createdBy,
		UpdatedAt:            nil,
		UpdatedBy:            nil,
		DeletedAt:            nil,
		DeletedBy:            nil,
	}

	if err := b.validate(); err != nil {
		return Brand{}, err
	}
	return b, nil
}

// NewMinimal constructs with only required fields.
func NewMinimal(
	id, companyID, name, description, walletAddress string,
	createdAt time.Time,
) (Brand, error) {
	return New(id, companyID, name, description, walletAddress, "", "", "", true, nil, nil, createdAt)
}

func (b Brand) validate() error {

	// ★ 新規作成時は ID="" を許容する
	// if b.ID == "" { return ErrInvalidID }

	if b.CompanyID == "" {
		return ErrInvalidCompanyID
	}
	if b.Name == "" {
		return ErrInvalidName
	}

	// Description は空でも OK
	// if b.Description == "" { return ErrInvalidDescription }

	if b.URL != "" && !isValidURL(b.URL) {
		return ErrInvalidURL
	}

	// Brand 作成直後は walletAddress が空でもよい。
	// SolanaBrandWalletService により後から付与される想定。
	// if b.WalletAddress == "" {
	// 	return ErrInvalidWalletAddress
	// }

	if b.ManagerID != nil && *b.ManagerID == "" {
		return ErrInvalidID
	}
	if b.CreatedBy != nil && *b.CreatedBy == "" {
		return ErrInvalidID
	}
	if b.UpdatedBy != nil && *b.UpdatedBy == "" {
		return ErrInvalidID
	}
	if b.DeletedBy != nil && *b.DeletedBy == "" {
		return ErrInvalidID
	}

	if b.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if b.UpdatedAt != nil && b.UpdatedAt.Before(b.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if b.DeletedAt != nil && b.UpdatedAt != nil && b.DeletedAt.Before(*b.UpdatedAt) {
		return ErrInvalidUpdatedAt
	}
	return nil
}

// ===============================
// Utility Functions
// ===============================

// URL validator
func isValidURL(s string) bool {
	u, err := url.ParseRequestURI(s)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// Patch struct
type BrandPatch struct {
	CompanyID            *string
	Name                 *string
	Description          *string
	URL                  *string
	BrandIcon            *string
	BrandBackgroundImage *string
	IsActive             *bool
	ManagerID            *string
	WalletAddress        *string
	UpdatedAt            *time.Time
	UpdatedBy            *string
	DeletedAt            *time.Time
	DeletedBy            *string
	CreatedBy            *string
}
