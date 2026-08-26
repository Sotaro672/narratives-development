// backend/internal/adapters/in/http/console/handler/inquiry_handler.go
package consoleHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	middleware "narratives/internal/adapters/in/http/middleware"
	consolequery "narratives/internal/application/query/console"
	usecase "narratives/internal/application/usecase"
	inquirydom "narratives/internal/domain/inquiry"
	orderdom "narratives/internal/domain/order"
	refunddom "narratives/internal/domain/refund"
)

// InquiryHandler は /inquiries 関連のエンドポイントを担当します。
type InquiryHandler struct {
	uc                    *usecase.InquiryUsecase
	returnReceiptUC       *usecase.ReturnReceiptUsecase
	openedReturnReceiptUC *usecase.OpenedReturnReceiptUsecase
	managementQuery       *consolequery.InquiryManagementQuery
	detailQuery           *consolequery.InquiryDetailQuery
}

// NewInquiryHandler はHTTPハンドラを初期化します。
func NewInquiryHandler(
	uc *usecase.InquiryUsecase,
	returnReceiptUC *usecase.ReturnReceiptUsecase,
	openedReturnReceiptUC *usecase.OpenedReturnReceiptUsecase,
	managementQuery *consolequery.InquiryManagementQuery,
	detailQuery *consolequery.InquiryDetailQuery,
) http.Handler {
	return &InquiryHandler{
		uc:                    uc,
		returnReceiptUC:       returnReceiptUC,
		openedReturnReceiptUC: openedReturnReceiptUC,
		managementQuery:       managementQuery,
		detailQuery:           detailQuery,
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
	// GET /inquiries/company/{companyId}/action-required-count
	//
	// URL 上の companyId は既存 route 互換のため受け取るが、
	// 実際の company boundary は middleware の companyId を正とする。
	if parts[0] == "company" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		if len(parts) == 2 && parts[1] != "" {
			h.listByCompanyID(w, r)
			return
		}

		if len(parts) == 3 && parts[1] != "" && parts[2] == "action-required-count" {
			h.countActionRequiredByCompanyID(w, r)
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid company id"})
		return
	}

	id := parts[0]

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

		case "reply":
			if r.Method != http.MethodPost {
				methodNotAllowed(w)
				return
			}
			h.reply(w, r, id)
			return

		case "receive-return":
			if r.Method != http.MethodPost {
				methodNotAllowed(w)
				return
			}
			h.receiveReturn(w, r, id)
			return

		case "receive-opened-return":
			if r.Method != http.MethodPost {
				methodNotAllowed(w)
				return
			}
			h.receiveOpenedReturn(w, r, id)
			return

		case "resolve":
			if r.Method != http.MethodPost {
				methodNotAllowed(w)
				return
			}
			h.resolve(w, r, id)
			return

		case "reopen":
			if r.Method != http.MethodPost {
				methodNotAllowed(w)
				return
			}
			h.reopen(w, r, id)
			return

		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}
	}

	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	h.get(w, r, id)
}

// GET /inquiries/company/{companyId}
//
// Query:
//   - searchQuery
//   - productId
//   - orderId
//   - avatarId
//   - status
//   - inquiryType
//   - updatedBy
//   - deletedBy
//   - resolvedBy
//   - closedBy
//   - imageFileName
//   - deleted=true|false
//   - resolved=true|false
//   - closed=true|false
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

// GET /inquiries/company/{companyId}/action-required-count
func (h *InquiryHandler) countActionRequiredByCompanyID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	companyID, ok := currentCompanyID(w, r)
	if !ok {
		return
	}

	filter := inquiryFilterFromRequest(r)

	count, err := h.managementQuery.CountActionRequiredByCompanyID(
		ctx,
		companyID,
		filter,
	)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]int{"count": count})
}

