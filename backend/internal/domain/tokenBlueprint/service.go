package tokenBlueprint

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ========================================
// 参照用の軽量DTO（外部ドメイン依存を避ける）
// ========================================

type TokenBlueprintRef struct {
	ID           string
	Name         string
	Symbol       string
	BrandID      string
	Description  string
	AssigneeID   string
	IconURL      *string
	ContentFiles []string
	BurnDate     *time.Time
	CreatedAt    time.Time
}

type BrandRef struct {
	ID   string
	Name string
}

type OrganizationMemberRef struct {
	ID          string
	Email       string
	FirstName   string
	LastName    string
	DisplayName string
	Username    string
}

// UI補助用
type AssigneeInfo struct {
	ID   string
	Name string
}

// ========================================
// 共通ユーティリティ
// ========================================

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// ========================================
// ファイルタイプ・サイズフォーマット
// ========================================

type ContentType string

const (
	ContentTypeImage    ContentType = "image"
	ContentTypeVideo    ContentType = "video"
	ContentTypePDF      ContentType = "pdf"
	ContentTypeDocument ContentType = "document"
)

func DetermineFileType(fileType string) ContentType {
	ft := strings.ToLower(strings.TrimSpace(fileType))
	switch {
	case strings.HasPrefix(ft, "image/"):
		return ContentTypeImage
	case strings.HasPrefix(ft, "video/"):
		return ContentTypeVideo
	case ft == "application/pdf" || strings.HasSuffix(ft, "/pdf"):
		return ContentTypePDF
	default:
		return ContentTypeDocument
	}
}

func FormatFileSize(bytes int64, detailed bool) string {
	if !detailed {
		// シンプル表記（MB）
		return fmt.Sprintf("%.2f MB", float64(bytes)/1024.0/1024.0)
	}
	const (
		kb = 1024.0
		mb = 1024.0 * kb
		gb = 1024.0 * mb
	)
	b := float64(bytes)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.2f GB", b/gb)
	case b >= mb:
		return fmt.Sprintf("%.2f MB", b/mb)
	case b >= kb:
		return fmt.Sprintf("%.2f KB", b/kb)
	default:
		return fmt.Sprintf("%.0f Bytes", b)
	}
}

func FormatDateForDisplay(date time.Time) string {
	// 日本語ロケール相当の簡易表記
	return date.Format("2006/01/02")
}

type FormattedDate struct {
	DateString string
	IsValid    bool
}

func FormatBurnDateString(burnDate *time.Time) string {
	if burnDate == nil {
		return "無期限"
	}
	return FormatDateForDisplay(*burnDate)
}

func FormatBurnDateInfo(burnDate *time.Time) FormattedDate {
	if burnDate == nil {
		return FormattedDate{DateString: "", IsValid: false}
	}
	return FormattedDate{DateString: FormatDateForDisplay(*burnDate), IsValid: true}
}

func NormalizeTokenSymbol(symbol string, maxLength int) string {
	if maxLength <= 0 {
		maxLength = 10
	}
	s := strings.ToUpper(strings.TrimSpace(symbol))
	if len(s) > maxLength {
		return s[:maxLength]
	}
	return s
}

// ========================================
// メンバー/ブランド名解決
// ========================================

func GetMemberNameByID(memberID string, members []OrganizationMemberRef) string {
	for _, m := range members {
		if m.ID != memberID {
			continue
		}
		if s := strings.TrimSpace(m.DisplayName); s != "" {
			return s
		}
		fn := strings.TrimSpace(m.FirstName)
		ln := strings.TrimSpace(m.LastName)
		if fn != "" || ln != "" {
			full := strings.TrimSpace(strings.Join([]string{ln, fn}, " "))
			if full != "" {
				return full
			}
		}
		if s := strings.TrimSpace(m.Username); s != "" {
			return s
		}
		if m.Email != "" {
			parts := strings.SplitN(m.Email, "@", 2)
			if parts[0] != "" {
				return parts[0]
			}
		}
		break
	}
	return ""
}

