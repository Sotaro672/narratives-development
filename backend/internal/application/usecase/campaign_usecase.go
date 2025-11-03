// backend/internal/application/usecase/campaign_usecase.go
package usecase

import (
	"context"
	"time"

	campdom "narratives/internal/domain/campaign"
	cimgdom "narratives/internal/domain/campaignImage"
	cperfdom "narratives/internal/domain/campaignPerformance"
)

// CampaignReader はキャンペーン単体取得の契約です。
type CampaignReader interface {
	GetByID(ctx context.Context, id string) (campdom.Campaign, error)
}

// CampaignImageLister は Filter ベースの画像一覧取得の契約です。
// 既存の campaignImage.Repository に合わせています。
type CampaignImageLister interface {
	List(ctx context.Context, filter cimgdom.Filter, sort cimgdom.Sort, page cimgdom.Page) (cimgdom.PageResult[cimgdom.CampaignImage], error)
}

// CampaignImageObjectSaver は GCS に配置済みオブジェクト情報から保存する契約です。
// PG 実装(CampaignImageRepositoryPG)の SaveFromBucketObject をアダプトしてください。
type CampaignImageObjectSaver interface {
	SaveFromBucketObject(
		ctx context.Context,
		id string,
		campaignID string,
		bucket string,
		objectPath string,
		width, height *int,
		fileSize *int64,
		mimeType *string,
		createdBy *string,
		now time.Time,
	) (cimgdom.CampaignImage, error)
}

// CampaignPerformanceLister はキャンペーン別のパフォーマンス一覧取得の契約です。
// 実装が異なる場合はアダプタでこの契約に合わせてください。
type CampaignPerformanceLister interface {
	ListByCampaignID(ctx context.Context, campaignID string) ([]cperfdom.CampaignPerformance, error)
}

// CampaignAggregate は統合ビューです。
type CampaignAggregate struct {
	Campaign     campdom.Campaign               `json:"campaign"`
	Images       []cimgdom.CampaignImage        `json:"images"`
	Performances []cperfdom.CampaignPerformance `json:"performances"`
}

// CampaignUsecase は campaign / campaignImage / campaignPerformance を統合して扱います。
type CampaignUsecase struct {
	campaignReader CampaignReader

	imageLister      CampaignImageLister
	imageObjectSaver CampaignImageObjectSaver

	perfLister CampaignPerformanceLister
}

// NewCampaignUsecase はユースケースを初期化します。
// 不要な依存は nil でも構いません（該当機能はスキップ/空配列返却）。
func NewCampaignUsecase(
	campaignReader CampaignReader,
	imageLister CampaignImageLister,
	perfLister CampaignPerformanceLister,
	imageObjectSaver CampaignImageObjectSaver,
) *CampaignUsecase {
	return &CampaignUsecase{
		campaignReader:   campaignReader,
		imageLister:      imageLister,
		perfLister:       perfLister,
		imageObjectSaver: imageObjectSaver,
	}
}

// GetByID はキャンペーン本体を返します。
func (uc *CampaignUsecase) GetByID(ctx context.Context, id string) (campdom.Campaign, error) {
	return uc.campaignReader.GetByID(ctx, id)
}

// ListImages はキャンペーンに紐づく画像一覧を返します。
func (uc *CampaignUsecase) ListImages(ctx context.Context, campaignID string, page, perPage int) ([]cimgdom.CampaignImage, error) {
	if uc.imageLister == nil {
		return []cimgdom.CampaignImage{}, nil
	}
	f := cimgdom.Filter{
		CampaignID: &campaignID, // campaignImage.Filter は Deleted 等を持たない
	}
	sort := cimgdom.Sort{}
	pg := cimgdom.Page{Number: page, PerPage: perPage}
	res, err := uc.imageLister.List(ctx, f, sort, pg)
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

// ListPerformances はキャンペーンのパフォーマンス一覧を返します。
func (uc *CampaignUsecase) ListPerformances(ctx context.Context, campaignID string) ([]cperfdom.CampaignPerformance, error) {
	if uc.perfLister == nil {
		return []cperfdom.CampaignPerformance{}, nil
	}
	return uc.perfLister.ListByCampaignID(ctx, campaignID)
}

// GetAggregate はキャンペーン本体 + 画像 + パフォーマンスをまとめて返します。
func (uc *CampaignUsecase) GetAggregate(ctx context.Context, id string) (CampaignAggregate, error) {
	c, err := uc.campaignReader.GetByID(ctx, id)
	if err != nil {
		return CampaignAggregate{}, err
	}

	var (
		images []cimgdom.CampaignImage
		perfs  []cperfdom.CampaignPerformance
	)

	// 画像一覧
	if uc.imageLister != nil {
		f := cimgdom.Filter{CampaignID: &id}
		res, err := uc.imageLister.List(ctx, f, cimgdom.Sort{}, cimgdom.Page{Number: 1, PerPage: 100})
		if err != nil {
			return CampaignAggregate{}, err
		}
		images = res.Items
	}

	// パフォーマンス一覧
	if uc.perfLister != nil {
		ps, err := uc.perfLister.ListByCampaignID(ctx, id)
		if err != nil {
			return CampaignAggregate{}, err
		}
		perfs = ps
	}

	return CampaignAggregate{
		Campaign:     c,
		Images:       images,
		Performances: perfs,
	}, nil
}

// SaveImageFromGCS は GCS に格納済みのオブジェクト情報から CampaignImage を保存します。
// bucket が空文字でも構いません（実装側でデフォルトを使用）。
func (uc *CampaignUsecase) SaveImageFromGCS(
	ctx context.Context,
	id string,
	campaignID string,
	bucket string,
	objectPath string,
	width, height *int,
	fileSize *int64,
	mimeType *string,
	createdBy *string,
	now time.Time,
) (cimgdom.CampaignImage, error) {
	if uc.imageObjectSaver == nil {
		return cimgdom.CampaignImage{}, ErrNotSupported("Campaign.SaveImageFromGCS")
	}
	return uc.imageObjectSaver.SaveFromBucketObject(ctx, id, campaignID, bucket, objectPath, width, height, fileSize, mimeType, createdBy, now)
}
