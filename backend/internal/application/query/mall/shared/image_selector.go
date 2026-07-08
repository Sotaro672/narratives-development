// backend/internal/application/query/mall/shared/image_selector.go
package shared

import (
	"sort"

	ldom "narratives/internal/domain/list"
	resaledom "narratives/internal/domain/resale"
)

// SelectPrimaryListImageURL selects a representative list image URL.
//
// Policy:
// - list.ImageID wins when it matches an image with URL.
// - Otherwise, images are sorted by:
//  1. DisplayOrder asc
//  2. CreatedAt asc
//  3. ID asc
//
// - The first non-empty URL wins.
// - The input slice is not mutated.
func SelectPrimaryListImageURL(
	l ldom.List,
	images []ldom.ListImage,
) string {
	if len(images) == 0 {
		return ""
	}

	primaryImageID := l.ImageID
	if primaryImageID != "" {
		for _, img := range images {
			if img.ID == primaryImageID && img.URL != "" {
				return img.URL
			}
		}
	}

	sorted := append([]ldom.ListImage(nil), images...)

	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].DisplayOrder != sorted[j].DisplayOrder {
			return sorted[i].DisplayOrder < sorted[j].DisplayOrder
		}

		if !sorted[i].CreatedAt.Equal(sorted[j].CreatedAt) {
			return sorted[i].CreatedAt.Before(sorted[j].CreatedAt)
		}

		return sorted[i].ID < sorted[j].ID
	})

	for _, img := range sorted {
		if img.URL != "" {
			return img.URL
		}
	}

	return ""
}

// SelectPrimaryResaleImageURL selects a representative resale image URL.
//
// Policy:
// - resale.ImageID wins when it matches an image with URL.
// - Otherwise, FirstResaleImageURL is used.
func SelectPrimaryResaleImageURL(
	item resaledom.Resale,
	images []resaledom.ResaleImage,
) string {
	if len(images) == 0 {
		return ""
	}

	primaryImageID := item.ImageID
	if primaryImageID != "" {
		for _, img := range images {
			if img.ID == primaryImageID && img.URL != "" {
				return img.URL
			}
		}
	}

	return FirstResaleImageURL(images)
}

// FirstResaleImageURL selects a representative resale image URL
// when the parent resale entity is not available.
//
// Existing CartQuery / OrderQuery behavior:
// - first URL is fallback
// - DisplayOrder == 0 wins when present
func FirstResaleImageURL(images []resaledom.ResaleImage) string {
	if len(images) == 0 {
		return ""
	}

	var fallback string

	for _, img := range images {
		if img.URL == "" {
			continue
		}

		if fallback == "" {
			fallback = img.URL
		}

		if img.DisplayOrder == 0 {
			return img.URL
		}
	}

	return fallback
}
