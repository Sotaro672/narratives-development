// backend/internal/adapters/in/http/mall/handler/inquiry_handler.go
package mallHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"narratives/internal/adapters/in/http/middleware"
	mallquery "narratives/internal/application/query/mall"
	usecase "narratives/internal/application/usecase"
	inquirydom "narratives/internal/domain/inquiry"
)

// InquiryHandler は mall 側の問い合わせエンドポイントを担当します。
type InquiryHandler struct {
	uc    *usecase.InquiryUsecase
	query *mallquery.InquiryQuery
}

// NewInquiryHandler は mall inquiry handler を初期化します。
func NewInquiryHandler(
	uc *usecase.InquiryUsecase,
	query *mallquery.InquiryQuery,
) http.Handler {
	return &InquiryHandler{
		uc:    uc,
		query: query,
	}
}

// ServeHTTP はHTTPルーティングの入口です。
//
// Supported:
//
//	GET  /mall/me/inquiries
//	POST /mall/me/inquiries
//	GET  /mall/me/inquiries/unread-count
//	GET  /mall/me/inquiries/{id}
//	GET  /mall/me/inquiries/{id}/replies
//	POST /mall/me/inquiries/{id}/mark-as-read
//	POST /mall/me/inquiries/{id}/reply
//	POST /mall/me/inquiries/{id}/close
func (h *InquiryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "inquiry usecase is nil"})
		return
	}

	if h.query == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "inquiry query is nil"})
		return
	}

	if r.URL.Path == "/mall/me/inquiries" {
		switch r.Method {
		case http.MethodGet:
			h.list(w, r)
			return

		case http.MethodPost:
			h.create(w, r)
			return

		default:
			methodNotAllowed(w)
			return
		}
	}

	if r.URL.Path == "/mall/me/inquiries/unread-count" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.countUnread(w, r)
		return
	}

	if !strings.HasPrefix(r.URL.Path, "/mall/me/inquiries/") {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	rest := strings.TrimPrefix(r.URL.Path, "/mall/me/inquiries/")
	parts := strings.Split(rest, "/")

	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid inquiry id"})
		return
	}

	inquiryID := strings.TrimSpace(parts[0])

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.get(w, r, inquiryID)
		return
	}

	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	switch parts[1] {
	case "replies":
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		h.listReplies(w, r, inquiryID)
		return

	case "mark-as-read":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		h.markAsRead(w, r, inquiryID)
		return

	case "reply":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		h.reply(w, r, inquiryID)
		return

	case "close":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		h.close(w, r, inquiryID)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

// createInquiryRequest は mall 画面から問い合わせを起票する request です。
//
// productId は /mall/me/preview?productId=... の productId を渡します。
// avatarId は request body では受け取らず、AvatarContextMiddleware 由来で解決します。
type createInquiryRequest struct {
	ProductID   string                 `json:"productId"`
	Subject     string                 `json:"subject"`
	Content     string                 `json:"content"`
	InquiryType string                 `json:"inquiryType"`
	Images      []createInquiryImageIn `json:"images"`
}

// createInquiryImageIn は問い合わせ画像メタデータです。
//
// 画像バイナリは frontend から Firebase Storage へ直接保存します。
// backend は Firebase Storage の downloadURL(fileUrl) と objectPath のみ保存します。
type createInquiryImageIn struct {
	FileName   string  `json:"fileName"`
	FileURL    string  `json:"fileUrl"`
	ObjectPath string  `json:"objectPath"`
	FileSize   int64   `json:"fileSize"`
	MimeType   string  `json:"mimeType"`
	CreatedAt  *string `json:"createdAt"`
}

// replyInquiryRequest は avatar 側の返信 request です。
//
// reply は Inquiry.Content に追記せず、
// inquiries/{inquiryId}/replies/{replyId} に保存します。
type replyInquiryRequest struct {
	Content string                 `json:"content"`
	Images  []createInquiryImageIn `json:"images"`
}

// GET /mall/me/inquiries
//
// avatar 側が自分の起票した問い合わせ一覧を取得します。
func (h *InquiryHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	avatarID, ok := currentAvatarIDFromRequest(w, r)
	if !ok {
		return
	}

	filter := buildInquiryFilterFromQuery(r)
	page := buildInquiryPageFromQuery(r)

	result, err := h.query.ListByAvatarID(
		ctx,
		avatarID,
		filter,
		inquirydom.Sort{},
		page,
	)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"items":   result.Items,
		"page":    page.Number,
		"perPage": page.PerPage,
	})
}

