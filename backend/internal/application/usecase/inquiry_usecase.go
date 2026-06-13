// backend/internal/application/usecase/inquiry_usecase.go
package usecase

import (
	"context"

	inquirydom "narratives/internal/domain/inquiry"
)

// InquiryAggregate は Inquiry とその画像一覧をまとめたビューです。
//
// inquiryImage ドメインは inquiry ドメインへ統合済みのため、
// Images は Inquiry.Images から取得します。
type InquiryAggregate struct {
	Inquiry inquirydom.Inquiry     `json:"inquiry"`
	Images  []inquirydom.ImageFile `json:"images"`
}

// InquiryUsecase は Inquiry 集約を扱います。
//
// 画像は Inquiry.Images として Inquiry 集約内で管理します。
// Firebase Storage への保存・削除は frontend / application 層の責務とし、
// domain / repository では fileUrl と objectPath のメタデータのみ扱います。
type InquiryUsecase struct {
	repo inquirydom.Repository
}

// NewInquiryUsecase はユースケースを初期化します。
func NewInquiryUsecase(repo inquirydom.Repository) *InquiryUsecase {
	return &InquiryUsecase{
		repo: repo,
	}
}

// ListByCompanyID は companyID に紐づく Inquiry 一覧を返します。
func (uc *InquiryUsecase) ListByCompanyID(
	ctx context.Context,
	companyID string,
	filter inquirydom.Filter,
	sort inquirydom.Sort,
	page inquirydom.Page,
) (inquirydom.PageResult[inquirydom.Inquiry], error) {
	return uc.repo.ListByCompanyID(ctx, companyID, filter, sort, page)
}

// GetByID は Inquiry を返します。
func (uc *InquiryUsecase) GetByID(ctx context.Context, id string) (inquirydom.Inquiry, error) {
	return uc.repo.GetByID(ctx, id)
}

// Create は Inquiry を作成します。
func (uc *InquiryUsecase) Create(ctx context.Context, inq inquirydom.Inquiry) (inquirydom.Inquiry, error) {
	return uc.repo.Create(ctx, inq)
}

// Update は Inquiry を部分更新します。
//
// 画像追加・更新・削除は InquiryPatch.Images に更新後の Images 全体を渡して行います。
func (uc *InquiryUsecase) Update(
	ctx context.Context,
	id string,
	patch inquirydom.InquiryPatch,
) (inquirydom.Inquiry, error) {
	return uc.repo.Update(ctx, id, patch)
}

// Delete は Inquiry を削除します。
func (uc *InquiryUsecase) Delete(ctx context.Context, id string) error {
	return uc.repo.Delete(ctx, id)
}

// GetImages は Inquiry に紐づく画像一覧を返します。
//
// inquiryImage ドメインは廃止済みのため、別 repository へは問い合わせず、
// Inquiry.Images をそのまま返します。
func (uc *InquiryUsecase) GetImages(ctx context.Context, inquiryID string) ([]inquirydom.ImageFile, error) {
	in, err := uc.repo.GetByID(ctx, inquiryID)
	if err != nil {
		return nil, err
	}
	if len(in.Images) == 0 {
		return []inquirydom.ImageFile{}, nil
	}
	return in.Images, nil
}

// GetAggregate は Inquiry と画像一覧をまとめて返します。
//
// 画像は Inquiry.Images を正として扱います。
func (uc *InquiryUsecase) GetAggregate(ctx context.Context, id string) (InquiryAggregate, error) {
	in, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return InquiryAggregate{}, err
	}

	images := in.Images
	if images == nil {
		images = []inquirydom.ImageFile{}
	}

	return InquiryAggregate{
		Inquiry: in,
		Images:  images,
	}, nil
}
