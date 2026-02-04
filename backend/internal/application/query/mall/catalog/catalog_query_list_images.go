// backend/internal/application/query/mall/catalog/catalog_query_list_images.go
package catalogQuery

import (
	"context"
	"log"
	"sort"
	"strings"

	dto "narratives/internal/application/query/mall/dto"
	listimgdom "narratives/internal/domain/listImage"
)

// ============================================================
// ListImages (listId -> listImage[])
// - best-effort: ListImageRepo が nil の場合はエラーにせず空で返す
// - sort: displayOrder asc (known first), then id asc
// ============================================================

// loadListImages returns DTO-ready list images + error string (empty means OK).
func (q *CatalogQuery) loadListImages(
	ctx context.Context,
	listID string,
) ([]dto.CatalogListImageDTO, string) {
	listID = strings.TrimSpace(listID)
	if listID == "" {
		return nil, "listId is empty"
	}

	// best-effort: repo が無ければ壊さない（catalogの必須要件にしない）
	if q == nil || q.ListImageRepo == nil {
		log.Printf("[catalog] listImages repo is nil (best-effort) listId=%q", listID)
		return nil, ""
	}

	// ✅ FIX: CatalogQueryTypes の interface は ListByListID
	imgs, err := q.ListImageRepo.ListByListID(ctx, listID)
	if err != nil {
		log.Printf("[catalog] listImages ListByListID error listId=%q err=%q", listID, err.Error())
		return nil, err.Error()
	}

	out := make([]dto.CatalogListImageDTO, 0, len(imgs))
	seen := map[string]struct{}{}

	for _, it := range imgs {
		id := strings.TrimSpace(it.ID)
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
		return strings.TrimSpace(a.ID) < strings.TrimSpace(b.ID)
	})

	log.Printf("[catalog] listImages ok listId=%q count=%d", listID, len(out))
	return out, ""
}

// ============================================================
// mapper (domain -> dto)
// ============================================================

// NOTE: dto 側に CatalogListImageDTO を追加している前提（絶対正スキーマのみ）
func toCatalogListImageDTO(img listimgdom.ListImage) dto.CatalogListImageDTO {
	return dto.CatalogListImageDTO{
		ID:         strings.TrimSpace(img.ID),
		ListID:     strings.TrimSpace(img.ListID),
		URL:        strings.TrimSpace(img.URL),
		ObjectPath: strings.TrimSpace(img.ObjectPath),
		FileName:   strings.TrimSpace(img.FileName),
		Size:       img.Size,
		DisplayOrder: func() int {
			if img.DisplayOrder <= 0 {
				return 0
			}
			return img.DisplayOrder
		}(),
	}
}
