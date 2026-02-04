// backend/internal/domain/listImage/repository_port.go
package listImage

import (
	"context"
	"errors"

	common "narratives/internal/domain/common"
)

// UploadImageInput - アップロード入力（インフラ非依存の契約）
//
// NOTE:
//   - entity.go を正として、GCS は「環境ごとに1 bucket固定」＋「listId を prefix」に統一する前提。
//   - data URL を直接受け取ってアップロードまでやる方式は開発では便利だが、
//     将来は Signed URL 発行(別Port)に寄せてもよい。
//   - 本Portでは「どこへ保存するか」を安定させるため ObjectPath / ImageID を明示できるようにする。
type UploadImageInput struct {
	// data URL 形式を推奨: data:<mime>;base64,<payload>
	// ※ entity.go の ValidateDataURL と整合
	ImageData string `json:"imageData"`

	// 元ファイル名（拡張子チェック等に利用）
	FileName string `json:"fileName"`

	// 紐づくリストID（canonical objectPath の prefix に必須）
	ListID string `json:"listId"`

	// ✅ Optional: アップロード先バケット（任意）
	// 空なら実装側で env or domain DefaultBucket を使う（推奨: env）
	Bucket string `json:"bucket,omitempty"`

	// ✅ Optional: 画像ID（docId / canonical objectPath の末尾に使用）
	// - 空なら実装側で採番してよい
	// - 既存画像の「差し替え」をこの Upload で扱う場合は必須（同じ ImageID を指定）
	ImageID string `json:"imageId,omitempty"`

	// ✅ Optional: objectPath を明示したい場合
	// - 空なら canonical: lists/{listId}/images/{imageId} を使用
	// - もし指定するなら entity.go の ObjectPath と同じルール（URLではなくパス）
	ObjectPath string `json:"objectPath,omitempty"`

	// ✅ Optional: displayOrder（images サブコレクションに保存する並び順）
	// - nil なら実装側で決める（例: 末尾に追加）
	DisplayOrder *int `json:"displayOrder,omitempty"`
}

// ListImagePatch - 部分更新: nil のフィールドは更新しない
type ListImagePatch struct {
	URL        *string
	ObjectPath *string
	FileName   *string
	Size       *int64

	DisplayOrder *int
}

// Filter - フィルタ/検索条件（実装側で解釈）
type Filter struct {
	SearchQuery string // fileName/url/objectPath などの部分一致は実装側で解釈

	IDs    []string
	ListID *string
	// ListIDs は必要なら残す（Firestore だと IN 制約に注意）
	ListIDs []string

	FileNameLike  *string
	MinSize       *int64
	MaxSize       *int64
	MinDisplayOrd *int
	MaxDisplayOrd *int
}

// 共通型エイリアス（インフラ非依存）
type Sort = common.Sort
type SortOrder = common.SortOrder
type Page = common.Page
type PageResult[T any] = common.PageResult[T]
type CursorPage = common.CursorPage
type CursorPageResult[T any] = common.CursorPageResult[T]
type SaveOptions = common.SaveOptions

const (
	SortAsc  = common.SortAsc
	SortDesc = common.SortDesc
)

// 代表的なエラー（契約）
var (
	ErrNotFound = errors.New("listImage: not found")
	ErrConflict = errors.New("listImage: conflict")
)

// RepositoryPort - 画像管理のデータアクセスを抽象化（契約）
//
// NOTE:
// - entity.go を正として、ListImage は URL に加え ObjectPath を持つ（安定した更新/削除/署名URL発行のため）
// - Create/Update/Save は「メタデータ（Firestore等）」の永続化が主目的
// - Upload は「data URL を受け取り、GCSへ保存＋メタデータ作成」まで行う convenience 契約（開発向け）
type RepositoryPort interface {
	// 一覧・検索
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[ListImage], error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[ListImage], error)

	// 取得
	// NOTE: imageID は ListImage.ID（docId）
	FindByID(ctx context.Context, imageID string) (*ListImage, error)

	// listId 配下の画像一覧（displayOrder を扱う前提で利用頻度高）
	FindByListID(ctx context.Context, listID string) ([]ListImage, error)

	Exists(ctx context.Context, imageID string) (bool, error)

	// 作成/変更（メタデータ）
	Create(ctx context.Context, img ListImage) (ListImage, error)

	// 部分更新（メタデータ）
	Update(ctx context.Context, imageID string, patch ListImagePatch) (ListImage, error)

	// Upsert（メタデータ）
	Save(ctx context.Context, img ListImage, opts *SaveOptions) (ListImage, error)

	// アップロード（data URL 等を受け取り、GCSへ保存＋メタデータ作成まで行う）
	//
	// 推奨 canonical:
	//   bucket: env固定（in.Bucket が空なら env / DefaultBucket）
	//   objectPath: lists/{listId}/images/{imageId}
	//
	// 差し替え運用:
	// - in.ImageID を指定した場合は同じ objectPath に上書きすること（bucket create/delete をしない）
	Upload(ctx context.Context, in UploadImageInput) (*ListImage, error)

	// 削除
	// NOTE: GCS object 削除を含めるかは実装/ユースケース側で決める。
	// ここでは「image メタデータの削除」を最低保証とし、必要なら GCS も削除してよい。
	Delete(ctx context.Context, imageID string) error
}
