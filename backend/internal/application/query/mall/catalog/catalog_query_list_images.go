// backend/internal/application/query/mall/catalog/catalog_query_list_images.go
package catalogQuery

import (
	"context"
	"log"
	"sort"

	dto "narratives/internal/application/query/mall/dto"
	listdom "narratives/internal/domain/list"
)

// ============================================================
// ListImages (listId -> listImage[])
// - best-effort: ListImageRepo が nil の場合はエラーにせず空で返す
// - sort: displayOrder asc (known first), then id asc
//
// Firebase Storage migration policy:
// - domain/listImage は削除済み
// - ListImage は domain/list.ListImage を使う
// - ListImage.URL は Firebase Storage downloadURL
// - backend は GCS bucket / public URL を組み立てない
// ============================================================

// loadListImages returns DTO-ready list images + error string (empty means OK).
func (q *CatalogQuery) loadListImages(ctx context.Context, listID string) ([]dto.CatalogListImageDTO, string) {
	if listID == "" {
		return nil, "listId is empty"
	}

	// best-effort: repo が無ければ壊さない（catalogの必須要件にしない）
	if q == nil || q.ListImageRepo == nil {
		log.Printf("[catalog] listImages repo is nil best-effort listId=%q", listID)
		return nil, ""
	}

	imgs, err := q.ListImageRepo.FindByListID(ctx, listID)
	if err != nil {
		log.Printf("[catalog] listImages FindByListID error listId=%q err=%q", listID, err.Error())
		return nil, err.Error()
	}

	out := make([]dto.CatalogListImageDTO, 0, len(imgs))
	seen := map[string]struct{}{}

	for _, it := range imgs {
		id := it.ID
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		out = append(out, toCatalogListImageDTO(it))
	}

	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]

		ao := a.DisplayOrder
		bo := b.DisplayOrder

		aKnown := ao > 0
		bKnown := bo > 0

		// known first
		if aKnown != bKnown {
			return aKnown
		}

		// both known: order asc
		if aKnown && bKnown && ao != bo {
			return ao < bo
		}

		// fallback: id asc
		return a.ID < b.ID
	})

	log.Printf("[catalog] listImages ok listId=%q count=%d", listID, len(out))
	return out, ""
}

// ============================================================
// mapper (domain -> dto)
// ============================================================

func toCatalogListImageDTO(img listdom.ListImage) dto.CatalogListImageDTO {
	return dto.CatalogListImageDTO{
		ID:         img.ID,
		ListID:     img.ListID,
		URL:        img.URL,
		ObjectPath: img.ObjectPath,
		FileName:   img.FileName,
		Size:       img.Size,
		DisplayOrder: func() int {
			if img.DisplayOrder <= 0 {
				return 0
			}
			return img.DisplayOrder
		}(),
	}
}
