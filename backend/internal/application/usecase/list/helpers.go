// backend/internal/application/usecase/list/helpers.go
//
// Responsibility:
// - package list 内でのみ使うヘルパー関数を提供する。
// - 文字列の判定/patch生成/readableId生成等の「純粋関数」を集約し、他機能ファイルを薄く保つ。
//
// Features:
// - isImageURL / generateReadableID
// - getPatchUpdater / buildPatchFromItem
// - normalizeStrPtr
package list

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	listdom "narratives/internal/domain/list"
)

func isImageURL(v string) bool {
	s := strings.TrimSpace(v)
	return strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "gs://")
}

// ✅ readableId を生成（衝突してもOKな可読ID）
func generateReadableID(listID string, createdAt time.Time) string {
	t := createdAt
	if t.IsZero() {
		t = time.Now().UTC()
	}
	date := t.UTC().Format("20060102")

	base := strings.TrimSpace(listID)
	if base == "" {
		base = fmt.Sprintf("noid-%d", time.Now().UTC().UnixNano())
	}

	sum := sha1.Sum([]byte(base))
	hex6 := hex.EncodeToString(sum[:])
	if len(hex6) > 6 {
		hex6 = hex6[:6]
	}
	return fmt.Sprintf("L-%s-%s", date, hex6)
}

func (uc *ListUsecase) getPatchUpdater() ListPatchUpdater {
	if uc == nil {
		return nil
	}
	if uc.listReader != nil {
		if pu, ok := any(uc.listReader).(ListPatchUpdater); ok {
			return pu
		}
	}
	if uc.listCreator != nil {
		if pu, ok := any(uc.listCreator).(ListPatchUpdater); ok {
			return pu
		}
	}
	return nil
}

func buildPatchFromItem(item listdom.List) listdom.ListPatch {
	statusV := item.Status
	assigneeV := strings.TrimSpace(item.AssigneeID)
	imageV := strings.TrimSpace(item.ImageID) // URL格納方針
	titleV := strings.TrimSpace(item.Title)
	descV := strings.TrimSpace(item.Description)

	var updatedByV *string
	if item.UpdatedBy != nil {
		v := strings.TrimSpace(*item.UpdatedBy)
		if v != "" {
			updatedByV = &v
		}
	}

	now := time.Now().UTC()
	updatedAtV := now
	if item.UpdatedAt != nil && !item.UpdatedAt.IsZero() {
		updatedAtV = item.UpdatedAt.UTC()
	}

	// prices: nil(未指定)なら patch に入れない（意図せず全削除を防ぐ）
	var pricesPtr *[]listdom.ListPriceRow
	if item.Prices != nil {
		pv := item.Prices
		pricesPtr = &pv
	}

	// readableId: 変更対象として渡された場合のみ patch に入れる
	var readableIDPtr *string
	if strings.TrimSpace(item.ReadableID) != "" {
		v := strings.TrimSpace(item.ReadableID)
		readableIDPtr = &v
	}

	return listdom.ListPatch{
		Status:      &statusV,
		AssigneeID:  &assigneeV,
		ImageID:     &imageV,
		Title:       &titleV,
		Description: &descV,
		ReadableID:  readableIDPtr,
		UpdatedBy:   updatedByV,
		UpdatedAt:   &updatedAtV,
		Prices:      pricesPtr,
	}
}

func normalizeStrPtr(p *string) *string {
	if p == nil {
		return nil
	}
	t := strings.TrimSpace(*p)
	if t == "" {
		return nil
	}
	return &t
}
