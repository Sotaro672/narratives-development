package permission

import (
	"context"
	"sort"
	"strings"
)

// ========================================
// ポート（他コンテキストへの依存）
// ========================================

// 管理画面向けの集約データ取得アダプタ
type ManagementAdapterPort interface {
	FetchPermissionManagementData(ctx context.Context) (PermissionManagementData, error)
}

// メンバー更新ポート（権限更新に使用）
type MembersPort interface {
	UpdateMemberPermissions(ctx context.Context, memberID string, permissions []string) (Member, error)
}

// ========================================
// ビュー/DTO相当
// ========================================

type PermissionManagementData struct {
	Permissions []Permission
	Loading     bool
}

type LoadPermissionsResult struct {
	Success     bool
	Permissions []Permission
	Error       string
}

type CreatePermissionResult struct {
	Success    bool
	Permission *Permission
	Error      string
}

type UpdatePermissionResult struct {
	Success    bool
	Permission *Permission
	Error      string
}

type DeletePermissionResult struct {
	Success bool
	Error   string
}

type UpdateMemberPermissionsResult struct {
	Success bool
	Member  *Member
	Error   string
}

// メンバー（必要最小限）
type Member struct {
	ID          string
	Permissions []string
}

// ========================================
// サービス
// ========================================

type Service struct {
	repo    Repository
	adapter ManagementAdapterPort
	members MembersPort
}

// 任意のポートを与えない場合は nil のままで動作
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// 依存をまとめて差し込むための補助
func NewServiceWithPorts(repo Repository, adapter ManagementAdapterPort, members MembersPort) *Service {
	return &Service{repo: repo, adapter: adapter, members: members}
}

func (s *Service) SetManagementAdapter(a ManagementAdapterPort) { s.adapter = a }
func (s *Service) SetMembersPort(m MembersPort)                 { s.members = m }

// 一覧オプション（Filter/Sort/Page の束ね）
type ListPermissionsOptions struct {
	Filter Filter
	Sort   Sort
	Page   Page
}

// ========================================
// データ取得（管理画面）
// ========================================

func (s *Service) GetPermissionManagementData(ctx context.Context) PermissionManagementData {
	// アダプタがあれば委譲
	if s.adapter != nil {
		if data, err := s.adapter.FetchPermissionManagementData(ctx); err == nil {
			return data
		}
	}
	// フォールバック: リポジトリ直取得（署名: List(ctx, Filter, Sort, Page)）
	pr, err := s.repo.List(ctx, Filter{}, Sort{}, Page{})
	if err != nil {
		return PermissionManagementData{Permissions: []Permission{}, Loading: false}
	}
	// PageResult[Permission] をフラット配列へ
	return PermissionManagementData{Permissions: pr.Items, Loading: false}
}

func (s *Service) HandleLoadPermissions(ctx context.Context) LoadPermissionsResult {
	data := s.GetPermissionManagementData(ctx)
	return LoadPermissionsResult{
		Success:     true,
		Permissions: data.Permissions,
	}
}

// ========================================
// CRUD
// ========================================

func (s *Service) HandleCreatePermission(ctx context.Context, p Permission) CreatePermissionResult {
	created, err := s.repo.Create(ctx, p)
	if err != nil {
		return CreatePermissionResult{Success: false, Error: err.Error()}
	}
	return CreatePermissionResult{Success: true, Permission: &created}
}

func (s *Service) HandleUpdatePermission(ctx context.Context, id string, patch PermissionPatch) UpdatePermissionResult {
	updated, err := s.repo.Update(ctx, id, patch)
	if err != nil {
		return UpdatePermissionResult{Success: false, Error: err.Error()}
	}
	return UpdatePermissionResult{Success: true, Permission: &updated}
}

func (s *Service) HandleDeletePermission(ctx context.Context, id string) DeletePermissionResult {
	if err := s.repo.Delete(ctx, id); err != nil {
		return DeletePermissionResult{Success: false, Error: err.Error()}
	}
	return DeletePermissionResult{Success: true}
}

// ========================================
// メンバー権限更新
// ========================================

func (s *Service) HandleUpdateMemberPermissions(ctx context.Context, memberID string, newPermissions []string) UpdateMemberPermissionsResult {
	if s.members == nil {
		return UpdateMemberPermissionsResult{Success: false, Error: "members port not configured"}
	}
	mem, err := s.members.UpdateMemberPermissions(ctx, memberID, dedupTrim(newPermissions))
	if err != nil {
		return UpdateMemberPermissionsResult{Success: false, Error: err.Error()}
	}
	return UpdateMemberPermissionsResult{Success: true, Member: &mem}
}

func (s *Service) HandleBulkUpdateMemberPermissions(ctx context.Context, memberIDs []string, newPermissions []string) []UpdateMemberPermissionsResult {
	results := make([]UpdateMemberPermissionsResult, 0, len(memberIDs))
	for _, id := range memberIDs {
		results = append(results, s.HandleUpdateMemberPermissions(ctx, id, newPermissions))
	}
	return results
}

