package token

import (
	"errors"
	"strings"
)

// Token mirrors web-app/src/shared/types/token.ts
// Solana token on-chain metadata relation
type Token struct {
	MintAddress   string // Solana mint (base58, 32-byte pubkey)
	MintRequestID string // foreign key to mintRequest domain
	Owner         string // wallet address (base58, 32-byte pubkey)
}

// Errors
var (
	ErrInvalidMintRequestID = errors.New("token: invalid mintRequestId")
	ErrInvalidOwner         = errors.New("token: invalid owner")
	ErrInvalidMintAddress   = errors.New("token: invalid mintAddress")
	ErrInvalidSymbol        = errors.New("token: invalid symbol")
	ErrInvalidDecimals      = errors.New("token: invalid decimals")
	ErrInvalidUpdatedAt     = errors.New("token: invalid updatedAt")
)

// Policy (align with shared/constants/tokenConstants.ts as needed)
var (
	// Solana pubkey is 32 bytes base58-encoded; observed length typically 32..44.
	Base58MinLen     = 32
	Base58MaxLen     = 44
	base58Alphabet   = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	MaxMintRequestID = 128 // adjust if tokenConstants.ts defines a different limit
)

// Constructors

func New(mintAddress, mintRequestID, owner string) (Token, error) {
	t := Token{
		MintAddress:   strings.TrimSpace(mintAddress),
		MintRequestID: strings.TrimSpace(mintRequestID),
		Owner:         strings.TrimSpace(owner),
	}
	if err := t.validate(); err != nil {
		return Token{}, err
	}
	return t, nil
}

// Mutators

func (t *Token) UpdateOwner(owner string) error {
	owner = strings.TrimSpace(owner)
	if !isValidBase58Pubkey(owner) {
		return ErrInvalidOwner
	}
	t.Owner = owner
	return nil
}

func (t *Token) UpdateMintRequestID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" || (MaxMintRequestID > 0 && len(id) > MaxMintRequestID) {
		return ErrInvalidMintRequestID
	}
	t.MintRequestID = id
	return nil
}

// Validation

func (t Token) validate() error {
	if !isValidBase58Pubkey(t.MintAddress) {
		return ErrInvalidMintAddress
	}
	if t.MintRequestID == "" || (MaxMintRequestID > 0 && len(t.MintRequestID) > MaxMintRequestID) {
		return ErrInvalidMintRequestID
	}
	if !isValidBase58Pubkey(t.Owner) {
		return ErrInvalidOwner
	}
	return nil
}

// Helpers

func isValidBase58Pubkey(s string) bool {
	if s = strings.TrimSpace(s); s == "" {
		return false
	}
	// length check (approximate for Solana pubkeys)
	if len(s) < Base58MinLen || (Base58MaxLen > 0 && len(s) > Base58MaxLen) {
		return false
	}
	// character set check
	for i := 0; i < len(s); i++ {
		if !strings.ContainsRune(base58Alphabet, rune(s[i])) {
			return false
		}
	}
	return true
}
