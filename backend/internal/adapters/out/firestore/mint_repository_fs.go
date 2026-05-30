// backend/internal/adapters/out/firestore/mint_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	usecase "narratives/internal/application/usecase"
	mintdom "narratives/internal/domain/mint"
	tokendom "narratives/internal/domain/token"
)

// MintRepositoryFS implements mint.MintRepository using Firestore.
// It also implements usecase.MintRequestPort for mint execution flow.
type MintRepositoryFS struct {
	Client *firestore.Client
}

var _ usecase.MintRequestPort = (*MintRepositoryFS)(nil)

func NewMintRepositoryFS(client *firestore.Client) *MintRepositoryFS {
	return &MintRepositoryFS{Client: client}
}

func (r *MintRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("mints")
}

func (r *MintRepositoryFS) tokensCol() *firestore.CollectionRef {
	return r.Client.Collection("tokens")
}

func (r *MintRepositoryFS) brandsCol() *firestore.CollectionRef {
	return r.Client.Collection("brands")
}

func (r *MintRepositoryFS) tokenBlueprintsCol() *firestore.CollectionRef {
	return r.Client.Collection("token_blueprints")
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

func nonEmptyStringAny(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func hasNonZeroTimestampAny(v any) bool {
	if v == nil {
		return false
	}

	switch t := v.(type) {
	case time.Time:
		return !t.IsZero()
	case *time.Time:
		return t != nil && !t.IsZero()
	default:
		return false
	}
}

// minted=false なのに mintedAt / 署名があるなど、
// 「既にミント済みの痕跡」があるかを判定する。
func hasMintedEvidence(raw map[string]any) bool {
	if raw == nil {
		return false
	}

	if v, ok := raw["mintedAt"]; ok && hasNonZeroTimestampAny(v) {
		return true
	}

	for _, k := range []string{"onChainTxSignature", "onchainTxSignature", "txSignature", "signature"} {
		if s := nonEmptyStringAny(raw[k]); s != "" {
			return true
		}
	}

	return false
}

type tokenBlueprintDoc struct {
	Name        string `firestore:"name"`
	Symbol      string `firestore:"symbol"`
	MetadataURI string `firestore:"metadataUri"`
}

type brandDoc struct {
	WalletAddress string `firestore:"walletAddress"`
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

// ============================================================
// MintRequestPort implementation
// ============================================================

// LoadForMinting は mintID を受け取り、
// mints + token_blueprints + brands から MintRequestForUsecase を構築して返します。
func (r *MintRepositoryFS) LoadForMinting(
	ctx context.Context,
	id string,
) (*usecase.MintRequestForUsecase, error) {
	if r == nil || r.Client == nil {
		return nil, fmt.Errorf("MintRepositoryFS is not initialized")
	}

	mintID := id
	if mintID == "" {
		return nil, fmt.Errorf("mint id is empty")
	}

	mintSnap, err := r.col().Doc(mintID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("mint %s not found", mintID)
		}
		return nil, fmt.Errorf("get mint %s: %w", mintID, err)
	}

	raw := mintSnap.Data()

	minted := asBool(raw["minted"])
	if !minted && hasMintedEvidence(raw) {
		_, _ = r.col().Doc(mintID).Update(ctx, []firestore.Update{
			{Path: "minted", Value: true},
		})
		return nil, fmt.Errorf("mint %s is already minted", mintID)
	}

	if minted {
		return nil, fmt.Errorf("mint %s is already minted", mintID)
	}

	brandID := s(raw["brandId"])
	if brandID == "" {
		return nil, fmt.Errorf("mint %s has empty brandId", mintID)
	}

	tbID := s(raw["tokenBlueprintId"])
	if tbID == "" {
		return nil, fmt.Errorf("mint %s has empty tokenBlueprintId", mintID)
	}

	productIDs := decodeStringSlice(raw["products"])

	if len(productIDs) > 0 {
		already := make([]string, 0, len(productIDs))

		for _, pid := range productIDs {
			if pid == "" {
				continue
			}

			_, err := r.tokensCol().Doc(pid).Get(ctx)
			if err == nil {
				already = append(already, pid)
				continue
			}

			if status.Code(err) == codes.NotFound {
				continue
			}

			return nil, fmt.Errorf("check token for product %s: %w", pid, err)
		}

		if len(already) > 0 {
			return nil, fmt.Errorf("tokens already exist for products: %v", already)
		}
	}

	tbSnap, err := r.tokenBlueprintsCol().Doc(tbID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("tokenBlueprint %s not found for mint %s", tbID, mintID)
		}
		return nil, fmt.Errorf("get tokenBlueprint %s: %w", tbID, err)
	}

	var tb tokenBlueprintDoc
	if err := tbSnap.DataTo(&tb); err != nil {
		return nil, fmt.Errorf("decode tokenBlueprint %s: %w", tbID, err)
	}

	name := tb.Name
	symbol := tb.Symbol
	metadataURI := tb.MetadataURI

	if name == "" || symbol == "" {
		return nil, fmt.Errorf("tokenBlueprint %s has empty name or symbol", tbID)
	}

	if metadataURI == "" {
		return nil, fmt.Errorf("tokenBlueprint %s has empty metadataUri", tbID)
	}

	brandSnap, err := r.brandsCol().Doc(brandID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("brand %s not found for mint %s", brandID, mintID)
		}
		return nil, fmt.Errorf("get brand %s: %w", brandID, err)
	}

	var b brandDoc
	if err := brandSnap.DataTo(&b); err != nil {
		return nil, fmt.Errorf("decode brand %s: %w", brandID, err)
	}

	toAddress := b.WalletAddress
	if toAddress == "" {
		return nil, fmt.Errorf("brand %s has empty walletAddress", brandID)
	}

	dto := &usecase.MintRequestForUsecase{
		ID:              mintID,
		ToAddress:       toAddress,
		ProductIDs:      productIDs,
		BlueprintName:   name,
		BlueprintSymbol: symbol,
		MetadataURI:     metadataURI,
	}

	return dto, nil
}

