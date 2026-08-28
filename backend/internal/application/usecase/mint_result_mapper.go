// backend/internal/application/usecase/mint_result_mapper.go
package usecase

import (
	"errors"

	mintdom "narratives/internal/domain/mint"
	tokendom "narratives/internal/domain/token"
)

// ============================================================
// MintResultMapper
// ============================================================

type MintResultMapper struct{}

func NewMintResultMapper() *MintResultMapper {
	return &MintResultMapper{}
}

func (m *MintResultMapper) FromMint(ent mintdom.Mint) *tokendom.MintResult {
	return &tokendom.MintResult{
		Signature:   ent.OnChainTxSignature,
		AssetID:     "",
		TreeAddress: "",
		LeafIndex:   0,
		Slot:        0,
	}
}

func (m *MintResultMapper) ApplyOnchainResult(
	ent *mintdom.Mint,
	result *tokendom.MintResult,
) error {
	if ent == nil {
		return errors.New("mint entity is nil")
	}

	if result == nil {
		return nil
	}

	if result.Signature != "" {
		ent.OnChainTxSignature = result.Signature
	}

	return nil
}

func lastMintResult(
	minted []MintedTokenForUsecase,
) *tokendom.MintResult {
	for i := len(minted) - 1; i >= 0; i-- {
		if minted[i].Result != nil {
			return minted[i].Result
		}
	}

	return nil
}
