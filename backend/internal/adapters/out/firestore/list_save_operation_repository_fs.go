// backend/internal/adapters/out/firestore/list_save_operation_repository_fs.go
package firestore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	listdom "narratives/internal/domain/list"
	"sort"
	"strings"
	"time"

	gfs "cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	listSaveOperationsCollection               = "listSaveOperations"
	listSaveOperationIdempotencyKeysCollection = "listSaveOperationIdempotencyKeys"
	defaultSaveOperationRetryLimit             = 50
	maxSaveOperationRetryLimit                 = 500
)

var errInvalidSaveOperationDocument = errors.New(
	"firestore: invalid list save operation document",
)

// ListSaveOperationRepositoryFS persists List save Saga operations.
type ListSaveOperationRepositoryFS struct {
	Client *gfs.Client
}

func NewListSaveOperationRepositoryFS(
	client *gfs.Client,
) *ListSaveOperationRepositoryFS {
	return &ListSaveOperationRepositoryFS{
		Client: client,
	}
}

var _ listdom.SaveOperationRepository = (*ListSaveOperationRepositoryFS)(nil)

// ============================================================
// Firestore document models
// ============================================================
type listSaveOperationImageDoc struct {
	ImageID      string `firestore:"image_id"`
	URL          string `firestore:"url"`
	StoragePath  string `firestore:"storage_path"`
	DisplayOrder int    `firestore:"display_order"`
}
type listSaveOperationPayloadDoc struct {
	TargetList             listdom.List                `firestore:"target_list"`
	PreviousList           *listdom.List               `firestore:"previous_list,omitempty"`
	NewImages              []listSaveOperationImageDoc `firestore:"new_images"`
	DeleteImageIDs         []string                    `firestore:"delete_image_ids"`
	PreviousImages         []listdom.ListImage         `firestore:"previous_images"`
	PrimaryImageID         string                      `firestore:"primary_image_id"`
	PreviousPrimaryImageID string                      `firestore:"previous_primary_image_id"`
}
type listSaveOperationProgressDoc struct {
	UploadedImageIDs        []string `firestore:"uploaded_image_ids"`
	RegisteredImageIDs      []string `firestore:"registered_image_ids"`
	DeletedImageIDs         []string `firestore:"deleted_image_ids"`
	CompensatedStoragePaths []string `firestore:"compensated_storage_paths"`
	ListUpdated             bool     `firestore:"list_updated"`
	PrimaryImageUpdated     bool     `firestore:"primary_image_updated"`
}
type listSaveOperationDoc struct {
	ID             string                       `firestore:"id"`
	IdempotencyKey string                       `firestore:"idempotency_key"`
	ListID         string                       `firestore:"list_id"`
	Type           string                       `firestore:"type"`
	Status         string                       `firestore:"status"`
	ResumeStatus   string                       `firestore:"resume_status"`
	Payload        listSaveOperationPayloadDoc  `firestore:"payload"`
	Progress       listSaveOperationProgressDoc `firestore:"progress"`
	RetryCount     int                          `firestore:"retry_count"`
	MaxRetries     int                          `firestore:"max_retries"`
	LastError      string                       `firestore:"last_error"`
	Version        int64                        `firestore:"version"`
	CreatedAt      time.Time                    `firestore:"created_at"`
	UpdatedAt      time.Time                    `firestore:"updated_at"`
	FailedAt       *time.Time                   `firestore:"failed_at,omitempty"`
	CompletedAt    *time.Time                   `firestore:"completed_at,omitempty"`
	CompensatedAt  *time.Time                   `firestore:"compensated_at,omitempty"`
}
type listSaveOperationIdempotencyDoc struct {
	OperationID        string    `firestore:"operation_id"`
	IdempotencyKey     string    `firestore:"idempotency_key"`
	RequestFingerprint string    `firestore:"request_fingerprint"`
	CreatedAt          time.Time `firestore:"created_at"`
}

