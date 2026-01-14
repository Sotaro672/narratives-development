package firestore

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
)

var (
	ErrTokenOwnerUpdaterNotConfigured = errors.New("token_owner_updater_fs: not configured")
	ErrTokenOwnerUpdaterInvalidID     = errors.New("token_owner_updater_fs: productId is empty")
)

type TokenOwnerUpdaterFS struct {
	Client *firestore.Client

	// collection name (default "tokens")
	TokensCollection string
}

func NewTokenOwnerUpdaterFS(client *firestore.Client) *TokenOwnerUpdaterFS {
	return &TokenOwnerUpdaterFS{
		Client:           client,
		TokensCollection: "tokens",
	}
}

func (r *TokenOwnerUpdaterFS) UpdateToAddressByProductID(
	ctx context.Context,
	productID string,
	newToAddress string,
	now time.Time,
	txSignature string,
) error {
	if r == nil || r.Client == nil {
		return ErrTokenOwnerUpdaterNotConfigured
	}

	pid := strings.TrimSpace(productID)
	if pid == "" {
		return ErrTokenOwnerUpdaterInvalidID
	}

	col := strings.TrimSpace(r.TokensCollection)
	if col == "" {
		col = "tokens"
	}

	to := strings.TrimSpace(newToAddress)

	updates := map[string]any{
		"toAddress": to,
		"updatedAt": now.UTC(),
	}

	// 任意：運用上便利なので保存（フィールド名は必要なら調整）
	if sig := strings.TrimSpace(txSignature); sig != "" {
		updates["onChainTxSignature"] = sig
		updates["transferredAt"] = now.UTC()
	}

	_, err := r.Client.Collection(col).Doc(pid).Set(ctx, updates, firestore.MergeAll)
	return err
}
