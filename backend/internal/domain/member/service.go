// backend/internal/domain/member/service.go
package member

import (
	"context"
	"strings"
)

// Service は member 領域のユースケース的な便宜関数を提供します。
type Service struct {
	repo Repository
}

// NewService は member.Service を生成します。
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetNameLastFirstByID は memberID から Member を取得し、
// 「lastName firstName」の順で整形した表示名を返します。
// - lastName / firstName の両方が存在: "last first"
// - 片方のみ存在: その値のみ
// - どちらも空: ""
func (s *Service) GetNameLastFirstByID(ctx context.Context, memberID string) (string, error) {
	memberID = strings.TrimSpace(memberID)
	if memberID == "" {
		return "", ErrInvalidID
	}

	m, err := s.repo.GetByID(ctx, memberID)
	if err != nil {
		// そのままドメインエラー（ErrNotFound 等）を返却
		return "", err
	}

	return FormatLastFirst(m.LastName, m.FirstName), nil
}

// FormatLastFirst は「姓→名」の順で半角スペース区切りの表示名を返します。
// 空要素は除外され、両方空の場合は空文字を返します。
func FormatLastFirst(lastName, firstName string) string {
	ln := strings.TrimSpace(lastName)
	fn := strings.TrimSpace(firstName)

	switch {
	case ln != "" && fn != "":
		return ln + " " + fn
	case ln != "":
		return ln
	case fn != "":
		return fn
	default:
		return ""
	}
}
