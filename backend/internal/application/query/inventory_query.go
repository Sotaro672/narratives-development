// backend/internal/application/query/inventory_query.go
package query

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	resolver "narratives/internal/application/resolver"
	invdom "narratives/internal/domain/inventory"
	pbdom "narratives/internal/domain/productBlueprint"
)

// ============================================================
// DTOs for InventoryDetail page (read-only)
// ============================================================

type InventoryDetailDTO struct {
	InventoryID           string                     `json:"inventoryId"`
	TokenBlueprintID      string                     `json:"tokenBlueprintId"`   // pbId query の場合は空
	ProductBlueprintID    string                     `json:"productBlueprintId"` // pbId query の場合に必ず入る
	ModelID               string                     `json:"modelId"`            // pbId query の場合は空
	ProductBlueprintPatch ProductBlueprintPatchDTO   `json:"productBlueprintPatch"`
	TokenBlueprint        TokenBlueprintSummaryDTO   `json:"tokenBlueprint"`
	ProductBlueprint      ProductBlueprintSummaryDTO `json:"productBlueprint"`
	Rows                  []InventoryRowDTO          `json:"rows"`
	TotalStock            int                        `json:"totalStock"`
	UpdatedAt             time.Time                  `json:"updatedAt"`
}

type TokenBlueprintSummaryDTO struct {
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`
	Symbol string `json:"symbol,omitempty"`
}

type ProductBlueprintSummaryDTO struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type ProductBlueprintPatchDTO struct {
	ProductName      *string             `json:"productName,omitempty"`
	BrandID          *string             `json:"brandId,omitempty"`
	ItemType         *pbdom.ItemType     `json:"itemType,omitempty"`
	Fit              *string             `json:"fit,omitempty"`
	Material         *string             `json:"material,omitempty"`
	Weight           *float64            `json:"weight,omitempty"`
	QualityAssurance *[]string           `json:"qualityAssurance,omitempty"`
	ProductIdTag     *pbdom.ProductIDTag `json:"productIdTag,omitempty"`
	AssigneeID       *string             `json:"assigneeId,omitempty"`
}

// InventoryCard 用（フロント側の命名に合わせる）
//   - token 列を左に追加
//   - modelCode -> modelNumber
//   - colorName -> color
//   - colorCode -> rgb（数値 or 変換可能な文字列を想定）
//     ※ JSON では数値(int)で返す（rgbIntToHex が使える）
type InventoryRowDTO struct {
	Token       string `json:"token"`
	ModelNumber string `json:"modelNumber"`
	Size        string `json:"size"`
	Color       string `json:"color"`
	RGB         *int   `json:"rgb,omitempty"`
	Stock       int    `json:"stock"`
}

// ============================================================
// DTOs (Inventory Management List)
// - 列: プロダクト名 / トークン名 / 型番 / 在庫数
// ============================================================

type InventoryManagementRowDTO struct {
	ProductBlueprintID string `json:"productBlueprintId"`
	ProductName        string `json:"productName"`
	TokenName          string `json:"tokenName"`
	ModelNumber        string `json:"modelNumber"`
	Stock              int    `json:"stock"`
}

// ============================================================
// Query Service (Read-model assembler)
// ============================================================

type InventoryQuery struct {
	invRepo       inventoryReader
	pbRepo        productBlueprintIDsByCompanyReader // companyId -> その会社の productBlueprintId 一覧
	pbPatchReader productBlueprintPatchReader
	prReader      productReader
	nameResolver  *resolver.NameResolver // ✅ tokenName / modelNumber / productName 解決
}

func NewInventoryQuery(
	invRepo inventoryReader,
	pbRepo productBlueprintIDsByCompanyReader,
	pbPatchReader productBlueprintPatchReader,
	prReader productReader,
	nameResolver *resolver.NameResolver,
) *InventoryQuery {
	return &InventoryQuery{
		invRepo:       invRepo,
		pbRepo:        pbRepo,
		pbPatchReader: pbPatchReader,
		prReader:      prReader,
		nameResolver:  nameResolver,
	}
}

// ------------------------------------------------------------
// A) inventoryId(docId) を直接指定して Detail DTO
// ------------------------------------------------------------

func (q *InventoryQuery) GetDetail(ctx context.Context, inventoryID string) (InventoryDetailDTO, error) {
	if q == nil || q.invRepo == nil {
		return InventoryDetailDTO{}, errors.New("inventory query/invRepo is nil")
	}
	id := strings.TrimSpace(inventoryID)
	if id == "" {
		return InventoryDetailDTO{}, invdom.ErrInvalidMintID
	}

	m, err := q.invRepo.GetByID(ctx, id)
	if err != nil {
		return InventoryDetailDTO{}, err
	}

	return q.buildDetailFromSingleMint(ctx, m)
}

// ------------------------------------------------------------
// B) productBlueprintId で inventories を引いて rows を返す
//    GET /inventory?productBlueprintId={pbId}
// ------------------------------------------------------------

func (q *InventoryQuery) GetDetailByProductBlueprintID(ctx context.Context, productBlueprintID string) (InventoryDetailDTO, error) {
	if q == nil || q.invRepo == nil {
		return InventoryDetailDTO{}, errors.New("inventory query/invRepo is nil")
	}

	pbID := strings.TrimSpace(productBlueprintID)
	if pbID == "" {
		return InventoryDetailDTO{}, invdom.ErrInvalidProductBlueprintID
	}

	mints, err := q.invRepo.ListByProductBlueprintID(ctx, pbID)
	if err != nil {
		return InventoryDetailDTO{}, err
	}
	if len(mints) == 0 {
		return InventoryDetailDTO{}, invdom.ErrNotFound
	}

	// ProductBlueprint patch は 1回だけ
	pbSummary := ProductBlueprintSummaryDTO{ID: pbID}
	pbPatchDTO := ProductBlueprintPatchDTO{}
	if q.pbPatchReader != nil {
		patch, err := q.pbPatchReader.GetPatchByID(ctx, pbID)
		if err != nil {
			return InventoryDetailDTO{}, err
		}
		pbPatchDTO = ProductBlueprintPatchDTO{
			ProductName:      patch.ProductName,
			BrandID:          patch.BrandID,
			ItemType:         patch.ItemType,
			Fit:              patch.Fit,
			Material:         patch.Material,
			Weight:           patch.Weight,
			QualityAssurance: patch.QualityAssurance,
			ProductIdTag:     patch.ProductIdTag,
			AssigneeID:       patch.AssigneeID,
		}
		if patch.ProductName != nil && strings.TrimSpace(*patch.ProductName) != "" {
			pbSummary.Name = strings.TrimSpace(*patch.ProductName)
		}
	}

	type rowKey struct {
		token       string
		modelNumber string
		size        string
		color       string
		rgb         int // -1 = nil
	}

	group := map[rowKey]*InventoryRowDTO{}
	maxUpdated := time.Time{}

	for _, m := range mints {
		t := m.UpdatedAt
		if t.IsZero() {
			t = m.CreatedAt
		}
		if maxUpdated.IsZero() || t.After(maxUpdated) {
			maxUpdated = t
		}

		// ✅ tokenName 解決（NameResolver）
		tokenLabel := q.resolveTokenName(ctx, m.TokenBlueprintID)
		if tokenLabel == "" {
			tokenLabel = strings.TrimSpace(m.TokenBlueprintID)
		}
		if tokenLabel == "" {
			tokenLabel = "-"
		}

		// ✅ modelNumber 解決（NameResolver）
		defaultModelNumber := q.resolveModelNumber(ctx, m.ModelID)
		if defaultModelNumber == "" {
			defaultModelNumber = strings.TrimSpace(m.ModelID)
		}
		if defaultModelNumber == "" {
			defaultModelNumber = "-"
		}

		if q.prReader != nil && len(m.Products) > 0 {
			prods, err := q.prReader.ListByIDs(ctx, m.Products)
			if err != nil {
				return InventoryDetailDTO{}, err
			}

			for _, p := range prods {
				modelNumber := strings.TrimSpace(p.ModelCode)
				if modelNumber == "" {
					modelNumber = defaultModelNumber
				}
				if modelNumber == "" {
					modelNumber = "-"
				}

				size := strings.TrimSpace(p.Size)
				if size == "" {
					size = "-"
				}

				color := strings.TrimSpace(p.ColorName)
				if color == "" {
					color = "-"
				}

				rgbPtr := parseColorCodeToRGBPtr(p.ColorCode)
				rgbKey := -1
				if rgbPtr != nil {
					rgbKey = *rgbPtr
				}

				k := rowKey{
					token:       tokenLabel,
					modelNumber: modelNumber,
					size:        size,
					color:       color,
					rgb:         rgbKey,
				}

				if group[k] == nil {
					group[k] = &InventoryRowDTO{
						Token:       tokenLabel,
						ModelNumber: modelNumber,
						Size:        size,
						Color:       color,
						RGB:         rgbPtr,
						Stock:       0,
					}
				}
				group[k].Stock++
			}
			continue
		}

		stock := m.Accumulation
		if stock <= 0 {
			stock = len(m.Products)
		}
		if stock < 0 {
			stock = 0
		}

		k := rowKey{
			token:       tokenLabel,
			modelNumber: defaultModelNumber,
			size:        "-",
			color:       "-",
			rgb:         -1,
		}
		if group[k] == nil {
			group[k] = &InventoryRowDTO{
				Token:       tokenLabel,
				ModelNumber: defaultModelNumber,
				Size:        "-",
				Color:       "-",
				RGB:         nil,
				Stock:       0,
			}
		}
		group[k].Stock += stock
	}

	rows := make([]InventoryRowDTO, 0, len(group))
	for _, v := range group {
		rows = append(rows, *v)
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Token != rows[j].Token {
			return rows[i].Token < rows[j].Token
		}
		if rows[i].ModelNumber != rows[j].ModelNumber {
			return rows[i].ModelNumber < rows[j].ModelNumber
		}
		if rows[i].Size != rows[j].Size {
			return rows[i].Size < rows[j].Size
		}
		if rows[i].Color != rows[j].Color {
			return rows[i].Color < rows[j].Color
		}
		ri := -1
		if rows[i].RGB != nil {
			ri = *rows[i].RGB
		}
		rj := -1
		if rows[j].RGB != nil {
			rj = *rows[j].RGB
		}
		return ri < rj
	})

	total := 0
	for _, r := range rows {
		total += r.Stock
	}

	return InventoryDetailDTO{
		InventoryID: pbID, // 互換

		TokenBlueprintID:   "",
		ProductBlueprintID: pbID,
		ModelID:            "",

		ProductBlueprintPatch: pbPatchDTO,

		TokenBlueprint:   TokenBlueprintSummaryDTO{},
		ProductBlueprint: pbSummary,

		Rows:       rows,
		TotalStock: total,
		UpdatedAt:  maxUpdated,
	}, nil
}

// ============================================================
// ✅ NEW: currentMember.companyId -> productBlueprintIds -> inventories list
// ============================================================

func (q *InventoryQuery) ListByCurrentCompany(ctx context.Context) ([]InventoryManagementRowDTO, error) {
	if q == nil || q.invRepo == nil || q.pbRepo == nil {
		return nil, errors.New("inventory query repositories are not configured")
	}

	// NOTE: companyIDFromContext は package query 内で 1箇所だけ定義してください（重複禁止）。
	companyID := companyIDFromContext(ctx)
	if companyID == "" {
		return nil, errors.New("companyId is missing in context")
	}

	pbIDs, err := q.pbRepo.ListIDsByCompanyID(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if len(pbIDs) == 0 {
		return []InventoryManagementRowDTO{}, nil
	}

	type key struct {
		pbID      string
		tokenName string
		modelNum  string
	}

	group := map[key]int{}
	productNameCache := map[string]string{}

	for _, pbID := range pbIDs {
		pbID = strings.TrimSpace(pbID)
		if pbID == "" {
			continue
		}

		if _, ok := productNameCache[pbID]; !ok {
			name := q.resolveProductName(ctx, pbID)
			if name == "" {
				name = pbID
			}
			productNameCache[pbID] = name
		}

		mints, err := q.invRepo.ListByProductBlueprintID(ctx, pbID)
		if err != nil {
			return nil, err
		}
		if len(mints) == 0 {
			continue
		}

		for _, m := range mints {
			tokenName := q.resolveTokenName(ctx, m.TokenBlueprintID)
			if tokenName == "" {
				tokenName = strings.TrimSpace(m.TokenBlueprintID)
			}
			if tokenName == "" {
				tokenName = "-"
			}

			modelNumber := q.resolveModelNumber(ctx, m.ModelID)
			if modelNumber == "" {
				modelNumber = strings.TrimSpace(m.ModelID)
			}
			if modelNumber == "" {
				modelNumber = "-"
			}

			stock := m.Accumulation
			if stock <= 0 {
				stock = len(m.Products)
			}
			if stock < 0 {
				stock = 0
			}

			k := key{pbID: pbID, tokenName: tokenName, modelNum: modelNumber}
			group[k] += stock
		}
	}

	rows := make([]InventoryManagementRowDTO, 0, len(group))
	for k, stock := range group {
		rows = append(rows, InventoryManagementRowDTO{
			ProductBlueprintID: k.pbID,
			ProductName:        productNameCache[k.pbID],
			TokenName:          k.tokenName,
			ModelNumber:        k.modelNum,
			Stock:              stock,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].ProductName != rows[j].ProductName {
			return rows[i].ProductName < rows[j].ProductName
		}
		if rows[i].TokenName != rows[j].TokenName {
			return rows[i].TokenName < rows[j].TokenName
		}
		if rows[i].ModelNumber != rows[j].ModelNumber {
			return rows[i].ModelNumber < rows[j].ModelNumber
		}
		return rows[i].Stock < rows[j].Stock
	})

	return rows, nil
}

// ------------------------------------------------------------
// internal: 単一 inventory(mint) 用の DTO
// ------------------------------------------------------------

func (q *InventoryQuery) buildDetailFromSingleMint(ctx context.Context, m invdom.Mint) (InventoryDetailDTO, error) {
	tokenLabel := q.resolveTokenName(ctx, m.TokenBlueprintID)
	if tokenLabel == "" {
		tokenLabel = strings.TrimSpace(m.TokenBlueprintID)
	}

	tb := TokenBlueprintSummaryDTO{ID: strings.TrimSpace(m.TokenBlueprintID)}

	pbID := strings.TrimSpace(m.ProductBlueprintID)
	pbSummary := ProductBlueprintSummaryDTO{ID: pbID}
	pbPatchDTO := ProductBlueprintPatchDTO{}
	if q.pbPatchReader != nil && pbID != "" {
		patch, err := q.pbPatchReader.GetPatchByID(ctx, pbID)
		if err != nil {
			return InventoryDetailDTO{}, err
		}
		pbPatchDTO = ProductBlueprintPatchDTO{
			ProductName:      patch.ProductName,
			BrandID:          patch.BrandID,
			ItemType:         patch.ItemType,
			Fit:              patch.Fit,
			Material:         patch.Material,
			Weight:           patch.Weight,
			QualityAssurance: patch.QualityAssurance,
			ProductIdTag:     patch.ProductIdTag,
			AssigneeID:       patch.AssigneeID,
		}
		if patch.ProductName != nil && strings.TrimSpace(*patch.ProductName) != "" {
			pbSummary.Name = strings.TrimSpace(*patch.ProductName)
		}
	}

	defaultModelNumber := q.resolveModelNumber(ctx, m.ModelID)
	if defaultModelNumber == "" {
		defaultModelNumber = strings.TrimSpace(m.ModelID)
	}
	if defaultModelNumber == "" {
		defaultModelNumber = "-"
	}

	rows := make([]InventoryRowDTO, 0, 16)
	if q.prReader != nil && len(m.Products) > 0 {
		prods, err := q.prReader.ListByIDs(ctx, m.Products)
		if err != nil {
			return InventoryDetailDTO{}, err
		}

		type key struct {
			modelNumber string
			size        string
			color       string
			rgb         int
		}
		group := map[key]*InventoryRowDTO{}

		for _, p := range prods {
			modelNumber := strings.TrimSpace(p.ModelCode)
			if modelNumber == "" {
				modelNumber = defaultModelNumber
			}
			if modelNumber == "" {
				modelNumber = "-"
			}

			size := strings.TrimSpace(p.Size)
			if size == "" {
				size = "-"
			}

			color := strings.TrimSpace(p.ColorName)
			if color == "" {
				color = "-"
			}

			rgbPtr := parseColorCodeToRGBPtr(p.ColorCode)
			rgbKey := -1
			if rgbPtr != nil {
				rgbKey = *rgbPtr
			}

			k := key{modelNumber: modelNumber, size: size, color: color, rgb: rgbKey}

			if group[k] == nil {
				group[k] = &InventoryRowDTO{
					Token:       tokenLabel,
					ModelNumber: modelNumber,
					Size:        size,
					Color:       color,
					RGB:         rgbPtr,
					Stock:       0,
				}
			}
			group[k].Stock++
		}

		for _, v := range group {
			rows = append(rows, *v)
		}
	} else {
		stock := m.Accumulation
		if stock <= 0 {
			stock = len(m.Products)
		}
		rows = append(rows, InventoryRowDTO{
			Token:       tokenLabel,
			ModelNumber: defaultModelNumber,
			Size:        "-",
			Color:       "-",
			RGB:         nil,
			Stock:       stock,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].ModelNumber != rows[j].ModelNumber {
			return rows[i].ModelNumber < rows[j].ModelNumber
		}
		if rows[i].Size != rows[j].Size {
			return rows[i].Size < rows[j].Size
		}
		if rows[i].Color != rows[j].Color {
			return rows[i].Color < rows[j].Color
		}
		ri := -1
		if rows[i].RGB != nil {
			ri = *rows[i].RGB
		}
		rj := -1
		if rows[j].RGB != nil {
			rj = *rows[j].RGB
		}
		return ri < rj
	})

	total := 0
	for _, r := range rows {
		total += r.Stock
	}

	return InventoryDetailDTO{
		InventoryID: strings.TrimSpace(m.ID),

		TokenBlueprintID:   strings.TrimSpace(m.TokenBlueprintID),
		ProductBlueprintID: pbID,
		ModelID:            strings.TrimSpace(m.ModelID),

		ProductBlueprintPatch: pbPatchDTO,

		TokenBlueprint:   tb,
		ProductBlueprint: pbSummary,

		Rows:       rows,
		TotalStock: total,

		UpdatedAt: m.UpdatedAt,
	}, nil
}

// ============================================================
// helpers
// ============================================================

func (q *InventoryQuery) resolveTokenName(ctx context.Context, tokenBlueprintID string) string {
	if q == nil || q.nameResolver == nil {
		return ""
	}
	return strings.TrimSpace(q.nameResolver.ResolveTokenName(ctx, tokenBlueprintID))
}

func (q *InventoryQuery) resolveModelNumber(ctx context.Context, modelVariationID string) string {
	if q == nil || q.nameResolver == nil {
		return ""
	}
	return strings.TrimSpace(q.nameResolver.ResolveModelNumber(ctx, modelVariationID))
}

func (q *InventoryQuery) resolveProductName(ctx context.Context, productBlueprintID string) string {
	if q == nil || q.nameResolver == nil {
		return ""
	}
	return strings.TrimSpace(q.nameResolver.ResolveProductName(ctx, productBlueprintID))
}

func parseColorCodeToRGBPtr(colorCode string) *int {
	s := strings.TrimSpace(colorCode)
	if s == "" {
		return nil
	}

	s = strings.TrimPrefix(s, "#")
	s = strings.TrimPrefix(strings.ToLower(s), "0x")

	if len(s) == 6 {
		if n, err := strconv.ParseInt(s, 16, 32); err == nil {
			v := int(n)
			return &v
		}
	}

	if n, err := strconv.ParseInt(s, 10, 32); err == nil {
		v := int(n)
		return &v
	}

	return nil
}

// ============================================================
// Minimal readers (ports)
// ============================================================

type inventoryReader interface {
	GetByID(ctx context.Context, id string) (invdom.Mint, error)
	ListByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]invdom.Mint, error)
}

type productBlueprintIDsByCompanyReader interface {
	ListIDsByCompanyID(ctx context.Context, companyID string) ([]string, error)
}

type productBlueprintPatchReader interface {
	GetPatchByID(ctx context.Context, id string) (pbdom.Patch, error)
}

type productReader interface {
	ListByIDs(ctx context.Context, ids []string) ([]ProductView, error)
}

type ProductView struct {
	ID        string
	ModelCode string
	Size      string
	ColorName string
	ColorCode string
}
