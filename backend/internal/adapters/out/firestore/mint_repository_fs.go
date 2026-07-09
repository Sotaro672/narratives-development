// backend/internal/adapters/out/firestore/mint_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	usecase "narratives/internal/application/usecase"
	mintdom "narratives/internal/domain/mint"
	tokendom "narratives/internal/domain/token"
)

// MintRepositoryFS implements mint.MintRepository using Firestore.
// It also implements:
// - usecase.MintRequestPort
// - usecase.MintProductMintRecorder
// - mint.MintProductTaskRepository
type MintRepositoryFS struct {
	Client *firestore.Client
}

var _ mintdom.MintRepository = (*MintRepositoryFS)(nil)
var _ mintdom.MintProductTaskRepository = (*MintRepositoryFS)(nil)
var _ usecase.MintRequestPort = (*MintRepositoryFS)(nil)
var _ usecase.MintProductMintRecorder = (*MintRepositoryFS)(nil)

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

func (r *MintRepositoryFS) taskCol(mintID string) *firestore.CollectionRef {
	return r.col().Doc(mintID).Collection("products")
}

func (r *MintRepositoryFS) taskDoc(mintID string, productID string) *firestore.DocumentRef {
	return r.taskCol(mintID).Doc(productID)
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

func mintStatusFromRaw(raw map[string]any, minted bool) mintdom.MintStatus {
	statusText := ""
	if raw != nil {
		statusText = s(raw["status"])
	}

	status := mintdom.MintStatus(statusText)
	if status.IsValid() {
		return status
	}

	if minted {
		return mintdom.MintStatusMinted
	}

	return mintdom.MintStatusCreated
}

func taskStatusFromRaw(raw map[string]any) mintdom.MintProductTaskStatus {
	statusText := ""
	if raw != nil {
		statusText = s(raw["status"])
	}

	status := mintdom.MintProductTaskStatus(statusText)
	if status.IsValid() {
		return status
	}

	return mintdom.MintProductTaskStatusPending
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
	minted := asBool(data["minted"])

	m := mintdom.Mint{
		ID:                 doc.Ref.ID,
		BrandID:            s(data["brandId"]),
		TokenBlueprintID:   s(data["tokenBlueprintId"]),
		Products:           decodeStringSlice(data["products"]),
		Status:             mintStatusFromRaw(data, minted),
		CreatedBy:          s(data["createdBy"]),
		CreatedAt:          asTimeUTC(data["createdAt"]),
		Minted:             minted,
		MintedAt:           asTimePtr(data["mintedAt"]),
		ScheduledBurnDate:  asTimePtr(data["scheduledBurnDate"]),
		OnChainTxSignature: s(data["onChainTxSignature"]),
	}

	if err := m.Validate(); err != nil {
		return mintdom.Mint{}, err
	}

	return m, nil
}

func encodeMintProductTask(t mintdom.MintProductTask) map[string]any {
	data := map[string]any{
		"mintId":       t.MintID,
		"productId":    t.ProductID,
		"status":       string(t.Status),
		"attemptCount": t.AttemptCount,
		"createdAt":    t.CreatedAt.UTC(),
		"updatedAt":    t.UpdatedAt.UTC(),
	}

	if t.MintAddress != "" {
		data["mintAddress"] = t.MintAddress
	}

	if t.Signature != "" {
		data["signature"] = t.Signature
	}

	if t.ErrorMessage != "" {
		data["errorMessage"] = t.ErrorMessage
	}

	if t.MintingStartedAt != nil && !t.MintingStartedAt.IsZero() {
		data["mintingStartedAt"] = t.MintingStartedAt.UTC()
	}

	if t.MintedAt != nil && !t.MintedAt.IsZero() {
		data["mintedAt"] = t.MintedAt.UTC()
	}

	if t.LastFailedAt != nil && !t.LastFailedAt.IsZero() {
		data["lastFailedAt"] = t.LastFailedAt.UTC()
	}

	return data
}

func decodeMintProductTaskFromDoc(
	mintID string,
	doc *firestore.DocumentSnapshot,
) (mintdom.MintProductTask, error) {
	if doc == nil || !doc.Exists() {
		return mintdom.MintProductTask{}, mintdom.ErrMintProductTaskNotFound
	}

	data := doc.Data()

	productID := s(data["productId"])
	if productID == "" {
		productID = doc.Ref.ID
	}

	taskMintID := s(data["mintId"])
	if taskMintID == "" {
		taskMintID = mintID
	}

	t := mintdom.MintProductTask{
		MintID:    taskMintID,
		ProductID: productID,

		Status: taskStatusFromRaw(data),

		AttemptCount: int(asInt64(data["attemptCount"])),

		MintAddress:  s(data["mintAddress"]),
		Signature:    s(data["signature"]),
		ErrorMessage: s(data["errorMessage"]),

		CreatedAt: asTimeUTC(data["createdAt"]),
		UpdatedAt: asTimeUTC(data["updatedAt"]),

		MintingStartedAt: asTimePtr(data["mintingStartedAt"]),
		MintedAt:         asTimePtr(data["mintedAt"]),
		LastFailedAt:     asTimePtr(data["lastFailedAt"]),
	}

	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}
	if t.UpdatedAt.IsZero() {
		t.UpdatedAt = t.CreatedAt
	}

	if err := t.Validate(); err != nil {
		return mintdom.MintProductTask{}, err
	}

	return t, nil
}

