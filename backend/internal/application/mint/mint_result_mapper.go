// backend/internal/application/mint/mint_result_mapper.go
package mint

import (
	"errors"

	mintdom "narratives/internal/domain/mint"
	tokendom "narratives/internal/domain/token"
)

// MintResultMapper は、mints レコードのオンチェーン結果を MintResult へ変換し、
// オンチェーン結果を mints エンティティへ反映する Mapper。
type MintResultMapper struct{}

func NewMintResultMapper() *MintResultMapper {
	return &MintResultMapper{}
}

// FromMint builds tokendom.MintResult from mintdom.Mint.
func (m *MintResultMapper) FromMint(ent mintdom.Mint) *tokendom.MintResult {
	return &tokendom.MintResult{
		Signature:   ent.OnChainTxSignature,
		MintAddress: "",
		Slot:        0,
	}
}

// ApplyOnchainResult applies signature to the mint entity.
//
// Policy A で ID/InspectionID を揃える責務は usecase 側に残す。
// この Mapper はオンチェーン結果フィールドのみを扱う。
func (m *MintResultMapper) ApplyOnchainResult(ent *mintdom.Mint, result *tokendom.MintResult) error {
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
