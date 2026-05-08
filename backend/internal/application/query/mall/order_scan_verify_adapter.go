// backend\internal\application\query\mall\order_scan_verify_adapter.go
package mall

import (
	"context"

	"narratives/internal/application/usecase"
)

type scanVerifierAdapter struct {
	q *OrderScanVerifyQuery
}

func NewScanVerifierAdapter(q *OrderScanVerifyQuery) usecase.ScanVerifier {
	return &scanVerifierAdapter{q: q}
}

func (a *scanVerifierAdapter) Verify(ctx context.Context, avatarID, productID string) (usecase.ScanVerifyResult, error) {
	out, err := a.q.VerifyScanPurchasedByAvatarID(ctx, avatarID, productID)
	if err != nil {
		return usecase.ScanVerifyResult{}, err
	}

	return usecase.ScanVerifyResult{
		AvatarID:                out.AvatarID,
		ProductID:               out.ProductID,
		ScannedModelID:          out.ScannedModelID,
		ScannedTokenBlueprintID: out.ScannedTokenBlueprintID,
		Matched:                 out.Matched,
	}, nil
}
