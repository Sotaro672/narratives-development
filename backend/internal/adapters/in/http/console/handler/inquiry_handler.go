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

type InquiryHandler struct {
	uc                    *usecase.InquiryUsecase
	returnReceiptUC       *usecase.ReturnReceiptUsecase
	openedReturnReceiptUC *usecase.OpenedReturnReceiptUsecase
	managementQuery       *consolequery.InquiryManagementQuery
	detailQuery           *consolequery.InquiryDetailQuery
}

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

	var req struct {
		MerchandiseRefundAmount int  `json:"merchandiseRefundAmount"`
		RefundOutboundShipping  bool `json:"refundOutboundShipping"`
		CoverReturnShipping     bool `json:"coverReturnShipping"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	selection := refunddom.ReturnRefundSelection{
		MerchandiseRefundAmount: req.MerchandiseRefundAmount,
		RefundOutboundShipping:  req.RefundOutboundShipping,
		CoverReturnShipping:     req.CoverReturnShipping,
	}

	if err := refunddom.ValidateReturnRefundSelection(selection); err != nil {
		writeInquiryErr(w, err)
		return
	}

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
			Selection: selection,
		},
	)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	totalSellerBurdenAmount, err := result.Refund.TotalSellerBurdenAmount()
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	response := struct {
		Inquiry                   inquirydom.Inquiry `json:"inquiry"`
		RefundID                  string             `json:"refundId"`
		MerchandiseRefundAmount   int                `json:"merchandiseRefundAmount"`
		RefundOutboundShipping    bool               `json:"refundOutboundShipping"`
		CoverReturnShipping       bool               `json:"coverReturnShipping"`
		MerchandiseAmount         int                `json:"merchandiseAmount"`
		MerchandiseTaxAmount      int                `json:"merchandiseTaxAmount"`
		OutboundShippingAmount    int                `json:"outboundShippingAmount"`
		OutboundShippingTaxAmount int                `json:"outboundShippingTaxAmount"`
		ReturnShippingAmount      int                `json:"returnShippingAmount"`
		ReturnShippingTaxAmount   int                `json:"returnShippingTaxAmount"`
		RefundAmount              int                `json:"refundAmount"`
		TotalSellerBurdenAmount   int                `json:"totalSellerBurdenAmount"`
		RefundStatus              string             `json:"refundStatus"`
		TransferReversalStatus    string             `json:"transferReversalStatus"`
		FinanciallyCompleted      bool               `json:"financiallyCompleted"`
		OrderCompleted            bool               `json:"orderCompleted"`
		InquiryResolved           bool               `json:"inquiryResolved"`
		AlreadyCompleted          bool               `json:"alreadyCompleted"`
	}{
		Inquiry:                   result.Inquiry,
		RefundID:                  result.Refund.ID,
		MerchandiseRefundAmount:   result.Refund.RequestedMerchandiseRefundAmount,
		RefundOutboundShipping:    result.Refund.RefundOutboundShipping,
		CoverReturnShipping:       result.Refund.CoverReturnShipping,
		MerchandiseAmount:         result.Refund.MerchandiseAmount,
		MerchandiseTaxAmount:      result.Refund.MerchandiseTaxAmount,
		OutboundShippingAmount:    result.Refund.OutboundShippingAmount,
		OutboundShippingTaxAmount: result.Refund.OutboundShippingTaxAmount,
		ReturnShippingAmount:      result.Refund.ReturnShippingAmount,
		ReturnShippingTaxAmount:   result.Refund.ReturnShippingTaxAmount,
		RefundAmount:              result.Refund.RefundAmount,
		TotalSellerBurdenAmount:   totalSellerBurdenAmount,
		RefundStatus:              string(result.Refund.Status),
		TransferReversalStatus:    string(result.Refund.TransferReversalStatus),
		FinanciallyCompleted:      result.FinanciallyCompleted,
		OrderCompleted:            result.OrderCompleted,
		InquiryResolved:           result.InquiryResolved,
		AlreadyCompleted:          result.AlreadyCompleted,
	}

	if !result.FinanciallyCompleted {
		w.WriteHeader(http.StatusAccepted)
	}

	_ = json.NewEncoder(w).Encode(response)
}

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
		MerchandiseRefundAmount int  `json:"merchandiseRefundAmount"`
		RefundOutboundShipping  bool `json:"refundOutboundShipping"`
		CoverReturnShipping     bool `json:"coverReturnShipping"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	selection := refunddom.ReturnRefundSelection{
		MerchandiseRefundAmount: req.MerchandiseRefundAmount,
		RefundOutboundShipping:  req.RefundOutboundShipping,
		CoverReturnShipping:     req.CoverReturnShipping,
	}

	if err := refunddom.ValidateReturnRefundSelection(selection); err != nil {
		writeInquiryErr(w, err)
		return
	}

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
			Selection: selection,
		},
	)
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	totalSellerBurdenAmount, err := result.Refund.TotalSellerBurdenAmount()
	if err != nil {
		writeInquiryErr(w, err)
		return
	}

	response := struct {
		Inquiry                   inquirydom.Inquiry `json:"inquiry"`
		RefundID                  string             `json:"refundId"`
		MerchandiseRefundAmount   int                `json:"merchandiseRefundAmount"`
		RefundOutboundShipping    bool               `json:"refundOutboundShipping"`
		CoverReturnShipping       bool               `json:"coverReturnShipping"`
		MerchandiseAmount         int                `json:"merchandiseAmount"`
		MerchandiseTaxAmount      int                `json:"merchandiseTaxAmount"`
		OutboundShippingAmount    int                `json:"outboundShippingAmount"`
		OutboundShippingTaxAmount int                `json:"outboundShippingTaxAmount"`
		ReturnShippingAmount      int                `json:"returnShippingAmount"`
		ReturnShippingTaxAmount   int                `json:"returnShippingTaxAmount"`
		RefundAmount              int                `json:"refundAmount"`
		TotalSellerBurdenAmount   int                `json:"totalSellerBurdenAmount"`
		RefundStatus              string             `json:"refundStatus"`
		TransferReversalStatus    string             `json:"transferReversalStatus"`
		FinanciallyCompleted      bool               `json:"financiallyCompleted"`
		OrderCompleted            bool               `json:"orderCompleted"`
		InquiryResolved           bool               `json:"inquiryResolved"`
		AlreadyCompleted          bool               `json:"alreadyCompleted"`
	}{
		Inquiry:                   result.Inquiry,
		RefundID:                  result.Refund.ID,
		MerchandiseRefundAmount:   result.Refund.RequestedMerchandiseRefundAmount,
		RefundOutboundShipping:    result.Refund.RefundOutboundShipping,
		CoverReturnShipping:       result.Refund.CoverReturnShipping,
		MerchandiseAmount:         result.Refund.MerchandiseAmount,
		MerchandiseTaxAmount:      result.Refund.MerchandiseTaxAmount,
		OutboundShippingAmount:    result.Refund.OutboundShippingAmount,
		OutboundShippingTaxAmount: result.Refund.OutboundShippingTaxAmount,
		ReturnShippingAmount:      result.Refund.ReturnShippingAmount,
		ReturnShippingTaxAmount:   result.Refund.ReturnShippingTaxAmount,
		RefundAmount:              result.Refund.RefundAmount,
		TotalSellerBurdenAmount:   totalSellerBurdenAmount,
		RefundStatus:              string(result.Refund.Status),
		TransferReversalStatus:    string(result.Refund.TransferReversalStatus),
		FinanciallyCompleted:      result.FinanciallyCompleted,
		OrderCompleted:            result.OrderCompleted,
		InquiryResolved:           result.InquiryResolved,
		AlreadyCompleted:          result.AlreadyCompleted,
	}

	if !result.FinanciallyCompleted {
		w.WriteHeader(http.StatusAccepted)
	}

	_ = json.NewEncoder(w).Encode(response)
}

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

func writeInquiryErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, inquirydom.ErrInvalidID),
		errors.Is(err, usecase.ErrReturnReceiptInvalidInquiryType),
		errors.Is(err, usecase.ErrOpenedReturnReceiptInvalidInquiryType),
		errors.Is(err, refunddom.ErrInvalidReturnRefundAmount),
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
