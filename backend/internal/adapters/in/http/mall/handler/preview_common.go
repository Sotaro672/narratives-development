// backend/internal/adapters/in/http/mall/handler/preview_common.go
package mallHandler

import (
	"context"
	"errors"
	"net/http"

	dto "narratives/internal/application/query/mall/dto"
	sharedquery "narratives/internal/application/query/shared"
	tokenbpdom "narratives/internal/domain/tokenBlueprint"
)

// Preview共通レスポンスDTO
type tokenBlueprintPatchDTO struct {
	ID          string `json:"id"`
	TokenName   string `json:"tokenName"`
	Symbol      string `json:"symbol"`
	BrandName   string `json:"brandName,omitempty"`
	CompanyName string `json:"companyName,omitempty"`
	Description string `json:"description,omitempty"`
	TokenIcon   string `json:"tokenIcon,omitempty"`
}

// validatePreviewGETRequest はPreview API共通のHTTPメソッド検証を行います。
//
// falseを返した場合は、この関数内でレスポンスを書き込み済みです。
func validatePreviewGETRequest(
	w http.ResponseWriter,
	r *http.Request,
) bool {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return false
	}

	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"error": "method not allowed",
		})
		return false
	}

	return true
}

// resolvePreviewModelInfoFromRequest はPreview API共通の入力検証と
// PreviewModelInfo取得処理を行います。
//
// extraResponseFieldsには、Preview Me固有のavatarIdなど、
// エラー応答へ追加したい値を渡します。
//
// falseを返した場合は、この関数内でレスポンスを書き込み済みです。
func resolvePreviewModelInfoFromRequest(
	w http.ResponseWriter,
	r *http.Request,
	q PreviewQuery,
	extraResponseFields map[string]any,
) (*dto.PreviewModelInfo, bool) {
	if q == nil {
		writePreviewError(
			w,
			http.StatusInternalServerError,
			"preview query not configured",
			"",
			extraResponseFields,
		)
		return nil, false
	}

	productID := r.URL.Query().Get("productId")
	if productID == "" {
		writePreviewError(
			w,
			http.StatusBadRequest,
			"productId is required",
			"",
			extraResponseFields,
		)
		return nil, false
	}

	info, err := q.ResolveModelInfoByProductID(
		r.Context(),
		productID,
	)
	if err != nil {
		switch {
		case isNotFoundLike(err):
			writePreviewError(
				w,
				http.StatusNotFound,
				"not found",
				productID,
				extraResponseFields,
			)
		case errors.Is(err, context.Canceled),
			errors.Is(err, context.DeadlineExceeded):
			writePreviewError(
				w,
				http.StatusRequestTimeout,
				"request canceled",
				productID,
				extraResponseFields,
			)
		default:
			writePreviewError(
				w,
				http.StatusInternalServerError,
				"resolve failed",
				productID,
				extraResponseFields,
			)
		}

		return nil, false
	}

	if info == nil {
		writePreviewError(
			w,
			http.StatusInternalServerError,
			"resolve failed (nil result)",
			productID,
			extraResponseFields,
		)
		return nil, false
	}

	return info, true
}

func writePreviewError(
	w http.ResponseWriter,
	status int,
	message string,
	productID string,
	extraResponseFields map[string]any,
) {
	response := make(
		map[string]any,
		len(extraResponseFields)+2,
	)

	for key, value := range extraResponseFields {
		response[key] = value
	}

	response["error"] = message

	if productID != "" {
		response["productId"] = productID
	}

	writeJSON(w, status, response)
}

// buildPreviewData はPreview API共通のレスポンスdataを生成します。
func buildPreviewData(
	ctx context.Context,
	info *dto.PreviewModelInfo,
	ownerQ *sharedquery.OwnerResolveQuery,
	tbRepo TokenBlueprintPatchReader,
	nameR PreviewNameResolver,
) map[string]any {
	if info == nil {
		return nil
	}

	// owner resolve（best-effort）
	if info.Owner == nil && ownerQ != nil && info.Token != nil {
		addr := info.Token.ToAddress
		if addr != "" {
			resolvedOwner, err := ownerQ.Resolve(ctx, addr)
			if err == nil {
				info.Owner = resolvedOwner
			}
		}
	}

	// tokenBlueprint patch（best-effort）
	tbPatch := resolveTokenBlueprintPatch(ctx, info, tbRepo)
	tbDTO := buildTokenBlueprintPatchDTO(ctx, tbPatch, nameR)

	return map[string]any{
		"productId":   info.ProductID,
		"modelId":     info.ModelID,
		"modelKind":   info.ModelKind,
		"modelNumber": info.ModelNumber,
		"modelLabel":  info.ModelLabel,

		// apparel
		"size":         info.Size,
		"color":        info.Color,
		"rgb":          info.RGB,
		"measurements": info.Measurements,

		// alcohol
		"volumeValue": info.VolumeValue,
		"volumeUnit":  info.VolumeUnit,

		// category / productBlueprint
		"productBlueprintId": info.ProductBlueprintID,
		"productBlueprintCategoryPath": append(
			[]string(nil),
			info.ProductBlueprintCategoryPath...,
		),
		"productBlueprintPatch": info.ProductBlueprintPatch,
		"categoryInputSchema":   info.CategoryInputSchema,

		// display
		"brandName":   info.BrandName,
		"companyName": info.CompanyName,

		// token / owner / transfer
		"token":               info.Token,
		"owner":               info.Owner,
		"transfers":           info.Transfers,
		"tokenBlueprintPatch": tbDTO,
	}
}

func resolveTokenBlueprintPatch(
	ctx context.Context,
	info *dto.PreviewModelInfo,
	tbRepo TokenBlueprintPatchReader,
) *tokenbpdom.Patch {
	if info == nil {
		return nil
	}

	if info.TokenBlueprintPatch != nil {
		return info.TokenBlueprintPatch
	}

	if tbRepo == nil || info.Token == nil {
		return nil
	}

	tbID := info.Token.TokenBlueprintID
	if tbID == "" {
		return nil
	}

	patch, err := tbRepo.GetPatchByID(ctx, tbID)
	if err != nil {
		return nil
	}

	return &patch
}

func buildTokenBlueprintPatchDTO(
	ctx context.Context,
	patch *tokenbpdom.Patch,
	nameR PreviewNameResolver,
) *tokenBlueprintPatchDTO {
	if patch == nil {
		return nil
	}

	brandName := patch.BrandName
	companyName := ""

	if nameR != nil {
		brandID := patch.BrandID
		companyID := patch.CompanyID

		if brandName == "" && brandID != "" {
			brandName = nameR.ResolveBrandName(ctx, brandID)
		}

		if companyID != "" {
			companyName = nameR.ResolveCompanyName(ctx, companyID)
		}

		if companyName == "" && brandID != "" {
			brandCompanyID := nameR.ResolveBrandCompanyID(ctx, brandID)
			if brandCompanyID != "" {
				companyName = nameR.ResolveCompanyName(ctx, brandCompanyID)
			}
		}
	}

	return &tokenBlueprintPatchDTO{
		ID:          patch.ID,
		TokenName:   patch.TokenName,
		Symbol:      patch.Symbol,
		BrandName:   brandName,
		CompanyName: companyName,
		Description: patch.Description,
		TokenIcon:   patch.IconURL,
	}
}
