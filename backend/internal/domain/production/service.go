package production

import (
	"strings"
	"time"
)

// 補助型（他パッケージの最小サマリをローカルDTOとして定義）
type Model struct {
	ID                 string
	Name               string
	ProductBlueprintID string
	BrandID            string
}

// 既存の ProductBlueprint/Brand/OrganizationMember が無ければ最小定義を追加
// ↓ entity.go に正式定義があるため、ローカル定義は削除してください。
// type ProductBlueprint struct { ... }  // ← このローカル定義を削除

// 参照している箇所は entity の ProductBlueprint を使用します。
// pb.Colors を使用している箇所があれば pb.ModelVariation に置換してください。

type ProductIdStatus string

const (
	ProductIdStatusQR   ProductIdStatus = "QR"
	ProductIdStatusNFC  ProductIdStatus = "NFC"
	ProductIdStatusNone ProductIdStatus = "印刷前"
)

type ProductIDTag struct {
	Type string
}

type Brand struct {
	ID   string
	Name string
}

type OrganizationMember struct {
	ID        string
	FirstName string
	LastName  string
}

func (m OrganizationMember) FullName() string {
	fn := strings.TrimSpace(m.FirstName)
	ln := strings.TrimSpace(m.LastName)
	if fn == "" && ln == "" {
		return ""
	}
	if fn == "" {
		return ln
	}
	if ln == "" {
		return fn
	}
	return ln + " " + fn
}

type PlanModel struct {
	ModelID  string
	Quantity int
}

// UI用行DTO

type ProductionPlanTableRow struct {
	PlanID          string
	ProductName     string
	BrandName       string
	AssigneeName    string
	Quantity        int
	ProductIdStatus ProductIdStatus
	PrintedAt       *time.Time
	CreatedAt       time.Time
}

// フィルタ・ソート・ページング

type ProductionPlanFilter struct {
	SearchTerm        string
	SelectedProducts  []string
	SelectedBrands    []string
	SelectedAssignees []string
	SelectedStatuses  []ProductIdStatus
}

type FilterResult struct {
	FilteredPlans []Production
	TotalCount    int
}

// 追加: フィルタオプション型
type FilterOptions struct {
	Products  []string
	Brands    []string
	Assignees []string
	Statuses  []ProductIdStatus
}

type SortConfig struct {
	Column string // "productionPlan" | "printDate" | "createDate"
	Order  string // "asc" | "desc"
}

type PaginationConfig struct {
	ItemsPerPage int
	CurrentPage  int
}

type PaginationResult struct {
	PaginatedItems []Production
	TotalPages     int
	CurrentPage    int
	TotalItems     int
}

// モデル・サイズ/カラー関連

type SizeVariation struct {
	ID           string
	Size         string
	Measurements map[string]float64
}

type ProductionQuantity struct {
	Size     string
	Color    string
	Quantity int
}

type ModelNumber struct {
	Size        string
	Color       string
	ModelNumber string
}

type SizeTotal struct {
	Size  string
	Total int
}

type ColorTotal struct {
	Color string
	Total int
}

type ProductionQuantityTotals struct {
	SizeTotals  []SizeTotal
	ColorTotals []ColorTotal
	GrandTotal  int
}

type ProductionQuantityMatrixResult struct {
	ProductionQuantities []ProductionQuantity
	TotalCombinations    int
}

type ModelNumberMatrixResult struct {
	ModelNumbers      []ModelNumber
	TotalCombinations int
}

// ステータス判定

func GetProductIdStatus(pb ProductBlueprint) ProductIdStatus {
	switch strings.ToLower(string(pb.ProductIdTag.Type)) {
	case "qr":
		return ProductIdStatusQR
	case "nfc":
		return ProductIdStatusNFC
	default:
		return ProductIdStatusNone
	}
}

// 名称解決

func GetAssigneeName(assigneeID string, members []OrganizationMember) string {
	for _, m := range members {
		if m.ID == assigneeID {
			name := m.FullName()
			if name == "-" {
				return ""
			}
			return name
		}
	}
	return ""
}

func GetBrandName(brandID string, brands []Brand) string {
	for _, b := range brands {
		if b.ID == brandID {
			return b.Name
		}
	}
	return ""
}

// 集計

func CalculatePlanQuantity(plan Production) int {
	// 複数モデル対応（旧: return plan.Quantity）
	return TotalPlanQuantity(plan)
}

// helper: 合計数量（plan.Quantity の代替）
func TotalPlanQuantity(plan Production) int {
	total := 0
	for _, mq := range plan.Models {
		total += mq.Quantity
	}
	return total
}

// helper: plan.Models から最初に解決できた Model を返す（単一表示用など）
func firstModelFromPlan(models []Model, plan Production) *Model {
	for _, mq := range plan.Models {
		if m := findModelByID(models, mq.ModelID); m != nil {
			return m
		}
	}
	return nil
}

// helper: PB解決。plan.ProductBlueprintID を優先し、なければ Model 経由。
func pbFromPlan(productBlueprints []ProductBlueprint, plan Production, models []Model) *ProductBlueprint {
	if pb := findPBByID(productBlueprints, plan.ProductBlueprintID); pb != nil {
		return pb
	}
	if m := firstModelFromPlan(models, plan); m != nil {
		return findPBByID(productBlueprints, m.ProductBlueprintID)
	}
	return nil
}

