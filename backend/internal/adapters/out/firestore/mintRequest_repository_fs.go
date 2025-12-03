// backend/internal/adapters/out/firestore/mintRequest_repository_fs.go
package firestore

import (
	"context"
	"strings"
	"time"

	common "narratives/internal/adapters/out/firestore/common"
	mintdom "narratives/internal/domain/mintRequest"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MintRequestRepositoryFS は mintRequest 用の Firestore 実装です。
// backend/internal/domain/mintRequest/repository_port.go の
// mintrequest.Repository インターフェースを満たします。
type MintRequestRepositoryFS struct {
	client *firestore.Client
}

// コンパイル時チェック
var _ mintdom.Repository = (*MintRequestRepositoryFS)(nil)

// NewMintRequestRepositoryFS はリポジトリを生成します。
func NewMintRequestRepositoryFS(client *firestore.Client) *MintRequestRepositoryFS {
	return &MintRequestRepositoryFS{client: client}
}

// Firestore 上のドキュメント構造
type mintRequestDoc struct {
	ID                string     `firestore:"id"`
	ProductionID      string     `firestore:"productionId"`
	Status            string     `firestore:"status"`
	MintQuantity      int        `firestore:"mintQuantity"`
	RequestedBy       *string    `firestore:"requestedBy"`
	RequestedAt       *time.Time `firestore:"requestedAt"`
	MintedAt          *time.Time `firestore:"mintedAt"`
	ScheduledBurnDate *time.Time `firestore:"scheduledBurnDate"`
	TokenBlueprintID  *string    `firestore:"tokenBlueprintId"`
}

func (r *MintRequestRepositoryFS) collection() *firestore.CollectionRef {
	return r.client.Collection("mintRequests")
}

// GetByID は ID で 1 件取得します。
func (r *MintRequestRepositoryFS) GetByID(ctx context.Context, id string) (mintdom.MintRequest, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return mintdom.MintRequest{}, mintdom.ErrNotFound
	}

	doc, err := r.collection().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return mintdom.MintRequest{}, mintdom.ErrNotFound
		}
		return mintdom.MintRequest{}, err
	}

	var d mintRequestDoc
	if err := doc.DataTo(&d); err != nil {
		return mintdom.MintRequest{}, err
	}

	// docID を優先して ID に入れておく（フィールド側が空の場合の保険）
	if strings.TrimSpace(d.ID) == "" {
		d.ID = doc.Ref.ID
	}

	return docToDomain(d)
}

// ListByProductionIDs は、指定された複数の productionId のいずれかに紐づく
// すべての MintRequest を取得します。
// Firestore の "in" クエリは 10 個までの制約があるため、必要に応じてチャンク分割します。
func (r *MintRequestRepositoryFS) ListByProductionIDs(
	ctx context.Context,
	productionIDs []string,
) ([]mintdom.MintRequest, error) {

	// 前処理: trim + 空文字除外 + 重複除去
	uniq := make(map[string]struct{}, len(productionIDs))
	for _, id := range productionIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		uniq[id] = struct{}{}
	}

	if len(uniq) == 0 {
		return []mintdom.MintRequest{}, nil
	}

	normalized := make([]string, 0, len(uniq))
	for id := range uniq {
		normalized = append(normalized, id)
	}

	const chunkSize = 10 // Firestore "in" クエリ上限

	var results []mintdom.MintRequest

	for i := 0; i < len(normalized); i += chunkSize {
		end := i + chunkSize
		if end > len(normalized) {
			end = len(normalized)
		}
		chunk := normalized[i:end]

		iter := r.collection().
			Where("productionId", "in", chunk).
			Documents(ctx)
		defer iter.Stop()

		for {
			snap, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, err
			}

			var d mintRequestDoc
			if err := snap.DataTo(&d); err != nil {
				return nil, err
			}
			if strings.TrimSpace(d.ID) == "" {
				d.ID = snap.Ref.ID
			}

			mr, err := docToDomain(d)
			if err != nil {
				return nil, err
			}
			results = append(results, mr)
		}
	}

	return results, nil
}

// Update は既存の MintRequest を更新します。
// 対象が存在しない場合は mintdom.ErrNotFound を返します。
func (r *MintRequestRepositoryFS) Update(ctx context.Context, mr mintdom.MintRequest) (mintdom.MintRequest, error) {
	id := strings.TrimSpace(mr.ID)
	if id == "" {
		return mintdom.MintRequest{}, mintdom.ErrNotFound
	}

	// 一度存在確認（upsert ではなく update 想定）
	_, err := r.collection().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return mintdom.MintRequest{}, mintdom.ErrNotFound
		}
		return mintdom.MintRequest{}, err
	}

	doc := domainToDoc(mr)
	if _, err := r.collection().Doc(id).Set(ctx, doc); err != nil {
		return mintdom.MintRequest{}, err
	}

	return mr, nil
}

// ===============================
// Mapping helpers
// ===============================

func docToDomain(d mintRequestDoc) (mintdom.MintRequest, error) {
	m := mintdom.MintRequest{
		ID:                strings.TrimSpace(d.ID),
		ProductionID:      strings.TrimSpace(d.ProductionID),
		Status:            mintdom.MintRequestStatus(strings.TrimSpace(d.Status)),
		MintQuantity:      d.MintQuantity,
		RequestedBy:       common.TrimPtr(d.RequestedBy),
		RequestedAt:       common.NormalizeTimePtr(d.RequestedAt),
		MintedAt:          common.NormalizeTimePtr(d.MintedAt),
		ScheduledBurnDate: common.NormalizeTimePtr(d.ScheduledBurnDate),
		TokenBlueprintID:  common.TrimPtr(d.TokenBlueprintID),
	}

	// ドメインの一貫性チェック
	if err := m.Validate(); err != nil {
		return mintdom.MintRequest{}, err
	}
	return m, nil
}

func domainToDoc(m mintdom.MintRequest) mintRequestDoc {
	return mintRequestDoc{
		ID:                strings.TrimSpace(m.ID),
		ProductionID:      strings.TrimSpace(m.ProductionID),
		Status:            string(m.Status),
		MintQuantity:      m.MintQuantity,
		RequestedBy:       common.TrimPtr(m.RequestedBy),
		RequestedAt:       common.NormalizeTimePtr(m.RequestedAt),
		MintedAt:          common.NormalizeTimePtr(m.MintedAt),
		ScheduledBurnDate: common.NormalizeTimePtr(m.ScheduledBurnDate),
		TokenBlueprintID:  common.TrimPtr(m.TokenBlueprintID),
	}
}
