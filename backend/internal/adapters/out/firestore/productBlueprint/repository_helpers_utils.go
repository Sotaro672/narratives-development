// backend/internal/adapters/out/firestore/productBlueprint/repository_helpers_utils.go
// Responsibility: 変換補助（数値変換・文字列配列の正規化）や一覧取得の共通処理を提供する。
package productBlueprint

import (
	"context"
	"errors"
	"strings"
	"time"

	"google.golang.org/api/iterator"

	pbdom "narratives/internal/domain/productBlueprint"
)

func getFloat64(v any) float64 {
	switch x := v.(type) {
	case int:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	case float32:
		return float64(x)
	case float64:
		return x
	default:
		return 0
	}
}

func dedupTrimStrings(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

// ListByCompanyID returns non-deleted ProductBlueprints for the given companyID.
// NOTE:
//   - Firestore で deletedAt==nil を厳密に拾うのはフィールド未設定問題があるため、
//     companyId で取得してから deletedAt を in-memory で除外する。
func (r *ProductBlueprintRepositoryFS) ListByCompanyID(
	ctx context.Context,
	companyID string,
) ([]pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, pbdom.ErrInvalidCompanyID
	}

	iter := r.col().
		Where("companyId", "==", companyID).
		Documents(ctx)
	defer iter.Stop()

	out := make([]pbdom.ProductBlueprint, 0, 64)
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		pb, err := docToProductBlueprint(snap)
		if err != nil {
			return nil, err
		}

		// deleted は除外（live list）
		if pb.DeletedAt != nil && !pb.DeletedAt.IsZero() {
			continue
		}

		out = append(out, pb)
	}

	return out, nil
}

// ListDeletedByCompanyID returns only logically deleted ProductBlueprints for the given companyID.
func (r *ProductBlueprintRepositoryFS) ListDeletedByCompanyID(
	ctx context.Context,
	companyID string,
) ([]pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, pbdom.ErrInvalidCompanyID
	}

	q := r.col().Query.
		Where("companyId", "==", companyID).
		Where("deletedAt", ">", time.Time{})

	snaps, err := q.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	out := make([]pbdom.ProductBlueprint, 0, len(snaps))
	for _, snap := range snaps {
		pb, err := docToProductBlueprint(snap)
		if err != nil {
			return nil, err
		}
		out = append(out, pb)
	}
	return out, nil
}