// ============================================================
// Collection references
// ============================================================
func (r *ListSaveOperationRepositoryFS) operationsCol() *gfs.CollectionRef {
	return r.Client.Collection(
		listSaveOperationsCollection,
	)
}
func (r *ListSaveOperationRepositoryFS) idempotencyCol() *gfs.CollectionRef {
	return r.Client.Collection(
		listSaveOperationIdempotencyKeysCollection,
	)
}
func (r *ListSaveOperationRepositoryFS) operationRef(
	operationID string,
) *gfs.DocumentRef {
	return r.operationsCol().Doc(operationID)
}
func (r *ListSaveOperationRepositoryFS) idempotencyRef(
	idempotencyKey string,
) *gfs.DocumentRef {
	return r.idempotencyCol().Doc(
		hashIdempotencyKey(idempotencyKey),
	)
}

// ============================================================
// Create
// ============================================================
// Create persists a new SaveOperation.
//
// Idempotency behavior:
//   - The same key and equivalent request returns the existing operation.
//   - The same key with a different request returns
//     ErrSaveOperationIdempotencyConflict.
//   - The same operation ID with different content returns
//     ErrSaveOperationConflict.
func (r *ListSaveOperationRepositoryFS) Create(
	ctx context.Context,
	operation listdom.SaveOperation,
) (listdom.SaveOperation, error) {
	if err := r.validateClient(); err != nil {
		return listdom.SaveOperation{}, err
	}
	normalizeSaveOperation(&operation)
	if err := validateOperationDocumentID(
		operation.ID,
	); err != nil {
		return listdom.SaveOperation{}, err
	}
	if operation.Status != listdom.SaveOperationStatusPending {
		return listdom.SaveOperation{}, fmt.Errorf(
			"%w: Create requires pending status",
			listdom.ErrInvalidSaveOperation,
		)
	}
	if operation.Version != 1 {
		return listdom.SaveOperation{}, fmt.Errorf(
			"%w: new operation version must be 1",
			listdom.ErrInvalidSaveOperation,
		)
	}
	if err := operation.Validate(); err != nil {
		return listdom.SaveOperation{}, err
	}
	requestFingerprint, err :=
		saveOperationRequestFingerprint(operation)
	if err != nil {
		return listdom.SaveOperation{}, err
	}
	operationRef := r.operationRef(operation.ID)
	idempotencyRef := r.idempotencyRef(
		operation.IdempotencyKey,
	)
	var result listdom.SaveOperation
	err = r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *gfs.Transaction,
		) error {
			existingOperationDoc, err :=
				tx.Get(operationRef)
			if err == nil {
				existingOperation, decodeErr :=
					decodeListSaveOperationDoc(
						existingOperationDoc,
					)
				if decodeErr != nil {
					return decodeErr
				}
				existingFingerprint, fingerprintErr :=
					saveOperationRequestFingerprint(
						existingOperation,
					)
				if fingerprintErr != nil {
					return fingerprintErr
				}
				if existingOperation.IdempotencyKey ==
					operation.IdempotencyKey &&
					existingFingerprint ==
						requestFingerprint {
					result = existingOperation
					return nil
				}
				return listdom.ErrSaveOperationConflict
			}
			if status.Code(err) != codes.NotFound {
				return err
			}
			idempotencyDocSnapshot, err :=
				tx.Get(idempotencyRef)
			if err == nil {
				indexDoc, decodeErr :=
					decodeListSaveOperationIdempotencyDoc(
						idempotencyDocSnapshot,
					)
				if decodeErr != nil {
					return decodeErr
				}
				if indexDoc.IdempotencyKey !=
					operation.IdempotencyKey {
					return listdom.
						ErrSaveOperationIdempotencyConflict
				}
				existingRef := r.operationRef(
					indexDoc.OperationID,
				)
				existingDoc, getErr :=
					tx.Get(existingRef)
				if getErr != nil {
					if status.Code(getErr) ==
						codes.NotFound {
						return listdom.
							ErrSaveOperationConflict
					}
					return getErr
				}
				existingOperation, decodeErr :=
					decodeListSaveOperationDoc(
						existingDoc,
					)
				if decodeErr != nil {
					return decodeErr
				}
				if existingOperation.IdempotencyKey !=
					operation.IdempotencyKey {
					return listdom.
						ErrSaveOperationIdempotencyConflict
				}
				existingFingerprint, fingerprintErr :=
					saveOperationRequestFingerprint(
						existingOperation,
					)
				if fingerprintErr != nil {
					return fingerprintErr
				}
				if existingFingerprint != requestFingerprint {
					return listdom.
						ErrSaveOperationIdempotencyConflict
				}
				result = existingOperation
				return nil
			}
			if status.Code(err) != codes.NotFound {
				return err
			}
			if err := tx.Create(
				operationRef,
				encodeListSaveOperationDoc(operation),
			); err != nil {
				if status.Code(err) ==
					codes.AlreadyExists {
					return listdom.
						ErrSaveOperationConflict
				}
				return err
			}
			idempotencyDoc :=
				listSaveOperationIdempotencyDoc{
					OperationID:        operation.ID,
					IdempotencyKey:     operation.IdempotencyKey,
					RequestFingerprint: requestFingerprint,
					CreatedAt: normalizeFirestoreTime(
						operation.CreatedAt,
					),
				}
			if err := tx.Create(
				idempotencyRef,
				idempotencyDoc,
			); err != nil {
				if status.Code(err) ==
					codes.AlreadyExists {
					return listdom.
						ErrSaveOperationIdempotencyConflict
				}
				return err
			}
			result = operation
			return nil
		},
	)
	if err != nil {
		return listdom.SaveOperation{},
			mapSaveOperationRepositoryError(err)
	}
	return result, nil
}

