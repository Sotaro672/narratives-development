// backend\internal\domain\permission\service.go
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
// ========================================

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

// ------------------------------------------------------------
// 追加: 権限名から日本語名を取得するヘルパ
// ------------------------------------------------------------

// DisplayNameJaFromPermissionName は、権限名から日本語表示名を返します。
// - name は "wallet.view" などの Permission.Name
// - allPermissions カタログに存在する場合、その Description（日本語名）を返す
// - 見つからない場合は ("", false) を返す
func DisplayNameJaFromPermissionName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}

	for _, p := range allPermissions {
		if p.Name == name {
			// Permission の第3引数（Description）を日本語表示名として扱う
			return strings.TrimSpace(p.Description), true
		}
	}
	return "", false
}

// ------------------------------------------------------------
// 追加: 権限名からカテゴリを検索するヘルパ
// ------------------------------------------------------------

// CategoryFromPermissionName は、権限名から PermissionCategory を返します。
// name は "<category>[.<subscope>].<action>" の形式を前提とし、
// 1) カタログに登録済みの権限ならその Category
// 2) 見つからない場合は name の先頭プレフィックスから推論
func CategoryFromPermissionName(name string) (PermissionCategory, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}

	// 1) まず static カタログから探す
	for _, p := range allPermissions {
		if p.Name == name {
			return p.Category, true
		}
	}

	// 2) カタログに無い場合（例: 旧データ "wallet.edit" など）は
	//    先頭の "<category>" 部分から推論する
	//    "wallet.edit" → "wallet" → CategoryWallet
	firstDot := strings.IndexByte(name, '.')
	if firstDot <= 0 {
		return "", false
	}
	catStr := name[:firstDot]
	cat := PermissionCategory(catStr)
	if !IsValidCategory(cat) {
		return "", false
	}
	return cat, true
}

// GroupPermissionNamesByCategory は、権限名のスライスを Category ごとにまとめます。
// 例: ["wallet.view","member.view"] → { "wallet": [...], "member": [...] }
func GroupPermissionNamesByCategory(names []string) map[PermissionCategory][]string {
	out := make(map[PermissionCategory][]string)
	for _, n := range names {
		cat, ok := CategoryFromPermissionName(n)
		if !ok {
			continue
		}
		out[cat] = append(out[cat], n)
	}
	return out
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

type PermissionStatistics struct {
	TotalPermissions   int
	TotalCategories    int
	CategoryCounts     map[string]int
	MostCommonCategory string
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
