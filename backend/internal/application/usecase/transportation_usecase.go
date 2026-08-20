// backend/internal/application/usecase/transportation_usecase.go
package usecase

import (
	"context"
	"errors"
	"time"

	transportationdom "narratives/internal/domain/transportation"
)

// TransportationRepoはDomain層のRepositoryPortをそのまま使用します。
// Application層では同じinterfaceを再定義しません。
type TransportationRepo = transportationdom.RepositoryPort

// CreateTransportationFeeSettingInputは配送料金設定の新規作成入力です。
// CompanyID、CreatedAt、UpdatedAtはUsecase側で設定します。
// PrefectureRatesは47都道府県すべて必須です。
// IslandRatesは任意で、設定された場合は都道府県料金より優先されます。
type CreateTransportationFeeSettingInput struct {
	PrefectureRates []transportationdom.PrefectureRate
	IslandRates     []transportationdom.IslandRate
}

// UpdateTransportationFeeSettingInputは配送料金設定の更新入力です。
// PrefectureRatesは部分更新ではなく47都道府県すべてを受け取ります。
// IslandRatesも現在の完全な設定を受け取り、既存設定を置き換えます。
type UpdateTransportationFeeSettingInput struct {
	PrefectureRates []transportationdom.PrefectureRate
	IslandRates     []transportationdom.IslandRate
}

// TransportationUsecaseはcompany単位の配送料金設定を制御します。
// 1 companyにつきTransportationFeeSettingは最大1件です。
type TransportationUsecase struct {
	repo TransportationRepo
	now  func() time.Time
}

// NewTransportationUsecaseはTransportationUsecaseを生成します。
func NewTransportationUsecase(repo TransportationRepo) *TransportationUsecase {
	return &TransportationUsecase{
		repo: repo,
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

// GetByCompanyIDは指定companyの配送料金設定を取得します。
// 設定が存在しない場合はtransportation.ErrNotFoundを返します。
func (u *TransportationUsecase) GetByCompanyID(
	ctx context.Context,
	companyID string,
) (*transportationdom.TransportationFeeSetting, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	validCompanyID, err := validateTransportationCompanyID(companyID)
	if err != nil {
		return nil, err
	}

	return u.repo.GetByCompanyID(ctx, validCompanyID)
}

// ExistsByCompanyIDは指定companyに配送料金設定が存在するか返します。
func (u *TransportationUsecase) ExistsByCompanyID(
	ctx context.Context,
	companyID string,
) (bool, error) {
	if err := u.ensureRepo(); err != nil {
		return false, err
	}

	validCompanyID, err := validateTransportationCompanyID(companyID)
	if err != nil {
		return false, err
	}

	return u.repo.ExistsByCompanyID(ctx, validCompanyID)
}

// Createはcompanyの配送料金設定を新規作成します。
// PrefectureRatesは47都道府県すべて必須です。
// IslandRatesは任意です。
// 同一CompanyIDの設定が既に存在する場合はRepositoryからErrConflictが返されます。
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

	now := u.now().UTC()

	setting, err := transportationdom.NewWithNow(
		validCompanyID,
		in.PrefectureRates,
		in.IslandRates,
		now,
	)
	if err != nil {
		return nil, err
	}

	return u.repo.Create(ctx, setting)
}

// Updateは既存の配送料金設定を更新します。
// Updateはupsertではありません。
// 対象が存在しない場合はErrNotFoundを返します。
// CompanyIDとCreatedAtは既存値を維持し、UpdatedAtのみserver clockで更新します。
func (u *TransportationUsecase) Update(
	ctx context.Context,
	companyID string,
	in UpdateTransportationFeeSettingInput,
) (*transportationdom.TransportationFeeSetting, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	validCompanyID, err := validateTransportationCompanyID(companyID)
	if err != nil {
		return nil, err
	}

	current, err := u.repo.GetByCompanyID(ctx, validCompanyID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, transportationdom.ErrNotFound
	}

	now := u.now().UTC()

	if err := current.UpdateRates(
		in.PrefectureRates,
		in.IslandRates,
		now,
	); err != nil {
		return nil, err
	}

	return u.repo.Update(ctx, *current)
}

// ResolveFeeは配送先に適用する送料を解決します。
// islandCodeに一致するIslandRateが存在する場合はIslandRateを優先し、
// 存在しない場合はPrefectureRateへfallbackします。
// islandCodeが空の場合はPrefectureRateを使用します。
func (u *TransportationUsecase) ResolveFee(
	ctx context.Context,
	companyID string,
	prefectureCode string,
	islandCode string,
) (int64, error) {
	if err := u.ensureRepo(); err != nil {
		return 0, err
	}

	validCompanyID, err := validateTransportationCompanyID(companyID)
	if err != nil {
		return 0, err
	}

	code, err := transportationdom.ParsePrefectureCode(prefectureCode)
	if err != nil {
		return 0, err
	}

	setting, err := u.repo.GetByCompanyID(ctx, validCompanyID)
	if err != nil {
		return 0, err
	}
	if setting == nil {
		return 0, transportationdom.ErrNotFound
	}

	return setting.ResolveFee(code, islandCode)
}
