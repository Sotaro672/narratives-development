package firestore

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"

	mintdom "narratives/internal/domain/mint"
)

// MintRepositoryFS implements mint.MintRepository using Firestore.
type MintRepositoryFS struct {
	Client *firestore.Client
}

func NewMintRepositoryFS(client *firestore.Client) *MintRepositoryFS {
	return &MintRepositoryFS{Client: client}
}

func (r *MintRepositoryFS) Create(ctx context.Context, m mintdom.Mint) (mintdom.Mint, error) {
	if r.Client == nil {
		return mintdom.Mint{}, errors.New("firestore client is nil")
	}

	col := r.Client.Collection("mints")

	// ID が空なら自動採番
	var docRef *firestore.DocumentRef
	if m.ID == "" {
		docRef = col.NewDoc()
		m.ID = docRef.ID
	} else {
		docRef = col.Doc(m.ID)
	}

	// CreatedAt がゼロならここで補完（通常は usecase 側で埋めている想定）
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}

	// ドメインの Validate
	if err := m.Validate(); err != nil {
		return mintdom.Mint{}, err
	}

	// Firestore に保存するデータ
	data := map[string]interface{}{
		"brandId":          m.BrandID,
		"tokenBlueprintId": m.TokenBlueprintID,
		"products":         m.Products,
		"createdAt":        m.CreatedAt,
		"createdBy":        m.CreatedBy,
		"minted":           m.Minted,
	}

	// mintedAt（任意）
	if m.MintedAt != nil && !m.MintedAt.IsZero() {
		data["mintedAt"] = m.MintedAt.UTC()
	}

	// ★ ScheduledBurnDate（任意）も保存
	if m.ScheduledBurnDate != nil && !m.ScheduledBurnDate.IsZero() {
		data["scheduledBurnDate"] = m.ScheduledBurnDate.UTC()
	}

	if _, err := docRef.Set(ctx, data); err != nil {
		return mintdom.Mint{}, err
	}

	return m, nil
}