// ============================================================
// Read
// ============================================================
func (r *ListSaveOperationRepositoryFS) GetByID(
	ctx context.Context,
	operationID string,
) (listdom.SaveOperation, error) {
	if err := r.validateClient(); err != nil {
		return listdom.SaveOperation{}, err
	}
	if err := validateOperationDocumentID(
		operationID,
	); err != nil {
		return listdom.SaveOperation{}, err
	}
	doc, err := r.operationRef(operationID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return listdom.SaveOperation{},
				listdom.ErrSaveOperationNotFound
		}
		return listdom.SaveOperation{}, err
	}
	operation, err := decodeListSaveOperationDoc(doc)
	if err != nil {
		return listdom.SaveOperation{}, err
	}
	return operation, nil
}
func (r *ListSaveOperationRepositoryFS) GetByIdempotencyKey(
	ctx context.Context,
	idempotencyKey string,
) (listdom.SaveOperation, error) {
	if err := r.validateClient(); err != nil {
		return listdom.SaveOperation{}, err
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return listdom.SaveOperation{}, fmt.Errorf(
			"%w: idempotencyKey is required",
			listdom.ErrInvalidSaveOperation,
		)
	}
	indexDocSnapshot, err :=
		r.idempotencyRef(idempotencyKey).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return listdom.SaveOperation{},
				listdom.ErrSaveOperationNotFound
		}
		return listdom.SaveOperation{}, err
	}
	indexDoc, err :=
		decodeListSaveOperationIdempotencyDoc(
			indexDocSnapshot,
		)
	if err != nil {
		return listdom.SaveOperation{}, err
	}
	// SHA-256 collisions are practically infeasible, but the original key is
	// still checked so the index contract does not rely on that assumption.
	if indexDoc.IdempotencyKey != idempotencyKey {
		return listdom.SaveOperation{},
			listdom.ErrSaveOperationIdempotencyConflict
	}
	return r.GetByID(
		ctx,
		indexDoc.OperationID,
	)
}