func inquiryFilterFromRequest(r *http.Request) inquirydom.Filter {
	q := r.URL.Query()

	filter := inquirydom.Filter{
		SearchQuery: q.Get("searchQuery"),
	}

	if v := q.Get("productId"); v != "" {
		filter.ProductID = &v
	}
	if v := q.Get("orderId"); v != "" {
		filter.OrderID = &v
	}
	if v := q.Get("avatarId"); v != "" {
		filter.AvatarID = &v
	}
	if v := q.Get("status"); v != "" {
		status := inquirydom.InquiryStatus(v)
		filter.Status = &status
	}
	if v := q.Get("inquiryType"); v != "" {
		inquiryType := inquirydom.InquiryType(v)
		filter.InquiryType = &inquiryType
	}
	if v := q.Get("updatedBy"); v != "" {
		filter.UpdatedBy = &v
	}
	if v := q.Get("deletedBy"); v != "" {
		filter.DeletedBy = &v
	}
	if v := q.Get("resolvedBy"); v != "" {
		filter.ResolvedBy = &v
	}
	if v := q.Get("closedBy"); v != "" {
		filter.ClosedBy = &v
	}
	if v := q.Get("imageFileName"); v != "" {
		filter.ImageFileName = &v
	}
	if v := q.Get("deleted"); v != "" {
		deleted := v == "true"
		filter.Deleted = &deleted
	}
	if v := q.Get("resolved"); v != "" {
		resolved := v == "true"
		filter.Resolved = &resolved
	}
	if v := q.Get("closed"); v != "" {
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

	memberID, ok := currentMemberID(w, r)
	if !ok {
		return
	}

	detail, err := h.detailQuery.GetDetailByIDForCompany(ctx, id, companyID)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	if !detail.Inquiry.IsRead || hasUnreadAvatarReply(detail.Replies) {
		if _, err := h.uc.MarkAsRead(ctx, usecase.MarkInquiryAsReadInput{
			InquiryID:        id,
			ReaderSenderType: inquirydom.ReplySenderTypeMember,
			ReaderSenderID:   memberID,
		}); err != nil {
			writeInquiryErr(w, err)
			return
		}

		detail, err = h.detailQuery.GetDetailByIDForCompany(ctx, id, companyID)
		if err != nil {
			writeInquiryErr(w, err)
			return
		}
	}

	_ = json.NewEncoder(w).Encode(detail)
}

func hasUnreadAvatarReply(replies []inquirydom.Reply) bool {
	for _, reply := range replies {
		if reply.IsRead {
			continue
		}
		if reply.SenderType == inquirydom.ReplySenderTypeAvatar {
			return true
		}
	}
	return false
}

// POST /inquiries/{id}/reply
//
// Body:
//
//	{
//	  "content": "返信本文",
//	  "images": []
//	}
//
// memberId は request body から受け取らず、認証 context の memberId を正とします。
func (h *InquiryHandler) reply(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID, ok := currentCompanyID(w, r)
	if !ok {
		return
	}

	memberID, ok := currentMemberID(w, r)
	if !ok {
		return
	}

	var req struct {
		Content string `json:"content"`
		Images  []struct {
			FileName   string  `json:"fileName"`
			FileURL    string  `json:"fileUrl"`
			ObjectPath string  `json:"objectPath"`
			FileSize   int64   `json:"fileSize"`
			MimeType   string  `json:"mimeType"`
			CreatedAt  *string `json:"createdAt"`
		} `json:"images"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	if req.Content == "" && len(req.Images) == 0 {
		writeInquiryErr(w, inquirydom.ErrReplyContentOrImageRequired)
		return
	}

	detail, err := h.detailQuery.GetDetailByIDForCompany(ctx, id, companyID)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	if detail.Inquiry.Status == inquirydom.InquiryStatusClosed {
		writeInquiryErr(w, inquirydom.ErrInquiryAlreadyClosed)
		return
	}

	now := time.Now().UTC()

	images, err := buildInquiryImagesForConsoleReply(id, memberID, now, req.Images)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	created, err := h.uc.CreateReplyByMember(ctx, id, memberID, req.Content, images)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// POST /inquiries/{id}/receive-return
//
// 未開封返品の商品受領を確定し、商品代金 + 対象商品の消費税を返金します。
//
// request body から返金額・orderId・orderItemIndex は受け取りません。
// Inquiry.OrderID + Inquiry.OrderItemIndex と Order snapshot を正とします。
//
// companyId / memberId は request body から受け取らず、
// 認証 context の値を正とします。
func (h *InquiryHandler) receiveReturn(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID, ok := currentCompanyID(w, r)
	if !ok {
		return
	}

	memberID, ok := currentMemberID(w, r)
	if !ok {
		return
	}

	// ReturnReceiptUsecase 自体でも company boundary を再検証するが、
	// 先に Console detail query を通して他 company の Inquiry を
	// financial usecase へ渡さない。
	if _, err := h.detailQuery.GetDetailByIDForCompany(ctx, id, companyID); err != nil {
		writeInquiryErr(w, err)
		return
	}

	if h.returnReceiptUC == nil {
		writeInquiryErr(w, usecase.ErrReturnReceiptUsecaseNotConfigured)
		return
	}

	result, err := h.returnReceiptUC.ReceiveReturn(
		ctx,
		usecase.ReceiveReturnInput{
			InquiryID: id,
			CompanyID: companyID,
			MemberID:  memberID,
		},
	)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	response := struct {
		Inquiry                inquirydom.Inquiry `json:"inquiry"`
		RefundID               string             `json:"refundId"`
		RefundStatus           string             `json:"refundStatus"`
		TransferReversalStatus string             `json:"transferReversalStatus"`
		FinanciallyCompleted   bool               `json:"financiallyCompleted"`
		OrderCompleted         bool               `json:"orderCompleted"`
		InquiryResolved        bool               `json:"inquiryResolved"`
		AlreadyCompleted       bool               `json:"alreadyCompleted"`
	}{
		Inquiry:                result.Inquiry,
		RefundID:               result.Refund.ID,
		RefundStatus:           string(result.Refund.Status),
		TransferReversalStatus: string(result.Refund.TransferReversalStatus),
		FinanciallyCompleted:   result.FinanciallyCompleted,
		OrderCompleted:         result.OrderCompleted,
		InquiryResolved:        result.InquiryResolved,
		AlreadyCompleted:       result.AlreadyCompleted,
	}

	if !result.FinanciallyCompleted {
		w.WriteHeader(http.StatusAccepted)
	}

	_ = json.NewEncoder(w).Encode(response)
}

// POST /inquiries/{id}/receive-opened-return
//
// 開封後返品の商品受領を確定し、選択された refund policy に従って返金します。
//
// Body:
//
//	{
//	  "policy": "half_merchandise"
//	}
//
// policy は次の3択のみです:
//
// - half_merchandise
// - merchandise_only
// - merchandise_round_trip_shipping
//
// request body から返金額・送料・orderId・orderItemIndex は受け取りません。
// Inquiry.OrderID + Inquiry.OrderItemIndex と Order snapshot を正とします。
//
// companyId / memberId は request body から受け取らず、
// 認証 context の値を正とします。
func (h *InquiryHandler) receiveOpenedReturn(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID, ok := currentCompanyID(w, r)
	if !ok {
		return
	}

	memberID, ok := currentMemberID(w, r)
	if !ok {
		return
	}

	var req struct {
		Policy refunddom.OpenedReturnRefundPolicy `json:"policy"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	if err := refunddom.ValidateOpenedReturnRefundPolicy(req.Policy); err != nil {
		writeInquiryErr(w, err)
		return
	}

	// OpenedReturnReceiptUsecase 自体でも company boundary を再検証するが、
	// 先に Console detail query を通して他 company の Inquiry を
	// financial usecase へ渡さない。
	if _, err := h.detailQuery.GetDetailByIDForCompany(ctx, id, companyID); err != nil {
		writeInquiryErr(w, err)
		return
	}

	if h.openedReturnReceiptUC == nil {
		writeInquiryErr(w, usecase.ErrOpenedReturnReceiptUsecaseNotConfigured)
		return
	}

	result, err := h.openedReturnReceiptUC.ReceiveOpenedReturn(
		ctx,
		usecase.ReceiveOpenedReturnInput{
			InquiryID: id,
			CompanyID: companyID,
			MemberID:  memberID,
			Policy:    req.Policy,
		},
	)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	totalBrandBurdenAmount, err := result.Refund.TotalBrandBurdenAmount()
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	response := struct {
		Inquiry                 inquirydom.Inquiry `json:"inquiry"`
		RefundID                string             `json:"refundId"`
		Policy                  string             `json:"policy"`
		RefundAmount            int                `json:"refundAmount"`
		ReturnShippingAmount    int                `json:"returnShippingAmount"`
		ReturnShippingTaxAmount int                `json:"returnShippingTaxAmount"`
		TotalBrandBurdenAmount  int                `json:"totalBrandBurdenAmount"`
		RefundStatus            string             `json:"refundStatus"`
		TransferReversalStatus  string             `json:"transferReversalStatus"`
		FinanciallyCompleted    bool               `json:"financiallyCompleted"`
		OrderCompleted          bool               `json:"orderCompleted"`
		InquiryResolved         bool               `json:"inquiryResolved"`
		AlreadyCompleted        bool               `json:"alreadyCompleted"`
	}{
		Inquiry:                 result.Inquiry,
		RefundID:                result.Refund.ID,
		Policy:                  string(result.Refund.Policy),
		RefundAmount:            result.Refund.RefundAmount,
		ReturnShippingAmount:    result.Refund.ReturnShippingAmount,
		ReturnShippingTaxAmount: result.Refund.ReturnShippingTaxAmount,
		TotalBrandBurdenAmount:  totalBrandBurdenAmount,
		RefundStatus:            string(result.Refund.Status),
		TransferReversalStatus:  string(result.Refund.TransferReversalStatus),
		FinanciallyCompleted:    result.FinanciallyCompleted,
		OrderCompleted:          result.OrderCompleted,
		InquiryResolved:         result.InquiryResolved,
		AlreadyCompleted:        result.AlreadyCompleted,
	}

	if !result.FinanciallyCompleted {
		w.WriteHeader(http.StatusAccepted)
	}

	_ = json.NewEncoder(w).Encode(response)
}

// POST /inquiries/{id}/resolve
//
// memberId は request body から受け取らず、認証 context の memberId を正とします。
func (h *InquiryHandler) resolve(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID, ok := currentCompanyID(w, r)
	if !ok {
		return
	}

	memberID, ok := currentMemberID(w, r)
	if !ok {
		return
	}

	detail, err := h.detailQuery.GetDetailByIDForCompany(ctx, id, companyID)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	// 返品 Inquiry は返金処理を必須とするため generic resolve では
	// 対応済みにしない。
	//
	// return_unopened:
	// POST /inquiries/{id}/receive-return
	//
	// return_opened:
	// POST /inquiries/{id}/receive-opened-return
	//
	// どちらも Refund -> Transfer Reversal -> Order return completion の
	// 完了後に専用 Usecase から Inquiry を resolved へ遷移させる。
	if detail.Inquiry.InquiryType == inquirydom.InquiryTypeReturnUnopened ||
		detail.Inquiry.InquiryType == inquirydom.InquiryTypeReturnOpened {
		writeInquiryErr(w, inquirydom.ErrInquiryInvalidWorkflow)
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

// POST /inquiries/{id}/reopen
//
// memberId は request body から受け取らず、認証 context の memberId を正とします。
func (h *InquiryHandler) reopen(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID, ok := currentCompanyID(w, r)
	if !ok {
		return
	}

	memberID, ok := currentMemberID(w, r)
	if !ok {
		return
	}

	if _, err := h.detailQuery.GetDetailByIDForCompany(ctx, id, companyID); err != nil {
		writeInquiryErr(w, err)
		return
	}

	updated, err := h.uc.ReopenByMember(ctx, usecase.ReopenInquiryInput{
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
//	  "createdAt": "2026-01-01T00:00:00Z"
//	}
//
// 画像バイナリは frontend から Firebase Storage へ直接保存します。
// backend は Firebase Storage downloadURL と objectPath のみ保存します。
// createdBy は request body ではなく認証 context の memberId を正とします。
func (h *InquiryHandler) addImage(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID, ok := currentCompanyID(w, r)
	if !ok {
		return
	}

	memberID, ok := currentMemberID(w, r)
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

	var objectPath *string
	if req.ObjectPath != "" {
		v := req.ObjectPath
		objectPath = &v
	}

	image, err := inquirydom.NewImageFileMinimal(
		id,
		req.FileName,
		req.FileURL,
		objectPath,
		req.FileSize,
		req.MimeType,
		createdAt,
		memberID,
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
	updatedBy := memberID

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

	memberID, ok := currentMemberID(w, r)
	if !ok {
		return
	}

	fileName := r.URL.Query().Get("fileName")
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

	if !in.RemoveImageByFileName(fileName) {
		writeInquiryErr(w, inquirydom.ErrNotFound)
		return
	}

	now := time.Now().UTC()
	updatedBy := memberID

	updated, err := h.uc.Update(ctx, id, inquirydom.InquiryPatch{
		Images:    &in.Images,
		UpdatedAt: &now,
		UpdatedBy: &updatedBy,
	})
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(updated.Images)
}

func buildInquiryImagesForConsoleReply(
	inquiryID string,
	memberID string,
	now time.Time,
	rawImages []struct {
		FileName   string  `json:"fileName"`
		FileURL    string  `json:"fileUrl"`
		ObjectPath string  `json:"objectPath"`
		FileSize   int64   `json:"fileSize"`
		MimeType   string  `json:"mimeType"`
		CreatedAt  *string `json:"createdAt"`
	},
) ([]inquirydom.ImageFile, error) {
	if len(rawImages) == 0 {
		return []inquirydom.ImageFile{}, nil
	}

	images := make([]inquirydom.ImageFile, 0, len(rawImages))

	for _, raw := range rawImages {
		imgCreatedAt := now
		if raw.CreatedAt != nil && *raw.CreatedAt != "" {
			t, err := time.Parse(time.RFC3339, *raw.CreatedAt)
			if err != nil {
				return nil, inquirydom.ErrInvalidImageCreatedAt
			}
			imgCreatedAt = t.UTC()
		}

		var objectPath *string
		if raw.ObjectPath != "" {
			v := raw.ObjectPath
			objectPath = &v
		}

		img, err := inquirydom.NewImageFileMinimal(
			inquiryID,
			raw.FileName,
			raw.FileURL,
			objectPath,
			raw.FileSize,
			raw.MimeType,
			imgCreatedAt,
			memberID,
		)
		if err != nil {
			return nil, err
		}

		images = append(images, img)
	}

	return images, nil
}

func currentCompanyID(w http.ResponseWriter, r *http.Request) (string, bool) {
	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId not found"})
		return "", false
	}
	return companyID, true
}

func currentMemberID(w http.ResponseWriter, r *http.Request) (string, bool) {
	memberID := usecase.MemberIDFromContext(r.Context())
	if memberID == "" {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "memberId not found"})
		return "", false
	}
	return memberID, true
}

// エラーハンドリング
func writeInquiryErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, inquirydom.ErrInvalidID),
		errors.Is(err, usecase.ErrReturnReceiptInvalidInquiryType),
		errors.Is(err, usecase.ErrOpenedReturnReceiptInvalidInquiryType),
		errors.Is(err, refunddom.ErrInvalidOpenedReturnRefundPolicy),
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
		errors.Is(err, inquirydom.ErrInvalidReplyID),
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
		errors.Is(err, inquirydom.ErrReplyTooManyImages),
		errors.Is(err, inquirydom.ErrReplyInconsistentImage),
		errors.Is(err, inquirydom.ErrReplyDuplicateImage),
		errors.Is(err, inquirydom.ErrReplyContentOrImageRequired),
		errors.Is(err, inquirydom.ErrInconsistentInquiry),
		errors.Is(err, inquirydom.ErrDuplicateImage),
		errors.Is(err, inquirydom.ErrTooManyImages),
		errors.Is(err, inquirydom.ErrInquiryAlreadyClosed),
		errors.Is(err, inquirydom.ErrInquiryInvalidWorkflow):
		code = http.StatusBadRequest

	case errors.Is(err, inquirydom.ErrInquiryForbidden),
		errors.Is(err, usecase.ErrReturnReceiptInvalidCompanyID),
		errors.Is(err, usecase.ErrReturnReceiptInvalidMemberID),
		errors.Is(err, usecase.ErrOpenedReturnReceiptInvalidCompanyID),
		errors.Is(err, usecase.ErrOpenedReturnReceiptInvalidMemberID):
		code = http.StatusForbidden

	case errors.Is(err, inquirydom.ErrNotFound),
		errors.Is(err, orderdom.ErrNotFound),
		errors.Is(err, refunddom.ErrNotFound),
		errors.Is(err, usecase.ErrReturnReceiptCompanyMismatch),
		errors.Is(err, usecase.ErrOpenedReturnReceiptCompanyMismatch):
		code = http.StatusNotFound

	case errors.Is(err, inquirydom.ErrConflict),
		errors.Is(err, orderdom.ErrConflict),
		errors.Is(err, refunddom.ErrConflict),
		errors.Is(err, usecase.ErrReturnReceiptInquiryNotOpen),
		errors.Is(err, usecase.ErrReturnReceiptInquiryClosed),
		errors.Is(err, usecase.ErrReturnReceiptOrderMismatch),
		errors.Is(err, usecase.ErrReturnReceiptOrderNotPaid),
		errors.Is(err, usecase.ErrReturnReceiptReturnNotRequested),
		errors.Is(err, usecase.ErrReturnReceiptReturnNotUnopened),
		errors.Is(err, usecase.ErrReturnReceiptRefundMismatch),
		errors.Is(err, usecase.ErrReturnReceiptOrderCompletionMismatch),
		errors.Is(err, usecase.ErrReturnReceiptInquiryResolutionMismatch),
		errors.Is(err, usecase.ErrOpenedReturnReceiptInquiryNotOpen),
		errors.Is(err, usecase.ErrOpenedReturnReceiptInquiryClosed),
		errors.Is(err, usecase.ErrOpenedReturnReceiptOrderMismatch),
		errors.Is(err, usecase.ErrOpenedReturnReceiptOrderNotPaid),
		errors.Is(err, usecase.ErrOpenedReturnReceiptReturnNotRequested),
		errors.Is(err, usecase.ErrOpenedReturnReceiptReturnNotOpened),
		errors.Is(err, usecase.ErrOpenedReturnReceiptRefundMismatch),
		errors.Is(err, usecase.ErrOpenedReturnReceiptOrderCompletionMismatch),
		errors.Is(err, usecase.ErrOpenedReturnReceiptInquiryResolutionMismatch):
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
