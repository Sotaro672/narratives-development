// backend/internal/application/query/mall/inquiry_query.go
package mall

import (
	"context"
	"errors"
	"fmt"
	"strings"

	inquirydom "narratives/internal/domain/inquiry"
	modeldom "narratives/internal/domain/model"
	productdom "narratives/internal/domain/product"
	productblueprintdom "narratives/internal/domain/productBlueprint"
)

// InquiryQuery は mall 側の Inquiry read model を扱います。
//
// usecase は command 専用に寄せるため、mall 画面で必要な Inquiry 取得は
// この query service に集約します。
// また、Inquiry.ProductID から Product.ModelID を解決し、
// ModelVariation.GetProductBlueprintID() から productBlueprintId を解決し、
// ProductBlueprint.CompanyID を解決します。
type InquiryQuery struct {
	repo                 inquirydom.Repository
	productRepo          productdom.Repository
	modelRepo            modeldom.RepositoryPort
	productBlueprintRepo productblueprintdom.Repository
}

// NewInquiryQuery は InquiryQuery を初期化します。
func NewInquiryQuery(
	repo inquirydom.Repository,
	productRepo productdom.Repository,
	modelRepo modeldom.RepositoryPort,
	productBlueprintRepo productblueprintdom.Repository,
) *InquiryQuery {
	return &InquiryQuery{
		repo:                 repo,
		productRepo:          productRepo,
		modelRepo:            modelRepo,
		productBlueprintRepo: productBlueprintRepo,
	}
}

// InquiryDetail は mall 側の Inquiry 詳細表示用 read model です。
//
// Inquiry.ProductID から解決した modelId / productBlueprintId / companyId を含めます。
type InquiryDetail struct {
	Inquiry            inquirydom.Inquiry `json:"inquiry"`
	ModelID            string             `json:"modelId"`
	ProductBlueprintID string             `json:"productBlueprintId"`
	CompanyID          string             `json:"companyId"`
}

// GetByID は Inquiry を取得します。
//
// command 処理前の現在状態取得など、domain entity が必要な場合に使います。
func (q *InquiryQuery) GetByID(ctx context.Context, id string) (inquirydom.Inquiry, error) {
	if q == nil || q.repo == nil {
		return inquirydom.Inquiry{}, fmt.Errorf("mall inquiry query: repository is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return inquirydom.Inquiry{}, inquirydom.ErrInvalidID
	}

	return q.repo.GetByID(ctx, id)
}

// GetByIDForAvatar は avatar 所有確認込みで Inquiry を取得します。
//
// reply / close 前の現在状態確認など、domain entity が必要な command 補助で使います。
func (q *InquiryQuery) GetByIDForAvatar(
	ctx context.Context,
	id string,
	avatarID string,
) (inquirydom.Inquiry, error) {
	if q == nil || q.repo == nil {
		return inquirydom.Inquiry{}, fmt.Errorf("mall inquiry query: repository is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return inquirydom.Inquiry{}, inquirydom.ErrInvalidID
	}

	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return inquirydom.Inquiry{}, inquirydom.ErrInvalidAvatarID
	}

	inq, err := q.repo.GetByID(ctx, id)
	if err != nil {
		return inquirydom.Inquiry{}, err
	}

	if strings.TrimSpace(inq.AvatarID) != avatarID {
		return inquirydom.Inquiry{}, inquirydom.ErrInquiryForbidden
	}

	return inq, nil
}

// GetDetailByIDForAvatar は avatar 所有確認込みで Inquiry 詳細 read model を取得します。
//
// Inquiry.ProductID から product repository を使って Product.ModelID を取得し、
// model repository を使って ModelVariation.GetProductBlueprintID() を取得し、
// productBlueprint repository を使って ProductBlueprint.CompanyID を取得します。
func (q *InquiryQuery) GetDetailByIDForAvatar(
	ctx context.Context,
	id string,
	avatarID string,
) (InquiryDetail, error) {
	inq, err := q.GetByIDForAvatar(ctx, id, avatarID)
	if err != nil {
		return InquiryDetail{}, err
	}

	modelID, productBlueprintID, companyID, err := q.resolveProductModelRefByInquiryProductID(ctx, inq.ProductID)
	if err != nil {
		return InquiryDetail{}, err
	}

	return InquiryDetail{
		Inquiry:            inq,
		ModelID:            modelID,
		ProductBlueprintID: productBlueprintID,
		CompanyID:          companyID,
	}, nil
}

func (q *InquiryQuery) resolveProductModelRefByInquiryProductID(
	ctx context.Context,
	productID string,
) (modelID string, productBlueprintID string, companyID string, err error) {
	if q == nil {
		return "", "", "", fmt.Errorf("mall inquiry query: query is nil")
	}

	productID = strings.TrimSpace(productID)
	if productID == "" {
		return "", "", "", nil
	}

	if q.productRepo == nil {
		return "", "", "", fmt.Errorf("mall inquiry query: product repository is nil")
	}

	product, err := q.productRepo.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, productdom.ErrNotFound) {
			return "", "", "", nil
		}
		return "", "", "", err
	}

	modelID = strings.TrimSpace(product.ModelID)
	if modelID == "" {
		return "", "", "", nil
	}

	if q.modelRepo == nil {
		return modelID, "", "", fmt.Errorf("mall inquiry query: model repository is nil")
	}

	model, err := q.modelRepo.GetByID(ctx, modelID)
	if err != nil {
		if errors.Is(err, modeldom.ErrNotFound) {
			return modelID, "", "", nil
		}
		return modelID, "", "", err
	}

	productBlueprintID = strings.TrimSpace(model.GetProductBlueprintID())
	if productBlueprintID == "" {
		return modelID, "", "", nil
	}

	if q.productBlueprintRepo == nil {
		return modelID, productBlueprintID, "", fmt.Errorf("mall inquiry query: product blueprint repository is nil")
	}

	productBlueprint, err := q.productBlueprintRepo.GetByID(ctx, productBlueprintID)
	if err != nil {
		return modelID, productBlueprintID, "", err
	}

	return modelID, productBlueprintID, strings.TrimSpace(productBlueprint.CompanyID), nil
}
