// backend/internal/application/mint/mint_result_mapper.go
package mint

import (
	"errors"
	"strings"

	mintdom "narratives/internal/domain/mint"
	tokendom "narratives/internal/domain/token"
)

// MintResultMapper は、mints レコードのフィールド揺れを吸収して MintResult を生成する Mapper。
// また、オンチェーン結果（署名/アドレス）を mints エンティティへ反映する責務も担う。
type MintResultMapper struct{}

func NewMintResultMapper() *MintResultMapper { return &MintResultMapper{} }

// FromMint builds tokendom.MintResult from mintdom.Mint with field-name compatibility.
func (m *MintResultMapper) FromMint(ent mintdom.Mint) *tokendom.MintResult {
	sig := ""
	for _, name := range []string{
		"OnChainTxSignature",
		"OnchainTxSignature",
		"TxSignature",
		"Signature",
	} {
		if v := getIfExistsString(ent, name); v != "" {
			sig = v
			break
		}
	}

	addr := ""
	for _, name := range []string{
		"MintAddress",
		"OnChainMintAddress",
		"OnchainMintAddress",
	} {
		if v := getIfExistsString(ent, name); v != "" {
			addr = v
			break
		}
	}

	return &tokendom.MintResult{
		Signature:   sig,
		MintAddress: addr,
		Slot:        0,
	}
}

// ApplyOnchainResult applies signature/mintAddress to the mint entity with field-name compatibility.
// Policy A で ID/InspectionID を揃える責務は usecase 側に残す（この Mapper は結果フィールドのみを扱う）。
func (m *MintResultMapper) ApplyOnchainResult(ent *mintdom.Mint, result *tokendom.MintResult) error {
	if ent == nil {
		return errors.New("mint entity is nil")
	}
	if result == nil {
		return nil
	}

	sig := strings.TrimSpace(result.Signature)
	addr := strings.TrimSpace(result.MintAddress)

	if sig != "" {
		setIfExistsString(ent, "OnChainTxSignature", sig)
		setIfExistsString(ent, "OnchainTxSignature", sig)
		setIfExistsString(ent, "TxSignature", sig)
		setIfExistsString(ent, "Signature", sig)
	}
	if addr != "" {
		setIfExistsString(ent, "MintAddress", addr)
		setIfExistsString(ent, "OnChainMintAddress", addr)
		setIfExistsString(ent, "OnchainMintAddress", addr)
	}

	return nil
}
