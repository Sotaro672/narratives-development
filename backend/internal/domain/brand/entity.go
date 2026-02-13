// backend/internal/domain/brand/entity.go
package brand

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	domcommon "narratives/internal/domain/common"
)

// TS: Brand を正として定義（status は持たず isActive のみ）
type Brand struct {
	ID            string     `json:"id"`
	CompanyID     string     `json:"companyId"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	URL           string     `json:"websiteUrl,omitempty"`
	IsActive      bool       `json:"isActive"`
	ManagerID     *string    `json:"manager,omitempty"`
	WalletAddress string     `json:"walletAddress"`
	CreatedAt     time.Time  `json:"createdAt"`
	CreatedBy     *string    `json:"createdBy,omitempty"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy     *string    `json:"updatedBy,omitempty"`
	DeletedAt     *time.Time `json:"deletedAt,omitempty"`
	DeletedBy     *string    `json:"deletedBy,omitempty"`
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
	websiteURL string,
	isActive bool,
	managerID, createdBy *string,
	createdAt time.Time,
) (Brand, error) {
	createdAt = createdAt.UTC()
	b := Brand{
		ID:            id,
		CompanyID:     companyID,
		Name:          name,
		Description:   description,
		URL:           websiteURL,
		IsActive:      isActive,
		ManagerID:     domcommon.NormalizeStringPtr(managerID),
		WalletAddress: walletAddress,
		CreatedAt:     createdAt,
		CreatedBy:     domcommon.NormalizeStringPtr(createdBy),
		UpdatedAt:     nil,
		UpdatedBy:     nil,
		DeletedAt:     nil,
		DeletedBy:     nil,
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
	return New(id, companyID, name, description, walletAddress, "", true, nil, nil, createdAt)
}

// NewFromStringTime allows createdAt as string.
func NewFromStringTime(
	id, companyID, name, description, walletAddress, createdAt string,
) (Brand, error) {
	t, err := domcommon.ParseTime(createdAt)
	if err != nil {
		return Brand{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
	}
	return NewMinimal(id, companyID, name, description, walletAddress, t)
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

	// 🔽🔽 ここを「空を許容」に変更 🔽🔽
	// Brand 作成直後は walletAddress が空でもよい。
	// SolanaBrandWalletService により後から付与される想定。
	// 形式チェックが必要になったら、「非空かつ base58 っぽい場合のみチェック」などにする。
	//
	// if b.WalletAddress == "" {
	// 	return ErrInvalidWalletAddress
	// }

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
	CompanyID     *string
	Name          *string
	Description   *string
	URL           *string
	IsActive      *bool
	ManagerID     *string
	WalletAddress *string
	UpdatedAt     *time.Time
	UpdatedBy     *string
	DeletedAt     *time.Time
	DeletedBy     *string
	CreatedBy     *string
}

// ここに SolanaWallet などの追加定義がある場合はそのまま残して OK
