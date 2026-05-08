// backend/internal/adapters/out/firestore/order_query_fs.go
package firestore

import (
	"cloud.google.com/go/firestore"

	uc "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
)

// --- filter/sort helpers ---

func applyOrderSort(q firestore.Query, sort common.Sort) firestore.Query {
	dir := firestore.Desc
	if sort.Order == common.SortAsc {
		dir = firestore.Asc
	}

	// absolute source of truth: createdAt only
	if sort.Column != "" && sort.Column != "createdAt" {
		return q.OrderBy("createdAt", firestore.Desc).
			OrderBy(firestore.DocumentID, firestore.Desc)
	}

	return q.OrderBy("createdAt", dir).
		OrderBy(firestore.DocumentID, dir)
}

func matchOrderFilter(o orderdom.Order, f uc.OrderFilter) bool {
	if f.ID != "" && o.ID != f.ID {
		return false
	}

	if f.UserID != nil && *f.UserID != "" && o.UserID != *f.UserID {
		return false
	}

	if f.AvatarID != nil && *f.AvatarID != "" && o.AvatarID != *f.AvatarID {
		return false
	}

	if f.CartID != nil && *f.CartID != "" && o.CartID != *f.CartID {
		return false
	}

	if f.CreatedFrom != nil {
		if o.CreatedAt.IsZero() || o.CreatedAt.Before(f.CreatedFrom.UTC()) {
			return false
		}
	}

	if f.CreatedTo != nil {
		// upper bound exclusive
		if o.CreatedAt.IsZero() || !o.CreatedAt.Before(f.CreatedTo.UTC()) {
			return false
		}
	}

	return true
}
