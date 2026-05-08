// backend/internal/domain/list/errors_policy.go
package list

import "errors"

// Errors
var (
	// For persisted List entity
	ErrInvalidID = errors.New("list: invalid id")

	ErrInvalidReadableID   = errors.New("list: invalid readableId")
	ErrInvalidStatus       = errors.New("list: invalid status")
	ErrInvalidAssigneeID   = errors.New("list: invalid assigneeId")
	ErrInvalidTitle        = errors.New("list: invalid title")
	ErrInvalidInventoryID  = errors.New("list: invalid inventoryId")
	ErrInvalidDescription  = errors.New("list: invalid description")
	ErrInvalidPrices       = errors.New("list: invalid prices")
	ErrInvalidPrice        = errors.New("list: invalid price")
	ErrInvalidPriceModelID = errors.New("list: invalid modelId in prices")

	ErrInvalidCreatedBy = errors.New("list: invalid createdBy")
	ErrInvalidCreatedAt = errors.New("list: invalid createdAt")

	ErrInvalidUpdatedAt = errors.New("list: invalid updatedAt")
	ErrInvalidUpdatedBy = errors.New("list: invalid updatedBy")
	ErrInvalidDeletedAt = errors.New("list: invalid deletedAt")
	ErrInvalidDeletedBy = errors.New("list: invalid deletedBy")

	// Primary image linkage errors
	ErrEmptyImageID   = errors.New("list: imageId must not be empty")
	ErrInvalidImageID = errors.New("list: invalid imageId")

	// ListImage errors
	ErrInvalidListImageID           = errors.New("list: invalid listImage id")
	ErrInvalidListImageListID       = errors.New("list: invalid listImage listId")
	ErrInvalidListImageURL          = errors.New("list: invalid listImage url")
	ErrInvalidListImageObjectPath   = errors.New("list: invalid listImage objectPath")
	ErrInvalidListImageFileName     = errors.New("list: invalid listImage fileName")
	ErrInvalidListImageContentType  = errors.New("list: invalid listImage contentType")
	ErrInvalidListImageSize         = errors.New("list: invalid listImage size")
	ErrInvalidListImageDisplayOrder = errors.New("list: invalid listImage displayOrder")
	ErrInvalidListImageCreatedAt    = errors.New("list: invalid listImage createdAt")
	ErrInvalidListImageCreatedBy    = errors.New("list: invalid listImage createdBy")
	ErrListImageNotFound            = errors.New("list: listImage not found")
	ErrListImageConflict            = errors.New("list: listImage conflict")

	// UI-facing image validation errors
	ErrInvalidFileType = errors.New("無効なファイル形式です")
	ErrFileTooLarge    = errors.New("ファイルサイズが大きすぎます")
	ErrUploadFailed    = errors.New("画像のアップロードに失敗しました")
)

// Policy
var (
	MaxTitleLength       = 200
	MaxDescriptionLength = 2000
	MinPrice             = 0
	MaxPrice             = 10_000_000

	// human-friendly id guard
	MaxReadableIDLength = 64

	// primary image id guard
	MaxImageIDLength = 128

	// list image policy
	DefaultMaxImageSizeBytes int64 = 5 * 1024 * 1024  // 5MB
	MaxListImageSize         int64 = 20 * 1024 * 1024 // 20MB

	SupportedImageMIMEs = map[string]struct{}{
		"image/jpeg": {},
		"image/jpg":  {},
		"image/png":  {},
		"image/webp": {},
	}

	AllowedImageExtensions = map[string]struct{}{
		".png":  {},
		".jpg":  {},
		".jpeg": {},
		".webp": {},
	}
)
