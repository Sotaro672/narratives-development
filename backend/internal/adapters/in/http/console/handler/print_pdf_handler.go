// backend/internal/adapters/in/http/console/handler/print_pdf_handler.go
package consoleHandler

import (
	"context"
	"errors"
	"mime"
	"net/http"
	"strconv"
	"strings"

	consolequery "narratives/internal/application/query/console"
	usecase "narratives/internal/application/usecase"
	printdom "narratives/internal/domain/print"
	productbpdom "narratives/internal/domain/productBlueprint"
	productiondom "narratives/internal/domain/production"
)

const printPDFPath = "/products/print-pdf"

// PrintPDFProductionAccessQuery は、署名チケットに含まれる companyId が
// 対象 Production にアクセス可能か確認するための最小ポートです。
// CompanyProductionQueryService がこの interface を満たします。
type PrintPDFProductionAccessQuery interface {
	GetProductionDetailByID(
		ctx context.Context,
		id string,
	) (consolequery.ProductionDetailDTO, error)
}

// PrintPDFLogQuery は PDF 生成対象の print_log を取得するための最小ポートです。
// PrintQueryService がこの interface を満たします。
type PrintPDFLogQuery interface {
	ListPrintLogsByProductionID(
		ctx context.Context,
		productionID string,
	) ([]consolequery.PrintLogForPrintDTO, error)
}

// PrintPDFHandler は短時間有効の署名チケットを検証し、
// QR PDF を application/pdf として直接返す public handler です。
//
// Firebase ID Token は URL に含めません。
// 認証済み endpoint で発行された ticket のみを認証情報として扱います。
type PrintPDFHandler struct {
	signer          *PrintPDFTicketSigner
	productionQuery PrintPDFProductionAccessQuery
	printQuery      PrintPDFLogQuery
}

func NewPrintPDFHandler(
	signer *PrintPDFTicketSigner,
	productionQuery PrintPDFProductionAccessQuery,
	printQuery PrintPDFLogQuery,
) http.Handler {
	return &PrintPDFHandler{
		signer:          signer,
		productionQuery: productionQuery,
		printQuery:      printQuery,
	}
}

func (h *PrintPDFHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.URL.Path != printPDFPath {
		writePrintPDFJSONError(
			w,
			http.StatusNotFound,
			"not_found",
		)
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set(
			"Allow",
			http.MethodGet,
		)

		writePrintPDFJSONError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
		)
		return
	}

	h.handleGet(
		w,
		r,
	)
}

func (h *PrintPDFHandler) handleGet(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h.signer == nil {
		writePrintPDFJSONError(
			w,
			http.StatusInternalServerError,
			"print_pdf_ticket_signer_not_configured",
		)
		return
	}

	if h.productionQuery == nil {
		writePrintPDFJSONError(
			w,
			http.StatusInternalServerError,
			"print_pdf_production_query_not_configured",
		)
		return
	}

	if h.printQuery == nil {
		writePrintPDFJSONError(
			w,
			http.StatusInternalServerError,
			"print_pdf_query_not_configured",
		)
		return
	}

	ticket :=
		strings.TrimSpace(
			r.URL.Query().Get(
				"ticket",
			),
		)

	if ticket == "" {
		writePrintPDFJSONError(
			w,
			http.StatusUnauthorized,
			"print_pdf_ticket_required",
		)
		return
	}

	claims, err :=
		h.signer.Verify(
			ticket,
		)
	if err != nil {
		h.writeTicketError(
			w,
			err,
		)
		return
	}

	// Public endpoint では AuthMiddleware が通らないため、
	// 署名済み claims の companyId を application context に復元する。
	//
	// CompanyProductionQueryService.GetProductionDetailByID が
	// Production → ProductBlueprint → CompanyID の境界を検証する。
	ctx :=
		usecase.WithCompanyID(
			r.Context(),
			claims.CompanyID,
		)

	if _, err :=
		h.productionQuery.GetProductionDetailByID(
			ctx,
			claims.ProductionID,
		); err != nil {
		h.writeProductionAccessError(
			w,
			err,
		)
		return
	}

	logs, err :=
		h.printQuery.ListPrintLogsByProductionID(
			ctx,
			claims.ProductionID,
		)
	if err != nil {
		h.writePrintLogError(
			w,
			err,
		)
		return
	}

	if len(logs) == 0 {
		writePrintPDFJSONError(
			w,
			http.StatusNotFound,
			"print_log_not_found",
		)
		return
	}

	pdfBytes, err :=
		BuildPrintPDF(
			logs,
		)
	if err != nil {
		writePrintPDFJSONError(
			w,
			http.StatusInternalServerError,
			"print_pdf_generation_failed",
		)
		return
	}

	if len(pdfBytes) == 0 {
		writePrintPDFJSONError(
			w,
			http.StatusInternalServerError,
			"print_pdf_is_empty",
		)
		return
	}

	filename :=
		buildPrintPDFFilename(
			claims.ProductionID,
		)

	contentDisposition :=
		mime.FormatMediaType(
			"inline",
			map[string]string{
				"filename": filename,
			},
		)

	if contentDisposition == "" {
		contentDisposition =
			`inline; filename="qr-output.pdf"`
	}

	// Chrome 標準 PDF Viewer が通常の HTTPS response として
	// 直接読み込めるよう application/pdf + inline で返す。
	w.Header().Set(
		"Content-Type",
		"application/pdf",
	)

	w.Header().Set(
		"Content-Disposition",
		contentDisposition,
	)

	// URL 自体が短時間有効の bearer credential に相当するため、
	// browser / intermediary cache へ保存しない。
	w.Header().Set(
		"Cache-Control",
		"private, no-store, max-age=0",
	)

	w.Header().Set(
		"Pragma",
		"no-cache",
	)

	w.Header().Set(
		"X-Content-Type-Options",
		"nosniff",
	)

	w.Header().Set(
		"Content-Length",
		strconv.Itoa(
			len(pdfBytes),
		),
	)

	w.WriteHeader(
		http.StatusOK,
	)

	_, _ =
		w.Write(
			pdfBytes,
		)
}

