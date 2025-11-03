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

// Wallets/Tokens DDL from domain

// TokensTableDDL defines the SQL for the tokens migration.
const TokensTableDDL = `
-- Migration: Initialize Token domain
-- Based on web-app/src/shared/types/token.ts

BEGIN;

CREATE TABLE IF NOT EXISTS tokens (
  mint_address    TEXT        PRIMARY KEY,                      -- TS: Token.mintAddress
  mint_request_id TEXT        NOT NULL,                         -- TS: Token.mintRequestId
  owner           TEXT        NOT NULL,                         -- TS: Token.owner (wallet address)
  minted_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_transferred_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT chk_tokens_mint_address_non_empty CHECK (char_length(trim(mint_address)) > 0),
  CONSTRAINT chk_tokens_owner_non_empty        CHECK (char_length(trim(owner)) > 0),

  -- Wallet must exist
  CONSTRAINT fk_tokens_owner_wallet
    FOREIGN KEY (owner) REFERENCES wallets(wallet_address) ON DELETE RESTRICT
);

-- Optional FK to mint_requests(id) if the table exists
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'mint_requests'
  ) THEN
    BEGIN
      ALTER TABLE tokens
        ADD CONSTRAINT fk_tokens_mint_request
        FOREIGN KEY (mint_request_id) REFERENCES mint_requests(id) ON DELETE RESTRICT;
    EXCEPTION WHEN duplicate_object THEN
      NULL;
    END;
  END IF;
END$$;

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_tokens_owner            ON tokens(owner);
CREATE INDEX IF NOT EXISTS idx_tokens_mint_request_id  ON tokens(mint_request_id);
CREATE INDEX IF NOT EXISTS idx_tokens_minted_at       ON tokens(minted_at);
CREATE INDEX IF NOT EXISTS idx_tokens_last_transferred_at       ON tokens(last_transferred_at);

COMMIT;
`
