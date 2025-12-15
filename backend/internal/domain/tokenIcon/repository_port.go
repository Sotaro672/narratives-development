package tokenIcon

import (
	"context"
	"errors"
	"time"
)

// 契約のみ（DB/ストレージ技術には依存しない）
// エンティティは entity.go の TokenIcon（ID, URL, FileName, Size）に準拠

// 作成入力（IDは実装側で採番可）
type CreateTokenIconInput struct {
	URL      string `json:"url"`
	FileName string `json:"fileName"`
	Size     int64  `json:"size"`
}

// 部分更新（nilは未更新）
type UpdateTokenIconInput struct {
	URL      *string `json:"url,omitempty"`
	FileName *string `json:"fileName,omitempty"`
	Size     *int64  `json:"size,omitempty"`
}

// 検索条件
type Filter struct {
	IDs          []string
	FileNameLike string

	SizeMin *int64
	SizeMax *int64
}

// 並び順
type Sort struct {
	Column SortColumn
	Order  SortOrder
}

type SortColumn string

const (
	SortByFileName SortColumn = "fileName"
	SortBySize     SortColumn = "size"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// ページング
type Page struct {
	Number  int
	PerPage int
}

type PageResult struct {
	Items      []TokenIcon
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// 統計（任意契約）
type TokenIconStats struct {
	Total       int
	TotalSize   int64
	AverageSize float64
	LargestIcon *struct {
		ID       string
		FileName string
		Size     int64
	}
	SmallestIcon *struct {
		ID       string
		FileName string
		Size     int64
	}
}

// Repository Port（契約のみ）
type RepositoryPort interface {
	// 取得
	GetByID(ctx context.Context, id string) (*TokenIcon, error)
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 作成/更新/削除
	Create(ctx context.Context, in CreateTokenIconInput) (*TokenIcon, error)
	Update(ctx context.Context, id string, in UpdateTokenIconInput) (*TokenIcon, error)
	Delete(ctx context.Context, id string) error

	// ───────────────────────────────────────────────────────────
	// 旧: バックエンドが直接アップロードする方式（互換のため残す）
	// ※ 今後は Signed URL を推奨（下の SignedUploadURLIssuer を参照）
	// 戻り値: 生成URL, 実サイズ(byte)
	// ───────────────────────────────────────────────────────────
	// UploadIcon(ctx context.Context, fileName, contentType string, r io.Reader) (url string, size int64, err error)

	// 統計（任意）
	GetTokenIconStats(ctx context.Context) (TokenIconStats, error)

	// （任意）トランザクション境界
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	// （任意）開発/テスト用メンテ
	Reset(ctx context.Context) error
}

// ─────────────────────────────────────────────────────────────
// ★ NEW: フロントが GCS に直接 PUT するための「署名付きURL」契約
// - 画像が無い create でも、後から icon を付け足せる想定
// - docId（= tokenBlueprintId 等）配下に置く objectPath を backend が決めて返す
// ─────────────────────────────────────────────────────────────

type SignedUploadURLInput struct {
	// 例: tokenBlueprintId（GCS のプレフィックスに使う）
	DocID string `json:"docId"`

	// 実ファイル名（表示用/監査用）
	FileName string `json:"fileName"`

	// Content-Type（例: image/png）
	ContentType string `json:"contentType"`

	// サイズ（任意）
	Size *int64 `json:"size,omitempty"`

	// 例: "icon" / "keep" など（用途識別、実装側で objectPath を決めるために使う）
	Purpose string `json:"purpose,omitempty"`
}

type SignedUploadURLResult struct {
	// ブラウザから PUT する先
	UploadURL string `json:"uploadUrl"`

	// GCS 上のオブジェクトパス（例: "{docId}/icon.png" や "{docId}/.keep"）
	ObjectPath string `json:"objectPath"`

	// 公開URL（例: https://storage.googleapis.com/<bucket>/<objectPath>）
	// ※ 実装により CDN / 署名付きGET に変える場合もあるが、ひとまず「image」に入れられるURLを返す想定
	PublicURL string `json:"publicUrl"`

	// 期限（任意）
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// SignedUploadURLIssuer は「署名付きURLを発行できる」実装が任意で満たす追加契約です。
// RepositoryPort に含めないことで、既存実装を一気に壊さず段階移行できます。
type SignedUploadURLIssuer interface {
	IssueSignedUploadURL(ctx context.Context, in SignedUploadURLInput) (*SignedUploadURLResult, error)
}

// 共通エラー（契約）
var (
	ErrNotFound = errors.New("tokenIcon: not found")
	ErrConflict = errors.New("tokenIcon: conflict")
)