func asInt64(v any) int64 {
	if v == nil {
		return 0
	}

	switch n := v.(type) {
	case int:
		return int64(n)
	case int8:
		return int64(n)
	case int16:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return n
	case uint:
		return int64(n)
	case uint8:
		return int64(n)
	case uint16:
		return int64(n)
	case uint32:
		return int64(n)
	case uint64:
		if n > uint64(^uint(0)>>1) {
			return 0
		}
		return int64(n)
	case float32:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
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

	if m.Status == "" {
		m.Status = mintdom.MintStatusCreated
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
		"status":           string(m.Status),
		"createdBy":        m.CreatedBy,
	}

	if exists && existingSnap != nil && existingSnap.Exists() {
		edata := existingSnap.Data()

		existingMinted := asBool(edata["minted"])
		existingStatus := mintStatusFromRaw(edata, existingMinted)

		data["minted"] = existingMinted
		data["status"] = string(existingStatus)

		m.Minted = existingMinted
		m.Status = existingStatus

		if createdAt := asTimeUTC(edata["createdAt"]); !createdAt.IsZero() {
			data["createdAt"] = createdAt
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

	if m.CreatedAt.IsZero() || m.CreatedBy == "" || m.BrandID == "" || m.TokenBlueprintID == "" {
		existing, err := r.GetByID(ctx, m.ID)
		if err != nil {
			return mintdom.Mint{}, err
		}

		if m.CreatedAt.IsZero() {
			m.CreatedAt = existing.CreatedAt
		}
		if m.CreatedBy == "" {
			m.CreatedBy = existing.CreatedBy
		}
		if m.BrandID == "" {
			m.BrandID = existing.BrandID
		}
		if m.TokenBlueprintID == "" {
			m.TokenBlueprintID = existing.TokenBlueprintID
		}
		if len(m.Products) == 0 {
			m.Products = existing.Products
		}
	}

	if m.Status == "" {
		if m.Minted {
			m.Status = mintdom.MintStatusMinted
		} else {
			m.Status = mintdom.MintStatusCreated
		}
	}

	if err := m.Validate(); err != nil {
		return mintdom.Mint{}, err
	}

	data := map[string]any{
		"brandId":          m.BrandID,
		"tokenBlueprintId": m.TokenBlueprintID,
		"products":         m.Products,
		"status":           string(m.Status),
		"createdBy":        m.CreatedBy,
		"minted":           m.Minted,
	}

	if m.MintedAt != nil && !m.MintedAt.IsZero() {
		data["mintedAt"] = m.MintedAt.UTC()
	} else {
		data["mintedAt"] = firestore.Delete
	}

	if m.ScheduledBurnDate != nil && !m.ScheduledBurnDate.IsZero() {
		data["scheduledBurnDate"] = m.ScheduledBurnDate.UTC()
	} else {
		data["scheduledBurnDate"] = firestore.Delete
	}

	if m.OnChainTxSignature != "" {
		data["onChainTxSignature"] = m.OnChainTxSignature
	} else {
		data["onChainTxSignature"] = firestore.Delete
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
// MintProductTaskRepository implementation
// ============================================================

func (r *MintRepositoryFS) CreateTasks(
	ctx context.Context,
	mintID string,
	productIDs []string,
) ([]mintdom.MintProductTask, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if mintID == "" {
		return nil, errors.New("mint id is empty")
	}
	if len(productIDs) == 0 {
		return nil, mintdom.ErrInvalidProducts
	}

	now := time.Now().UTC()
	tasks := make([]mintdom.MintProductTask, 0, len(productIDs))

	batch := r.Client.Batch()
	writeCount := 0

	for _, productID := range productIDs {
		if productID == "" {
			return nil, mintdom.ErrInvalidProducts
		}

		docRef := r.taskDoc(mintID, productID)
		snap, err := docRef.Get(ctx)
		if err == nil && snap.Exists() {
			existing, decErr := decodeMintProductTaskFromDoc(mintID, snap)
			if decErr != nil {
				return nil, decErr
			}
			tasks = append(tasks, existing)
			continue
		}
		if err != nil && status.Code(err) != codes.NotFound {
			return nil, fmt.Errorf("get mint product task mintID=%s productID=%s: %w", mintID, productID, err)
		}

		task, err := mintdom.NewMintProductTask(mintID, productID, now)
		if err != nil {
			return nil, err
		}

		batch.Create(docRef, encodeMintProductTask(task))
		writeCount++
		tasks = append(tasks, task)
	}

	if writeCount > 0 {
		if _, err := batch.Commit(ctx); err != nil {
			return nil, fmt.Errorf("create mint product tasks mintID=%s: %w", mintID, err)
		}
	}

	return tasks, nil
}

func (r *MintRepositoryFS) GetByProductID(
	ctx context.Context,
	mintID string,
	productID string,
) (mintdom.MintProductTask, error) {
	if r == nil || r.Client == nil {
		return mintdom.MintProductTask{}, errors.New("firestore client is nil")
	}
	if mintID == "" {
		return mintdom.MintProductTask{}, errors.New("mint id is empty")
	}
	if productID == "" {
		return mintdom.MintProductTask{}, errors.New("product id is empty")
	}

	snap, err := r.taskDoc(mintID, productID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return mintdom.MintProductTask{}, mintdom.ErrMintProductTaskNotFound
		}
		return mintdom.MintProductTask{}, err
	}

	return decodeMintProductTaskFromDoc(mintID, snap)
}

func (r *MintRepositoryFS) ListByMintID(
	ctx context.Context,
	mintID string,
) ([]mintdom.MintProductTask, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if mintID == "" {
		return nil, errors.New("mint id is empty")
	}

	iter := r.taskCol(mintID).
		OrderBy("createdAt", firestore.Asc).
		OrderBy("productId", firestore.Asc).
		Documents(ctx)
	defer iter.Stop()

	tasks := []mintdom.MintProductTask{}

	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		task, err := decodeMintProductTaskFromDoc(mintID, doc)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (r *MintRepositoryFS) GetNextExecutableTask(
	ctx context.Context,
	mintID string,
) (mintdom.MintProductTask, error) {
	if r == nil || r.Client == nil {
		return mintdom.MintProductTask{}, errors.New("firestore client is nil")
	}
	if mintID == "" {
		return mintdom.MintProductTask{}, errors.New("mint id is empty")
	}

	statuses := []mintdom.MintProductTaskStatus{
		mintdom.MintProductTaskStatusPending,
		mintdom.MintProductTaskStatusFailedRetryable,
	}

	for _, st := range statuses {
		iter := r.taskCol(mintID).
			Where("status", "==", string(st)).
			OrderBy("createdAt", firestore.Asc).
			OrderBy("productId", firestore.Asc).
			Limit(1).
			Documents(ctx)

		doc, err := iter.Next()
		iter.Stop()

		if errors.Is(err, iterator.Done) {
			continue
		}
		if err != nil {
			return mintdom.MintProductTask{}, err
		}

		return decodeMintProductTaskFromDoc(mintID, doc)
	}

	return mintdom.MintProductTask{}, mintdom.ErrMintProductTaskNotFound
}

func (r *MintRepositoryFS) MarkMinting(
	ctx context.Context,
	mintID string,
	productID string,
) (mintdom.MintProductTask, error) {
	if r == nil || r.Client == nil {
		return mintdom.MintProductTask{}, errors.New("firestore client is nil")
	}
	if mintID == "" {
		return mintdom.MintProductTask{}, errors.New("mint id is empty")
	}
	if productID == "" {
		return mintdom.MintProductTask{}, errors.New("product id is empty")
	}

	docRef := r.taskDoc(mintID, productID)
	var updated mintdom.MintProductTask

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(docRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return mintdom.ErrMintProductTaskNotFound
			}
			return err
		}

		task, err := decodeMintProductTaskFromDoc(mintID, snap)
		if err != nil {
			return err
		}

		if err := task.MarkMinting(time.Now().UTC()); err != nil {
			return err
		}

		updated = task

		return tx.Set(docRef, encodeMintProductTask(task), firestore.MergeAll)
	})
	if err != nil {
		return mintdom.MintProductTask{}, err
	}

	return updated, nil
}

func (r *MintRepositoryFS) MarkMinted(
	ctx context.Context,
	mintID string,
	productID string,
	mintAddress string,
	signature string,
) (mintdom.MintProductTask, error) {
	if r == nil || r.Client == nil {
		return mintdom.MintProductTask{}, errors.New("firestore client is nil")
	}
	if mintID == "" {
		return mintdom.MintProductTask{}, errors.New("mint id is empty")
	}
	if productID == "" {
		return mintdom.MintProductTask{}, errors.New("product id is empty")
	}

	docRef := r.taskDoc(mintID, productID)
	var updated mintdom.MintProductTask

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(docRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return mintdom.ErrMintProductTaskNotFound
			}
			return err
		}

		task, err := decodeMintProductTaskFromDoc(mintID, snap)
		if err != nil {
			return err
		}

		if err := task.MarkMinted(time.Now().UTC(), mintAddress, signature); err != nil {
			return err
		}

		updated = task

		return tx.Set(docRef, encodeMintProductTask(task), firestore.MergeAll)
	})
	if err != nil {
		return mintdom.MintProductTask{}, err
	}

	return updated, nil
}

