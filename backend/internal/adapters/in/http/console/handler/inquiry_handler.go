// backend/internal/adapters/in/http/console/handler/inquiry_handler.go
package consoleHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	consolequery "narratives/internal/application/query/console"
	usecase "narratives/internal/application/usecase"

	middleware "narratives/internal/adapters/in/http/middleware"

	inquirydom "narratives/internal/domain/inquiry"
)

// InquiryHandler は /inquiries 関連のエンドポイントを担当します。
type InquiryHandler struct {
	uc              *usecase.InquiryUsecase
	managementQuery *consolequery.InquiryManagementQuery
	detailQuery     *consolequery.InquiryDetailQuery
}

// NewInquiryHandler はHTTPハンドラを初期化します。
func NewInquiryHandler(
	uc *usecase.InquiryUsecase,
	managementQuery *consolequery.InquiryManagementQuery,
	detailQuery *consolequery.InquiryDetailQuery,
) http.Handler {
	return &InquiryHandler{
		uc:              uc,
		managementQuery: managementQuery,
		detailQuery:     detailQuery,
	}
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

	if len(parts) == 0 || parts[0] == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	// GET /inquiries/company/{companyId}
	// GET /inquiries/company/{companyId}/unread-count
	//
	// NOTE:
	// URL 上の companyId は既存 route 互換のため受け取るが、
	// 実際の company boundary は middleware が context に入れた
	// ログイン中 member の companyId を使う。
	if parts[0] == "company" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		if len(parts) == 2 && parts[1] != "" {
			h.listByCompanyID(w, r)
			return
		}

		if len(parts) == 3 && parts[1] != "" && parts[2] == "unread-count" {
			h.countUnreadByCompanyID(w, r)
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid company id"})
		return
	}

	id := parts[0]

	// サブリソース
	if len(parts) > 1 {
		switch parts[1] {
		case "images":
			switch r.Method {
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

		case "resolve":
			if r.Method != http.MethodPost {
				methodNotAllowed(w)
				return
			}
			h.resolve(w, r, id)
			return

		case "close":
			if r.Method != http.MethodPost {
				methodNotAllowed(w)
				return
			}
			h.close(w, r, id)
			return

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}
	}

	// GET /inquiries/{id}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	h.get(w, r, id)
}

// GET /inquiries/company/{companyId}
//
// Query:
//
//	searchQuery
//	productId
//	avatarId
//	status
//	inquiryType
//	updatedBy
//	deletedBy
//	resolvedBy
//	closedBy
//	imageFileName
//	deleted=true|false
//	resolved=true|false
//	closed=true|false
func (h *InquiryHandler) listByCompanyID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	companyID, ok := currentCompanyID(w, r)
	if !ok {
		return
	}

	filter := inquiryFilterFromRequest(r)

	result, err := h.managementQuery.ListByCompanyID(
		ctx,
		companyID,
		filter,
		inquirydom.Sort{},
		inquirydom.Page{},
	)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(result)
}

// GET /inquiries/company/{companyId}/unread-count
//
// Query:
//
//	searchQuery
//	productId
//	avatarId
//	status
//	inquiryType
//	updatedBy
//	deletedBy
//	resolvedBy
//	closedBy
//	imageFileName
//	deleted=true|false
//	resolved=true|false
//	closed=true|false
func (h *InquiryHandler) countUnreadByCompanyID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	companyID, ok := currentCompanyID(w, r)
	if !ok {
		return
	}

	filter := inquiryFilterFromRequest(r)

	count, err := h.managementQuery.CountUnreadByCompanyID(ctx, companyID, filter)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]int{
		"count": count,
	})
}

