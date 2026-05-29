// backend/internal/adapters/out/firestore/mint_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mintdom "narratives/internal/domain/mint"
)

// MintRepositoryFS implements mint.MintRepository using Firestore.
type MintRepositoryFS struct {
	Client *firestore.Client
}

func NewMintRepositoryFS(client *firestore.Client) *MintRepositoryFS {
	return &MintRepositoryFS{Client: client}
}

func (r *MintRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("mints")
}

func asBool(v any) bool {
	b, ok := v.(bool)
	if ok {
		return b
	}
	return false
}

// s delegates to helper_repository_fs.go's asString(v any).
func s(v any) string {
	return asString(v)
}

// asTimeUTC adapts helper_repository_fs.go's asTime(v any) (time.Time, bool) to UTC time.Time.
func asTimeUTC(v any) time.Time {
	if tt, ok := asTime(v); ok {
		return tt.UTC()
	}
	return time.Time{}
}

func asTimePtr(v any) *time.Time {
	if v == nil {
		return nil
	}

	switch t := v.(type) {
	case time.Time:
		if t.IsZero() {
			return nil
		}
		tt := t.UTC()
		return &tt
	case *time.Time:
		if t == nil || t.IsZero() {
			return nil
		}
		tt := t.UTC()
		return &tt
	default:
		return nil
	}
}

func decodeStringSlice(v any) []string {
	if v == nil {
		return []string{}
	}

	switch vv := v.(type) {
	case []string:
		out := make([]string, len(vv))
		copy(out, vv)
		return out

	case []any:
		out := make([]string, 0, len(vv))
		for _, elem := range vv {
			if sv, ok := elem.(string); ok {
				out = append(out, sv)
			}
		}
		return out

	default:
		return []string{}
	}
}

func decodeMintFromDoc(doc *firestore.DocumentSnapshot) (mintdom.Mint, error) {
	if doc == nil || !doc.Exists() {
		return mintdom.Mint{}, errors.New("doc is nil or not exists")
	}

	data := doc.Data()

	m := mintdom.Mint{
		ID:                 doc.Ref.ID,
		BrandID:            s(data["brandId"]),
		TokenBlueprintID:   s(data["tokenBlueprintId"]),
		Products:           decodeStringSlice(data["products"]),
		CreatedBy:          s(data["createdBy"]),
		CreatedAt:          asTimeUTC(data["createdAt"]),
		Minted:             asBool(data["minted"]),
		MintedAt:           asTimePtr(data["mintedAt"]),
		ScheduledBurnDate:  asTimePtr(data["scheduledBurnDate"]),
		OnChainTxSignature: s(data["onChainTxSignature"]),
	}

	if err := m.Validate(); err != nil {
		return mintdom.Mint{}, err
	}

	return m, nil
}

// ============================================================
// CRUD
// ============================================================

