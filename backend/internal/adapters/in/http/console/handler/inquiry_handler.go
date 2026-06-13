// backend/internal/adapters/in/http/console/handler/inquiry_handler.go
package consoleHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	inquirydom "narratives/internal/domain/inquiry"
)

// InquiryHandler は /inquiries 関連のエンドポイントを担当します。
type InquiryHandler struct {
	uc *usecase.InquiryUsecase
}

// NewInquiryHandler はHTTPハンドラを初期化します。
func NewInquiryHandler(uc *usecase.InquiryUsecase) http.Handler {
	return &InquiryHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *InquiryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if !strings.HasPrefix(r.URL.Path, "/inquiries/") {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	rest := strings.TrimPrefix(r.URL.Path, "/inquiries/")
	parts := strings.Split(rest, "/")
	id := parts[0]
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	// サブリソース
	if len(parts) > 1 {
		switch parts[1] {
		case "aggregate":
			if r.Method != http.MethodGet {
				methodNotAllowed(w)
				return
			}
			h.getAggregate(w, r, id)
			return

		case "images":
			switch r.Method {
			case http.MethodGet:
				h.listImages(w, r, id)
				return
			case http.MethodPost:
				h.addImage(w, r, id)
				return
			case http.MethodDelete:
				h.deleteImage(w, r, id)
				return
			default:
				methodNotAllowed(w)
				return
			}

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}
	}

	// /inquiries/{id}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	h.get(w, r, id)
}

// GET /inquiries/{id}
func (h *InquiryHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	in, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(in)
}

// GET /inquiries/{id}/images
func (h *InquiryHandler) listImages(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	items, err := h.uc.GetImages(ctx, id)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	if items == nil {
		items = []inquirydom.ImageFile{}
	}

	_ = json.NewEncoder(w).Encode(items)
}

// POST /inquiries/{id}/images
//
// Body:
//
//	{
//	  "fileName": "sample.png",
//	  "fileUrl": "https://firebasestorage.googleapis.com/...",
//	  "objectPath": "inquiry-images/{inquiryId}/{imageId}/sample.png",
//	  "fileSize": 123,
//	  "mimeType": "image/png",
//	  "width": 123,
//	  "height": 456,
//	  "createdAt": "2026-01-01T00:00:00Z",
//	  "createdBy": "uid_or_member_id"
//	}
//
// 画像バイナリは frontend から Firebase Storage へ直接保存します。
// backend は Firebase Storage の downloadURL(fileUrl) と objectPath のみ保存します。
func (h *InquiryHandler) addImage(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	var req struct {
		FileName   string  `json:"fileName"`
		FileURL    string  `json:"fileUrl"`
		ObjectPath string  `json:"objectPath"`
		FileSize   int64   `json:"fileSize"`
		MimeType   string  `json:"mimeType"`
		Width      *int    `json:"width"`
		Height     *int    `json:"height"`
		CreatedAt  *string `json:"createdAt"`
		CreatedBy  string  `json:"createdBy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	createdAt := time.Now().UTC()
	if req.CreatedAt != nil && *req.CreatedAt != "" {
		t, err := time.Parse(time.RFC3339, *req.CreatedAt)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid createdAt"})
			return
		}
		createdAt = t.UTC()
	}

	createdBy := req.CreatedBy
	if createdBy == "" {
		createdBy = "system"
	}

	var objectPath *string
	if req.ObjectPath != "" {
		objectPath = &req.ObjectPath
	}

	image, err := inquirydom.NewImageFileMinimal(
		id,
		req.FileName,
		req.FileURL,
		objectPath,
		req.FileSize,
		req.MimeType,
		req.Width,
		req.Height,
		createdAt,
		createdBy,
	)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	in, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	if err := in.AddImage(image); err != nil {
		writeInquiryErr(w, err)
		return
	}

	now := time.Now().UTC()
	updatedBy := createdBy

	updated, err := h.uc.Update(ctx, id, inquirydom.InquiryPatch{
		Images:    &in.Images,
		UpdatedAt: &now,
		UpdatedBy: &updatedBy,
	})
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	added := findImageByFileName(updated.Images, image.FileName)
	if added == nil {
		added = &image
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(added)
}

// DELETE /inquiries/{id}/images?fileName=sample.png
//
// Firestore 上の Inquiry.Images から対象画像メタデータを削除します。
// Firebase Storage の実ファイル削除は、この handler の外側、または usecase 側で
// 削除前に ObjectPath を取得して実行してください。
func (h *InquiryHandler) deleteImage(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	fileName := r.URL.Query().Get("fileName")
	if fileName == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "fileName is required"})
		return
	}

	in, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	removed := in.RemoveImageByFileName(fileName)
	if !removed {
		writeInquiryErr(w, inquirydom.ErrNotFound)
		return
	}

	now := time.Now().UTC()

	updated, err := h.uc.Update(ctx, id, inquirydom.InquiryPatch{
		Images:    &in.Images,
		UpdatedAt: &now,
	})
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(updated.Images)
}

// GET /inquiries/{id}/aggregate
func (h *InquiryHandler) getAggregate(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	agg, err := h.uc.GetAggregate(ctx, id)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(agg)
}

// エラーハンドリング
func writeInquiryErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, inquirydom.ErrInvalidID),
		errors.Is(err, inquirydom.ErrInvalidAvatarID),
		errors.Is(err, inquirydom.ErrInvalidSubject),
		errors.Is(err, inquirydom.ErrInvalidContent),
		errors.Is(err, inquirydom.ErrInvalidStatus),
		errors.Is(err, inquirydom.ErrInvalidInquiryType),
		errors.Is(err, inquirydom.ErrInvalidCreatedAt),
		errors.Is(err, inquirydom.ErrInvalidUpdatedAt),
		errors.Is(err, inquirydom.ErrInvalidUpdatedBy),
		errors.Is(err, inquirydom.ErrInvalidDeletedAt),
		errors.Is(err, inquirydom.ErrInvalidDeletedBy),
		errors.Is(err, inquirydom.ErrInvalidImageInquiryID),
		errors.Is(err, inquirydom.ErrInvalidImageFileName),
		errors.Is(err, inquirydom.ErrInvalidImageFileURL),
		errors.Is(err, inquirydom.ErrInvalidImageObjectPath),
		errors.Is(err, inquirydom.ErrInvalidImageFileSize),
		errors.Is(err, inquirydom.ErrInvalidImageMIMEType),
		errors.Is(err, inquirydom.ErrInvalidImageDimensions),
		errors.Is(err, inquirydom.ErrInvalidImageCreatedAt),
		errors.Is(err, inquirydom.ErrInvalidImageCreatedBy),
		errors.Is(err, inquirydom.ErrInvalidImageUpdatedAt),
		errors.Is(err, inquirydom.ErrInvalidImageUpdatedBy),
		errors.Is(err, inquirydom.ErrInvalidImageDeletedAt),
		errors.Is(err, inquirydom.ErrInvalidImageDeletedBy),
		errors.Is(err, inquirydom.ErrInconsistentInquiry),
		errors.Is(err, inquirydom.ErrDuplicateImage),
		errors.Is(err, inquirydom.ErrTooManyImages):
		code = http.StatusBadRequest

	case errors.Is(err, inquirydom.ErrNotFound):
		code = http.StatusNotFound

	case errors.Is(err, inquirydom.ErrConflict):
		code = http.StatusConflict
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func findImageByFileName(images []inquirydom.ImageFile, fileName string) *inquirydom.ImageFile {
	for i := range images {
		if images[i].FileName == fileName {
			return &images[i]
		}
	}
	return nil
}
