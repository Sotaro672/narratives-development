// backend/internal/domain/list/helpers.go
package list

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func RequireNonEmpty(name string, v string) error {
	if v == "" {
		return fmt.Errorf("%s is required", name)
	}

	return nil
}

func (l *List) touch(now time.Time) {
	if now.IsZero() {
		now = time.Now().UTC()
	}

	t := now.UTC()
	l.UpdatedAt = &t
}

func normalizePriceRows(in []ListPriceRow) []ListPriceRow {
	if in == nil {
		return nil
	}

	seen := map[string]struct{}{}
	out := make([]ListPriceRow, 0, len(in))

	for _, v := range in {
		mid := strings.TrimSpace(v.ModelID)
		if mid == "" {
			continue
		}

		if !priceAllowed(v.Price) {
			continue
		}

		if _, ok := seen[mid]; ok {
			continue
		}

		seen[mid] = struct{}{}
		out = append(out, ListPriceRow{
			ModelID: mid,
			Price:   v.Price,
		})
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

func validatePriceRows(rows []ListPriceRow) error {
	if rows == nil {
		return nil
	}

	for _, r := range rows {
		mid := strings.TrimSpace(r.ModelID)
		if mid == "" {
			return ErrInvalidPriceModelID
		}

		if !priceAllowed(r.Price) {
			return ErrInvalidPrice
		}
	}

	return nil
}

func priceAllowed(v int) bool {
	return v >= MinPrice && v <= MaxPrice
}

var readableIDRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

func isValidReadableID(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	if len(s) > MaxReadableIDLength {
		return false
	}

	return readableIDRe.MatchString(s)
}

var imageIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func isValidImageID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}

	if len(id) > MaxImageIDLength {
		return false
	}

	if strings.Contains(id, "/") {
		return false
	}

	return imageIDRe.MatchString(id)
}

func validateURL(u string) error {
	u = strings.TrimSpace(u)
	if u == "" {
		return ErrInvalidListImageURL
	}

	pu, err := url.ParseRequestURI(u)
	if err != nil {
		return ErrInvalidListImageURL
	}

	if pu.Scheme == "" || pu.Host == "" {
		return ErrInvalidListImageURL
	}

	return nil
}

func validateObjectPath(p string) error {
	p = normalizeObjectPath(p)
	if p == "" {
		return ErrInvalidListImageObjectPath
	}

	if strings.Contains(p, "://") {
		return ErrInvalidListImageObjectPath
	}

	if strings.Contains(p, `\`) {
		return ErrInvalidListImageObjectPath
	}

	return nil
}

func normalizeObjectPath(p string) string {
	return strings.TrimLeft(strings.TrimSpace(p), "/")
}

func normalizeImageFileName(name string) string {
	fn := strings.TrimSpace(name)
	if fn == "" {
		fn = "image.png"
	}

	if filepath.Ext(fn) == "" {
		fn += ".png"
	}

	return fn
}

func isAllowedImageExtension(name string) bool {
	if len(AllowedImageExtensions) == 0 {
		return true
	}

	ext := strings.ToLower(filepath.Ext(name))
	_, ok := AllowedImageExtensions[ext]

	return ok
}

func isSupportedImageMIME(mime string) bool {
	mime = strings.ToLower(strings.TrimSpace(mime))
	if mime == "" {
		return false
	}

	_, ok := SupportedImageMIMEs[mime]
	return ok
}

func inferContentTypeFromFileName(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))

	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return ""
	}
}

// CanonicalListImageObjectPath returns:
//
//	lists/{listId}/images/{imageId}/{fileName}
func CanonicalListImageObjectPath(
	listID string,
	imageID string,
	fileName string,
) string {
	lid := strings.Trim(strings.TrimSpace(listID), "/")
	iid := strings.Trim(strings.TrimSpace(imageID), "/")
	fn := strings.Trim(strings.TrimSpace(fileName), "/")

	return strings.TrimLeft(
		fmt.Sprintf("lists/%s/images/%s/%s", lid, iid, fn),
		"/",
	)
}
