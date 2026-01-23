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

// Repository Port（契約のみ）
type RepositoryPort interface {
	// 取得
	GetByID(ctx context.Context, id string) (*TokenIcon, error)

	// 作成/更新/削除
	Create(ctx context.Context, in CreateTokenIconInput) (*TokenIcon, error)
	Update(ctx context.Context, id string, in UpdateTokenIconInput) (*TokenIcon, error)
	Delete(ctx context.Context, id string) error
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
