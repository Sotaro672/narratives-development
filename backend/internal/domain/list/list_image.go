// backend/internal/domain/list/list_image.go
package list

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ListImage is an image record under:
//
//	/lists/{listId}/images/{imageId}
//
// Firebase Storage 移行後:
// - URL is Firebase Storage getDownloadURL()
// - ObjectPath is Firebase Storage object path
// - backend only persists and reads image metadata
// - backend does not upload/delete GCS objects
type ListImage struct {
	ID           string    `json:"id"`
	ListID       string    `json:"listId"`
	URL          string    `json:"url"`
	ObjectPath   string    `json:"objectPath"`
	FileName     string    `json:"fileName"`
	ContentType  string    `json:"contentType,omitempty"`
	Size         int64     `json:"size"`
	DisplayOrder int       `json:"displayOrder"`
	CreatedAt    time.Time `json:"createdAt"`
	CreatedBy    string    `json:"createdBy,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt,omitempty"`
	UpdatedBy    string    `json:"updatedBy,omitempty"`
}

// ImageFileValidation - 画像ファイルのバリデーション結果
type ImageFileValidation struct {
	IsValid      bool   `json:"isValid"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

func NewImageFileValidation(err error) ImageFileValidation {
	if err == nil {
		return ImageFileValidation{IsValid: true}
	}
	return ImageFileValidation{
		IsValid:      false,
		ErrorMessage: err.Error(),
	}
}

// NewListImage creates a ListImage with validation.
//
// Firebase Storage 前提:
//   - url は Firebase Storage getDownloadURL() の値
//   - objectPath は Firebase Storage object path
//     例: lists/{listId}/images/{imageId}/{fileName}
func NewListImage(
	id string,
	listID string,
	u string,
	objectPath string,
	fileName string,
	contentType string,
	size int64,
	displayOrder int,
	createdAt time.Time,
	createdBy string,
) (ListImage, error) {
	li := ListImage{
		ID:           strings.TrimSpace(id),
		ListID:       strings.TrimSpace(listID),
		URL:          strings.TrimSpace(u),
		ObjectPath:   normalizeObjectPath(objectPath),
		FileName:     normalizeImageFileName(fileName),
		ContentType:  strings.TrimSpace(contentType),
		Size:         size,
		DisplayOrder: displayOrder,
		CreatedAt:    createdAt.UTC(),
		CreatedBy:    strings.TrimSpace(createdBy),
	}

	if li.ContentType == "" {
		li.ContentType = inferContentTypeFromFileName(li.FileName)
	}

	if err := li.Validate(); err != nil {
		return ListImage{}, err
	}

	return li, nil
}

// NewListImageWithCanonicalPath builds objectPath as:
//
//	lists/{listId}/images/{imageId}/{fileName}
func NewListImageWithCanonicalPath(
	id string,
	listID string,
	downloadURL string,
	fileName string,
	contentType string,
	size int64,
	displayOrder int,
	createdAt time.Time,
	createdBy string,
) (ListImage, error) {
	fn := normalizeImageFileName(fileName)

	return NewListImage(
		id,
		listID,
		downloadURL,
		CanonicalListImageObjectPath(listID, id, fn),
		fn,
		contentType,
		size,
		displayOrder,
		createdAt,
		createdBy,
	)
}

func (li *ListImage) UpdateURL(u string) error {
	u = strings.TrimSpace(u)
	if err := validateURL(u); err != nil {
		return err
	}

	li.URL = u
	return nil
}

func (li *ListImage) UpdateObjectPath(objectPath string) error {
	obj := normalizeObjectPath(objectPath)
	if err := validateObjectPath(obj); err != nil {
		return err
	}

	li.ObjectPath = obj
	return nil
}

func (li *ListImage) UpdateFileName(name string) error {
	fn := normalizeImageFileName(name)
	if !isAllowedImageExtension(fn) {
		return ErrInvalidListImageFileName
	}

	li.FileName = fn
	return nil
}

func (li *ListImage) UpdateContentType(contentType string) error {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if !isSupportedImageMIME(ct) {
		return ErrInvalidListImageContentType
	}

	li.ContentType = ct
	return nil
}

func (li *ListImage) UpdateSize(size int64) error {
	if size < 0 {
		return ErrInvalidListImageSize
	}

	if MaxListImageSize > 0 && size > MaxListImageSize {
		return ErrInvalidListImageSize
	}

	li.Size = size
	return nil
}

func (li *ListImage) SetDisplayOrder(order int) error {
	if order < 0 {
		return ErrInvalidListImageDisplayOrder
	}

	li.DisplayOrder = order
	return nil
}

func (li *ListImage) Touch(now time.Time, actor string) {
	if now.IsZero() {
		now = time.Now().UTC()
	}

	t := now.UTC()
	li.UpdatedAt = t
	li.UpdatedBy = strings.TrimSpace(actor)
}

// Validate validates a ListImage.
func (li ListImage) Validate() error {
	if li.ID == "" {
		return ErrInvalidListImageID
	}

	if li.ListID == "" {
		return ErrInvalidListImageListID
	}

	if err := validateURL(li.URL); err != nil {
		return err
	}

	if err := validateObjectPath(li.ObjectPath); err != nil {
		return err
	}

	if li.FileName == "" || !isAllowedImageExtension(li.FileName) {
		return ErrInvalidListImageFileName
	}

	if li.ContentType != "" && !isSupportedImageMIME(li.ContentType) {
		return ErrInvalidListImageContentType
	}

	if li.Size < 0 {
		return ErrInvalidListImageSize
	}

	if MaxListImageSize > 0 && li.Size > MaxListImageSize {
		return ErrInvalidListImageSize
	}

	if li.DisplayOrder < 0 {
		return ErrInvalidListImageDisplayOrder
	}

	if li.CreatedAt.IsZero() {
		return ErrInvalidListImageCreatedAt
	}

	if li.CreatedBy == "" {
		return ErrInvalidListImageCreatedBy
	}

	return nil
}

// ValidateDataURL validates a data URL: data:<mime>;base64,<payload>.
func ValidateDataURL(
	data string,
	maxBytes int,
	supported map[string]struct{},
) (mime string, payload []byte, err error) {
	if !strings.HasPrefix(data, "data:") {
		return "", nil, errors.New("invalid data URL: missing 'data:' prefix")
	}

	parts := strings.SplitN(data, ",", 2)
	if len(parts) != 2 {
		return "", nil, errors.New("invalid data URL: missing payload")
	}

	meta := parts[0]
	raw := parts[1]

	if !strings.Contains(meta, ";base64") {
		return "", nil, errors.New("invalid data URL: not base64 encoded")
	}

	mime = strings.TrimPrefix(strings.SplitN(meta, ";", 2)[0], "data:")
	if mime == "" {
		return "", nil, errors.New("invalid data URL: missing mime type")
	}

	if supported == nil {
		supported = SupportedImageMIMEs
	}

	if _, ok := supported[mime]; !ok {
		return "", nil, fmt.Errorf("unsupported content type: %s", mime)
	}

	decoded, decErr := base64.StdEncoding.DecodeString(raw)
	if decErr != nil {
		return "", nil, fmt.Errorf("invalid base64 payload: %w", decErr)
	}

	if len(decoded) == 0 {
		return "", nil, errors.New("empty image payload")
	}

	if maxBytes <= 0 {
		maxBytes = int(DefaultMaxImageSizeBytes)
	}

	if len(decoded) > maxBytes {
		return "", nil, fmt.Errorf("file too large: %d bytes (max %d)", len(decoded), maxBytes)
	}

	return mime, decoded, nil
}
