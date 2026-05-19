// backend/internal/adapters/out/firestore/productBlueprint/repository_helpers_utils.go
// Responsibility: 変換補助（数値変換・文字列配列の正規化）や一覧取得の共通処理を提供する。
package productBlueprint

import (
	"context"
	"errors"

	"google.golang.org/api/iterator"

	pbdom "narratives/internal/domain/productBlueprint"
)

func dedupTrimStrings(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := make([]string, 0, len(xs))
	for _, x := range xs {
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

// ListByCompanyID returns ProductBlueprints for the given companyID.
func (r *ProductBlueprintRepositoryFS) ListByCompanyID(
	ctx context.Context,
	companyID string,
) ([]pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

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

		out = append(out, pb)
	}

	return out, nil
}
