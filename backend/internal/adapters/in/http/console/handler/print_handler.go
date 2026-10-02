// backend/internal/adapters/in/http/console/handler/print_handler.go
package consoleHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	consolequery "narratives/internal/application/query/console"
	usecase "narratives/internal/application/usecase"
	printdom "narratives/internal/domain/print"
	productdom "narratives/internal/domain/product"
)

type PrintHandler struct {
	uc             *usecase.PrintUsecase
	query          *consolequery.PrintQueryService
	printPDFSigner *PrintPDFTicketSigner
}

type createPrintPDFURLResponse struct {
	URL string `json:"url"`
}

func NewPrintHandler(
	uc *usecase.PrintUsecase,
	query *consolequery.PrintQueryService,
) http.Handler {
	return &PrintHandler{
		uc:    uc,
		query: query,
	}
}

// NewPrintHandlerWithPDFSigner は既存の Print API に加え、
// QR PDF 表示用の短時間署名URL発行を有効にする。
func NewPrintHandlerWithPDFSigner(
	uc *usecase.PrintUsecase,
	query *consolequery.PrintQueryService,
	printPDFSigner *PrintPDFTicketSigner,
) http.Handler {
	return &PrintHandler{
		uc:             uc,
		query:          query,
		printPDFSigner: printPDFSigner,
	}
}

func (h *PrintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/products/print-logs":
		h.createPrintLog(w, r)
		return

	case r.Method == http.MethodGet && r.URL.Path == "/products/print-logs":
		productionID := strings.Trim(r.URL.Query().Get("productionId"), " \t\r\n/")
		if productionID == "" {
			writePrintHandlerJSONError(w, http.StatusBadRequest, "productionId query parameter is required")
			return
		}
		h.listPrintLogsByProductionID(w, r, productionID)
		return

	case r.Method == http.MethodPost && r.URL.Path == "/products/print-pdf-url":
		h.createPrintPDFURL(w, r)
		return

	case r.Method == http.MethodGet && r.URL.Path == "/products":
		productionID := strings.Trim(r.URL.Query().Get("productionId"), " \t\r\n/")
		if productionID == "" {
			writePrintHandlerJSONError(w, http.StatusBadRequest, "productionId query parameter is required")
			return
		}
		h.listByProductionID(w, r, productionID)
		return

	case r.Method == http.MethodPost && r.URL.Path == "/products":
		h.create(w, r)
		return

	default:
		writePrintHandlerJSONError(w, http.StatusNotFound, "not_found")
		return
	}
}

func (h *PrintHandler) listByProductionID(
	w http.ResponseWriter,
	r *http.Request,
	productionID string,
) {
	if h.query == nil {
		writePrintHandlerJSONError(w, http.StatusInternalServerError, "print query service is not configured")
		return
	}

	list, err := h.query.ListProductsByProductionID(r.Context(), productionID)
	if err != nil {
		writeProductErr(w, err)
		return
	}

	if list == nil {
		writePrintHandlerJSON(w, http.StatusOK, []any{})
		return
	}

	writePrintHandlerJSON(w, http.StatusOK, list)
}

func (h *PrintHandler) listPrintLogsByProductionID(
	w http.ResponseWriter,
	r *http.Request,
	productionID string,
) {
	if h.query == nil {
		writePrintHandlerJSONError(w, http.StatusInternalServerError, "print query service is not configured")
		return
	}

	logs, err := h.query.ListPrintLogsByProductionID(r.Context(), productionID)
	if err != nil {
		if errors.Is(err, printdom.ErrNotFound) {
			writePrintHandlerJSON(w, http.StatusOK, []any{})
			return
		}
		writeProductErr(w, err)
		return
	}

	if logs == nil {
		writePrintHandlerJSON(w, http.StatusOK, []any{})
		return
	}

	writePrintHandlerJSON(w, http.StatusOK, logs)
}