func (r *MintRepositoryFS) Create(ctx context.Context, m mintdom.Mint) (mintdom.Mint, error) {
	if r == nil || r.Client == nil {
		return mintdom.Mint{}, errors.New("firestore client is nil")
	}

	if m.ID == "" {
		return mintdom.Mint{}, errors.New("mint.ID is empty")
	}

	docRef := r.col().Doc(m.ID)

	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}

	if err := m.Validate(); err != nil {
		return mintdom.Mint{}, err
	}

	existingSnap, getErr := docRef.Get(ctx)
	exists := getErr == nil
	if getErr != nil && status.Code(getErr) != codes.NotFound {
		return mintdom.Mint{}, getErr
	}

	data := map[string]any{
		"brandId":          m.BrandID,
		"tokenBlueprintId": m.TokenBlueprintID,
		"products":         m.Products,
		"createdBy":        m.CreatedBy,
	}

	if exists && existingSnap != nil && existingSnap.Exists() {
		edata := existingSnap.Data()

		data["minted"] = asBool(edata["minted"])

		if createdAt := asTimeUTC(edata["createdAt"]); !createdAt.IsZero() {
			m.CreatedAt = createdAt
		}

		if mintedAt := asTimePtr(edata["mintedAt"]); mintedAt != nil && !mintedAt.IsZero() {
			data["mintedAt"] = mintedAt.UTC()
			m.MintedAt = mintedAt
		}

		if scheduledBurnDate := asTimePtr(edata["scheduledBurnDate"]); scheduledBurnDate != nil && !scheduledBurnDate.IsZero() {
			data["scheduledBurnDate"] = scheduledBurnDate.UTC()
			m.ScheduledBurnDate = scheduledBurnDate
		}

		if onChainTxSignature := s(edata["onChainTxSignature"]); onChainTxSignature != "" {
			data["onChainTxSignature"] = onChainTxSignature
			m.OnChainTxSignature = onChainTxSignature
		}
	} else {
		data["createdAt"] = m.CreatedAt.UTC()
		data["minted"] = m.Minted

		if m.MintedAt != nil && !m.MintedAt.IsZero() {
			data["mintedAt"] = m.MintedAt.UTC()
		}

		if m.ScheduledBurnDate != nil && !m.ScheduledBurnDate.IsZero() {
			data["scheduledBurnDate"] = m.ScheduledBurnDate.UTC()
		}

		if m.OnChainTxSignature != "" {
			data["onChainTxSignature"] = m.OnChainTxSignature
		}
	}

	if !exists {
		if _, err := docRef.Create(ctx, data); err != nil {
			if status.Code(err) != codes.AlreadyExists {
				return mintdom.Mint{}, err
			}

			if _, err2 := docRef.Set(ctx, data, firestore.MergeAll); err2 != nil {
				return mintdom.Mint{}, err2
			}
		}
	} else {
		if _, err := docRef.Set(ctx, data, firestore.MergeAll); err != nil {
			return mintdom.Mint{}, err
		}
	}

	return m, nil
}

// Update updates a Mint.
// docId is fixed to m.ID.
// In AMOL/Narratives, m.ID is expected to be productionID == inspectionID == mintID.
func (r *MintRepositoryFS) Update(ctx context.Context, m mintdom.Mint) (mintdom.Mint, error) {
	if r == nil || r.Client == nil {
		return mintdom.Mint{}, errors.New("firestore client is nil")
	}

	if m.ID == "" {
		return mintdom.Mint{}, errors.New("mint.ID is empty")
	}

	docRef := r.col().Doc(m.ID)

	// createdAt がゼロなら既存から補完（Validate を通すため）
	if m.CreatedAt.IsZero() {
		existing, err := r.GetByID(ctx, m.ID)
		if err != nil {
			return mintdom.Mint{}, err
		}
		m.CreatedAt = existing.CreatedAt
	}

	if err := m.Validate(); err != nil {
		return mintdom.Mint{}, err
	}

	data := map[string]any{
		"brandId":          m.BrandID,
		"tokenBlueprintId": m.TokenBlueprintID,
		"products":         m.Products,
		"createdBy":        m.CreatedBy,
		"minted":           m.Minted,
	}

	if m.MintedAt != nil && !m.MintedAt.IsZero() {
		data["mintedAt"] = m.MintedAt.UTC()
	}

	if m.ScheduledBurnDate != nil && !m.ScheduledBurnDate.IsZero() {
		data["scheduledBurnDate"] = m.ScheduledBurnDate.UTC()
	}

	if m.OnChainTxSignature != "" {
		data["onChainTxSignature"] = m.OnChainTxSignature
	}

	_, err := docRef.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return mintdom.Mint{}, mintdom.ErrNotFound
		}
		return mintdom.Mint{}, err
	}

	return m, nil
}

// ============================================================
// Queries
// ============================================================

// GetByID returns a Mint by docId.
// docId is expected to be productionID == inspectionID == mintID.
func (r *MintRepositoryFS) GetByID(ctx context.Context, id string) (mintdom.Mint, error) {
	if r == nil || r.Client == nil {
		return mintdom.Mint{}, errors.New("firestore client is nil")
	}

	if id == "" {
		return mintdom.Mint{}, errors.New("id is empty")
	}

	doc, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return mintdom.Mint{}, mintdom.ErrNotFound
		}
		return mintdom.Mint{}, err
	}

	return decodeMintFromDoc(doc)
}