func (h *PrintPDFHandler) writeTicketError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrPrintPDFTicketExpired,
	):
		writePrintPDFJSONError(
			w,
			http.StatusUnauthorized,
			"print_pdf_ticket_expired",
		)

	case errors.Is(
		err,
		ErrPrintPDFTicketInvalid,
	):
		writePrintPDFJSONError(
			w,
			http.StatusUnauthorized,
			"print_pdf_ticket_invalid",
		)

	case errors.Is(
		err,
		ErrPrintPDFTicketSecretRequired,
	),
		errors.Is(
			err,
			ErrPrintPDFTicketSecretTooShort,
		),
		errors.Is(
			err,
			ErrPrintPDFTicketInvalidTTL,
		):
		writePrintPDFJSONError(
			w,
			http.StatusInternalServerError,
			"print_pdf_ticket_signer_invalid",
		)

	default:
		writePrintPDFJSONError(
			w,
			http.StatusUnauthorized,
			"print_pdf_ticket_invalid",
		)
	}
}

func (h *PrintPDFHandler) writeProductionAccessError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		productiondom.ErrNotFound,
	),
		errors.Is(
			err,
			productiondom.ErrInvalidID,
		),
		errors.Is(
			err,
			productbpdom.ErrNotFound,
		),
		errors.Is(
			err,
			productbpdom.ErrInvalidCompanyID,
		):
		// 他社 Production の存在を公開 endpoint から判別できないよう、
		// company mismatch も not found として扱う。
		writePrintPDFJSONError(
			w,
			http.StatusNotFound,
			"production_not_found",
		)

	case errors.Is(
		err,
		productbpdom.ErrInternal,
	):
		writePrintPDFJSONError(
			w,
			http.StatusInternalServerError,
			"production_access_check_failed",
		)

	default:
		writePrintPDFJSONError(
			w,
			http.StatusInternalServerError,
			"production_access_check_failed",
		)
	}
}

func (h *PrintPDFHandler) writePrintLogError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		printdom.ErrNotFound,
	):
		writePrintPDFJSONError(
			w,
			http.StatusNotFound,
			"print_log_not_found",
		)

	case errors.Is(
		err,
		printdom.ErrInvalidPrintLogProductionID,
	):
		writePrintPDFJSONError(
			w,
			http.StatusBadRequest,
			"invalid_production_id",
		)

	default:
		writePrintPDFJSONError(
			w,
			http.StatusInternalServerError,
			"print_log_load_failed",
		)
	}
}

func writePrintPDFJSONError(
	w http.ResponseWriter,
	status int,
	message string,
) {
	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	w.Header().Set(
		"Cache-Control",
		"no-store",
	)

	writeJSONError(
		w,
		status,
		message,
	)
}

func buildPrintPDFFilename(
	productionID string,
) string {
	const maxIDLength = 80

	productionID =
		strings.TrimSpace(
			productionID,
		)

	var builder strings.Builder

	for _, char := range productionID {
		if builder.Len() >= maxIDLength {
			break
		}

		switch {
		case char >= 'a' && char <= 'z':
			builder.WriteRune(
				char,
			)

		case char >= 'A' && char <= 'Z':
			builder.WriteRune(
				char,
			)

		case char >= '0' && char <= '9':
			builder.WriteRune(
				char,
			)

		case char == '-',
			char == '_':
			builder.WriteRune(
				char,
			)

		default:
			builder.WriteByte(
				'_',
			)
		}
	}

	safeID :=
		strings.Trim(
			builder.String(),
			"_",
		)

	if safeID == "" {
		return "qr-output.pdf"
	}

	return "qr-output-" +
		safeID +
		".pdf"
}
