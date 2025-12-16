// backend/internal/application/query/inventory_query.go
package query

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	invdom "narratives/internal/domain/inventory"
	pbdom "narratives/internal/domain/productBlueprint"
)

// ============================================================
// DTOs for InventoryDetail page (read-only)
// ============================================================

type InventoryDetailDTO struct {
	// 互換のため残す（pbId query の場合は pbId を入れる）
	InventoryID string `json:"inventoryId"`

	TokenBlueprintID   string `json:"tokenBlueprintId"`   // pbId query の場合は空
	ProductBlueprintID string `json:"productBlueprintId"` // pbId query の場合に必ず入る
	ModelID            string `json:"modelId"`            // pbId query の場合は空

	ProductBlueprintPatch ProductBlueprintPatchDTO `json:"productBlueprintPatch"`

	// 表示用途の summary（pbId query の場合、token は rows の token 列で表現）
	TokenBlueprint   TokenBlueprintSummaryDTO   `json:"tokenBlueprint"`
	ProductBlueprint ProductBlueprintSummaryDTO `json:"productBlueprint"`

	Rows       []InventoryRowDTO `json:"rows"`
	TotalStock int               `json:"totalStock"`

	UpdatedAt time.Time `json:"updatedAt"`
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
//     ※ JSON では数値にして返す（rgbIntToHex が使える）
type InventoryRowDTO struct {
	Token       string `json:"token"`
	ModelNumber string `json:"modelNumber"`
	Size        string `json:"size"`
	Color       string `json:"color"`
	RGB         *int   `json:"rgb,omitempty"`
	Stock       int    `json:"stock"`
}

// ============================================================
// NEW: companyId から PB -> inventories を引く（一覧用）
// ============================================================

type CompanyInventoryListDTO struct {
	CompanyID string               `json:"companyId"`
	Items     []InventoryDetailDTO `json:"items"`
	Count     int                  `json:"count"`
	UpdatedAt time.Time            `json:"updatedAt"`
}

// ============================================================
// Query Service (Read-model assembler)
// ============================================================

type InventoryQuery struct {
	// inventories
	invRepo inventoryReader

	// token label resolve (optional)
	tbReader tokenBlueprintReader

	// product blueprint patch (optional but recommended)
	pbPatchReader productBlueprintPatchReader

	// products resolve (optional)
	prReader productReader

	// company -> productBlueprintIds (optional; for ListByCompanyID)
	pbIDsReader companyProductBlueprintIDsReader
}

// 互換: 既存 DI を壊さないコンストラクタ
func NewInventoryQuery(
	invRepo inventoryReader,
	tbReader tokenBlueprintReader,
	pbPatchReader productBlueprintPatchReader,
	prReader productReader,
) *InventoryQuery {
	return &InventoryQuery{
		invRepo:       invRepo,
		tbReader:      tbReader,
		pbPatchReader: pbPatchReader,
		prReader:      prReader,
		pbIDsReader:   nil,
	}
}

// ✅ 追加: company -> pbIDs を使う版（必要な場合だけ DI で採用）
func NewInventoryQueryWithCompany(
	invRepo inventoryReader,
	tbReader tokenBlueprintReader,
	pbPatchReader productBlueprintPatchReader,
	prReader productReader,
	pbIDsReader companyProductBlueprintIDsReader,
) *InventoryQuery {
	return &InventoryQuery{
		invRepo:       invRepo,
		tbReader:      tbReader,
		pbPatchReader: pbPatchReader,
		prReader:      prReader,
		pbIDsReader:   pbIDsReader,
	}
}

// ------------------------------------------------------------
// A) 既存: inventoryId(docId) を直接指定して Detail DTO を組み立てる
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
// ✅ B) 期待値: productBlueprintId で inventories を引いて rows を返す
//    GET /inventory?productBlueprintId={pbId}
// ------------------------------------------------------------

func (q *InventoryQuery) GetDetailByProductBlueprintID(
	ctx context.Context,
	productBlueprintID string,
) (InventoryDetailDTO, error) {
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

	// tokenBlueprintId -> token表示名 をキャッシュ
	tokenLabelCache := map[string]string{}
	tokenSummaryCache := map[string]TokenBlueprintSummaryDTO{}

	resolveTokenLabel := func(tokenBlueprintID string) (label string, summary TokenBlueprintSummaryDTO, err error) {
		tbID := strings.TrimSpace(tokenBlueprintID)
		if tbID == "" {
			return "", TokenBlueprintSummaryDTO{}, nil
		}

		if v, ok := tokenLabelCache[tbID]; ok {
			return v, tokenSummaryCache[tbID], nil
		}

		label = tbID
		summary = TokenBlueprintSummaryDTO{ID: tbID}

		if q.tbReader != nil {
			view, e := q.tbReader.GetByID(ctx, tbID)
			if e != nil {
				return "", TokenBlueprintSummaryDTO{}, e
			}
			summary = TokenBlueprintSummaryDTO(view)
			if strings.TrimSpace(view.Name) != "" {
				label = strings.TrimSpace(view.Name)
			}
		}

		tokenLabelCache[tbID] = label
		tokenSummaryCache[tbID] = summary
		return label, summary, nil
	}

	// rows を全 inventories から集計
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
		// updatedAt の最大を採用（fallback createdAt）
		t := m.UpdatedAt
		if t.IsZero() {
			t = m.CreatedAt
		}
		if maxUpdated.IsZero() || t.After(maxUpdated) {
			maxUpdated = t
		}

		tokenLabel, _, err := resolveTokenLabel(m.TokenBlueprintID)
		if err != nil {
			return InventoryDetailDTO{}, err
		}
		if tokenLabel == "" {
			tokenLabel = "-"
		}

		// products を引けるなら variant 単位で group
		if q.prReader != nil && len(m.Products) > 0 {
			prods, err := q.prReader.ListByIDs(ctx, m.Products)
			if err != nil {
				return InventoryDetailDTO{}, err
			}

			for _, p := range prods {
				modelNumber := strings.TrimSpace(p.ModelNumber)
				if modelNumber == "" {
					modelNumber = strings.TrimSpace(m.ModelID)
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

				rgbPtr := parseColorCodeToRGBPtr(p.RGB)
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

		// fallback（product を引けない場合）
		modelNumber := strings.TrimSpace(m.ModelID)
		if modelNumber == "" {
			modelNumber = "-"
		}

		stock := m.Accumulation
		if stock <= 0 {
			stock = len(m.Products)
		}
		if stock <= 0 {
			stock = 0
		}

		k := rowKey{
			token:       tokenLabel,
			modelNumber: modelNumber,
			size:        "-",
			color:       "-",
			rgb:         -1,
		}
		if group[k] == nil {
			group[k] = &InventoryRowDTO{
				Token:       tokenLabel,
				ModelNumber: modelNumber,
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

	// pbId query の場合、上段 tokenBlueprint/modelId は意味を持たないので空にする
	dto := InventoryDetailDTO{
		InventoryID: pbID, // 互換: フロントが inventoryId と呼んでいるため pbID を詰める

		TokenBlueprintID:   "",
		ProductBlueprintID: pbID,
		ModelID:            "",

		ProductBlueprintPatch: pbPatchDTO,

		TokenBlueprint:   TokenBlueprintSummaryDTO{}, // rows 内 token 列で表現
		ProductBlueprint: pbSummary,

		Rows:       rows,
		TotalStock: total,

		UpdatedAt: maxUpdated,
	}
	return dto, nil
}

// ------------------------------------------------------------
// ✅ companyId -> productBlueprintIds -> inventories を list（一覧用）
// ------------------------------------------------------------

func (q *InventoryQuery) ListByCompanyID(ctx context.Context, companyID string) (CompanyInventoryListDTO, error) {
	if q == nil {
		return CompanyInventoryListDTO{}, errors.New("inventory query is nil")
	}
	if q.pbIDsReader == nil {
		return CompanyInventoryListDTO{}, errors.New("inventory query/pbIDsReader is nil")
	}

	cid := strings.TrimSpace(companyID)
	if cid == "" {
		return CompanyInventoryListDTO{}, errors.New("companyId is empty")
	}

	pbIDs, err := q.pbIDsReader.ListIDsByCompany(ctx, cid)
	if err != nil {
		return CompanyInventoryListDTO{}, err
	}
	if len(pbIDs) == 0 {
		return CompanyInventoryListDTO{
			CompanyID: cid,
			Items:     []InventoryDetailDTO{},
			Count:     0,
			UpdatedAt: time.Time{},
		}, nil
	}

	items := make([]InventoryDetailDTO, 0, len(pbIDs))
	maxUpdated := time.Time{}

	for _, raw := range pbIDs {
		pbID := strings.TrimSpace(raw)
		if pbID == "" {
			continue
		}

		dto, err := q.GetDetailByProductBlueprintID(ctx, pbID)
		if err != nil {
			// inventory 無しはスキップ（一覧用途）
			if errors.Is(err, invdom.ErrNotFound) {
				continue
			}
			return CompanyInventoryListDTO{}, err
		}

		t := dto.UpdatedAt
		if !t.IsZero() && (maxUpdated.IsZero() || t.After(maxUpdated)) {
			maxUpdated = t
		}
		items = append(items, dto)
	}

	// updated desc -> pbId
	sort.Slice(items, func(i, j int) bool {
		if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		return items[i].ProductBlueprintID < items[j].ProductBlueprintID
	})

	return CompanyInventoryListDTO{
		CompanyID: cid,
		Items:     items,
		Count:     len(items),
		UpdatedAt: maxUpdated,
	}, nil
}

// ------------------------------------------------------------
// internal: 単一 inventory(mint) 用の DTO
// ------------------------------------------------------------

func (q *InventoryQuery) buildDetailFromSingleMint(ctx context.Context, m invdom.Mint) (InventoryDetailDTO, error) {
	// token
	tokenLabel := strings.TrimSpace(m.TokenBlueprintID)
	tb := TokenBlueprintSummaryDTO{ID: tokenLabel}
	if q.tbReader != nil && tokenLabel != "" {
		v, err := q.tbReader.GetByID(ctx, tokenLabel)
		if err != nil {
			return InventoryDetailDTO{}, err
		}
		tb = TokenBlueprintSummaryDTO(v)
		if strings.TrimSpace(v.Name) != "" {
			tokenLabel = strings.TrimSpace(v.Name)
		}
	}

	// product blueprint patch
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

	// rows
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
			modelNumber := strings.TrimSpace(p.ModelNumber)
			if modelNumber == "" {
				modelNumber = strings.TrimSpace(m.ModelID)
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

			rgbPtr := parseColorCodeToRGBPtr(p.RGB)
			rgbKey := -1
			if rgbPtr != nil {
				rgbKey = *rgbPtr
			}

			k := key{
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

		for _, v := range group {
			rows = append(rows, *v)
		}
	} else {
		modelNumber := strings.TrimSpace(m.ModelID)
		if modelNumber == "" {
			modelNumber = "-"
		}
		stock := m.Accumulation
		if stock <= 0 {
			stock = len(m.Products)
		}
		rows = append(rows, InventoryRowDTO{
			Token:       tokenLabel,
			ModelNumber: modelNumber,
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

// ------------------------------------------------------------
// helpers
// ------------------------------------------------------------

func parseColorCodeToRGBPtr(colorCode string) *int {
	s := strings.TrimSpace(colorCode)
	if s == "" {
		return nil
	}

	// "#RRGGBB" / "RRGGBB" / "0xRRGGBB" / "0XRRGGBB" を吸収
	s = strings.TrimPrefix(s, "#")
	s = strings.TrimPrefix(strings.ToLower(s), "0x")

	// hex 6桁
	if len(s) == 6 {
		if n, err := strconv.ParseInt(s, 16, 32); err == nil {
			v := int(n)
			return &v
		}
	}

	// decimal
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

type companyProductBlueprintIDsReader interface {
	ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)
}

type tokenBlueprintReader interface {
	GetByID(ctx context.Context, id string) (TokenBlueprintView, error)
}

// must have identical fields to TokenBlueprintSummaryDTO to allow conversion
type TokenBlueprintView struct {
	ID     string
	Name   string
	Symbol string
}

type productBlueprintPatchReader interface {
	GetPatchByID(ctx context.Context, id string) (pbdom.Patch, error)
}

type productReader interface {
	ListByIDs(ctx context.Context, ids []string) ([]ProductView, error)
}

// adapter 側で合わせる view（内部用）
type ProductView struct {
	ID          string
	ModelNumber string
	Size        string
	ColorName   string
	RGB         string
}