// POST /mall/me/inquiries
func (h *InquiryHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	avatarID, ok := currentAvatarIDFromRequest(w, r)
	if !ok {
		return
	}

	var req createInquiryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	req.ProductID = strings.TrimSpace(req.ProductID)
	req.Subject = strings.TrimSpace(req.Subject)
	req.Content = strings.TrimSpace(req.Content)
	req.InquiryType = strings.TrimSpace(req.InquiryType)

	if req.InquiryType == "" {
		req.InquiryType = "product"
	}

	now := time.Now().UTC()

	inq := inquirydom.Inquiry{
		ID:          "",
		ProductID:   req.ProductID,
		AvatarID:    avatarID,
		Subject:     req.Subject,
		Content:     req.Content,
		Status:      inquirydom.InquiryStatusOpen,
		InquiryType: inquirydom.InquiryType(req.InquiryType),
		IsRead:      false,
		Images:      []inquirydom.ImageFile{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if len(req.Images) > 0 {
		images, err := buildInquiryImagesForMall("", avatarID, now, req.Images)
		if err != nil {
			writeInquiryErr(w, err)
			return
		}

		inq.Images = images
	}

	created, err := h.uc.Create(ctx, inq)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": created,
	})
}

// GET /mall/me/inquiries/unread-count
//
// avatar 側が受け取る未読 reply 数を avatarId のみで返します。
// companyId は使いません。
// 自分が送信した reply は count 対象外です。
func (h *InquiryHandler) countUnread(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	avatarID, ok := currentAvatarIDFromRequest(w, r)
	if !ok {
		return
	}

	filter := buildInquiryFilterFromQuery(r)

	count, err := h.uc.CountUnreadByAvatarID(ctx, usecase.CountUnreadByAvatarIDInput{
		AvatarID: avatarID,
		Filter:   filter,
	})
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"unreadCount": count,
		"count":       count,
	})
}

// GET /mall/me/inquiries/{id}
//
// avatar 側が自分の問い合わせ本体を取得します。
func (h *InquiryHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	avatarID, ok := currentAvatarIDFromRequest(w, r)
	if !ok {
		return
	}

	in, err := h.query.GetByIDForAvatar(ctx, id, avatarID)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": in,
	})
}

// GET /mall/me/inquiries/{id}/replies
//
// avatar 側が問い合わせ配下の reply 一覧を取得します。
//
// Expected flow:
//
//	ListByAvatarID -> GetByID -> ListByInquiryID
func (h *InquiryHandler) listReplies(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	avatarID, ok := currentAvatarIDFromRequest(w, r)
	if !ok {
		return
	}

	replies, err := h.query.ListRepliesByInquiryIDForAvatar(ctx, id, avatarID)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"items": replies,
	})
}

// POST /mall/me/inquiries/{id}/mark-as-read
//
// avatar 側が問い合わせを開いたタイミングなどで、
// 自分が送信した reply 以外を既読化します。
func (h *InquiryHandler) markAsRead(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	avatarID, ok := currentAvatarIDFromRequest(w, r)
	if !ok {
		return
	}

	if _, err := h.query.GetByIDForAvatar(ctx, id, avatarID); err != nil {
		writeInquiryErr(w, err)
		return
	}

	updated, err := h.uc.MarkAsRead(ctx, usecase.MarkInquiryAsReadInput{
		InquiryID:        id,
		ReaderSenderType: inquirydom.ReplySenderTypeAvatar,
		ReaderSenderID:   avatarID,
	})
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": updated,
	})
}

// POST /mall/me/inquiries/{id}/reply
//
// Body:
//
//	{
//	  "content": "追加の返信本文",
//	  "images": [
//	    {
//	      "fileName": "sample.png",
//	      "fileUrl": "https://firebasestorage.googleapis.com/...",
//	      "objectPath": "inquiry-replies/{inquiryId}/{imageId}/sample.png",
//	      "fileSize": 123,
//	      "mimeType": "image/png",
//	      "createdAt": "2026-01-01T00:00:00Z"
//	    }
//	  ]
//	}
//
// avatar 側が company member からの返信を受けた後、追加返信を入力する endpoint です。
func (h *InquiryHandler) reply(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	avatarID, ok := currentAvatarIDFromRequest(w, r)
	if !ok {
		return
	}

	var req replyInquiryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" && len(req.Images) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "content or images is required"})
		return
	}

	in, err := h.query.GetByIDForAvatar(ctx, id, avatarID)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	if in.Status == inquirydom.InquiryStatusClosed {
		writeInquiryErr(w, inquirydom.ErrInquiryAlreadyClosed)
		return
	}

	now := time.Now().UTC()

	images := []inquirydom.ImageFile{}
	if len(req.Images) > 0 {
		replyImages, err := buildInquiryImagesForMall(id, avatarID, now, req.Images)
		if err != nil {
			writeInquiryErr(w, err)
			return
		}

		images = replyImages
	}

	created, err := h.uc.CreateReplyByAvatar(ctx, id, avatarID, req.Content, images)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": created,
	})
}

