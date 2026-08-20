// backend/internal/adapters/out/firestore/transportation_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	transportationdom "narratives/internal/domain/transportation"
)

const transportationCollection = "transportations"

// RepositoryPortの実装漏れをcompile時に検出します。
var _ transportationdom.RepositoryPort = (*TransportationRepositoryFS)(nil)

// TransportationRepositoryFSはFirestoreを使用したTransportationFeeSetting Repository実装です。
//
// Firestore schema:
//
//	transportations/{transportationId}
//	  companyId
//	  name
//	  prefectureRates
//	  islandRates
//	  createdAt
//	  createdBy
//	  updatedAt
//	  updatedBy
//
// document IDはTransportationFeeSetting.IDと同一です。
// CompanyIDは1 companyが複数TransportationFeeSettingを所有できるようdocument fieldとして保存します。
type TransportationRepositoryFS struct {
	Client *firestore.Client
}

// transportationPrefectureRateDocumentは都道府県別送料のFirestore保存schemaです。
type transportationPrefectureRateDocument struct {
	PrefectureCode string `firestore:"prefectureCode"`
	Amount         int64  `firestore:"amount"`
}

// transportationIslandRateDocumentは島嶼部override送料のFirestore保存schemaです。
type transportationIslandRateDocument struct {
	IslandCode     string `firestore:"islandCode"`
	PrefectureCode string `firestore:"prefectureCode"`
	Amount         int64  `firestore:"amount"`
}

// transportationDocumentはtransportations/{transportationId}のFirestore保存schemaです。
// IDはdocument IDを正本とし、CompanyIDとNameはdocument fieldとして保存します。
type transportationDocument struct {
	CompanyID       string                                 `firestore:"companyId"`
	Name            string                                 `firestore:"name"`
	PrefectureRates []transportationPrefectureRateDocument `firestore:"prefectureRates"`
	IslandRates     []transportationIslandRateDocument     `firestore:"islandRates"`
	CreatedAt       time.Time                              `firestore:"createdAt"`
	CreatedBy       string                                 `firestore:"createdBy"`
	UpdatedAt       time.Time                              `firestore:"updatedAt"`
	UpdatedBy       string                                 `firestore:"updatedBy"`
}

func NewTransportationRepositoryFS(client *firestore.Client) *TransportationRepositoryFS {
	return &TransportationRepositoryFS{Client: client}
}

func (r *TransportationRepositoryFS) ensureClient() error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}
	return nil
}

func (r *TransportationRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection(transportationCollection)
}

func validateTransportationRepositoryID(id string) (string, error) {
	if id == "" || len([]rune(id)) > transportationdom.MaxTransportationIDLength {
		return "", transportationdom.ErrInvalidID
	}
	return id, nil
}

func validateTransportationRepositoryCompanyID(companyID string) (string, error) {
	if companyID == "" || len([]rune(companyID)) > transportationdom.MaxCompanyIDLength {
		return "", transportationdom.ErrInvalidCompanyID
	}
	return companyID, nil
}

func transportationNotFound(err error) bool {
	return status.Code(err) == codes.NotFound
}

// --------------------
// Read
// --------------------