func inquiryFilterFromRequest(r *http.Request) inquirydom.Filter {
	q := r.URL.Query()

	filter := inquirydom.Filter{
		SearchQuery: q.Get("searchQuery"),
	}

	if v := strings.TrimSpace(q.Get("productId")); v != "" {
		filter.ProductID = &v
	}

	if v := strings.TrimSpace(q.Get("avatarId")); v != "" {
		filter.AvatarID = &v
	}

	if v := strings.TrimSpace(q.Get("status")); v != "" {
		status := inquirydom.InquiryStatus(v)
		filter.Status = &status
	}

	if v := strings.TrimSpace(q.Get("inquiryType")); v != "" {
		inquiryType := inquirydom.InquiryType(v)
		filter.InquiryType = &inquiryType
	}

	if v := strings.TrimSpace(q.Get("updatedBy")); v != "" {
		filter.UpdatedBy = &v
	}

	if v := strings.TrimSpace(q.Get("deletedBy")); v != "" {
		filter.DeletedBy = &v
	}

	if v := strings.TrimSpace(q.Get("resolvedBy")); v != "" {
		filter.ResolvedBy = &v
	}

	if v := strings.TrimSpace(q.Get("closedBy")); v != "" {
		filter.ClosedBy = &v
	}

	if v := strings.TrimSpace(q.Get("imageFileName")); v != "" {
		filter.ImageFileName = &v
	}

	if v := strings.TrimSpace(q.Get("deleted")); v != "" {
		deleted := v == "true"
		filter.Deleted = &deleted
	}

	if v := strings.TrimSpace(q.Get("resolved")); v != "" {
		resolved := v == "true"
		filter.Resolved = &resolved
	}

	if v := strings.TrimSpace(q.Get("closed")); v != "" {
		closed := v == "true"
		filter.Closed = &closed
	}

	return filter
}

// GET /inquiries/{id}
func (h *InquiryHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID, ok := currentCompanyID(w, r)
	if !ok {
		return
	}

	detail, err := h.detailQuery.GetDetailByIDForCompany(ctx, id, companyID)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	if !detail.Inquiry.IsRead {
		updated, err := h.uc.MarkAsRead(ctx, usecase.MarkInquiryAsReadInput{
			InquiryID: id,
		})
		if err != nil {
			writeInquiryErr(w, err)
			return
		}

		detail.Inquiry = updated
	}

	_ = json.NewEncoder(w).Encode(detail)
}

