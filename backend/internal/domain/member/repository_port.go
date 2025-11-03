package member

import (
	"context"
	"time"

	common "narratives/internal/domain/common"
)

// MemberRole は後方互換のため string のエイリアスとして公開。
// adapter 側で string をそのまま代入可能にします。

// Filter は一覧取得時のフィルタ条件です。
type Filter struct {
	SearchQuery string   // 名前/フリガナ/メールの部分一致など
	RoleIDs     []string // ロールID（名称ではなくID/コードを推奨）
	BrandIDs    []string // 担当ブランドID
	CompanyID   string   // 所属会社フィルタ
	Status      string   // "active" | "inactive" など、必要に応じて
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time

	// 権限（entity.Member.Permissions に準拠）
	Permissions []string

	// --- 後方互換: 旧adapterが参照するフィールド名 ---
	// f.Roles -> RoleIDs と同義
	Roles []string
	// f.Brands -> BrandIDs と同義
	Brands []string
}

// Sort は一覧取得時のソート条件です。
type Sort struct {
	Column SortColumn
	Order  SortOrder
}

type SortColumn string

const (
	SortByJoinedAt      SortColumn = "joinedAt"
	SortByPermissions   SortColumn = "permissions"
	SortByAssigneeCount SortColumn = "assigneeCount"

	// よく使う列を追加で想定
	SortByName      SortColumn = "name"
	SortByEmail     SortColumn = "email"
	SortByUpdatedAt SortColumn = "updatedAt"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// 共通定義のエイリアス
type Page = common.Page
type PageResult = common.PageResult[Member]
type CursorPage = common.CursorPage
type CursorPageResult = common.CursorPageResult[Member]
type SaveOptions = common.SaveOptions

// Repository はメンバー集約の永続化ポートです。
type Repository interface {
	// 共通CRUD/一覧（GetByID, Create, Update, Delete, List を共通側に委譲）
	common.RepositoryCRUD[Member, MemberPatch]
	common.RepositoryList[Member, Filter]

	// 追加要件（共通に無いものはここで定義）
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult, error)
	GetByEmail(ctx context.Context, email string) (Member, error)
	Exists(ctx context.Context, id string) (bool, error)
	Count(ctx context.Context, filter Filter) (int, error)
	Save(ctx context.Context, m Member, opts *SaveOptions) (Member, error)
	Reset(ctx context.Context) error
}