// ============================================================
// Update
// ============================================================
// Update persists a state transition using optimistic concurrency control.
//
// expectedVersion must equal the currently persisted Version.
// A successful update increments Version by one.
func (r *ListSaveOperationRepositoryFS) Update(
	ctx context.Context,
	operation listdom.SaveOperation,
	expectedVersion int64,
) (listdom.SaveOperation, error) {
	if err := r.validateClient(); err != nil {
		return listdom.SaveOperation{}, err
	}
	normalizeSaveOperation(&operation)
	if err := validateOperationDocumentID(
		operation.ID,
	); err != nil {
		return listdom.SaveOperation{}, err
	}
	if expectedVersion <= 0 {
		return listdom.SaveOperation{}, fmt.Errorf(
			"%w: expectedVersion must be greater than zero",
			listdom.ErrInvalidSaveOperation,
		)
	}
	if err := operation.Validate(); err != nil {
		return listdom.SaveOperation{}, err
	}
	ref := r.operationRef(operation.ID)
	var updated listdom.SaveOperation
	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *gfs.Transaction,
		) error {
			doc, err := tx.Get(ref)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return listdom.ErrSaveOperationNotFound
				}
				return err
			}
			current, err := decodeListSaveOperationDoc(doc)
			if err != nil {
				return err
			}
			if current.Version != expectedVersion {
				return fmt.Errorf(
					"%w: expected version %d, current version %d",
					listdom.ErrSaveOperationConflict,
					expectedVersion,
					current.Version,
				)
			}
			if operation.ID != current.ID {
				return fmt.Errorf(
					"%w: operation id changed",
					listdom.ErrSaveOperationConflict,
				)
			}
			if normalizeFirestoreTime(
				operation.UpdatedAt,
			).Before(
				normalizeFirestoreTime(
					current.UpdatedAt,
				),
			) {
				return fmt.Errorf(
					"%w: updatedAt moved backwards",
					listdom.ErrSaveOperationConflict,
				)
			}
			// Values fixed at creation are always restored from Firestore.
			operation.ID = current.ID
			operation.IdempotencyKey = current.IdempotencyKey
			operation.ListID = current.ListID
			operation.Type = current.Type
			operation.Payload = current.Payload
			operation.MaxRetries = current.MaxRetries
			operation.CreatedAt = current.CreatedAt
			operation.Version = current.Version + 1
			normalizeSaveOperation(&operation)
			if err := operation.Validate(); err != nil {
				return err
			}
			if err := tx.Set(
				ref,
				encodeListSaveOperationDoc(operation),
			); err != nil {
				return err
			}
			updated = operation
			return nil
		},
	)
	if err != nil {
		return listdom.SaveOperation{},
			mapSaveOperationRepositoryError(err)
	}
	return updated, nil
}