// GetByIDはTransportationFeeSetting.IDをdocument IDとして1件取得します。
// 対象documentが存在しない場合はErrNotFoundを返します。
func (r *TransportationRepositoryFS) GetByID(ctx context.Context, id string) (*transportationdom.TransportationFeeSetting, error) {
	if err := r.ensureClient(); err != nil {
		return nil, err
	}

	validID, err := validateTransportationRepositoryID(id)
	if err != nil {
		return nil, err
	}

	snapshot, err := r.col().Doc(validID).Get(ctx)
	if transportationNotFound(err) {
		return nil, transportationdom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	entity, err := docToTransportation(snapshot)
	if err != nil {
		return nil, err
	}

	return &entity, nil
}

// ListByCompanyIDは指定companyが所有するすべての配送料金設定を取得します。
// companyId fieldによるqueryを行います。
// 対象が0件の場合は空sliceを返します。
func (r *TransportationRepositoryFS) ListByCompanyID(ctx context.Context, companyID string) ([]transportationdom.TransportationFeeSetting, error) {
	if err := r.ensureClient(); err != nil {
		return nil, err
	}

	validCompanyID, err := validateTransportationRepositoryCompanyID(companyID)
	if err != nil {
		return nil, err
	}

	iter := r.col().Where("companyId", "==", validCompanyID).Documents(ctx)
	defer iter.Stop()

	result := make([]transportationdom.TransportationFeeSetting, 0)

	for {
		snapshot, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		entity, err := docToTransportation(snapshot)
		if err != nil {
			return nil, err
		}

		if entity.CompanyID != validCompanyID {
			return nil, transportationdom.ErrInvalidCompanyID
		}

		result = append(result, entity)
	}

	return result, nil
}

// --------------------
// Write
// --------------------

// Createは新しいtransportations/{transportationId}を作成します。
// Createは既存documentを上書きしません。
// 同じTransportationFeeSetting.IDのdocumentが存在する場合はErrConflictを返します。
// 同一CompanyIDを持つ複数documentの作成は許可します。
// 保存前にDomain constructorを使用してEntity全体を再構築し、ID、CompanyID、Name、
// 47都道府県の完全性、重複、料金、島嶼部override、監査情報、timestampを検証します。
func (r *TransportationRepositoryFS) Create(ctx context.Context, value transportationdom.TransportationFeeSetting) (*transportationdom.TransportationFeeSetting, error) {
	if err := r.ensureClient(); err != nil {
		return nil, err
	}

	validID, err := validateTransportationRepositoryID(value.ID)
	if err != nil {
		return nil, err
	}

	validCompanyID, err := validateTransportationRepositoryCompanyID(value.CompanyID)
	if err != nil {
		return nil, err
	}

	validated, err := transportationdom.New(
		validID,
		validCompanyID,
		value.Name,
		value.PrefectureRates,
		value.IslandRates,
		value.CreatedAt,
		value.CreatedBy,
		value.UpdatedAt,
		value.UpdatedBy,
	)
	if err != nil {
		return nil, err
	}

	ref := r.col().Doc(validated.ID)

	_, err = ref.Create(ctx, transportationToDocData(validated))
	if status.Code(err) == codes.AlreadyExists {
		return nil, transportationdom.ErrConflict
	}
	if err != nil {
		return nil, err
	}

	return &validated, nil
}

// Updateは既存のtransportations/{transportationId}を更新します。
// Updateはupsertではありません。
// transaction内で対象documentを取得した後、tx.Updateを使用します。
// documentが存在しない場合はErrNotFoundを返します。
// ID、CompanyID、CreatedAt、CreatedByは変更できません。
// Name、PrefectureRates、IslandRates、UpdatedAt、UpdatedByのみ更新します。
func (r *TransportationRepositoryFS) Update(ctx context.Context, value transportationdom.TransportationFeeSetting) (*transportationdom.TransportationFeeSetting, error) {
	if err := r.ensureClient(); err != nil {
		return nil, err
	}

	validID, err := validateTransportationRepositoryID(value.ID)
	if err != nil {
		return nil, err
	}

	validCompanyID, err := validateTransportationRepositoryCompanyID(value.CompanyID)
	if err != nil {
		return nil, err
	}

	ref := r.col().Doc(validID)
	var updated transportationdom.TransportationFeeSetting

	err = r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snapshot, err := tx.Get(ref)
		if transportationNotFound(err) {
			return transportationdom.ErrNotFound
		}
		if err != nil {
			return err
		}

		current, err := docToTransportation(snapshot)
		if err != nil {
			return err
		}

		if value.ID != current.ID {
			return transportationdom.ErrInvalidID
		}
		if validCompanyID != current.CompanyID {
			return transportationdom.ErrInvalidCompanyID
		}
		if !value.CreatedAt.Equal(current.CreatedAt) {
			return transportationdom.ErrInvalidCreatedAt
		}
		if value.CreatedBy != current.CreatedBy {
			return transportationdom.ErrInvalidCreatedBy
		}

		next, err := transportationdom.New(
			current.ID,
			current.CompanyID,
			value.Name,
			value.PrefectureRates,
			value.IslandRates,
			current.CreatedAt,
			current.CreatedBy,
			value.UpdatedAt,
			value.UpdatedBy,
		)
		if err != nil {
			return err
		}

		doc := transportationToDocData(next)
		updates := []firestore.Update{
			{Path: "name", Value: doc.Name},
			{Path: "prefectureRates", Value: doc.PrefectureRates},
			{Path: "islandRates", Value: doc.IslandRates},
			{Path: "updatedAt", Value: doc.UpdatedAt},
			{Path: "updatedBy", Value: doc.UpdatedBy},
		}

		if err := tx.Update(ref, updates); err != nil {
			if transportationNotFound(err) {
				return transportationdom.ErrNotFound
			}
			return err
		}

		updated = next
		return nil
	})
	if transportationNotFound(err) {
		return nil, transportationdom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &updated, nil
}