// POST /mall/me/inquiries/{id}/close
//
// avatar 側が案件対応済みを確認した後、チケットを close する endpoint です。
func (h *InquiryHandler) close(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	avatarID, ok := currentAvatarIDFromRequest(w, r)
	if !ok {
		return
	}

	if _, err := h.query.GetByIDForAvatar(ctx, id, avatarID); err != nil {
		writeInquiryErr(w, err)
		return
	}

	updated, err := h.uc.CloseByAvatar(ctx, usecase.CloseInquiryByAvatarInput{
		InquiryID: id,
		AvatarID:  avatarID,
	})
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": updated,
	})
}

func buildInquiryFilterFromQuery(r *http.Request) inquirydom.Filter {
	q := r.URL.Query()
	filter := inquirydom.Filter{}

	if searchQuery := strings.TrimSpace(q.Get("searchQuery")); searchQuery != "" {
		filter.SearchQuery = searchQuery
	}

	if productID := strings.TrimSpace(q.Get("productId")); productID != "" {
		filter.ProductID = &productID
	}

	if statusRaw := strings.TrimSpace(q.Get("status")); statusRaw != "" {
		status := inquirydom.InquiryStatus(statusRaw)
		filter.Status = &status
	}

	if inquiryTypeRaw := strings.TrimSpace(q.Get("inquiryType")); inquiryTypeRaw != "" {
		inquiryType := inquirydom.InquiryType(inquiryTypeRaw)
		filter.InquiryType = &inquiryType
	}

	return filter
}

func buildInquiryPageFromQuery(r *http.Request) inquirydom.Page {
	q := r.URL.Query()

	pageNumber := parsePositiveInt(q.Get("page"), 1)
	perPage := parsePositiveInt(q.Get("perPage"), 100)

	if perPage > 100 {
		perPage = 100
	}

	return inquirydom.Page{
		Number:  pageNumber,
		PerPage: perPage,
	}
}

func buildInquiryImagesForMall(
	inquiryID string,
	avatarID string,
	now time.Time,
	rawImages []createInquiryImageIn,
) ([]inquirydom.ImageFile, error) {
	if len(rawImages) == 0 {
		return []inquirydom.ImageFile{}, nil
	}

	images := make([]inquirydom.ImageFile, 0, len(rawImages))

	for _, raw := range rawImages {
		imgCreatedAt := now
		if raw.CreatedAt != nil && strings.TrimSpace(*raw.CreatedAt) != "" {
			t, err := time.Parse(time.RFC3339, strings.TrimSpace(*raw.CreatedAt))
			if err != nil {
				return nil, inquirydom.ErrInvalidImageCreatedAt
			}
			imgCreatedAt = t.UTC()
		}

		var objectPath *string
		if strings.TrimSpace(raw.ObjectPath) != "" {
			v := strings.TrimSpace(raw.ObjectPath)
			objectPath = &v
		}

		img := inquirydom.ImageFile{
			InquiryID:  inquiryID,
			FileName:   strings.TrimSpace(raw.FileName),
			FileURL:    strings.TrimSpace(raw.FileURL),
			ObjectPath: objectPath,
			FileSize:   raw.FileSize,
			MimeType:   strings.TrimSpace(raw.MimeType),
			CreatedAt:  imgCreatedAt,
			CreatedBy:  avatarID,
		}

		images = append(images, img)
	}

	return images, nil
}

func currentAvatarIDFromRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || strings.TrimSpace(avatarID) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar context is required"})
		return "", false
	}

	return strings.TrimSpace(avatarID), true
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
		errors.Is(err, inquirydom.ErrInvalidReplyInquiryID),
		errors.Is(err, inquirydom.ErrInvalidReplySenderType),
		errors.Is(err, inquirydom.ErrInvalidReplySenderID),
		errors.Is(err, inquirydom.ErrInvalidReplyContent),
		errors.Is(err, inquirydom.ErrInvalidReplyCreatedAt),
		errors.Is(err, inquirydom.ErrInvalidReplyCreatedBy),
		errors.Is(err, inquirydom.ErrInvalidReplyUpdatedAt),
		errors.Is(err, inquirydom.ErrInvalidReplyUpdatedBy),
		errors.Is(err, inquirydom.ErrInvalidReplyDeletedAt),
		errors.Is(err, inquirydom.ErrInvalidReplyDeletedBy),
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
