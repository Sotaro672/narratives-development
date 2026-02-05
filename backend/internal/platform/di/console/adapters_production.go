// backend/internal/platform/di/console/adapters_production.go
package console

import (
	"context"
	"errors"
	"sort"
	"strings"

	fs "narratives/internal/adapters/out/firestore"
	productiondom "narratives/internal/domain/production"
)

// ✅ Adapter: ProductionRepositoryFS に GetTotalQuantityByModelID を付与
type productionRepoTotalQuantityAdapter struct {
	*fs.ProductionRepositoryFS
}

func (a *productionRepoTotalQuantityAdapter) GetTotalQuantityByModelID(
	ctx context.Context,
	productBlueprintIDs []string,
) ([]productiondom.ModelTotalQuantity, error) {
	if a == nil || a.ProductionRepositoryFS == nil {
		return nil, errors.New("production repo adapter is nil")
	}

	// sanitize + dedup ids
	ids := make([]string, 0, len(productBlueprintIDs))
	seen := make(map[string]struct{}, len(productBlueprintIDs))
	for _, id := range productBlueprintIDs {
		t := strings.TrimSpace(id)
		if t == "" {
			continue
		}
		k := strings.ToLower(t)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		ids = append(ids, t)
	}
	if len(ids) == 0 {
		return []productiondom.ModelTotalQuantity{}, nil
	}

	prods, err := a.ListByProductBlueprintID(ctx, ids)
	if err != nil {
		return nil, err
	}

	totalByKey := make(map[string]int, 64)
	origByKey := make(map[string]string, 64)

	for _, p := range prods {
		// deleted/status の概念は廃止（物理削除前提）なので、全件を集計対象とする
		for _, mq := range p.Models {
			mid := strings.TrimSpace(mq.ModelID)
			if mid == "" || mq.Quantity <= 0 {
				continue
			}
			key := strings.ToLower(mid)
			if _, ok := origByKey[key]; !ok {
				origByKey[key] = mid
			}
			totalByKey[key] += mq.Quantity
		}
	}

	out := make([]productiondom.ModelTotalQuantity, 0, len(totalByKey))
	for k, total := range totalByKey {
		out = append(out, productiondom.ModelTotalQuantity{
			ModelID:       origByKey[k],
			TotalQuantity: total,
		})
	}

	// stable order
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].ModelID) < strings.ToLower(out[j].ModelID)
	})

	return out, nil
}
