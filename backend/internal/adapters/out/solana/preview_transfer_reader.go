// backend/internal/adapters/out/solana/preview_transfer_reader.go
package solana

import (
	"context"

	dto "narratives/internal/application/query/mall/dto"
	solanainfra "narratives/internal/infra/solana"
)

type PreviewTransferReader struct {
	Reader *solanainfra.TokenTransferReaderSolana
}

func NewPreviewTransferReader(
	reader *solanainfra.TokenTransferReaderSolana,
) *PreviewTransferReader {
	return &PreviewTransferReader{
		Reader: reader,
	}
}

func (r *PreviewTransferReader) ListByAssetID(
	ctx context.Context,
	assetID string,
) ([]dto.PreviewTransferInfo, error) {
	if r == nil || r.Reader == nil {
		return []dto.PreviewTransferInfo{}, nil
	}

	if assetID == "" {
		return []dto.PreviewTransferInfo{}, nil
	}

	res, err := r.Reader.ListAssetTransfers(
		ctx,
		solanainfra.ListAssetTransfersInput{
			AssetID: assetID,
		},
	)
	if err != nil {
		return nil, err
	}

	out := make(
		[]dto.PreviewTransferInfo,
		0,
		len(res.Transfers),
	)

	for _, tr := range res.Transfers {
		out = append(
			out,
			dto.PreviewTransferInfo{
				TransferredAt:     tr.TransferredAt,
				FromWalletAddress: tr.FromWalletAddress,
				ToWalletAddress:   tr.ToWalletAddress,
			},
		)
	}

	return out, nil
}