func (h *PrintHandler) createPrintLog(w http.ResponseWriter, r *http.Request) {
	if h.uc == nil {
		writePrintHandlerJSONError(w, http.StatusInternalServerError, "print usecase is not configured")
		return
	}

	if h.query == nil {
		writePrintHandlerJSONError(w, http.StatusInternalServerError, "print query service is not configured")
		return
	}

	var req struct {
		ProductionID string `json:"productionId"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writePrintHandlerJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}

	productionID := strings.Trim(req.ProductionID, " \t\r\n/")
	if productionID == "" {
		writePrintHandlerJSONError(w, http.StatusBadRequest, "productionId is required")
		return
	}

	ctx := r.Context()

	if _, err := h.uc.CreatePrintLogForProduction(ctx, productionID); err != nil {
		writeProductErr(w, err)
		return
	}

	logs, err := h.query.ListPrintLogsByProductionID(ctx, productionID)
	if err != nil {
		writeProductErr(w, err)
		return
	}

	if len(logs) == 0 {
		writePrintHandlerJSONError(w, http.StatusInternalServerError, "created print log could not be loaded")
		return
	}

	writePrintHandlerJSON(w, http.StatusOK, logs[0])
}

// createPrintPDFURL は認証済みConsoleリクエストから短時間有効のPDF表示URLを発行する。
// Firebase ID Token はURLへ含めず、productionId / companyId を署名済みticketへ格納する。
func (h *PrintHandler) createPrintPDFURL(w http.ResponseWriter, r *http.Request) {
	if h.printPDFSigner == nil {
		writePrintHandlerJSONError(w, http.StatusInternalServerError, "print pdf ticket signer is not configured")
		return
	}

	var req struct {
		ProductionID string `json:"productionId"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writePrintHandlerJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}

	productionID := strings.Trim(req.ProductionID, " \t\r\n/")
	if productionID == "" {
		writePrintHandlerJSONError(w, http.StatusBadRequest, "productionId is required")
		return
	}

	// このendpoint自体は AuthMiddleware 配下で呼ばれるため、
	// companyId はクライアント入力ではなくContextを正とする。
	companyID := strings.TrimSpace(usecase.CompanyIDFromContext(r.Context()))
	if companyID == "" {
		writePrintHandlerJSONError(w, http.StatusForbidden, "companyId not resolved for current user")
		return
	}

	ticket, err := h.printPDFSigner.Issue(productionID, companyID)
	if err != nil {
		h.writePrintPDFTicketIssueError(w, err)
		return
	}

	// Host Header を署名URL生成に使用しない。
	// Backend-relative URL を返し、frontend 側で API_BASE を基準に絶対URLへ変換する。
	values := url.Values{}
	values.Set("ticket", ticket)

	printPDFURL := printPDFPath + "?" + values.Encode()

	writePrintHandlerJSON(w, http.StatusOK, createPrintPDFURLResponse{
		URL: printPDFURL,
	})
}

func (h *PrintHandler) writePrintPDFTicketIssueError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(err, ErrPrintPDFTicketProductionID):
		writePrintHandlerJSONError(w, http.StatusBadRequest, "productionId is required")

	case errors.Is(err, ErrPrintPDFTicketCompanyID):
		writePrintHandlerJSONError(w, http.StatusForbidden, "companyId is required")

	case errors.Is(err, ErrPrintPDFTicketSecretRequired),
		errors.Is(err, ErrPrintPDFTicketSecretTooShort),
		errors.Is(err, ErrPrintPDFTicketInvalidTTL):
		writePrintHandlerJSONError(w, http.StatusInternalServerError, "print pdf ticket signer is invalid")

	default:
		writePrintHandlerJSONError(w, http.StatusInternalServerError, "print pdf ticket could not be issued")
	}
}

func (h *PrintHandler) create(w http.ResponseWriter, r *http.Request) {
	if h.uc == nil {
		writePrintHandlerJSONError(w, http.StatusInternalServerError, "print usecase is not configured")
		return
	}

	var req struct {
		ModelID      string    `json:"modelId"`
		ProductionID string    `json:"productionId"`
		PrintedAt    time.Time `json:"printedAt"`
		PrintedBy    *string   `json:"printedBy"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writePrintHandlerJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}

	req.ModelID = strings.Trim(req.ModelID, " \t\r\n/")
	req.ProductionID = strings.Trim(req.ProductionID, " \t\r\n/")

	if req.ModelID == "" || req.ProductionID == "" {
		writePrintHandlerJSONError(w, http.StatusBadRequest, "modelId and productionId are required")
		return
	}

	if req.PrintedAt.IsZero() {
		writePrintHandlerJSONError(w, http.StatusBadRequest, "printedAt is required")
		return
	}

	printedAt := req.PrintedAt.UTC()

	p := productdom.Product{
		ModelID:          req.ModelID,
		ProductionID:     req.ProductionID,
		InspectionResult: productdom.InspectionNotYet,
		PrintedAt:        &printedAt,
		InspectedAt:      nil,
		InspectedBy:      nil,
	}

	created, err := h.uc.Create(r.Context(), p)
	if err != nil {
		writeProductErr(w, err)
		return
	}

	writePrintHandlerJSON(w, http.StatusOK, created)
}

func writeProductErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, productdom.ErrInvalidID):
		code = http.StatusBadRequest
	case errors.Is(err, productdom.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, productdom.ErrConflict):
		code = http.StatusConflict
	case errors.Is(err, printdom.ErrInvalidPrintLogProductionID):
		code = http.StatusBadRequest
	case errors.Is(err, printdom.ErrNotFound):
		code = http.StatusNotFound
	}

	writePrintHandlerJSONError(w, code, err.Error())
}

func writePrintHandlerJSONError(
	w http.ResponseWriter,
	status int,
	message string,
) {
	writePrintHandlerJSON(w, status, map[string]string{
		"error": message,
	})
}

func writePrintHandlerJSON(
	w http.ResponseWriter,
	status int,
	value any,
) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