func FormatMemberNameForDisplay(m OrganizationMemberRef) string {
	ln, fn := strings.TrimSpace(m.LastName), strings.TrimSpace(m.FirstName)
	if ln != "" && fn != "" {
		return ln + " " + fn
	}
	if ln != "" {
		return ln
	}
	if fn != "" {
		return fn
	}
	return m.ID
}

func GetBrandNameByID(brandID string, brands []BrandRef) string {
	for _, b := range brands {
		if b.ID == brandID {
			return b.Name
		}
	}
	return ""
}

func GetTokenBlueprintBrandName(bp TokenBlueprintRef, brands []BrandRef) string {
	return GetBrandNameByID(bp.BrandID, brands)
}

func GetTokenBlueprintAssigneeName(bp TokenBlueprintRef, members []OrganizationMemberRef) string {
	return GetMemberNameByID(bp.AssigneeID, members)
}

// ========================================
// ID/マップユーティリティ
// ========================================

func GetAllTokenBlueprintIDs(tokenBlueprints []TokenBlueprintRef) []string {
	out := make([]string, 0, len(tokenBlueprints))
	for _, bp := range tokenBlueprints {
		out = append(out, bp.ID)
	}
	return out
}

type TokenBlueprintBrandInfo struct {
	Name      string
	Symbol    string
	BrandName string
}

func BuildTokenBlueprintMapWithBrandNames(
	tokenBlueprints []TokenBlueprintRef,
	brands []BrandRef,
) map[string]TokenBlueprintBrandInfo {
	brandNameByID := make(map[string]string, len(brands))
	for _, b := range brands {
		brandNameByID[b.ID] = b.Name
	}
	out := make(map[string]TokenBlueprintBrandInfo, len(tokenBlueprints))
	for _, bp := range tokenBlueprints {
		out[bp.ID] = TokenBlueprintBrandInfo{
			Name:      bp.Name,
			Symbol:    bp.Symbol,
			BrandName: brandNameByID[bp.BrandID],
		}
	}
	return out
}

// ========================================
// フィルター・ソート・ページネーション
// ========================================

type TokenBlueprintFilter struct {
	SearchQuery    string
	BrandFilter    []string
	AssigneeFilter []string
}

type TokenBlueprintSortConfig struct {
	Column string // "createdAt" | "name" | "symbol"
	Order  string // "asc" | "desc"
}

type TokenBlueprintPaginationConfig struct {
	CurrentPage  int
	ItemsPerPage int
}

type TokenBlueprintPaginationResult struct {
	PaginatedItems []TokenBlueprintRef
	TotalPages     int
	CurrentPage    int
	TotalItems     int
}

type TokenBlueprintFilterOptions struct {
	Brands    []BrandRef
	Assignees []AssigneeInfo
}

func GenerateAssigneeList(tokenBlueprints []TokenBlueprintRef, members []OrganizationMemberRef) []AssigneeInfo {
	ids := make([]string, 0, len(tokenBlueprints))
	for _, bp := range tokenBlueprints {
		if bp.AssigneeID != "" {
			ids = append(ids, bp.AssigneeID)
		}
	}
	ids = uniqueStrings(ids)

	out := make([]AssigneeInfo, 0, len(ids))
	for _, id := range ids {
		if name := GetMemberNameByID(id, members); name != "" && name != "-" {
			out = append(out, AssigneeInfo{ID: id, Name: name})
		}
	}
	return out
}

func GenerateFilterOptions(
	tokenBlueprints []TokenBlueprintRef,
	brands []BrandRef,
	members []OrganizationMemberRef,
) TokenBlueprintFilterOptions {
	return TokenBlueprintFilterOptions{
		Brands:    brands,
		Assignees: GenerateAssigneeList(tokenBlueprints, members),
	}
}