func (s *Service) HandleAddPermissionsToMember(ctx context.Context, memberID string, current []string, toAdd []string) UpdateMemberPermissionsResult {
	merged := dedupTrim(append(append([]string{}, current...), toAdd...))
	return s.HandleUpdateMemberPermissions(ctx, memberID, merged)
}

func (s *Service) HandleRemovePermissionsFromMember(ctx context.Context, memberID string, current []string, toRemove []string) UpdateMemberPermissionsResult {
	toRemoveSet := toSet(toRemove)
	next := make([]string, 0, len(current))
	for _, v := range current {
		if _, ok := toRemoveSet[v]; !ok {
			next = append(next, v)
		}
	}
	return s.HandleUpdateMemberPermissions(ctx, memberID, next)
}

// ========================================
// 表示/検索/フィルタ/ソート/統計ヘルパ
// ========================================

func FindPermissionByID(list []Permission, id string) (Permission, bool) {
	for _, a := range list {
		if a.ID == id {
			return a, true
		}
	}
	return Permission{}, false
}

func FindPermissionByName(list []Permission, name string) (Permission, bool) {
	for _, a := range list {
		if a.Name == name {
			return a, true
		}
	}
	return Permission{}, false
}

func IsValidPermissionID(list []Permission, id string) bool {
	_, ok := FindPermissionByID(list, id)
	return ok
}

func IsValidPermissionName(list []Permission, name string) bool {
	_, ok := FindPermissionByName(list, name)
	return ok
}

func IsPermissionCategoryExists(list []Permission, category string) bool {
	for _, a := range list {
		if string(a.Category) == category {
			return true
		}
	}
	return false
}

type FilterOptions struct {
	SearchQuery    string
	CategoryFilter []PermissionCategory
}

type SortColumn string

const (
	SortByName     SortColumn = "name"
	SortByCategory SortColumn = "category"
)

type SortOptions struct {
	Column SortColumn
	Order  SortOrder // asc|desc (既存定義)
}

func FilterPermissions(list []Permission, opt FilterOptions) []Permission {
	q := strings.ToLower(strings.TrimSpace(opt.SearchQuery))
	hasQ := q != ""

	catSet := make(map[PermissionCategory]struct{}, len(opt.CategoryFilter))
	for _, c := range opt.CategoryFilter {
		catSet[c] = struct{}{}
	}
	filterByCat := len(catSet) > 0

	out := make([]Permission, 0, len(list))
	for _, a := range list {
		if filterByCat {
			if _, ok := catSet[a.Category]; !ok {
				continue
			}
		}
		if hasQ {
			name := strings.ToLower(a.Name)
			cat := strings.ToLower(string(a.Category))
			desc := strings.ToLower(a.Description)
			if !(strings.Contains(name, q) || strings.Contains(cat, q) || strings.Contains(desc, q)) {
				continue
			}
		}
		out = append(out, a)
	}
	return out
}

func SortPermissions(list []Permission, opt SortOptions) []Permission {
	if opt.Column == "" {
		return list
	}
	out := append([]Permission(nil), list...)
	asc := strings.ToLower(string(opt.Order)) == "asc"

	less := func(i, j int) bool { return true }
	switch opt.Column {
	case SortByName:
		less = func(i, j int) bool {
			if asc {
				return out[i].Name < out[j].Name
			}
			return out[j].Name < out[i].Name
		}
	case SortByCategory:
		less = func(i, j int) bool {
			ci := string(out[i].Category)
			cj := string(out[j].Category)
			if asc {
				return ci < cj
			}
			return cj < ci
		}
	}
	sort.Slice(out, less)
	return out
}

func FilterAndSortPermissions(list []Permission, f FilterOptions, s SortOptions) []Permission {
	return SortPermissions(FilterPermissions(list, f), s)
}

func GetUniqueCategories(list []Permission) []string {
	seen := map[string]struct{}{}
	for _, a := range list {
		seen[string(a.Category)] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func GroupPermissionsByCategory(list []Permission) map[string][]Permission {
	out := map[string][]Permission{}
	for _, a := range list {
		k := string(a.Category)
		out[k] = append(out[k], a)
	}
	return out
}

func GetPermissionCountByCategory(list []Permission, category string) int {
	c := 0
	for _, a := range list {
		if string(a.Category) == category {
			c++
		}
	}
	return c
}

type PermissionStatistics struct {
	TotalPermissions   int
	TotalCategories    int
	CategoryCounts     map[string]int
	MostCommonCategory string
}

func GetPermissionStatistics(list []Permission) PermissionStatistics {
	cats := GetUniqueCategories(list)
	counts := make(map[string]int, len(cats))
	for _, c := range cats {
		counts[c] = GetPermissionCountByCategory(list, c)
	}
	most := ""
	max := -1
	for c, n := range counts {
		if n > max {
			max = n
			most = c
		}
	}
	return PermissionStatistics{
		TotalPermissions:   len(list),
		TotalCategories:    len(cats),
		CategoryCounts:     counts,
		MostCommonCategory: most,
	}
}

// ========================================
// 内部ヘルパ
// ========================================

func toSet(xs []string) map[string]struct{} {
	m := make(map[string]struct{}, len(xs))
	for _, x := range xs {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		m[x] = struct{}{}
	}
	return m
}

func dedupTrim(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}
