// backend\internal\domain\tokenBlueprint\arweave_port.go
package tokenBlueprint

import "context"

type ArweaveUploader interface {
	// metadataJSON は JSON エンコード済みの []byte を想定
	UploadJSON(ctx context.Context, metadataJSON []byte) (string, error)
}