func FilterTokenBlueprints(tokenBlueprints []TokenBlueprintRef, filter TokenBlueprintFilter) []TokenBlueprintRef {
	search := strings.ToLower(strings.TrimSpace(filter.SearchQuery))
	brandSet := map[string]struct{}{}
	for _, id := range filter.BrandFilter {
		brandSet[id] = struct{}{}
	}
	assigneeSet := map[string]struct{}{}
	for _, id := range filter.AssigneeFilter {
		assigneeSet[id] = struct{}{}
	}

	out := make([]TokenBlueprintRef, 0, len(tokenBlueprints))
	for _, bp := range tokenBlueprints {
		// 検索
		if search != "" {
			if !(strings.Contains(strings.ToLower(bp.Name), search) ||
				strings.Contains(strings.ToLower(bp.Symbol), search) ||
				strings.Contains(strings.ToLower(bp.ID), search)) {
				continue
			}
		}
		// ブランド
		if len(brandSet) > 0 {
			if _, ok := brandSet[bp.BrandID]; !ok {
				continue
			}
		}
		// 担当者
		if len(assigneeSet) > 0 {
			if _, ok := assigneeSet[bp.AssigneeID]; !ok {
				continue
			}
		}
		out = append(out, bp)
	}
	return out
}

func SortTokenBlueprints(items []TokenBlueprintRef, cfg TokenBlueprintSortConfig) []TokenBlueprintRef {
	if cfg.Column == "" {
		return items
	}
	out := make([]TokenBlueprintRef, len(items))
	copy(out, items)

	var less func(i, j int) bool
	switch cfg.Column {
	case "createdAt":
		less = func(i, j int) bool {
			ti := out[i].CreatedAt
			tj := out[j].CreatedAt
			if cfg.Order == "asc" {
				return ti.Before(tj)
			}
			return ti.After(tj)
		}
	case "name":
		less = func(i, j int) bool {
			ai := strings.ToLower(out[i].Name)
			aj := strings.ToLower(out[j].Name)
			if cfg.Order == "asc" {
				return ai < aj
			}
			return ai > aj
		}
	case "symbol":
		less = func(i, j int) bool {
			ai := strings.ToLower(out[i].Symbol)
			aj := strings.ToLower(out[j].Symbol)
			if cfg.Order == "asc" {
				return ai < aj
			}
			return ai > aj
		}
	default:
		return out
	}

	sort.Slice(out, less)
	return out
}

func PaginateTokenBlueprints(items []TokenBlueprintRef, cfg TokenBlueprintPaginationConfig) TokenBlueprintPaginationResult {
	if cfg.ItemsPerPage <= 0 {
		cfg.ItemsPerPage = 10
	}
	if cfg.CurrentPage <= 0 {
		cfg.CurrentPage = 1
	}
	total := len(items)
	totalPages := (total + cfg.ItemsPerPage - 1) / cfg.ItemsPerPage
	start := (cfg.CurrentPage - 1) * cfg.ItemsPerPage
	if start > total {
		start = total
	}
	end := start + cfg.ItemsPerPage
	if end > total {
		end = total
	}
	return TokenBlueprintPaginationResult{
		PaginatedItems: items[start:end],
		TotalPages:     totalPages,
		CurrentPage:    cfg.CurrentPage,
		TotalItems:     total,
	}
}

func ProcessTokenBlueprintsForDisplay(
	tokenBlueprints []TokenBlueprintRef,
	filter TokenBlueprintFilter,
	sortCfg TokenBlueprintSortConfig,
	pageCfg TokenBlueprintPaginationConfig,
) TokenBlueprintPaginationResult {
	filtered := FilterTokenBlueprints(tokenBlueprints, filter)
	sorted := SortTokenBlueprints(filtered, sortCfg)
	return PaginateTokenBlueprints(sorted, pageCfg)
}
