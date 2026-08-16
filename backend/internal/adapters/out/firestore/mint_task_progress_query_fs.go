// backend/internal/adapters/out/firestore/mint_task_progress_query_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	querydto "narratives/internal/application/query/console/dto"
	mintdom "narratives/internal/domain/mint"
)

// MintTaskProgressQueryFS は mint task の進捗表示専用 read model query です。
// command 側の MintRepositoryFS とは分離し、mints/{mintID}/products の集計だけを担当します。
type MintTaskProgressQueryFS struct {
	Client *firestore.Client
}

func NewMintTaskProgressQueryFS(client *firestore.Client) *MintTaskProgressQueryFS {
	return &MintTaskProgressQueryFS{Client: client}
}

func (q *MintTaskProgressQueryFS) mintsCol() *firestore.CollectionRef {
	return q.Client.Collection("mints")
}

func (q *MintTaskProgressQueryFS) productsCol(mintID string) *firestore.CollectionRef {
	return q.mintsCol().Doc(mintID).Collection("products")
}

// GetMintTaskProgress は mints/{mintID}/products を集計して MintDetail 用の進捗を返します。
//
// 集計:
// - Total: products サブコレクションの総数
// - Pending: PENDING
// - Minting: MINTING
// - Minted: MINTED
// - FailedRetryable: FAILED_RETRYABLE
// - FailedFatal: FAILED_FATAL
// - Percentage: Minted / Total * 100
//
// status は domain/mint.MintProductTaskStatus を正として扱います。
// status 未設定または未知値は、処理未完了として Pending に含めます。
func (q *MintTaskProgressQueryFS) GetMintTaskProgress(
	ctx context.Context,
	mintID string,
) (*querydto.MintTaskProgressDTO, error) {
	if q == nil || q.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if mintID == "" {
		return nil, errors.New("mintID is empty")
	}

	iter := q.productsCol(mintID).Documents(ctx)
	defer iter.Stop()

	progress := &querydto.MintTaskProgressDTO{}

	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list mint product tasks mintID=%s: %w", mintID, err)
		}
		if doc == nil || !doc.Exists() {
			continue
		}

		progress.Total++

		data := doc.Data()
		status := mintdom.MintProductTaskStatus(asString(data["status"]))

		switch status {
		case mintdom.MintProductTaskStatusPending:
			progress.Pending++
		case mintdom.MintProductTaskStatusMinting:
			progress.Minting++
		case mintdom.MintProductTaskStatusMinted:
			progress.Minted++
		case mintdom.MintProductTaskStatusFailedRetryable:
			progress.FailedRetryable++
		case mintdom.MintProductTaskStatusFailedFatal:
			progress.FailedFatal++
		default:
			progress.Pending++
		}
	}

	progress.Percentage = calculateMintProgressPercentage(
		progress.Minted,
		progress.Total,
	)

	return progress, nil
}

func calculateMintProgressPercentage(minted int, total int) int {
	if total <= 0 || minted <= 0 {
		return 0
	}
	if minted >= total {
		return 100
	}
	return minted * 100 / total
}