func (r *MintRepositoryFS) MarkFailedRetryable(
	ctx context.Context,
	mintID string,
	productID string,
	message string,
) (mintdom.MintProductTask, error) {
	if r == nil || r.Client == nil {
		return mintdom.MintProductTask{}, errors.New("firestore client is nil")
	}
	if mintID == "" {
		return mintdom.MintProductTask{}, errors.New("mint id is empty")
	}
	if productID == "" {
		return mintdom.MintProductTask{}, errors.New("product id is empty")
	}

	docRef := r.taskDoc(mintID, productID)
	var updated mintdom.MintProductTask

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(docRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return mintdom.ErrMintProductTaskNotFound
			}
			return err
		}

		task, err := decodeMintProductTaskFromDoc(mintID, snap)
		if err != nil {
			return err
		}

		if err := task.MarkFailedRetryable(time.Now().UTC(), message); err != nil {
			return err
		}

		updated = task

		return tx.Set(docRef, encodeMintProductTask(task), firestore.MergeAll)
	})
	if err != nil {
		return mintdom.MintProductTask{}, err
	}

	return updated, nil
}

func (r *MintRepositoryFS) MarkFailedFatal(
	ctx context.Context,
	mintID string,
	productID string,
	message string,
) (mintdom.MintProductTask, error) {
	if r == nil || r.Client == nil {
		return mintdom.MintProductTask{}, errors.New("firestore client is nil")
	}
	if mintID == "" {
		return mintdom.MintProductTask{}, errors.New("mint id is empty")
	}
	if productID == "" {
		return mintdom.MintProductTask{}, errors.New("product id is empty")
	}

	docRef := r.taskDoc(mintID, productID)
	var updated mintdom.MintProductTask

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(docRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return mintdom.ErrMintProductTaskNotFound
			}
			return err
		}

		task, err := decodeMintProductTaskFromDoc(mintID, snap)
		if err != nil {
			return err
		}

		if err := task.MarkFailedFatal(time.Now().UTC(), message); err != nil {
			return err
		}

		updated = task

		return tx.Set(docRef, encodeMintProductTask(task), firestore.MergeAll)
	})
	if err != nil {
		return mintdom.MintProductTask{}, err
	}

	return updated, nil
}

