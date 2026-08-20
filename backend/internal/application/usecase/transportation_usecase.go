// backend/internal/application/usecase/transportation_usecase.go
package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	transportationdom "narratives/internal/domain/transportation"
)

// TransportationRepoはDomain層のRepositoryPortをそのまま使用します。
// Application層では同じinterfaceを再定義しません。
type TransportationRepo = transportationdom.RepositoryPort

// CreateTransportationFeeSettingInputは配送料金設定の新規作成入力です。
// ID、CompanyID、CreatedAt、UpdatedAtはUsecase側で設定します。
// Nameは必須です。
// PrefectureRatesは47都道府県すべて必須です。
// IslandRatesは任意で、設定された場合は都道府県料金より優先されます。
type CreateTransportationFeeSettingInput struct {
	Name            string
	PrefectureRates []transportationdom.PrefectureRate
	IslandRates     []transportationdom.IslandRate
}

// UpdateTransportationFeeSettingInputは配送料金設定の更新入力です。
// ID、CompanyID、CreatedAtは変更しません。
// Name、PrefectureRates、IslandRatesを完全な現在値として受け取ります。
type UpdateTransportationFeeSettingInput struct {
	Name            string
	PrefectureRates []transportationdom.PrefectureRate
	IslandRates     []transportationdom.IslandRate
}

// TransportationUsecaseはcompanyに紐づく配送料金設定を制御します。
// 1 companyは複数のTransportationFeeSettingを保持できます。
type TransportationUsecase struct {
	repo TransportationRepo

	// テスト時に差し替え可能な依存です。
	newDocID func() string
	now      func() time.Time
}

// NewTransportationUsecaseはTransportationUsecaseを生成します。
func NewTransportationUsecase(repo TransportationRepo) *TransportationUsecase {
	return &TransportationUsecase{
		repo:     repo,
		newDocID: uuid.NewString,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (u *TransportationUsecase) ensureRepo() error {
	if u == nil {
		return errors.New("transportation usecase is nil")
	}
	if u.repo == nil {
		return errors.New("transportation repo not configured")
	}
	if u.newDocID == nil {
		return errors.New("transportation id generator not configured")
	}
	if u.now == nil {
		return errors.New("transportation now not configured")
	}
	return nil
}

func validateTransportationCompanyID(companyID string) (string, error) {
	if companyID == "" || len([]rune(companyID)) > transportationdom.MaxCompanyIDLength {
		return "", transportationdom.ErrInvalidCompanyID
	}
	return companyID, nil
}

func validateTransportationID(id string) (string, error) {
	if id == "" || len([]rune(id)) > transportationdom.MaxTransportationIDLength {
		return "", transportationdom.ErrInvalidID
	}
	return id, nil
}

// ListByCompanyIDは認証済みcompanyが所有する配送料金設定一覧を取得します。
// 設定が0件の場合は空sliceを返します。
func (u *TransportationUsecase) ListByCompanyID(
	ctx context.Context,
	companyID string,
) ([]transportationdom.TransportationFeeSetting, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	validCompanyID, err := validateTransportationCompanyID(companyID)
	if err != nil {
		return nil, err
	}

	settings, err := u.repo.ListByCompanyID(ctx, validCompanyID)
	if err != nil {
		return nil, err
	}

	for _, setting := range settings {
		if setting.CompanyID != validCompanyID {
			return nil, errors.New("transportation repository returned setting owned by another company")
		}
	}

	return settings, nil
}

// GetByIDは指定Transportation IDの配送料金設定を取得します。
// 対象が存在しない場合、または別companyが所有する場合はErrNotFoundを返します。
func (u *TransportationUsecase) GetByID(
	ctx context.Context,
	companyID string,
	transportationID string,
) (*transportationdom.TransportationFeeSetting, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	validCompanyID, err := validateTransportationCompanyID(companyID)
	if err != nil {
		return nil, err
	}

	validTransportationID, err := validateTransportationID(transportationID)
	if err != nil {
		return nil, err
	}

	setting, err := u.repo.GetByID(ctx, validTransportationID)
	if err != nil {
		return nil, err
	}
	if setting == nil {
		return nil, transportationdom.ErrNotFound
	}
	if setting.CompanyID != validCompanyID {
		return nil, transportationdom.ErrNotFound
	}

	return setting, nil
}

// Createはcompanyに新しい配送料金設定を作成します。
// Transportation IDはUsecaseがUUIDで採番します。
// 同一companyに複数のTransportationFeeSettingを作成できます。
func (u *TransportationUsecase) Create(
	ctx context.Context,
	companyID string,
	in CreateTransportationFeeSettingInput,
) (*transportationdom.TransportationFeeSetting, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	validCompanyID, err := validateTransportationCompanyID(companyID)
	if err != nil {
		return nil, err
	}

	transportationID := u.newDocID()
	validTransportationID, err := validateTransportationID(transportationID)
	if err != nil {
		return nil, err
	}

	now := u.now().UTC()

	setting, err := transportationdom.NewWithNow(
		validTransportationID,
		validCompanyID,
		in.Name,
		in.PrefectureRates,
		in.IslandRates,
		now,
	)
	if err != nil {
		return nil, err
	}

	return u.repo.Create(ctx, setting)
}

// Updateは指定Transportation IDの既存配送料金設定を更新します。
// Updateはupsertではありません。
// ID、CompanyID、CreatedAtは既存値を維持し、Name、料金設定、UpdatedAtを更新します。
// 別companyが所有するTransportation IDを指定した場合はErrNotFoundを返します。
func (u *TransportationUsecase) Update(
	ctx context.Context,
	companyID string,
	transportationID string,
	in UpdateTransportationFeeSettingInput,
) (*transportationdom.TransportationFeeSetting, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	current, err := u.GetByID(ctx, companyID, transportationID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, transportationdom.ErrNotFound
	}

	now := u.now().UTC()

	if err := current.Update(
		in.Name,
		in.PrefectureRates,
		in.IslandRates,
		now,
	); err != nil {
		return nil, err
	}

	return u.repo.Update(ctx, *current)
}

// ResolveFeeは指定Transportation IDの料金設定から配送先に適用する送料を解決します。
// islandCodeに一致するIslandRateが存在する場合はIslandRateを優先し、
// 存在しない場合はPrefectureRateへfallbackします。
// islandCodeが空の場合はPrefectureRateを使用します。
// 別companyが所有するTransportation IDを指定した場合はErrNotFoundを返します。
func (u *TransportationUsecase) ResolveFee(
	ctx context.Context,
	companyID string,
	transportationID string,
	prefectureCode string,
	islandCode string,
) (int64, error) {
	if err := u.ensureRepo(); err != nil {
		return 0, err
	}

	code, err := transportationdom.ParsePrefectureCode(prefectureCode)
	if err != nil {
		return 0, err
	}

	setting, err := u.GetByID(ctx, companyID, transportationID)
	if err != nil {
		return 0, err
	}
	if setting == nil {
		return 0, transportationdom.ErrNotFound
	}

	return setting.ResolveFee(code, islandCode)
}
