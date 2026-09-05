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

func mintStatusFromRaw(raw map[string]any) (mintdom.MintStatus, error) {
	if raw == nil {
		return mintdom.MintStatusCreated, nil
	}

	statusText, err := firestoreOptionalString(raw, "status")
	if err != nil {
		return "", err
	}
	if statusText == nil {
		return mintdom.MintStatusCreated, nil
	}

	status := mintdom.MintStatus(*statusText)
	if status.IsValid() {
		return status, nil
	}

	return mintdom.MintStatusCreated, nil
}

func taskStatusFromRaw(raw map[string]any) (mintdom.MintProductTaskStatus, error) {
	if raw == nil {
		return mintdom.MintProductTaskStatusPending, nil
	}

	statusText, err := firestoreOptionalString(raw, "status")
	if err != nil {
		return "", err
	}
	if statusText == nil {
		return mintdom.MintProductTaskStatusPending, nil
	}

	status := mintdom.MintProductTaskStatus(*statusText)
	if status.IsValid() {
		return status, nil
	}

	return mintdom.MintProductTaskStatusPending, nil
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
	if data == nil {
		return mintdom.Mint{}, errors.New("mint document data is nil")
	}

	brandID, err := firestoreRequiredString(data, "brandId")
	if err != nil {
		return mintdom.Mint{}, err
	}
	tokenBlueprintID, err := firestoreRequiredString(data, "tokenBlueprintId")
	if err != nil {
		return mintdom.Mint{}, err
	}
	createdBy, err := firestoreRequiredString(data, "createdBy")
	if err != nil {
		return mintdom.Mint{}, err
	}
	requestedBy, err := firestoreOptionalString(data, "requestedBy")
	if err != nil {
		return mintdom.Mint{}, err
	}
	onChainTxSignature, err := firestoreOptionalString(data, "onChainTxSignature")
	if err != nil {
		return mintdom.Mint{}, err
	}
	mintStatus, err := mintStatusFromRaw(data)
	if err != nil {
		return mintdom.Mint{}, err
	}
	createdAt, err := firestoreRequiredTime(data, "createdAt")
	if err != nil {
		return mintdom.Mint{}, err
	}
	mintedAt, err := firestoreOptionalTime(data, "mintedAt")
	if err != nil {
		return mintdom.Mint{}, err
	}
	scheduledBurnDate, err := firestoreOptionalTime(data, "scheduledBurnDate")
	if err != nil {
		return mintdom.Mint{}, err
	}

	m := mintdom.Mint{
		ID:                 doc.Ref.ID,
		BrandID:            brandID,
		TokenBlueprintID:   tokenBlueprintID,
		Products:           decodeStringSlice(data["products"]),
		Status:             mintStatus,
		CreatedBy:          createdBy,
		RequestedBy:        ptrOrEmpty(requestedBy),
		CreatedAt:          createdAt,
		MintedAt:           mintedAt,
		ScheduledBurnDate:  scheduledBurnDate,
		OnChainTxSignature: ptrOrEmpty(onChainTxSignature),
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

	if t.AssetID != "" {
		data["assetId"] = t.AssetID
		data["treeAddress"] = t.TreeAddress
		data["leafIndex"] = int64(t.LeafIndex)
	}
	if t.Signature != "" {
		data["signature"] = t.Signature
	}
	if t.ErrorMessage != "" {
		data["errorMessage"] = t.ErrorMessage
	}

	setOptionalTime(data, "mintingStartedAt", t.MintingStartedAt)
	setOptionalTime(data, "mintedAt", t.MintedAt)
	setOptionalTime(data, "lastFailedAt", t.LastFailedAt)

	return data
}

func decodeMintProductTaskFromDoc(mintID string, doc *firestore.DocumentSnapshot) (mintdom.MintProductTask, error) {
	if doc == nil || !doc.Exists() {
		return mintdom.MintProductTask{}, mintdom.ErrMintProductTaskNotFound
	}

	data := doc.Data()
	if data == nil {
		return mintdom.MintProductTask{}, errors.New("mint product task document data is nil")
	}

	productIDValue, err := firestoreOptionalString(data, "productId")
	if err != nil {
		return mintdom.MintProductTask{}, err
	}
	productID := ptrOrEmpty(productIDValue)
	if productID == "" {
		productID = doc.Ref.ID
	}

	taskMintIDValue, err := firestoreOptionalString(data, "mintId")
	if err != nil {
		return mintdom.MintProductTask{}, err
	}
	taskMintID := ptrOrEmpty(taskMintIDValue)
	if taskMintID == "" {
		taskMintID = mintID
	}

	taskStatus, err := taskStatusFromRaw(data)
	if err != nil {
		return mintdom.MintProductTask{}, err
	}
	rawAttemptCount, err := firestoreRequiredInt64(data, "attemptCount")
	if err != nil {
		return mintdom.MintProductTask{}, err
	}
	attemptCount := int(rawAttemptCount)
	if int64(attemptCount) != rawAttemptCount {
		return mintdom.MintProductTask{}, fmt.Errorf("firestore: attemptCount is out of int range")
	}
	assetID, err := firestoreOptionalString(data, "assetId")
	if err != nil {
		return mintdom.MintProductTask{}, err
	}
	treeAddress, err := firestoreOptionalString(data, "treeAddress")
	if err != nil {
		return mintdom.MintProductTask{}, err
	}
	rawLeafIndex, err := firestoreOptionalInt64(data, "leafIndex")
	if err != nil {
		return mintdom.MintProductTask{}, err
	}
	leafIndex := uint64(0)
	if rawLeafIndex != nil {
		if *rawLeafIndex < 0 {
			return mintdom.MintProductTask{}, fmt.Errorf("firestore: leafIndex must not be negative")
		}
		leafIndex = uint64(*rawLeafIndex)
	}
	signature, err := firestoreOptionalString(data, "signature")
	if err != nil {
		return mintdom.MintProductTask{}, err
	}
	errorMessage, err := firestoreOptionalString(data, "errorMessage")
	if err != nil {
		return mintdom.MintProductTask{}, err
	}
	createdAt, err := firestoreRequiredTime(data, "createdAt")
	if err != nil {
		return mintdom.MintProductTask{}, err
	}
	updatedAt, err := firestoreRequiredTime(data, "updatedAt")
	if err != nil {
		return mintdom.MintProductTask{}, err
	}
	mintingStartedAt, err := firestoreOptionalTime(data, "mintingStartedAt")
	if err != nil {
		return mintdom.MintProductTask{}, err
	}
	mintedAt, err := firestoreOptionalTime(data, "mintedAt")
	if err != nil {
		return mintdom.MintProductTask{}, err
	}
	lastFailedAt, err := firestoreOptionalTime(data, "lastFailedAt")
	if err != nil {
		return mintdom.MintProductTask{}, err
	}

	t := mintdom.MintProductTask{
		MintID:           taskMintID,
		ProductID:        productID,
		Status:           taskStatus,
		AttemptCount:     attemptCount,
		AssetID:          ptrOrEmpty(assetID),
		TreeAddress:      ptrOrEmpty(treeAddress),
		LeafIndex:        leafIndex,
		Signature:        ptrOrEmpty(signature),
		ErrorMessage:     ptrOrEmpty(errorMessage),
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		MintingStartedAt: mintingStartedAt,
		MintedAt:         mintedAt,
		LastFailedAt:     lastFailedAt,
	}

	if err := t.Validate(); err != nil {
		return mintdom.MintProductTask{}, err
	}

	return t, nil
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

	if m.RequestedBy != "" {
		data["requestedBy"] = m.RequestedBy
	}

	if exists && existingSnap != nil && existingSnap.Exists() {
		edata := existingSnap.Data()
		existingStatus, err := mintStatusFromRaw(edata)
		if err != nil {
			return mintdom.Mint{}, err
		}

		data["status"] = string(existingStatus)
		m.Status = existingStatus

		createdBy, err := firestoreRequiredString(edata, "createdBy")
		if err != nil {
			return mintdom.Mint{}, err
		}
		data["createdBy"] = createdBy
		m.CreatedBy = createdBy

		requestedBy, err := firestoreOptionalString(edata, "requestedBy")
		if err != nil {
			return mintdom.Mint{}, err
		}
		if requestedBy != nil {
			data["requestedBy"] = *requestedBy
			m.RequestedBy = *requestedBy
		}

		createdAt, err := firestoreRequiredTime(edata, "createdAt")
		if err != nil {
			return mintdom.Mint{}, err
		}
		data["createdAt"] = createdAt
		m.CreatedAt = createdAt

		mintedAt, err := firestoreOptionalTime(edata, "mintedAt")
		if err != nil {
			return mintdom.Mint{}, err
		}
		m.MintedAt = mintedAt
		if mintedAt != nil {
			data["mintedAt"] = mintedAt.UTC()
		}

		scheduledBurnDate, err := firestoreOptionalTime(edata, "scheduledBurnDate")
		if err != nil {
			return mintdom.Mint{}, err
		}
		m.ScheduledBurnDate = scheduledBurnDate
		if scheduledBurnDate != nil {
			data["scheduledBurnDate"] = scheduledBurnDate.UTC()
		}

		onChainTxSignature, err := firestoreOptionalString(edata, "onChainTxSignature")
		if err != nil {
			return mintdom.Mint{}, err
		}
		if onChainTxSignature != nil {
			data["onChainTxSignature"] = *onChainTxSignature
			m.OnChainTxSignature = *onChainTxSignature
		}
	} else {
		data["createdAt"] = m.CreatedAt.UTC()
		setOptionalTime(data, "mintedAt", m.MintedAt)
		setOptionalTime(data, "scheduledBurnDate", m.ScheduledBurnDate)

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

	if m.Status == "" {
		m.Status = mintdom.MintStatusCreated
	}

	if m.CreatedAt.IsZero() ||
		m.CreatedBy == "" ||
		(m.Status != mintdom.MintStatusCreated && m.RequestedBy == "") ||
		m.BrandID == "" ||
		m.TokenBlueprintID == "" {
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
		if m.RequestedBy == "" {
			m.RequestedBy = existing.RequestedBy
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

	if err := m.Validate(); err != nil {
		return mintdom.Mint{}, err
	}

	data := map[string]any{
		"brandId":          m.BrandID,
		"tokenBlueprintId": m.TokenBlueprintID,
		"products":         m.Products,
		"status":           string(m.Status),
		"createdBy":        m.CreatedBy,
	}

	if m.RequestedBy != "" {
		data["requestedBy"] = m.RequestedBy
	} else {
		data["requestedBy"] = firestore.Delete
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

func (r *MintRepositoryFS) CreateTasks(ctx context.Context, mintID string, productIDs []string) ([]mintdom.MintProductTask, error) {
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

func (r *MintRepositoryFS) GetByProductID(ctx context.Context, mintID string, productID string) (mintdom.MintProductTask, error) {
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

func (r *MintRepositoryFS) ListByMintID(ctx context.Context, mintID string) ([]mintdom.MintProductTask, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if mintID == "" {
		return nil, errors.New("mint id is empty")
	}

	iter := r.taskCol(mintID).Documents(ctx)
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

func (r *MintRepositoryFS) GetNextExecutableTask(ctx context.Context, mintID string) (mintdom.MintProductTask, error) {
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

func (r *MintRepositoryFS) MarkMinting(ctx context.Context, mintID string, productID string) (mintdom.MintProductTask, error) {
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

func (r *MintRepositoryFS) MarkMinted(ctx context.Context, mintID string, productID string, assetID string, treeAddress string, leafIndex uint64, signature string) (mintdom.MintProductTask, error) {
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
		if err := task.MarkMinted(time.Now().UTC(), assetID, treeAddress, leafIndex, signature); err != nil {
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

func (r *MintRepositoryFS) MarkFailedRetryable(ctx context.Context, mintID string, productID string, message string) (mintdom.MintProductTask, error) {
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

func (r *MintRepositoryFS) MarkFailedFatal(ctx context.Context, mintID string, productID string, message string) (mintdom.MintProductTask, error) {
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

func (r *MintRepositoryFS) ResetRetryableToPending(ctx context.Context, mintID string, productID string) (mintdom.MintProductTask, error) {
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
func (r *MintRepositoryFS) LoadForMinting(ctx context.Context, id string) (*usecase.MintRequestForUsecase, error) {
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
	mintStatus, err := mintStatusFromRaw(raw)
	if err != nil {
		return nil, fmt.Errorf("decode mint %s status: %w", mintID, err)
	}
	if mintStatus == mintdom.MintStatusMinted {
		return nil, fmt.Errorf("mint %s is already minted", mintID)
	}

	brandID, err := firestoreRequiredString(raw, "brandId")
	if err != nil {
		return nil, fmt.Errorf("decode mint %s brandId: %w", mintID, err)
	}
	tbID, err := firestoreRequiredString(raw, "tokenBlueprintId")
	if err != nil {
		return nil, fmt.Errorf("decode mint %s tokenBlueprintId: %w", mintID, err)
	}
	actorID, err := firestoreRequiredString(raw, "requestedBy")
	if err != nil {
		return nil, fmt.Errorf("decode mint %s requestedBy: %w", mintID, err)
	}

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

// RecordProductAsMinted は productId 1件分の mint 結果を Firestore に反映します。
// - tokens コレクションに [productId, assetId] を 1:1 で保存（docID=productId）
// - 親 mints/{mintID} はここでは status=MINTED にしません。
// - 親の完了更新は、全 MintProductTask が MINTED になった後に MintUsecase.Update 経由で行います。
func (r *MintRepositoryFS) RecordProductAsMinted(ctx context.Context, id string, mt usecase.MintedTokenForUsecase) error {
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
	if mt.Result.AssetStandard == "" {
		return fmt.Errorf("mint result assetStandard is empty for product %s", productID)
	}
	if mt.Result.Cluster == "" {
		return fmt.Errorf("mint result cluster is empty for product %s", productID)
	}
	if mt.Result.AssetID == "" {
		return fmt.Errorf("mint result assetId is empty for product %s", productID)
	}
	if mt.Result.TreeAddress == "" {
		return fmt.Errorf("mint result treeAddress is empty for product %s", productID)
	}
	if mt.Result.CoreCollectionAddress == "" {
		return fmt.Errorf("mint result coreCollectionAddress is empty for product %s", productID)
	}
	if mt.Result.Signature == "" {
		return fmt.Errorf("mint result signature is empty for product %s", productID)
	}

	mintSnap, err := r.col().Doc(mintID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("mint %s not found when RecordProductAsMinted", mintID)
		}
		return fmt.Errorf("get mint %s in RecordProductAsMinted: %w", mintID, err)
	}

	raw := mintSnap.Data()

	brandID, err := firestoreRequiredString(raw, "brandId")
	if err != nil {
		return fmt.Errorf("decode mint %s brandId in RecordProductAsMinted: %w", mintID, err)
	}
	tbID, err := firestoreRequiredString(raw, "tokenBlueprintId")
	if err != nil {
		return fmt.Errorf("decode mint %s tokenBlueprintId in RecordProductAsMinted: %w", mintID, err)
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

	data := map[string]any{
		"brandId":               brandID,
		"tokenBlueprintId":      tbID,
		"assetStandard":         string(mt.Result.AssetStandard),
		"cluster":               mt.Result.Cluster,
		"assetId":               mt.Result.AssetID,
		"treeAddress":           mt.Result.TreeAddress,
		"leafIndex":             int64(mt.Result.LeafIndex),
		"coreCollectionAddress": mt.Result.CoreCollectionAddress,
		"onChainTxSignature":    mt.Result.Signature,
		"mintedAt":              firestore.ServerTimestamp,
		"toAddress":             toAddress,
		"metadataUri":           metadataURI,
	}

	if _, err := r.tokensCol().Doc(productID).Set(ctx, data, firestore.MergeAll); err != nil {
		return fmt.Errorf("set token productID=%s mintID=%s: %w", productID, mintID, err)
	}

	return nil
}
