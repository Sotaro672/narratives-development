// backend/internal/application/port/news_image_storage.go
package port

import (
	"context"
	"io"
)

// NewsImageUploadInput は News 画像を外部Storageへ保存するための入力。
//
// HTTP multipart / Firebase Storage / GCS 固有の型には依存しない。
// NewsID はStorage上のobject pathをNews単位に分離するために使用する。
type NewsImageUploadInput struct {
	NewsID      string
	FileName    string
	ContentType string
	FileSize    int64
	Reader      io.Reader
}

// NewsImageUploadResult はStorageへの保存結果を表す。
//
// この値から domain.NewsImage を生成できるだけの情報を返す。
// Storage実装固有のobjectやclientはapplication層へ公開しない。
type NewsImageUploadResult struct {
	FileURL    string
	ObjectPath string
	FileName   string
	MimeType   string
	FileSize   int64
}

// NewsImageStorage manages the optional main image associated with News.
//
// News画像は1件につき最大1枚を想定する。
// 画像本体は外部Storageに保存し、News documentにはUpload結果から生成した
// 画像メタデータのみを保存する。
type NewsImageStorage interface {
	// Upload stores one News image.
	//
	// Expected implementation policy:
	// - NewsIDごとに独立したobject pathへ保存する。
	// - Readerから受け取った画像データをStorageへ書き込む。
	// - FileURLはConsole / Mallから参照可能な永続URLを返す。
	// - ObjectPathは後続のrollback削除に使用できる値を返す。
	Upload(
		ctx context.Context,
		input NewsImageUploadInput,
	) (NewsImageUploadResult, error)

	// Delete physically deletes one News image object.
	//
	// News作成に失敗した場合のrollbackにも使用するため、
	// objectが既に存在しない場合は成功として扱える実装を想定する。
	Delete(
		ctx context.Context,
		objectPath string,
	) error
}
