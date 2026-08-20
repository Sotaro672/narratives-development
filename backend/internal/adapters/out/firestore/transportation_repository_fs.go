// backend/internal/adapters/out/firestore/transportation_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
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
//	transportations/{companyId}
//
// document IDはTransportationFeeSetting.CompanyIDと同一です。
// companyIdはdocument fieldとして重複保存せず、DocumentSnapshot.Ref.IDから復元します。
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

// transportationDocumentはtransportations/{companyId}のFirestore保存schemaです。
// CompanyIDはdocument IDを正本とするためfieldとして保存しません。
type transportationDocument struct {
	PrefectureRates []transportationPrefectureRateDocument `firestore:"prefectureRates"`
	IslandRates     []transportationIslandRateDocument     `firestore:"islandRates"`
	CreatedAt       time.Time                              `firestore:"createdAt"`
	UpdatedAt       time.Time                              `firestore:"updatedAt"`
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

// GetByCompanyIDは指定companyの配送料金設定を取得します。
// Firestore document ID = CompanyIDです。
// 対象documentが存在しない場合はErrNotFoundを返します。
func (r *TransportationRepositoryFS) GetByCompanyID(ctx context.Context, companyID string) (*transportationdom.TransportationFeeSetting, error) {
	if err := r.ensureClient(); err != nil {
		return nil, err
	}

	validCompanyID, err := validateTransportationRepositoryCompanyID(companyID)
	if err != nil {
		return nil, err
	}

	snapshot, err := r.col().Doc(validCompanyID).Get(ctx)
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

// ExistsByCompanyIDは指定companyの配送料金設定documentが存在するか返します。
// companyIDが不正な場合はfalseとErrInvalidCompanyIDを返します。
// documentが存在しない場合はfalse, nilを返します。
func (r *TransportationRepositoryFS) ExistsByCompanyID(ctx context.Context, companyID string) (bool, error) {
	if err := r.ensureClient(); err != nil {
		return false, err
	}

	validCompanyID, err := validateTransportationRepositoryCompanyID(companyID)
	if err != nil {
		return false, err
	}

	_, err = r.col().Doc(validCompanyID).Get(ctx)
	if transportationNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// --------------------
// Write
// --------------------

// Createは新しいtransportations/{companyId}を作成します。
// Createは既存documentを上書きしません。
// 同じCompanyIDのdocumentが存在する場合はErrConflictを返します。
// 保存前にDomain constructorを使用してEntityを再構築し、47都道府県の完全性、重複、料金、島嶼部override、timestampを検証します。
func (r *TransportationRepositoryFS) Create(ctx context.Context, value transportationdom.TransportationFeeSetting) (*transportationdom.TransportationFeeSetting, error) {
	if err := r.ensureClient(); err != nil {
		return nil, err
	}

	validCompanyID, err := validateTransportationRepositoryCompanyID(value.CompanyID)
	if err != nil {
		return nil, err
	}

	validated, err := transportationdom.New(validCompanyID, value.PrefectureRates, value.IslandRates, value.CreatedAt, value.UpdatedAt)
	if err != nil {
		return nil, err
	}

	ref := r.col().Doc(validated.CompanyID)

	_, err = ref.Create(ctx, transportationToDocData(validated))
	if status.Code(err) == codes.AlreadyExists {
		return nil, transportationdom.ErrConflict
	}
	if err != nil {
		return nil, err
	}

	return &validated, nil
}

// Updateは既存のtransportations/{companyId}を更新します。
// Updateはupsertではありません。
// transaction内で対象documentを取得した後、tx.Updateを使用します。
// documentが存在しない場合はErrNotFoundを返します。
// CompanyIDおよびCreatedAtは変更できません。
// PrefectureRates、IslandRates、UpdatedAtのみ更新します。
func (r *TransportationRepositoryFS) Update(ctx context.Context, value transportationdom.TransportationFeeSetting) (*transportationdom.TransportationFeeSetting, error) {
	if err := r.ensureClient(); err != nil {
		return nil, err
	}

	validCompanyID, err := validateTransportationRepositoryCompanyID(value.CompanyID)
	if err != nil {
		return nil, err
	}

	ref := r.col().Doc(validCompanyID)
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

		if value.CompanyID != current.CompanyID {
			return transportationdom.ErrInvalidCompanyID
		}
		if !value.CreatedAt.Equal(current.CreatedAt) {
			return transportationdom.ErrInvalidCreatedAt
		}

		next, err := transportationdom.New(current.CompanyID, value.PrefectureRates, value.IslandRates, current.CreatedAt, value.UpdatedAt)
		if err != nil {
			return err
		}

		doc := transportationToDocData(next)
		updates := []firestore.Update{
			{Path: "prefectureRates", Value: doc.PrefectureRates},
			{Path: "islandRates", Value: doc.IslandRates},
			{Path: "updatedAt", Value: doc.UpdatedAt},
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
		PrefectureRates: prefectureRates,
		IslandRates:     islandRates,
		CreatedAt:       value.CreatedAt.UTC(),
		UpdatedAt:       value.UpdatedAt.UTC(),
	}
}

func docToTransportation(snapshot *firestore.DocumentSnapshot) (transportationdom.TransportationFeeSetting, error) {
	if snapshot == nil || snapshot.Ref == nil {
		return transportationdom.TransportationFeeSetting{}, transportationdom.ErrInvalidCompanyID
	}

	companyID, err := validateTransportationRepositoryCompanyID(snapshot.Ref.ID)
	if err != nil {
		return transportationdom.TransportationFeeSetting{}, err
	}

	var doc transportationDocument
	if err := snapshot.DataTo(&doc); err != nil {
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
		companyID,
		prefectureRates,
		islandRates,
		doc.CreatedAt,
		doc.UpdatedAt,
	)
}
