// backend/internal/application/query/admin/report_name_query.go
package query

import (
	"context"

	avatardom "narratives/internal/domain/avatar"
	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	memberdom "narratives/internal/domain/member"
	reviewreport "narratives/internal/domain/reviewReport"
)

// ============================================================
// Read-only ports
// ============================================================

type reportAvatarReader interface {
	GetByID(ctx context.Context, id string) (avatardom.Avatar, error)
}

type reportBrandReader interface {
	GetByID(ctx context.Context, id string) (branddom.Brand, error)
}

type reportCompanyReader interface {
	GetByID(ctx context.Context, id string) (companydom.Company, error)
}

type reportMemberReader interface {
	GetByID(ctx context.Context, id string) (memberdom.Record, error)
	GetByUID(ctx context.Context, uid string) (memberdom.Record, error)
}

// ============================================================
// Query
// ============================================================

type ReportNameQuery struct {
	avatarRepo  reportAvatarReader
	brandRepo   reportBrandReader
	companyRepo reportCompanyReader
	memberRepo  reportMemberReader
}

func NewReportNameQuery(
	avatarRepo reportAvatarReader,
	brandRepo reportBrandReader,
	companyRepo reportCompanyReader,
	memberRepo reportMemberReader,
) *ReportNameQuery {
	return &ReportNameQuery{
		avatarRepo:  avatarRepo,
		brandRepo:   brandRepo,
		companyRepo: companyRepo,
		memberRepo:  memberRepo,
	}
}

// ============================================================
// Basic resolvers
// ============================================================

// ResolveAvatarName は avatarId から avatarName を解決する。
// 解決できない場合は空文字列を返す。
func (q *ReportNameQuery) ResolveAvatarName(ctx context.Context, avatarID string) string {
	if q == nil || q.avatarRepo == nil || avatarID == "" {
		return ""
	}

	entity, err := q.avatarRepo.GetByID(ctx, avatarID)
	if err != nil {
		return ""
	}

	return entity.AvatarName
}

// ResolveBrandName は brandId から brandName を解決する。
// 解決できない場合は空文字列を返す。
func (q *ReportNameQuery) ResolveBrandName(ctx context.Context, brandID string) string {
	if q == nil || q.brandRepo == nil || brandID == "" {
		return ""
	}

	entity, err := q.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return ""
	}

	return entity.Name
}

// ResolveCompanyName は companyId から companyName を解決する。
// 解決できない場合は空文字列を返す。
func (q *ReportNameQuery) ResolveCompanyName(ctx context.Context, companyID string) string {
	if q == nil || q.companyRepo == nil || companyID == "" {
		return ""
	}

	entity, err := q.companyRepo.GetByID(ctx, companyID)
	if err != nil {
		return ""
	}

	return entity.Name
}

// ResolveMemberName は member の Firestore document ID または Firebase Auth UID から
// 「姓 名」形式の memberName を解決する。
// 解決できない場合は空文字列を返す。
func (q *ReportNameQuery) ResolveMemberName(ctx context.Context, memberID string) string {
	if q == nil || q.memberRepo == nil || memberID == "" {
		return ""
	}

	if record, err := q.memberRepo.GetByID(ctx, memberID); err == nil {
		if name := memberdom.FormatLastFirst(record.Member.LastName, record.Member.FirstName); name != "" {
			return name
		}
	}

	if record, err := q.memberRepo.GetByUID(ctx, memberID); err == nil {
		if name := memberdom.FormatLastFirst(record.Member.LastName, record.Member.FirstName); name != "" {
			return name
		}
	}

	return ""
}

// ============================================================
// Review report resolvers
// ============================================================

// ResolveReporterName は通報者の表示名を解決する。
// AVATAR は avatarName、BRAND は brandName として表示する。
// memberName への変換は行わない。
func (q *ReportNameQuery) ResolveReporterName(
	ctx context.Context,
	reporterType reviewreport.ActorType,
	reporterID string,
) string {
	switch reporterType {
	case reviewreport.ActorTypeAvatar:
		return q.ResolveAvatarName(ctx, reporterID)
	case reviewreport.ActorTypeBrand:
		return q.ResolveBrandName(ctx, reporterID)
	default:
		return ""
	}
}

// ResolveTargetAuthorName は通報対象コンテンツの投稿者表示名を解決する。
// AVATAR は avatarName、BRAND は memberName として解決を試みる。
// 解決できない場合は空文字列を返し、レスポンス側で元 ID へフォールバックする。
func (q *ReportNameQuery) ResolveTargetAuthorName(
	ctx context.Context,
	authorType reviewreport.ActorType,
	authorID string,
) string {
	switch authorType {
	case reviewreport.ActorTypeAvatar:
		return q.ResolveAvatarName(ctx, authorID)
	case reviewreport.ActorTypeBrand:
		return q.ResolveMemberName(ctx, authorID)
	default:
		return ""
	}
}
