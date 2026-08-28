// backend/internal/application/port/post_transfer_resolver.go
package port

import "context"

// PostTransferResolveWarmer warms or refreshes transfer-related resolution
// state after a token transfer has completed.
type PostTransferResolveWarmer interface {
	ResolveAfterTransfer(
		ctx context.Context,
		avatarID string,
		assetID string,
	) error
}
