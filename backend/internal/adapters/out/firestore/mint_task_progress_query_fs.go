// backend/internal/adapters/out/firestore/mint_task_progress_query_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	querydto "narratives/internal/application/query/console/dto"
)

// MintTaskProgressQueryFS は、mint task の進捗表示用 read model query です。
//
// command 側の MintRepositoryFS とは分け、画面表示用の集計だけを担当します。
type MintTaskProgressQueryFS struct {
	Client *firestore.Client
}

func NewMintTaskProgressQueryFS(client *firestore.Client) *MintTaskProgressQueryFS {
	return &MintTaskProgressQueryFS{
		Client: client,
	}
}

func (q *MintTaskProgressQueryFS) mintsCol() *firestore.CollectionRef {
	return q.Client.Collection("mints")
}

func (q *MintTaskProgressQueryFS) productsCol(mintID string) *firestore.CollectionRef {
	return q.mintsCol().Doc(mintID).Collection("products")
}

// GetMintTaskProgress は mints/{mintID}/products を集計して進捗を返します。
//
// 仕様:
// - Total: products サブコレクションの総数
// - Minted: status == "MINTED" の件数
// - Percentage: Minted / Total * 100 の整数値
//
// 補足:
// - Firestore 側の status は現行実装では "MINTED" などの大文字を想定しています。
// - 念のため strings.ToUpper で大文字小文字差を吸収します。
func (q *MintTaskProgressQueryFS) GetMintTaskProgress(
	ctx context.Context,
	mintID string,
) (*querydto.MintTaskProgressDTO, error) {
	if q == nil || q.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(mintID)
	if id == "" {
		return nil, errors.New("mintID is empty")
	}

	iter := q.productsCol(id).Documents(ctx)
	defer iter.Stop()

	progress := &querydto.MintTaskProgressDTO{}

	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list mint product tasks mintID=%s: %w", id, err)
		}
		if doc == nil || !doc.Exists() {
			continue
		}

		progress.Total++

		data := doc.Data()
		statusText := strings.ToUpper(strings.TrimSpace(asString(data["status"])))

		switch statusText {
		case "MINTED":
			progress.Minted++
		case "MINTING":
			progress.Minting++
		case "FAILED_RETRYABLE":
			progress.FailedRetryable++
		case "FAILED_FATAL":
			progress.FailedFatal++
		case "PENDING":
			progress.Pending++
		default:
			// status 未設定・未知値は pending 相当として扱います。
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
	if total <= 0 {
		return 0
	}

	if minted <= 0 {
		return 0
	}

	if minted >= total {
		return 100
	}

	return int(float64(minted) / float64(total) * 100)
}