// ============================================================
// Retry query
// ============================================================
func (r *ListSaveOperationRepositoryFS) ListRetryable(
	ctx context.Context,
	filter listdom.SaveOperationRetryFilter,
) ([]listdom.SaveOperation, error) {
	if err := r.validateClient(); err != nil {
		return nil, err
	}
	statuses, err :=
		normalizeRetryStatuses(filter.Statuses)
	if err != nil {
		return nil, err
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = defaultSaveOperationRetryLimit
	}
	if limit > maxSaveOperationRetryLimit {
		limit = maxSaveOperationRetryLimit
	}
	updatedBefore := filter.UpdatedBefore
	if updatedBefore != nil {
		value := normalizeFirestoreTime(*updatedBefore)
		updatedBefore = &value
	}
	operationsByID :=
		make(map[string]listdom.SaveOperation)
	for _, operationStatus := range statuses {
		query := r.operationsCol().
			Where(
				"status",
				"==",
				string(operationStatus),
			)
		if updatedBefore != nil {
			query = query.Where(
				"updated_at",
				"<=",
				*updatedBefore,
			)
		}
		query = query.
			OrderBy("updated_at", gfs.Asc).
			OrderBy(gfs.DocumentID, gfs.Asc).
			Limit(limit)
		it := query.Documents(ctx)
		for {
			doc, nextErr := it.Next()
			if errors.Is(nextErr, iterator.Done) {
				break
			}
			if nextErr != nil {
				it.Stop()
				return nil, nextErr
			}
			operation, decodeErr :=
				decodeListSaveOperationDoc(doc)
			if decodeErr != nil {
				it.Stop()
				return nil, decodeErr
			}
			if !operation.CanRetry() {
				continue
			}
			operationsByID[operation.ID] =
				operation
		}
		it.Stop()
	}
	out := make(
		[]listdom.SaveOperation,
		0,
		len(operationsByID),
	)
	for _, operation := range operationsByID {
		out = append(out, operation)
	}
	sort.SliceStable(
		out,
		func(i int, j int) bool {
			left := out[i]
			right := out[j]
			if left.UpdatedAt.Equal(
				right.UpdatedAt,
			) {
				return left.ID < right.ID
			}
			return left.UpdatedAt.Before(
				right.UpdatedAt,
			)
		},
	)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// ============================================================
// Encode
// ============================================================
func encodeListSaveOperationDoc(
	operation listdom.SaveOperation,
) listSaveOperationDoc {
	return listSaveOperationDoc{
		ID:             operation.ID,
		IdempotencyKey: operation.IdempotencyKey,
		ListID:         operation.ListID,
		Type:           string(operation.Type),
		Status:         string(operation.Status),
		ResumeStatus:   string(operation.ResumeStatus),
		Payload: encodeListSaveOperationPayload(
			operation.Payload,
		),
		Progress: encodeListSaveOperationProgress(
			operation.Progress,
		),
		RetryCount: operation.RetryCount,
		MaxRetries: operation.MaxRetries,
		LastError:  operation.LastError,
		Version:    operation.Version,
		CreatedAt: normalizeFirestoreTime(
			operation.CreatedAt,
		),
		UpdatedAt: normalizeFirestoreTime(
			operation.UpdatedAt,
		),
		FailedAt:    utcTimePointer(operation.FailedAt),
		CompletedAt: utcTimePointer(operation.CompletedAt),
		CompensatedAt: utcTimePointer(
			operation.CompensatedAt,
		),
	}
}
func encodeListSaveOperationPayload(
	payload listdom.SaveOperationPayload,
) listSaveOperationPayloadDoc {
	newImages := make(
		[]listSaveOperationImageDoc,
		0,
		len(payload.NewImages),
	)
	for _, image := range payload.NewImages {
		newImages = append(
			newImages,
			listSaveOperationImageDoc{
				ImageID:      image.ImageID,
				URL:          image.URL,
				StoragePath:  image.StoragePath,
				DisplayOrder: image.DisplayOrder,
			},
		)
	}
	return listSaveOperationPayloadDoc{
		TargetList: payload.TargetList,
		PreviousList: cloneListPointer(
			payload.PreviousList,
		),
		NewImages: newImages,
		DeleteImageIDs: append(
			[]string(nil),
			payload.DeleteImageIDs...,
		),
		PreviousImages: append(
			[]listdom.ListImage(nil),
			payload.PreviousImages...,
		),
		PrimaryImageID:         payload.PrimaryImageID,
		PreviousPrimaryImageID: payload.PreviousPrimaryImageID,
	}
}
func encodeListSaveOperationProgress(
	progress listdom.SaveOperationProgress,
) listSaveOperationProgressDoc {
	return listSaveOperationProgressDoc{
		UploadedImageIDs: append(
			[]string(nil),
			progress.UploadedImageIDs...,
		),
		RegisteredImageIDs: append(
			[]string(nil),
			progress.RegisteredImageIDs...,
		),
		DeletedImageIDs: append(
			[]string(nil),
			progress.DeletedImageIDs...,
		),
		CompensatedStoragePaths: append(
			[]string(nil),
			progress.CompensatedStoragePaths...,
		),
		ListUpdated:         progress.ListUpdated,
		PrimaryImageUpdated: progress.PrimaryImageUpdated,
	}
}

// ============================================================
// Decode
// ============================================================
func decodeListSaveOperationDoc(
	doc *gfs.DocumentSnapshot,
) (listdom.SaveOperation, error) {
	if doc == nil || doc.Ref == nil {
		return listdom.SaveOperation{},
			errInvalidSaveOperationDocument
	}
	var raw listSaveOperationDoc
	if err := doc.DataTo(&raw); err != nil {
		return listdom.SaveOperation{}, fmt.Errorf(
			"%w: %v",
			errInvalidSaveOperationDocument,
			err,
		)
	}
	operationID := raw.ID
	if operationID == "" {
		operationID = doc.Ref.ID
	}
	operation := listdom.SaveOperation{
		ID:             operationID,
		IdempotencyKey: raw.IdempotencyKey,
		ListID:         raw.ListID,
		Type: listdom.SaveOperationType(
			raw.Type,
		),
		Status: listdom.SaveOperationStatus(
			raw.Status,
		),
		ResumeStatus: listdom.SaveOperationStatus(
			raw.ResumeStatus,
		),
		Payload: decodeListSaveOperationPayload(
			raw.Payload,
		),
		Progress: decodeListSaveOperationProgress(
			raw.Progress,
		),
		RetryCount: raw.RetryCount,
		MaxRetries: raw.MaxRetries,
		LastError:  raw.LastError,
		Version:    raw.Version,
		CreatedAt: normalizeFirestoreTime(
			raw.CreatedAt,
		),
		UpdatedAt: normalizeFirestoreTime(
			raw.UpdatedAt,
		),
		FailedAt:    utcTimePointer(raw.FailedAt),
		CompletedAt: utcTimePointer(raw.CompletedAt),
		CompensatedAt: utcTimePointer(
			raw.CompensatedAt,
		),
	}
	normalizeSaveOperation(&operation)
	if err := operation.Validate(); err != nil {
		return listdom.SaveOperation{}, fmt.Errorf(
			"%w: %v",
			errInvalidSaveOperationDocument,
			err,
		)
	}
	return operation, nil
}
func decodeListSaveOperationPayload(
	raw listSaveOperationPayloadDoc,
) listdom.SaveOperationPayload {
	newImages := make(
		[]listdom.SaveOperationImage,
		0,
		len(raw.NewImages),
	)
	for _, image := range raw.NewImages {
		newImages = append(
			newImages,
			listdom.SaveOperationImage{
				ImageID:      image.ImageID,
				URL:          image.URL,
				StoragePath:  image.StoragePath,
				DisplayOrder: image.DisplayOrder,
			},
		)
	}
	return listdom.SaveOperationPayload{
		TargetList: raw.TargetList,
		PreviousList: cloneListPointer(
			raw.PreviousList,
		),
		NewImages: newImages,
		DeleteImageIDs: append(
			[]string(nil),
			raw.DeleteImageIDs...,
		),
		PreviousImages: append(
			[]listdom.ListImage(nil),
			raw.PreviousImages...,
		),
		PrimaryImageID:         raw.PrimaryImageID,
		PreviousPrimaryImageID: raw.PreviousPrimaryImageID,
	}
}
func decodeListSaveOperationProgress(
	raw listSaveOperationProgressDoc,
) listdom.SaveOperationProgress {
	return listdom.SaveOperationProgress{
		UploadedImageIDs: append(
			[]string(nil),
			raw.UploadedImageIDs...,
		),
		RegisteredImageIDs: append(
			[]string(nil),
			raw.RegisteredImageIDs...,
		),
		DeletedImageIDs: append(
			[]string(nil),
			raw.DeletedImageIDs...,
		),
		CompensatedStoragePaths: append(
			[]string(nil),
			raw.CompensatedStoragePaths...,
		),
		ListUpdated:         raw.ListUpdated,
		PrimaryImageUpdated: raw.PrimaryImageUpdated,
	}
}
func decodeListSaveOperationIdempotencyDoc(
	doc *gfs.DocumentSnapshot,
) (listSaveOperationIdempotencyDoc, error) {
	if doc == nil || doc.Ref == nil {
		return listSaveOperationIdempotencyDoc{},
			errInvalidSaveOperationDocument
	}
	var raw listSaveOperationIdempotencyDoc
	if err := doc.DataTo(&raw); err != nil {
		return listSaveOperationIdempotencyDoc{},
			fmt.Errorf(
				"%w: %v",
				errInvalidSaveOperationDocument,
				err,
			)
	}
	if raw.OperationID == "" ||
		raw.IdempotencyKey == "" ||
		raw.RequestFingerprint == "" {
		return listSaveOperationIdempotencyDoc{},
			errInvalidSaveOperationDocument
	}
	return raw, nil
}

// ============================================================
// Validation and normalization
// ============================================================
func (r *ListSaveOperationRepositoryFS) validateClient() error {
	if r == nil || r.Client == nil {
		return errors.New(
			"firestore client is nil",
		)
	}
	return nil
}
func validateOperationDocumentID(
	operationID string,
) error {
	if operationID == "" {
		return fmt.Errorf(
			"%w: operationId is required",
			listdom.ErrInvalidSaveOperation,
		)
	}
	if strings.Contains(operationID, "/") ||
		strings.Contains(operationID, "://") ||
		strings.ContainsAny(operationID, "\r\n\x00") {
		return fmt.Errorf(
			"%w: invalid operationId",
			listdom.ErrInvalidSaveOperation,
		)
	}
	return nil
}
func normalizeSaveOperation(
	operation *listdom.SaveOperation,
) {
	if operation == nil {
		return
	}
	operation.CreatedAt =
		normalizeFirestoreTime(operation.CreatedAt)
	operation.UpdatedAt =
		normalizeFirestoreTime(operation.UpdatedAt)
	operation.FailedAt =
		utcTimePointer(operation.FailedAt)
	operation.CompletedAt =
		utcTimePointer(
			operation.CompletedAt,
		)
	operation.CompensatedAt =
		utcTimePointer(
			operation.CompensatedAt,
		)
}
func normalizeRetryStatuses(
	statuses []listdom.SaveOperationStatus,
) ([]listdom.SaveOperationStatus, error) {
	if len(statuses) == 0 {
		return []listdom.SaveOperationStatus{
			listdom.
				SaveOperationStatusFailedRetryable,
		}, nil
	}
	out := make(
		[]listdom.SaveOperationStatus,
		0,
		len(statuses),
	)
	seen := make(
		map[listdom.SaveOperationStatus]struct{},
		len(statuses),
	)
	for _, operationStatus := range statuses {
		if operationStatus !=
			listdom.
				SaveOperationStatusFailedRetryable {
			return nil, fmt.Errorf(
				"%w: status %q is not retryable",
				listdom.ErrInvalidSaveOperation,
				operationStatus,
			)
		}
		if _, exists :=
			seen[operationStatus]; exists {
			continue
		}
		seen[operationStatus] = struct{}{}
		out = append(out, operationStatus)
	}
	return out, nil
}

// ============================================================
// Request fingerprint
// ============================================================
func saveOperationRequestFingerprint(
	operation listdom.SaveOperation,
) (string, error) {
	request := struct {
		ListID     string                       `json:"listId"`
		Type       listdom.SaveOperationType    `json:"type"`
		Payload    listdom.SaveOperationPayload `json:"payload"`
		MaxRetries int                          `json:"maxRetries"`
	}{
		ListID:     operation.ListID,
		Type:       operation.Type,
		Payload:    operation.Payload,
		MaxRetries: operation.MaxRetries,
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf(
			"encode save operation fingerprint: %w",
			err,
		)
	}
	canonical, err := canonicalizeFingerprintJSON(encoded)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}
func canonicalizeFingerprintJSON(
	encoded []byte,
) ([]byte, error) {
	var value any
	if err := json.Unmarshal(encoded, &value); err != nil {
		return nil, fmt.Errorf(
			"decode save operation fingerprint: %w",
			err,
		)
	}
	normalizeFingerprintJSONValue(value)
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf(
			"canonicalize save operation fingerprint: %w",
			err,
		)
	}
	return canonical, nil
}
func normalizeFingerprintJSONValue(value any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if text, ok := item.(string); ok {
				if parsed, err := time.Parse(
					time.RFC3339Nano,
					text,
				); err == nil {
					typed[key] = normalizeFirestoreTime(
						parsed,
					).Format(time.RFC3339Nano)
					continue
				}
			}
			normalizeFingerprintJSONValue(item)
		}
	case []any:
		for _, item := range typed {
			normalizeFingerprintJSONValue(item)
		}
	}
}
func hashIdempotencyKey(
	idempotencyKey string,
) string {
	sum := sha256.Sum256(
		[]byte(idempotencyKey),
	)
	return hex.EncodeToString(sum[:])
}