// Deleteは既存のtransportations/{transportationId}を削除します。
// Deleteは対象documentが存在しない場合ErrNotFoundを返します。
func (r *TransportationRepositoryFS) Delete(ctx context.Context, id string) error {
	if err := r.ensureClient(); err != nil {
		return err
	}

	validID, err := validateTransportationRepositoryID(id)
	if err != nil {
		return err
	}

	ref := r.col().Doc(validID)

	err = r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		_, err := tx.Get(ref)
		if transportationNotFound(err) {
			return transportationdom.ErrNotFound
		}
		if err != nil {
			return err
		}

		if err := tx.Delete(ref); err != nil {
			if transportationNotFound(err) {
				return transportationdom.ErrNotFound
			}
			return err
		}

		return nil
	})
	if transportationNotFound(err) {
		return transportationdom.ErrNotFound
	}
	if err != nil {
		return err
	}

	return nil
}

// --------------------
// Mapper
// --------------------

func transportationToDocData(value transportationdom.TransportationFeeSetting) transportationDocument {
	prefectureRates := make([]transportationPrefectureRateDocument, len(value.PrefectureRates))
	for i, rate := range value.PrefectureRates {
		prefectureRates[i] = transportationPrefectureRateDocument{
			PrefectureCode: string(rate.PrefectureCode),
			Amount:         rate.Amount,
		}
	}

	islandRates := make([]transportationIslandRateDocument, len(value.IslandRates))
	for i, rate := range value.IslandRates {
		islandRates[i] = transportationIslandRateDocument{
			IslandCode:     string(rate.IslandCode),
			PrefectureCode: string(rate.PrefectureCode),
			Amount:         rate.Amount,
		}
	}

	return transportationDocument{
		CompanyID:       value.CompanyID,
		Name:            value.Name,
		PrefectureRates: prefectureRates,
		IslandRates:     islandRates,
		CreatedAt:       value.CreatedAt.UTC(),
		CreatedBy:       value.CreatedBy,
		UpdatedAt:       value.UpdatedAt.UTC(),
		UpdatedBy:       value.UpdatedBy,
	}
}

func docToTransportation(snapshot *firestore.DocumentSnapshot) (transportationdom.TransportationFeeSetting, error) {
	if snapshot == nil || snapshot.Ref == nil {
		return transportationdom.TransportationFeeSetting{}, transportationdom.ErrInvalidID
	}

	id, err := validateTransportationRepositoryID(snapshot.Ref.ID)
	if err != nil {
		return transportationdom.TransportationFeeSetting{}, err
	}

	var doc transportationDocument
	if err := snapshot.DataTo(&doc); err != nil {
		return transportationdom.TransportationFeeSetting{}, err
	}

	companyID, err := validateTransportationRepositoryCompanyID(doc.CompanyID)
	if err != nil {
		return transportationdom.TransportationFeeSetting{}, err
	}

	prefectureRates := make([]transportationdom.PrefectureRate, len(doc.PrefectureRates))
	for i, rate := range doc.PrefectureRates {
		prefectureCode, err := transportationdom.ParsePrefectureCode(rate.PrefectureCode)
		if err != nil {
			return transportationdom.TransportationFeeSetting{}, err
		}

		prefectureRates[i] = transportationdom.PrefectureRate{
			PrefectureCode: prefectureCode,
			Amount:         rate.Amount,
		}
	}

	islandRates := make([]transportationdom.IslandRate, len(doc.IslandRates))
	for i, rate := range doc.IslandRates {
		islandCode, err := transportationdom.ParseIslandCode(rate.IslandCode)
		if err != nil {
			return transportationdom.TransportationFeeSetting{}, err
		}

		prefectureCode, err := transportationdom.ParsePrefectureCode(rate.PrefectureCode)
		if err != nil {
			return transportationdom.TransportationFeeSetting{}, err
		}

		islandRates[i] = transportationdom.IslandRate{
			IslandCode:     islandCode,
			PrefectureCode: prefectureCode,
			Amount:         rate.Amount,
		}
	}

	return transportationdom.New(
		id,
		companyID,
		doc.Name,
		prefectureRates,
		islandRates,
		doc.CreatedAt,
		doc.CreatedBy,
		doc.UpdatedAt,
		doc.UpdatedBy,
	)
}
