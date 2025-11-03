package usecase

import (
	"context"
	"time"

	inquirydom "narratives/internal/domain/inquiry"
	imgdom "narratives/internal/domain/inquiryImage"
)

// InquiryReader は Inquiry の単体取得契約です。
type InquiryReader interface {
	GetByID(ctx context.Context, id string) (inquirydom.Inquiry, error)
}

// InquiryImageReader は InquiryID に紐づく画像集約の取得契約です。
type InquiryImageReader interface {
	GetImagesByInquiryID(ctx context.Context, inquiryID string) (*imgdom.InquiryImage, error)
}

// InquiryImageObjectSaver は GCS に配置済みオブジェクトから画像メタを保存する契約です。
type InquiryImageObjectSaver interface {
	SaveImageFromBucketObject(
		ctx context.Context,
		inquiryID string,
		fileName string,
		bucket string,
		objectPath string,
		fileSize int64,
		mimeType string,
		width, height *int,
		createdAt time.Time,
		createdBy string,
	) (*imgdom.ImageFile, error)
}

// InquiryAggregate は Inquiry とその画像一覧をまとめたビューです。
type InquiryAggregate struct {
	Inquiry inquirydom.Inquiry `json:"inquiry"`
	Images  []imgdom.ImageFile `json:"images"`
}

// InquiryUsecase は Inquiry と InquiryImage をまとめて扱います。
type InquiryUsecase struct {
	inquiryReader InquiryReader

	imageReader InquiryImageReader
	imageSaver  InquiryImageObjectSaver
}

// NewInquiryUsecase はユースケースを初期化します。
// imageReader / imageSaver は未接続なら nil でも動作します（該当機能はスキップ/エラーを返却）。
func NewInquiryUsecase(
	inquiryReader InquiryReader,
	imageReader InquiryImageReader,
	imageSaver InquiryImageObjectSaver,
) *InquiryUsecase {
	return &InquiryUsecase{
		inquiryReader: inquiryReader,
		imageReader:   imageReader,
		imageSaver:    imageSaver,
	}
}

// GetByID は Inquiry を返します。
func (uc *InquiryUsecase) GetByID(ctx context.Context, id string) (inquirydom.Inquiry, error) {
	return uc.inquiryReader.GetByID(ctx, id)
}

// GetImages は Inquiry に紐づく画像一覧を返します（存在しない場合は空配列）。
func (uc *InquiryUsecase) GetImages(ctx context.Context, inquiryID string) ([]imgdom.ImageFile, error) {
	if uc.imageReader == nil {
		return []imgdom.ImageFile{}, nil
	}
	agg, err := uc.imageReader.GetImagesByInquiryID(ctx, inquiryID)
	if err != nil {
		return nil, err
	}
	if agg == nil || len(agg.Images) == 0 {
		return []imgdom.ImageFile{}, nil
	}
	return agg.Images, nil
}

// GetAggregate は Inquiry と画像一覧をまとめて返します。
func (uc *InquiryUsecase) GetAggregate(ctx context.Context, id string) (InquiryAggregate, error) {
	in, err := uc.inquiryReader.GetByID(ctx, id)
	if err != nil {
		return InquiryAggregate{}, err
	}

	var images []imgdom.ImageFile
	if uc.imageReader != nil {
		agg, err := uc.imageReader.GetImagesByInquiryID(ctx, id)
		if err != nil {
			return InquiryAggregate{}, err
		}
		if agg != nil {
			images = agg.Images
		}
	}

	return InquiryAggregate{
		Inquiry: in,
		Images:  images,
	}, nil
}

// SaveImageFromGCS は GCS の bucket/objectPath から公開URLを構築し、画像メタを保存します。
// bucket が空なら entity 側の既定バケット（narratives_development_inquiry_image）を実装側で使用してください。
func (uc *InquiryUsecase) SaveImageFromGCS(
	ctx context.Context,
	inquiryID string,
	fileName string,
	bucket string,
	objectPath string,
	fileSize int64,
	mimeType string,
	width, height *int,
	createdAt time.Time,
	createdBy string,
) (imgdom.ImageFile, error) {
	if uc.imageSaver == nil {
		return imgdom.ImageFile{}, ErrNotSupported("Inquiry.SaveImageFromGCS")
	}
	im, err := uc.imageSaver.SaveImageFromBucketObject(
		ctx, inquiryID, fileName, bucket, objectPath,
		fileSize, mimeType, width, height, createdAt, createdBy,
	)
	if err != nil {
		return imgdom.ImageFile{}, err
	}
	return *im, nil
}