// ============================================================
// Error mapping
// ============================================================
func mapSaveOperationRepositoryError(
	err error,
) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(
		err,
		listdom.ErrSaveOperationNotFound,
	):
		return listdom.ErrSaveOperationNotFound
	case errors.Is(
		err,
		listdom.ErrSaveOperationConflict,
	):
		return err
	case errors.Is(
		err,
		listdom.
			ErrSaveOperationIdempotencyConflict,
	):
		return listdom.
			ErrSaveOperationIdempotencyConflict
	}
	switch status.Code(err) {
	case codes.NotFound:
		return listdom.ErrSaveOperationNotFound
	case codes.AlreadyExists,
		codes.Aborted,
		codes.FailedPrecondition:
		return fmt.Errorf(
			"%w: firestore transaction failed: %v",
			listdom.ErrSaveOperationConflict,
			err,
		)
	default:
		return err
	}
}

// ============================================================
// Small helpers
// ============================================================
func normalizeFirestoreTime(
	value time.Time,
) time.Time {
	if value.IsZero() {
		return value
	}
	return value.UTC().Truncate(time.Microsecond)
}
func utcTimePointer(
	value *time.Time,
) *time.Time {
	if value == nil {
		return nil
	}
	normalized := normalizeFirestoreTime(*value)
	return &normalized
}
func cloneListPointer(
	value *listdom.List,
) *listdom.List {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
