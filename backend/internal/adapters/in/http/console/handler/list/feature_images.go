// backend/internal/adapters/in/http/console/handler/list/feature_images.go
//
// Responsibility:
// - ListImage に関するエンドポイントを担当する。
//   - 画像レコード登録
//   - 画像一覧取得
//   - 画像削除
//   - primary image 設定
//
// Firebase Storage migration policy:
// - backend は GCS signed URL を発行しない
// - frontend が Firebase Storage へ直接 upload する
// - frontend が Firebase Storage の downloadURL / objectPath を backend に送る
// - backend は Firestore の /lists/{listId}/images/{imageId} record を保存・取得・削除する
package list

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	listdom "narratives/internal/domain/list"
)

// GET /lists/{id}/images
func (h *ListHandler) listImages(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid listId"})
		return
	}

	items, err := h.uc.GetImages(ctx, id)
	if err != nil {
		writeListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(items)
}

// DELETE /lists/{id}/images/{imageId}
func (h *ListHandler) deleteImage(w http.ResponseWriter, r *http.Request, listID string, imageID string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	if listID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid listId"})
		return
	}

	if imageID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "imageId is required"})
		return
	}

	// Firebase Storage の実体削除は frontend 側で deleteObject を使うか、
	// 将来的に Firebase Admin SDK を持つ専用 endpoint を作る。
	// backend では Firestore の画像 record 削除だけを担当する。
	if err := h.uc.DeleteImage(ctx, listID, imageID); err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}

		writeListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"listId":  listID,
		"imageId": imageID,
	})
}

// POST /lists/{id}/images
//
// Firebase Storage 前提:
// - frontend が Firebase Storage に upload 済み
// - request には Firebase Storage の downloadURL / objectPath を含める
// - backend は Firestore record として保存する
func (h *ListHandler) saveImageFromGCS(w http.ResponseWriter, r *http.Request, listID string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	if listID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid listId"})
		return
	}

	var req struct {
		// imageId = Firestore docID for /lists/{listId}/images/{imageId}
		ID string `json:"id"`

		// Firebase Storage getDownloadURL()
		URL string `json:"url"`

		// Firebase Storage object path
		// Example:
		//   lists/{listId}/images/{imageId}/{fileName}
		ObjectPath string `json:"objectPath"`

		FileName    string `json:"fileName"`
		ContentType string `json:"contentType"`

		Size         int64  `json:"size"`
		DisplayOrder int    `json:"displayOrder"`
		CreatedBy    string `json:"createdBy,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	req.ID = strings.TrimSpace(req.ID)
	req.URL = strings.TrimSpace(req.URL)
	req.ObjectPath = strings.TrimLeft(strings.TrimSpace(req.ObjectPath), "/")
	req.FileName = strings.TrimSpace(req.FileName)
	req.ContentType = strings.ToLower(strings.TrimSpace(req.ContentType))
	req.CreatedBy = strings.TrimSpace(req.CreatedBy)

	if req.ID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "id is required"})
		return
	}

	if req.URL == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "url is required"})
		return
	}

	if req.ObjectPath == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "objectPath is required"})
		return
	}

	if req.FileName == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "fileName is required"})
		return
	}

	if req.DisplayOrder < 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "displayOrder must be >= 0"})
		return
	}

	if req.ContentType == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "contentType is required"})
		return
	}

	if _, ok := listdom.SupportedImageMIMEs[req.ContentType]; !ok {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unsupported contentType"})
		return
	}

	if req.Size > 0 && req.Size > listdom.DefaultMaxImageSizeBytes {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "file too large"})
		return
	}

	// Firebase Storage canonical path:
	//   lists/{listId}/images/{imageId}/{fileName}
	prefix := "lists/" + listID + "/images/" + req.ID + "/"
	if !strings.HasPrefix(req.ObjectPath, prefix) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "objectPath_not_canonical"})
		return
	}

	now := time.Now().UTC()

	img, err := h.uc.SaveImage(
		ctx,
		listdom.ListImage{
			ID:           req.ID,
			ListID:       listID,
			URL:          req.URL,
			ObjectPath:   req.ObjectPath,
			FileName:     req.FileName,
			ContentType:  req.ContentType,
			Size:         req.Size,
			DisplayOrder: req.DisplayOrder,
			CreatedAt:    now,
			CreatedBy:    req.CreatedBy,
		},
	)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}

		writeListErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(img)
}

// PUT|POST|PATCH /lists/{id}/primary-image
func (h *ListHandler) setPrimaryImage(w http.ResponseWriter, r *http.Request, listID string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	var req struct {
		ImageID   string  `json:"imageId"`
		UpdatedBy *string `json:"updatedBy"`
		Now       *string `json:"now"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	imageID := strings.TrimSpace(req.ImageID)
	if imageID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "imageId is required"})
		return
	}

	now := time.Now().UTC()
	if req.Now != nil && *req.Now != "" {
		if t, err := time.Parse(time.RFC3339, *req.Now); err == nil {
			now = t.UTC()
		}
	}

	item, err := h.uc.SetPrimaryImage(ctx, listID, imageID, now, req.UpdatedBy)
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}

		writeListErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(item)
}