// POST /inquiries/{id}/resolve
//
// Body:
//
//	{
//	  "memberId": "member_document_id"
//	}
//
// company member が問い合わせを「対応済み」にします。
// ここでは status=resolved にします。
func (h *InquiryHandler) resolve(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID, ok := currentCompanyID(w, r)
	if !ok {
		return
	}

	if _, err := h.detailQuery.GetDetailByIDForCompany(ctx, id, companyID); err != nil {
		writeInquiryErr(w, err)
		return
	}

	var req struct {
		MemberID string `json:"memberId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	memberID := strings.TrimSpace(req.MemberID)
	if memberID == "" {
		writeInquiryErr(w, inquirydom.ErrInvalidResolvedBy)
		return
	}

	updated, err := h.uc.ResolveByMember(ctx, usecase.ResolveInquiryInput{
		InquiryID: id,
		MemberID:  memberID,
	})
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(updated)
}

// POST /inquiries/{id}/close
//
// Body:
//
//	{
//	  "memberId": "member_document_id"
//	}
//
// company member が問い合わせを close します。
func (h *InquiryHandler) close(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID, ok := currentCompanyID(w, r)
	if !ok {
		return
	}

	if _, err := h.detailQuery.GetDetailByIDForCompany(ctx, id, companyID); err != nil {
		writeInquiryErr(w, err)
		return
	}

	var req struct {
		MemberID string `json:"memberId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	memberID := strings.TrimSpace(req.MemberID)
	if memberID == "" {
		writeInquiryErr(w, inquirydom.ErrInvalidClosedBy)
		return
	}

	updated, err := h.uc.CloseByMember(ctx, usecase.CloseInquiryByMemberInput{
		InquiryID: id,
		MemberID:  memberID,
	})
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(updated)
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
//	  "createdAt": "2026-01-01T00:00:00Z",
//	  "createdBy": "uid_or_member_id"
//	}
//
// 画像バイナリは frontend から Firebase Storage へ直接保存します。
// backend は Firebase Storage の downloadURL(fileUrl) と objectPath のみ保存します。
func (h *InquiryHandler) addImage(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID, ok := currentCompanyID(w, r)
	if !ok {
		return
	}

	var req struct {
		FileName   string  `json:"fileName"`
		FileURL    string  `json:"fileUrl"`
		ObjectPath string  `json:"objectPath"`
		FileSize   int64   `json:"fileSize"`
		MimeType   string  `json:"mimeType"`
		CreatedAt  *string `json:"createdAt"`
		CreatedBy  string  `json:"createdBy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	createdAt := time.Now().UTC()
	if req.CreatedAt != nil && strings.TrimSpace(*req.CreatedAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.CreatedAt))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid createdAt"})
			return
		}
		createdAt = t.UTC()
	}

	createdBy := strings.TrimSpace(req.CreatedBy)
	if createdBy == "" {
		createdBy = "system"
	}

	var objectPath *string
	if strings.TrimSpace(req.ObjectPath) != "" {
		v := strings.TrimSpace(req.ObjectPath)
		objectPath = &v
	}

	image, err := inquirydom.NewImageFileMinimal(
		id,
		strings.TrimSpace(req.FileName),
		strings.TrimSpace(req.FileURL),
		objectPath,
		req.FileSize,
		strings.TrimSpace(req.MimeType),
		createdAt,
		createdBy,
	)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	detail, err := h.detailQuery.GetDetailByIDForCompany(ctx, id, companyID)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	in := detail.Inquiry

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

	companyID, ok := currentCompanyID(w, r)
	if !ok {
		return
	}

	fileName := strings.TrimSpace(r.URL.Query().Get("fileName"))
	if fileName == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "fileName is required"})
		return
	}

	detail, err := h.detailQuery.GetDetailByIDForCompany(ctx, id, companyID)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	in := detail.Inquiry

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

func currentCompanyID(w http.ResponseWriter, r *http.Request) (string, bool) {
	companyID, ok := middleware.CompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId not found"})
		return "", false
	}

	return companyID, true
}

// エラーハンドリング
func writeInquiryErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, inquirydom.ErrInvalidID),
		errors.Is(err, inquirydom.ErrInvalidProductID),
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
		errors.Is(err, inquirydom.ErrInvalidResolvedAt),
		errors.Is(err, inquirydom.ErrInvalidResolvedBy),
		errors.Is(err, inquirydom.ErrInvalidClosedAt),
		errors.Is(err, inquirydom.ErrInvalidClosedBy),
		errors.Is(err, inquirydom.ErrInvalidImageInquiryID),
		errors.Is(err, inquirydom.ErrInvalidImageFileName),
		errors.Is(err, inquirydom.ErrInvalidImageFileURL),
		errors.Is(err, inquirydom.ErrInvalidImageObjectPath),
		errors.Is(err, inquirydom.ErrInvalidImageFileSize),
		errors.Is(err, inquirydom.ErrInvalidImageMIMEType),
		errors.Is(err, inquirydom.ErrInvalidImageCreatedAt),
		errors.Is(err, inquirydom.ErrInvalidImageCreatedBy),
		errors.Is(err, inquirydom.ErrInvalidImageUpdatedAt),
		errors.Is(err, inquirydom.ErrInvalidImageUpdatedBy),
		errors.Is(err, inquirydom.ErrInvalidImageDeletedAt),
		errors.Is(err, inquirydom.ErrInvalidImageDeletedBy),
		errors.Is(err, inquirydom.ErrInconsistentInquiry),
		errors.Is(err, inquirydom.ErrDuplicateImage),
		errors.Is(err, inquirydom.ErrTooManyImages),
		errors.Is(err, inquirydom.ErrInquiryAlreadyClosed),
		errors.Is(err, inquirydom.ErrInquiryInvalidWorkflow):
		code = http.StatusBadRequest

	case errors.Is(err, inquirydom.ErrInquiryForbidden):
		code = http.StatusForbidden

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
