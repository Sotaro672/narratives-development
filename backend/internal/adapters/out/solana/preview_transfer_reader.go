// backend\internal\adapters\out\solana\preview_transfer_reader.go
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
	return &PreviewTransferReader{Reader: reader}
}

func (r *PreviewTransferReader) ListByMintAddress(
	ctx context.Context,
	mintAddress string,
) ([]dto.PreviewTransferInfo, error) {
	if r == nil || r.Reader == nil {
		return []dto.PreviewTransferInfo{}, nil
	}

	res, err := r.Reader.ListMintTransfers(ctx, solanainfra.ListMintTransfersInput{
		MintAddress: mintAddress,
	})
	if err != nil {
		return nil, err
	}

	out := make([]dto.PreviewTransferInfo, 0, len(res.Transfers))
	for _, tr := range res.Transfers {
		out = append(out, dto.PreviewTransferInfo{
			TransferredAt:     tr.TransferredAt,
			FromWalletAddress: tr.FromWalletAddress,
			ToWalletAddress:   tr.ToWalletAddress,
		})
	}
	return out, nil
}
