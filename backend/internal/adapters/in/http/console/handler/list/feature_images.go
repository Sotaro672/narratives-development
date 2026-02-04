// backend/internal/adapters/in/http/console/handler/list/feature_images.go
//
// Responsibility:
// - ListImage に関するエンドポイントを担当する。
//   - signed-url 発行
//   - GCS object からの保存
//   - 画像一覧取得
//   - 画像削除
//   - primary image 設定
//
// Policy:
// - 画像削除の実体は usecase に寄せる（handler は uc.DeleteImage(...) を呼ぶ）。
// - handler から imgDeleter を撤去済み（types.go に追従）。
package list

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	listuc "narratives/internal/application/usecase/list"
	listimgdom "narratives/internal/domain/listImage"
)

// POST /lists/{id}/images/signed-url
func (h *ListHandler) issueSignedURL(w http.ResponseWriter, r *http.Request, listID string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	var req struct {
		FileName         string `json:"fileName"`
		ContentType      string `json:"contentType"`
		Size             int64  `json:"size"`
		DisplayOrder     int    `json:"displayOrder"`
		ExpiresInSeconds int    `json:"expiresInSeconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid listId"})
		return
	}

	// ✅ displayOrder guard
	if req.DisplayOrder < 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "displayOrder must be >= 0"})
		return
	}

	ct := strings.ToLower(strings.TrimSpace(req.ContentType))
	if ct == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "contentType is required"})
		return
	}

	if _, ok := listimgdom.SupportedImageMIMEs[ct]; !ok {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unsupported contentType"})
		return
	}

	if req.Size > 0 && req.Size > int64(listimgdom.DefaultMaxImageSizeBytes) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "file too large"})
		return
	}

	out, err := h.uc.IssueImageSignedURL(ctx, listuc.ListImageIssueSignedURLInput{
		ListID:           listID,
		FileName:         strings.TrimSpace(req.FileName),
		ContentType:      ct,
		Size:             req.Size,
		DisplayOrder:     req.DisplayOrder,
		ExpiresInSeconds: req.ExpiresInSeconds,
	})
	if err != nil {
		if isNotSupported(err) {
			w.WriteHeader(http.StatusNotImplemented)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
			return
		}
		writeListErr(w, err)
		return
	}

	// ✅ contract checks:
	// - out.ID is imageId (Firestore docID)
	// - out.Bucket must be provided by issuer (env-fixed; no default)
	// - out.ObjectPath must be canonical: lists/{listId}/images/{imageId}
	if strings.TrimSpace(out.UploadURL) == "" ||
		strings.TrimSpace(out.Bucket) == "" ||
		strings.TrimSpace(out.ObjectPath) == "" ||
		strings.TrimSpace(out.ID) == "" {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "signed_url_response_invalid"})
		return
	}

	// canonical verify (defensive; usecase already checks)
	obj := strings.TrimLeft(strings.TrimSpace(out.ObjectPath), "/")
	imgID := strings.TrimSpace(out.ID)
	prefix := "lists/" + listID + "/images/"
	if !strings.HasPrefix(obj, prefix) || obj != prefix+imgID {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "signed_url_object_path_not_canonical"})
		return
	}

	type resp struct {
		// ✅ id is imageId (Firestore docID)
		ID string `json:"id"`

		// ✅ bucket is env-fixed and MUST be returned by issuer
		Bucket string `json:"bucket"`

		// ✅ canonical objectPath: lists/{listId}/images/{imageId}
		ObjectPath string `json:"objectPath"`

		PublicURL    string `json:"publicUrl"`
		UploadURL    string `json:"uploadUrl"`
		ExpiresAt    string `json:"expiresAt"`
		ContentType  string `json:"contentType"`
		Size         int64  `json:"size"`
		DisplayOrder int    `json:"displayOrder"`
		FileName     string `json:"fileName"`
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp{
		ID:           imgID,
		Bucket:       strings.TrimSpace(out.Bucket),
		ObjectPath:   obj,
		PublicURL:    strings.TrimSpace(out.PublicURL),
		UploadURL:    strings.TrimSpace(out.UploadURL),
		ExpiresAt:    strings.TrimSpace(out.ExpiresAt),
		ContentType:  strings.TrimSpace(out.ContentType),
		Size:         out.Size,
		DisplayOrder: out.DisplayOrder,
		FileName:     strings.TrimSpace(out.FileName),
	})
}

// GET /lists/{id}/images
func (h *ListHandler) listImages(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	id = strings.TrimSpace(id)
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

	listID = strings.TrimSpace(listID)
	if listID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid listId"})
		return
	}

	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "imageId is required"})
		return
	}

	// ✅ delete is handled by usecase (handler does not depend on adapters)
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
func (h *ListHandler) saveImageFromGCS(w http.ResponseWriter, r *http.Request, listID string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid listId"})
		return
	}

	var req struct {
		// ✅ id is imageId (Firestore docID)
		ID string `json:"id"`

		FileName string `json:"fileName"` // kept for request compatibility; not used by usecase

		// ✅ bucket is env-fixed and must be provided (from signed-url response)
		Bucket string `json:"bucket"`

		// ✅ canonical objectPath: lists/{listId}/images/{imageId}
		ObjectPath string `json:"objectPath"`

		Size         int64 `json:"size"`
		DisplayOrder int   `json:"displayOrder"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	req.ID = strings.TrimSpace(req.ID)
	req.Bucket = strings.TrimSpace(req.Bucket)
	req.ObjectPath = strings.TrimLeft(strings.TrimSpace(req.ObjectPath), "/")

	if req.DisplayOrder < 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "displayOrder must be >= 0"})
		return
	}

	// ✅ required fields
	if req.ID == "" || req.Bucket == "" || req.ObjectPath == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "id, bucket and objectPath are required"})
		return
	}

	// ✅ canonical objectPath validation: lists/{listId}/images/{imageId}
	prefix := "lists/" + listID + "/images/"
	if !strings.HasPrefix(req.ObjectPath, prefix) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "objectPath_not_canonical"})
		return
	}
	if req.ObjectPath != prefix+req.ID {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "objectPath_id_mismatch"})
		return
	}

	img, err := h.uc.SaveImageFromGCS(
		ctx,
		req.ID, // ✅ imageId
		listID,
		req.Bucket,
		req.ObjectPath,
		req.Size,
		req.DisplayOrder,
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
		// NOTE: client can send either a URL or an imageId (docID).
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
	if req.Now != nil && strings.TrimSpace(*req.Now) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.Now)); err == nil {
			now = t.UTC()
		}
	}

	item, err := h.uc.SetPrimaryImage(ctx, strings.TrimSpace(listID), imageID, now, normalizeStrPtr(req.UpdatedBy))
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