func (r *MintRepositoryFS) ResetRetryableToPending(
	ctx context.Context,
	mintID string,
	productID string,
) (mintdom.MintProductTask, error) {
	if r == nil || r.Client == nil {
		return mintdom.MintProductTask{}, errors.New("firestore client is nil")
	}
	if mintID == "" {
		return mintdom.MintProductTask{}, errors.New("mint id is empty")
	}
	if productID == "" {
		return mintdom.MintProductTask{}, errors.New("product id is empty")
	}

	docRef := r.taskDoc(mintID, productID)
	var updated mintdom.MintProductTask

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(docRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return mintdom.ErrMintProductTaskNotFound
			}
			return err
		}

		task, err := decodeMintProductTaskFromDoc(mintID, snap)
		if err != nil {
			return err
		}

		if err := task.ResetToPending(time.Now().UTC()); err != nil {
			return err
		}

		updated = task

		return tx.Set(docRef, encodeMintProductTask(task), firestore.MergeAll)
	})
	if err != nil {
		return mintdom.MintProductTask{}, err
	}

	return updated, nil
}

// ============================================================
// MintRequestPort implementation
// ============================================================

// LoadForMinting は mintID を受け取り、
// mints + token_blueprints + brands から MintRequestForUsecase を構築して返します。
// 初回 mint では tokenBlueprint.metadataUri が空の可能性があります。
// metadataUri の生成・保存は MintUsecase.ensureMetadataURI が担当するため、
// この adapter では metadataUri が空でもエラーにせず DTO に詰めて返します。
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
			{Path: "status", Value: string(mintdom.MintStatusMinted)},
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

	actorID := s(raw["createdBy"])

	productIDs := decodeStringSlice(raw["products"])

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
		ID:               mintID,
		TokenBlueprintID: tbID,
		ActorID:          actorID,
		ToAddress:        toAddress,
		ProductIDs:       productIDs,
		BlueprintName:    name,
		BlueprintSymbol:  symbol,
		MetadataURI:      metadataURI,
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
		{Path: "status", Value: string(mintdom.MintStatusMinted)},
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

