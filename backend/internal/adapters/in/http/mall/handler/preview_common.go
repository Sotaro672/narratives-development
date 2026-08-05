// backend/internal/adapters/in/http/mall/handler/preview_common.go
package mallHandler

import (
	"context"
	"strings"

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

// Preview共通のdata生成
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
		addr := strings.TrimSpace(info.Token.ToAddress)
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
		"productBlueprintId":           info.ProductBlueprintID,
		"productBlueprintCategoryCode": info.ProductBlueprintCategoryCode,
		"productBlueprintCategoryKind": info.ProductBlueprintCategoryKind,
		"productBlueprintCategoryName": info.ProductBlueprintCategoryName,
		"productBlueprintCategory":     info.ProductBlueprintCategory,
		"productBlueprintPatch":        info.ProductBlueprintPatch,
		"categoryInputSchema":          info.CategoryInputSchema,

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

	tbID := strings.TrimSpace(info.Token.TokenBlueprintID)
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

	brandName := strings.TrimSpace(patch.BrandName)
	companyName := ""

	if nameR != nil {
		brandID := strings.TrimSpace(patch.BrandID)
		companyID := strings.TrimSpace(patch.CompanyID)

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
		TokenIcon:   strings.TrimSpace(patch.IconURL),
	}
}
