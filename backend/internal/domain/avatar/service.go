package avatar

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	common "narratives/internal/domain/common"
)

// Service - アバタービジネスロジック層
type Service struct {
	repo Repository
}

// NewService - コンストラクタ
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ========================================
// バリデーション
// ========================================

// isValidWalletAddress - Solana想定のBase58(0OIl除外) 32-44文字
func isValidWalletAddress(wallet string) bool {
	if len(wallet) < 32 || len(wallet) > 44 {
		return false
	}
	base58 := regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]+$`)
	return base58.MatchString(wallet)
}

// ========================================
// 作成/更新/削除
// ========================================

type CreateAvatarInput struct {
	UserID        string
	AvatarName    string
	WalletAddress string // 任意
	Verified      bool   // 任意（エンティティ未対応のため利用しない）
}

func (s *Service) CreateAvatar(ctx context.Context, in CreateAvatarInput) (*Avatar, error) {
	if strings.TrimSpace(in.AvatarName) == "" {
		return nil, errors.New("アバター名を入力してください")
	}
	if utf8.RuneCountInString(in.AvatarName) > 50 {
		return nil, errors.New("アバター名は50文字以内で入力してください")
	}
	if strings.TrimSpace(in.UserID) == "" {
		return nil, errors.New("ユーザーIDを入力してください")
	}
	if in.WalletAddress != "" && !isValidWalletAddress(in.WalletAddress) {
		return nil, errors.New("無効なウォレットアドレス形式です")
	}

	a := Avatar{
		UserID:        in.UserID,
		AvatarName:    in.AvatarName,
		WalletAddress: toOptionalString(in.WalletAddress),
		// Verified/FollowersCount/PostsCount/JoinedAt はエンティティに存在しないため設定しない
	}
	created, err := s.repo.Create(ctx, a)
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func (s *Service) UpdateAvatar(ctx context.Context, id string, patch AvatarPatch) (*Avatar, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("アバターIDが指定されていません")
	}
	if patch.AvatarName != nil {
		name := strings.TrimSpace(*patch.AvatarName)
		if name == "" {
			return nil, errors.New("アバター名を入力してください")
		}
		if utf8.RuneCountInString(name) > 50 {
			return nil, errors.New("アバター名は50文字以内で入力してください")
		}
	}
	if patch.WalletAddress != nil && *patch.WalletAddress != "" {
		if !isValidWalletAddress(*patch.WalletAddress) {
			return nil, errors.New("無効なウォレットアドレス形式です")
		}
	}
	out, err := s.repo.Update(ctx, id, patch)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Service) DeleteAvatar(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("アバターIDが指定されていません")
	}
	return s.repo.Delete(ctx, id)
}

// リンク（ウォレット）
func (s *Service) LinkWallet(ctx context.Context, id, wallet string) (*Avatar, error) {
	if strings.TrimSpace(wallet) == "" {
		return nil, errors.New("ウォレットアドレスを入力してください")
	}
	if !isValidWalletAddress(wallet) {
		return nil, errors.New("無効なウォレットアドレス形式です")
	}
	return s.UpdateAvatar(ctx, id, AvatarPatch{WalletAddress: &wallet})
}

// ========================================
// ID/名前リゾルバー & キャッシュ
// ========================================

type AvatarCacheByWallet map[string]Avatar
type AvatarCacheByID map[string]Avatar

func (s *Service) ResolveAvatarNameByID(avatarID string, cache AvatarCacheByID) string {
	if avatarID == "" {
		return "不明なユーザー"
	}
	if a, ok := cache[avatarID]; ok {
		return a.AvatarName
	}
	return "不明なユーザー"
}

func (s *Service) ResolveAvatarNameByWallet(wallet string, cache AvatarCacheByWallet) string {
	if wallet == "" {
		return "不明なウォレット"
	}
	if a, ok := cache[wallet]; ok {
		return a.AvatarName
	}
	return ShortenWalletAddress(wallet)
}

func (s *Service) ResolveAvatarIDByWallet(wallet string, cache AvatarCacheByWallet) *string {
	if wallet == "" {
		return nil
	}
	if a, ok := cache[wallet]; ok {
		return &a.ID
	}
	return nil
}

func (s *Service) CreateAvatarCacheByWallet(ctx context.Context) (AvatarCacheByWallet, error) {
	pr, err := s.repo.List(ctx, Filter{}, common.Sort{}, common.Page{Number: 1, PerPage: 10000})
	if err != nil {
		return nil, err
	}
	cache := make(AvatarCacheByWallet, len(pr.Items))
	for _, a := range pr.Items {
		if a.WalletAddress != nil && *a.WalletAddress != "" {
			cache[*a.WalletAddress] = a
		}
	}
	return cache, nil
}

func (s *Service) GetAvatarNameByWalletAddress(ctx context.Context, wallet string) (string, error) {
	a, err := s.repo.GetByWalletAddress(ctx, wallet)
	if err != nil {
		return "", err
	}
	if a.ID == "" {
		return "", nil
	}
	return a.AvatarName, nil
}

func (s *Service) GetAvatarIDByWalletAddress(ctx context.Context, wallet string) (string, error) {
	a, err := s.repo.GetByWalletAddress(ctx, wallet)
	if err != nil {
		return "", err
	}
	if a.ID == "" {
		return "", nil
	}
	return a.ID, nil
}

// ========================================
// 表示名/ユーティリティ
// ========================================

func (s *Service) GetAvatarDisplayName(a *Avatar, walletFallback string) string {
	if a == nil {
		if walletFallback != "" {
			return ShortenWalletAddress(walletFallback)
		}
		return "不明なユーザー"
	}
	return a.AvatarName
}

func (s *Service) GetAvatarInitials(a Avatar) string {
	name := a.AvatarName
	if name == "" {
		return "??"
	}
	var out []rune
	for _, r := range name {
		out = append(out, r)
		if len(out) == 2 {
			break
		}
	}
	return string(out)
}

func (s *Service) GetAvatarUsername(a Avatar) string {
	// ドメインに username はないため既定値
	return "unknown"
}

func ShortenWalletAddress(wallet string) string {
	if len(wallet) < 16 {
		return wallet
	}
	return wallet[:7] + "..." + wallet[len(wallet)-5:]
}

// ========================================
// フィルタリング（最小）
// ========================================

type AvatarFilter struct {
	SearchQuery  string
	VerifiedOnly *bool // 互換のため残す（処理では未使用）
	MinFollowers *int  // 互換のため残す（処理では未使用）
}

func (s *Service) FilterAvatars(list []Avatar, f AvatarFilter) []Avatar {
	out := make([]Avatar, 0, len(list))
	q := strings.ToLower(strings.TrimSpace(f.SearchQuery))
	for _, a := range list {
		if q != "" {
			an := strings.ToLower(a.AvatarName)
			uid := strings.ToLower(a.UserID)
			w := ""
			if a.WalletAddress != nil {
				w = strings.ToLower(*a.WalletAddress)
			}
			if !strings.Contains(an, q) && !strings.Contains(uid, q) && !strings.Contains(w, q) {
				continue
			}
		}
		// VerifiedOnly / MinFollowers は非対応フィールドのため無視
		out = append(out, a)
	}
	return out
}

// ========================================
// 複合ヘルパー
// ========================================

// FetchAvatarNames - avatarID配列から名前を解決（全件取得の簡易版）
func (s *Service) FetchAvatarNames(ctx context.Context, ids []string) ([]string, error) {
	if len(ids) == 0 {
		return []string{}, nil
	}
	pr, err := s.repo.List(ctx, Filter{}, common.Sort{}, common.Page{Number: 1, PerPage: 10000})
	if err != nil {
		return nil, err
	}
	index := make(map[string]string, len(pr.Items))
	for _, a := range pr.Items {
		index[a.ID] = a.AvatarName
	}
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if name, ok := index[id]; ok && name != "" {
			names = append(names, name)
		} else {
			names = append(names, id)
		}
	}
	return names, nil
}

// Explain - デバッグ用の簡易説明
func (s *Service) Explain() string {
	return fmt.Sprintf("Avatar Service (repo=%T)", s.repo)
}

// Helper: optional string pointer (empty -> nil)
func toOptionalString(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	v := strings.TrimSpace(s)
	return &v
}