// 行変換（例）
func CreateTableRowData(
	plan Production,
	models []Model,
	productBlueprints []ProductBlueprint,
	brands []Brand,
	members []OrganizationMember,
) *ProductionPlanTableRow {
	model := firstModelFromPlan(models, plan) // 旧: findModelByID(models, plan.ModelID)
	pb := pbFromPlan(productBlueprints, plan, models) // 旧: plan.ProductBlueprintID 直参照
	if model == nil || pb == nil {
		return nil
	}
	row := &ProductionPlanTableRow{
		PlanID:          plan.ID,
		ProductName:     pb.ProductName,
		BrandName:       GetBrandName(pb.BrandID, brands),
		AssigneeName:    GetAssigneeName(pb.AssigneeID, members),
		Quantity:        TotalPlanQuantity(plan), // 旧: plan.Quantity
		ProductIdStatus: GetProductIdStatus(*pb),
		PrintedAt:       plan.PrintedAt,
		CreatedAt:       plan.CreatedAt,
	}
	return row
}

// フィルタ用オプション生成（例）
func GenerateFilterOptions(
	productionPlans []Production,
	models []Model,
	productBlueprints []ProductBlueprint,
	brands []Brand,
	members []OrganizationMember,
) FilterOptions {
	productNames := map[string]struct{}{}
	brandNames := map[string]struct{}{}
	assigneeNames := map[string]struct{}{}
	statusNames := map[ProductIdStatus]struct{}{}

	for _, plan := range productionPlans {
		pb := pbFromPlan(productBlueprints, plan, models) // 旧: modelId や plan.ProductBlueprintID 直参照
		if pb == nil {
			continue
		}
		if pb.ProductName != "" {
			productNames[pb.ProductName] = struct{}{}
		}
		if bn := GetBrandName(pb.BrandID, brands); bn != "" {
			brandNames[bn] = struct{}{}
		}
		if an := GetAssigneeName(pb.AssigneeID, members); an != "" {
			assigneeNames[an] = struct{}{}
		}
		statusNames[GetProductIdStatus(*pb)] = struct{}{}
	}

	toSlice := func(m map[string]struct{}) []string {
		out := make([]string, 0, len(m))
		for k := range m {
			out = append(out, k)
		}
		return out
	}
	toStatusSlice := func(m map[ProductIdStatus]struct{}) []ProductIdStatus {
		out := make([]ProductIdStatus, 0, len(m))
		for k := range m {
			out = append(out, k)
		}
		return out
	}

	return FilterOptions{
		Products:  toSlice(productNames),
		Brands:    toSlice(brandNames),
		Assignees: toSlice(assigneeNames),
		Statuses:  toStatusSlice(statusNames),
	}
}

// フィルタ本体（例）
func FilterProductionPlans(
	productionPlans []Production,
	filter ProductionPlanFilter,
	models []Model,
	productBlueprints []ProductBlueprint,
	brands []Brand,
	members []OrganizationMember,
) FilterResult {
	var filtered []Production

	for _, plan := range productionPlans {
		pb := pbFromPlan(productBlueprints, plan, models)
		if pb == nil {
			continue
		}
		statusName := GetProductIdStatus(*pb)
		assigneeName := GetAssigneeName(pb.AssigneeID, members)
		brandName := GetBrandName(pb.BrandID, brands)

		matchesSearch := true
		if filter.SearchTerm != "" {
			lterm := strings.ToLower(filter.SearchTerm)
			if !(strings.Contains(strings.ToLower(pb.ProductName), lterm) ||
				strings.Contains(strings.ToLower(brandName), lterm) ||
				strings.Contains(strings.ToLower(plan.ID), lterm)) {
				matchesSearch = false
			}
		}
		includes := func(list []string, v string) bool {
			for _, s := range list {
				if s == v {
					return true
				}
			}
			return false
		}
		includesStatus := func(list []ProductIdStatus, v ProductIdStatus) bool {
			for _, s := range list {
				if s == v {
					return true
				}
			}
			return false
		}

		if (len(filter.SelectedProducts) == 0 || includes(filter.SelectedProducts, pb.ProductName)) &&
			(len(filter.SelectedBrands) == 0 || includes(filter.SelectedBrands, brandName)) &&
			(len(filter.SelectedAssignees) == 0 || includes(filter.SelectedAssignees, assigneeName)) &&
			(len(filter.SelectedStatuses) == 0 || includesStatus(filter.SelectedStatuses, statusName)) &&
			matchesSearch {
			filtered = append(filtered, plan)
		}
	}

	return FilterResult{FilteredPlans: filtered, TotalCount: len(filtered)}
}

// 生産数量情報（例）
type InspectionProduct struct {
	InspectionResult string // "passed" など
}

func CalculateProductionQuantitiesInfo(plan Production, products []InspectionProduct) (totalProduced, totalPassed int) {
	totalProduced = TotalPlanQuantity(plan) // 旧: plan.Quantity
	for _, p := range products {
		if p.InspectionResult == "passed" {
			totalPassed++
		}
	}
	return
}

func findModelByID(models []Model, id string) *Model {
	for i := range models {
		if models[i].ID == id {
			return &models[i]
		}
	}
	return nil
}

func findPBByID(pbs []ProductBlueprint, id string) *ProductBlueprint {
	for i := range pbs {
		if pbs[i].ID == id {
			return &pbs[i]
		}
	}
	return nil
}