// MarkAsMinted はチェーンミント結果をもとに mints/{mintID} を更新します。
// mints には mintAddress を保存しない方針のため、mintAddress 更新は行いません。
func (r *MintRepositoryFS) MarkAsMinted(
	ctx context.Context,
	id string,
	result *tokendom.MintResult,
) error {
	if r == nil || r.Client == nil {
		return fmt.Errorf("MintRepositoryFS is not initialized")
	}

	if result == nil {
		return fmt.Errorf("mint result is nil")
	}

	mintID := id
	if mintID == "" {
		return fmt.Errorf("mint id is empty")
	}

	updates := []firestore.Update{
		{Path: "minted", Value: true},
		{Path: "mintedAt", Value: firestore.ServerTimestamp},
		{Path: "onChainTxSignature", Value: result.Signature},
	}

	_, err := r.col().Doc(mintID).Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("mint %s not found when updating as minted", mintID)
		}
		return fmt.Errorf("update mint %s as minted: %w", mintID, err)
	}

	return nil
}

// MarkProductsAsMinted は「1商品=1Mint」でミントした結果を Firestore に反映します。
// - tokens コレクションに [productId, mintAddress] を 1:1 で保存（docID=productId）
// - tokens には productId フィールドは保存しない（docID が productId なので不要）
// - tokens には tokenBlueprintId を保存する（商品型特定に必要）
// - tokens に toAddress / metadataUri をキャッシュとして保存する（体感高速化）
// - mints/{id} 自体も minted=true に更新（代表の MintResult を利用。ただし mintAddress は保存しない）
func (r *MintRepositoryFS) MarkProductsAsMinted(
	ctx context.Context,
	id string,
	minted []usecase.MintedTokenForUsecase,
) error {
	if r == nil || r.Client == nil {
		return fmt.Errorf("MintRepositoryFS is not initialized")
	}

	mintID := id
	if mintID == "" {
		return fmt.Errorf("mint id is empty")
	}

	if len(minted) == 0 {
		return fmt.Errorf("no minted results provided")
	}

	mintSnap, err := r.col().Doc(mintID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("mint %s not found when MarkProductsAsMinted", mintID)
		}
		return fmt.Errorf("get mint %s in MarkProductsAsMinted: %w", mintID, err)
	}

	raw := mintSnap.Data()

	brandID := s(raw["brandId"])
	if brandID == "" {
		return fmt.Errorf("mint %s has empty brandId in MarkProductsAsMinted", mintID)
	}

	tbID := s(raw["tokenBlueprintId"])
	if tbID == "" {
		return fmt.Errorf("mint %s has empty tokenBlueprintId in MarkProductsAsMinted", mintID)
	}

	brandSnap, err := r.brandsCol().Doc(brandID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("brand %s not found for mint %s", brandID, mintID)
		}
		return fmt.Errorf("get brand %s in MarkProductsAsMinted: %w", brandID, err)
	}

	var b brandDoc
	if err := brandSnap.DataTo(&b); err != nil {
		return fmt.Errorf("decode brand %s in MarkProductsAsMinted: %w", brandID, err)
	}

	toAddress := b.WalletAddress
	if toAddress == "" {
		return fmt.Errorf("brand %s has empty walletAddress (toAddress) in MarkProductsAsMinted", brandID)
	}

	tbSnap, err := r.tokenBlueprintsCol().Doc(tbID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("tokenBlueprint %s not found for mint %s", tbID, mintID)
		}
		return fmt.Errorf("get tokenBlueprint %s in MarkProductsAsMinted: %w", tbID, err)
	}

	var tb tokenBlueprintDoc
	if err := tbSnap.DataTo(&tb); err != nil {
		return fmt.Errorf("decode tokenBlueprint %s in MarkProductsAsMinted: %w", tbID, err)
	}

	metadataURI := tb.MetadataURI
	if metadataURI == "" {
		return fmt.Errorf("tokenBlueprint %s has empty metadataUri in MarkProductsAsMinted", tbID)
	}

	var lastResult *tokendom.MintResult
	for _, mt := range minted {
		if mt.Result != nil {
			lastResult = mt.Result
		}
	}

	if lastResult == nil {
		return fmt.Errorf("no valid MintResult found in minted list")
	}

	batch := r.Client.Batch()

	for _, mt := range minted {
		productID := mt.ProductID
		if productID == "" || mt.Result == nil {
			continue
		}

		data := map[string]interface{}{
			"brandId":            brandID,
			"tokenBlueprintId":   tbID,
			"mintAddress":        mt.Result.MintAddress,
			"onChainTxSignature": mt.Result.Signature,
			"mintedAt":           firestore.ServerTimestamp,
			"toAddress":          toAddress,
			"metadataUri":        metadataURI,
		}

		batch.Set(r.tokensCol().Doc(productID), data, firestore.MergeAll)
	}

	batch.Update(r.col().Doc(mintID), []firestore.Update{
		{Path: "minted", Value: true},
		{Path: "mintedAt", Value: firestore.ServerTimestamp},
		{Path: "onChainTxSignature", Value: lastResult.Signature},
	})

	_, err = batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("batch commit failed in MarkProductsAsMinted mintID=%s: %w", mintID, err)
	}

	return nil
}