// RecordProductAsMinted は productId 1件分の mint 結果を Firestore に反映します。
// - tokens コレクションに [productId, mintAddress] を 1:1 で保存（docID=productId）
// - 親 mints/{mintID} はここでは minted=true にしません。
// - 親の完了更新は、全 MintProductTask が MINTED になった後に MintUsecase.Update 経由で行います。
func (r *MintRepositoryFS) RecordProductAsMinted(
	ctx context.Context,
	id string,
	mt usecase.MintedTokenForUsecase,
) error {
	if r == nil || r.Client == nil {
		return fmt.Errorf("MintRepositoryFS is not initialized")
	}

	mintID := id
	if mintID == "" {
		return fmt.Errorf("mint id is empty")
	}

	productID := mt.ProductID
	if productID == "" {
		return fmt.Errorf("product id is empty")
	}

	if mt.Result == nil {
		return fmt.Errorf("mint result is nil for product %s", productID)
	}

	mintSnap, err := r.col().Doc(mintID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("mint %s not found when RecordProductAsMinted", mintID)
		}
		return fmt.Errorf("get mint %s in RecordProductAsMinted: %w", mintID, err)
	}

	raw := mintSnap.Data()

	brandID := s(raw["brandId"])
	if brandID == "" {
		return fmt.Errorf("mint %s has empty brandId in RecordProductAsMinted", mintID)
	}

	tbID := s(raw["tokenBlueprintId"])
	if tbID == "" {
		return fmt.Errorf("mint %s has empty tokenBlueprintId in RecordProductAsMinted", mintID)
	}

	brandSnap, err := r.brandsCol().Doc(brandID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("brand %s not found for mint %s", brandID, mintID)
		}
		return fmt.Errorf("get brand %s in RecordProductAsMinted: %w", brandID, err)
	}

	var b brandDoc
	if err := brandSnap.DataTo(&b); err != nil {
		return fmt.Errorf("decode brand %s in RecordProductAsMinted: %w", brandID, err)
	}

	toAddress := b.WalletAddress
	if toAddress == "" {
		return fmt.Errorf("brand %s has empty walletAddress (toAddress) in RecordProductAsMinted", brandID)
	}

	tbSnap, err := r.tokenBlueprintsCol().Doc(tbID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("tokenBlueprint %s not found for mint %s", tbID, mintID)
		}
		return fmt.Errorf("get tokenBlueprint %s in RecordProductAsMinted: %w", tbID, err)
	}

	var tb tokenBlueprintDoc
	if err := tbSnap.DataTo(&tb); err != nil {
		return fmt.Errorf("decode tokenBlueprint %s in RecordProductAsMinted: %w", tbID, err)
	}

	metadataURI := tb.MetadataURI
	if metadataURI == "" {
		return fmt.Errorf("tokenBlueprint %s has empty metadataUri in RecordProductAsMinted", tbID)
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

	if _, err := r.tokensCol().Doc(productID).Set(ctx, data, firestore.MergeAll); err != nil {
		return fmt.Errorf("set token productID=%s mintID=%s: %w", productID, mintID, err)
	}

	return nil
}

// MarkProductsAsMinted は旧互換用です。
// 新しい1件ずつmintフローでは、親 mints/{id} をここで minted=true にしません。
// 親 Mint の完了更新は、全 product task が MINTED になった後に MintUsecase 側で行います。
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

	for _, mt := range minted {
		if mt.ProductID == "" || mt.Result == nil {
			continue
		}

		if err := r.RecordProductAsMinted(ctx, mintID, mt); err != nil {
			return err
		}
	}

	return nil
}
