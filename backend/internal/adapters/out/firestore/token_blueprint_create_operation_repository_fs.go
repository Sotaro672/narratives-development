// backend/internal/adapters/out/firestore/token_blueprint_create_operation_repository_fs.go
package firestore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	gfs "cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	tbdom "narratives/internal/domain/tokenBlueprint"
)

const (
	tokenBlueprintCreateOperationsCollection = "tokenBlueprintCreateOperations"

	tokenBlueprintCreateOperationIdempotencyKeysCollection = "tokenBlueprintCreateOperationIdempotencyKeys"
)

var errInvalidTokenBlueprintCreateOperationDocument = errors.New(
	"firestore: invalid token blueprint create operation document",
)

// TokenBlueprintCreateOperationRepositoryFS persists
// TokenBlueprint create operations.
//
// TokenBlueprint本体とは別CollectionへOperationを保存し、
// frontendによるFirebase Storage uploadと、
// Cloud Tasksによるfinalize処理の境界を永続化します。
type TokenBlueprintCreateOperationRepositoryFS struct {
	Client *gfs.Client
}

func NewTokenBlueprintCreateOperationRepositoryFS(
	client *gfs.Client,
) *TokenBlueprintCreateOperationRepositoryFS {
	return &TokenBlueprintCreateOperationRepositoryFS{
		Client: client,
	}
}

var _ tbdom.CreateOperationRepository = (*TokenBlueprintCreateOperationRepositoryFS)(nil)

// ============================================================
// Firestore document models
// ============================================================

type tokenBlueprintCreateOperationIconDoc struct {
	FileName    string `firestore:"file_name"`
	ContentType string `firestore:"content_type"`
	Size        int64  `firestore:"size"`

	URL        string `firestore:"url"`
	ObjectPath string `firestore:"object_path"`

	Uploaded   bool       `firestore:"uploaded"`
	UploadedAt *time.Time `firestore:"uploaded_at,omitempty"`
}

type tokenBlueprintCreateOperationContentDoc struct {
	ID          string `firestore:"id"`
	Name        string `firestore:"name"`
	Type        string `firestore:"type"`
	ContentType string `firestore:"content_type"`
	Size        int64  `firestore:"size"`

	URL        string `firestore:"url"`
	ObjectPath string `firestore:"object_path"`

	Uploaded   bool       `firestore:"uploaded"`
	UploadedAt *time.Time `firestore:"uploaded_at,omitempty"`
}

type tokenBlueprintCreateOperationDoc struct {
	ID             string `firestore:"id"`
	IdempotencyKey string `firestore:"idempotency_key"`

	TokenBlueprintID string `firestore:"token_blueprint_id"`
	CompanyID        string `firestore:"company_id"`
	ActorID          string `firestore:"actor_id"`

	Status       string `firestore:"status"`
	ResumeStatus string `firestore:"resume_status"`

	Icon *tokenBlueprintCreateOperationIconDoc `firestore:"icon,omitempty"`

	Contents []tokenBlueprintCreateOperationContentDoc `firestore:"contents"`

	RetryCount int    `firestore:"retry_count"`
	MaxRetries int    `firestore:"max_retries"`
	LastError  string `firestore:"last_error"`

	Version int64 `firestore:"version"`

	CreatedAt   time.Time  `firestore:"created_at"`
	UpdatedAt   time.Time  `firestore:"updated_at"`
	FailedAt    *time.Time `firestore:"failed_at,omitempty"`
	CompletedAt *time.Time `firestore:"completed_at,omitempty"`
}

type tokenBlueprintCreateOperationIdempotencyDoc struct {
	OperationID string `firestore:"operation_id"`

	IdempotencyKey string `firestore:"idempotency_key"`

	RequestFingerprint string `firestore:"request_fingerprint"`

	CreatedAt time.Time `firestore:"created_at"`
}

// ============================================================
// Collection references
// ============================================================

func (
	r *TokenBlueprintCreateOperationRepositoryFS,
) operationsCol() *gfs.CollectionRef {
	return r.Client.Collection(
		tokenBlueprintCreateOperationsCollection,
	)
}

func (
	r *TokenBlueprintCreateOperationRepositoryFS,
) idempotencyCol() *gfs.CollectionRef {
	return r.Client.Collection(
		tokenBlueprintCreateOperationIdempotencyKeysCollection,
	)
}

func (
	r *TokenBlueprintCreateOperationRepositoryFS,
) operationRef(
	operationID string,
) *gfs.DocumentRef {
	return r.operationsCol().Doc(
		operationID,
	)
}

func (
	r *TokenBlueprintCreateOperationRepositoryFS,
) idempotencyRef(
	idempotencyKey string,
) *gfs.DocumentRef {
	return r.idempotencyCol().Doc(
		hashTokenBlueprintCreateOperationIdempotencyKey(
			idempotencyKey,
		),
	)
}

// ============================================================
// Create
// ============================================================

// Create persists a new waiting_upload CreateOperation.
//
// Idempotency behavior:
//   - 同じIdempotencyKey + 同一作成要求は既存Operationを返す。
//   - 同じIdempotencyKey + 異なる作成要求は
//     ErrCreateOperationIdempotencyConflict。
//   - 同じOperation ID + 異なる内容は
//     ErrCreateOperationConflict。
//
// request fingerprintにはupload後に変化する
// URL / ObjectPath / Uploaded / UploadedAtを含めません。
func (
	r *TokenBlueprintCreateOperationRepositoryFS,
) Create(
	ctx context.Context,
	operation tbdom.CreateOperation,
) (tbdom.CreateOperation, error) {
	if err := r.validateClient(); err != nil {
		return tbdom.CreateOperation{}, err
	}

	normalizeTokenBlueprintCreateOperation(
		&operation,
	)

	if err := validateTokenBlueprintCreateOperationDocumentID(
		operation.ID,
	); err != nil {
		return tbdom.CreateOperation{}, err
	}

	if operation.Status !=
		tbdom.CreateOperationStatusWaitingUpload {
		return tbdom.CreateOperation{}, fmt.Errorf(
			"%w: Create requires waiting_upload status",
			tbdom.ErrInvalidCreateOperation,
		)
	}

	if operation.Version != 1 {
		return tbdom.CreateOperation{}, fmt.Errorf(
			"%w: new operation version must be 1",
			tbdom.ErrInvalidCreateOperation,
		)
	}

	if err := operation.Validate(); err != nil {
		return tbdom.CreateOperation{}, err
	}

	requestFingerprint, err :=
		tokenBlueprintCreateOperationRequestFingerprint(
			operation,
		)
	if err != nil {
		return tbdom.CreateOperation{}, err
	}

	operationRef := r.operationRef(
		operation.ID,
	)

	idempotencyRef := r.idempotencyRef(
		operation.IdempotencyKey,
	)

	var result tbdom.CreateOperation

	err = r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *gfs.Transaction,
		) error {
			existingOperationDoc, err :=
				tx.Get(
					operationRef,
				)
			if err == nil {
				existingOperation, decodeErr :=
					decodeTokenBlueprintCreateOperationDoc(
						existingOperationDoc,
					)
				if decodeErr != nil {
					return decodeErr
				}

				existingFingerprint, fingerprintErr :=
					tokenBlueprintCreateOperationRequestFingerprint(
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

				return tbdom.ErrCreateOperationConflict
			}

			if status.Code(err) != codes.NotFound {
				return err
			}

			idempotencyDocSnapshot, err :=
				tx.Get(
					idempotencyRef,
				)
			if err == nil {
				indexDoc, decodeErr :=
					decodeTokenBlueprintCreateOperationIdempotencyDoc(
						idempotencyDocSnapshot,
					)
				if decodeErr != nil {
					return decodeErr
				}

				if indexDoc.IdempotencyKey !=
					operation.IdempotencyKey {
					return tbdom.
						ErrCreateOperationIdempotencyConflict
				}

				existingRef := r.operationRef(
					indexDoc.OperationID,
				)

				existingDoc, getErr :=
					tx.Get(
						existingRef,
					)
				if getErr != nil {
					if status.Code(getErr) ==
						codes.NotFound {
						return tbdom.
							ErrCreateOperationConflict
					}

					return getErr
				}

				existingOperation, decodeErr :=
					decodeTokenBlueprintCreateOperationDoc(
						existingDoc,
					)
				if decodeErr != nil {
					return decodeErr
				}

				if existingOperation.IdempotencyKey !=
					operation.IdempotencyKey {
					return tbdom.
						ErrCreateOperationIdempotencyConflict
				}

				existingFingerprint, fingerprintErr :=
					tokenBlueprintCreateOperationRequestFingerprint(
						existingOperation,
					)
				if fingerprintErr != nil {
					return fingerprintErr
				}

				if existingFingerprint !=
					requestFingerprint {
					return tbdom.
						ErrCreateOperationIdempotencyConflict
				}

				result = existingOperation
				return nil
			}

			if status.Code(err) != codes.NotFound {
				return err
			}

			if err := tx.Create(
				operationRef,
				encodeTokenBlueprintCreateOperationDoc(
					operation,
				),
			); err != nil {
				if status.Code(err) ==
					codes.AlreadyExists {
					return tbdom.
						ErrCreateOperationConflict
				}

				return err
			}

			idempotencyDoc :=
				tokenBlueprintCreateOperationIdempotencyDoc{
					OperationID: operation.ID,

					IdempotencyKey: operation.IdempotencyKey,

					RequestFingerprint: requestFingerprint,

					CreatedAt: normalizeTokenBlueprintCreateOperationTime(
						operation.CreatedAt,
					),
				}

			if err := tx.Create(
				idempotencyRef,
				idempotencyDoc,
			); err != nil {
				if status.Code(err) ==
					codes.AlreadyExists {
					return tbdom.
						ErrCreateOperationIdempotencyConflict
				}

				return err
			}

			result = operation
			return nil
		},
	)
	if err != nil {
		return tbdom.CreateOperation{},
			mapTokenBlueprintCreateOperationRepositoryError(
				err,
			)
	}

	return result, nil
}

// ============================================================
// Read
// ============================================================

func (
	r *TokenBlueprintCreateOperationRepositoryFS,
) GetByID(
	ctx context.Context,
	operationID string,
) (tbdom.CreateOperation, error) {
	if err := r.validateClient(); err != nil {
		return tbdom.CreateOperation{}, err
	}

	if err := validateTokenBlueprintCreateOperationDocumentID(
		operationID,
	); err != nil {
		return tbdom.CreateOperation{}, err
	}

	doc, err := r.operationRef(
		operationID,
	).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return tbdom.CreateOperation{},
				tbdom.ErrCreateOperationNotFound
		}

		return tbdom.CreateOperation{}, err
	}

	operation, err :=
		decodeTokenBlueprintCreateOperationDoc(
			doc,
		)
	if err != nil {
		return tbdom.CreateOperation{}, err
	}

	return operation, nil
}

func (
	r *TokenBlueprintCreateOperationRepositoryFS,
) GetByIdempotencyKey(
	ctx context.Context,
	idempotencyKey string,
) (tbdom.CreateOperation, error) {
	if err := r.validateClient(); err != nil {
		return tbdom.CreateOperation{}, err
	}

	idempotencyKey = strings.TrimSpace(
		idempotencyKey,
	)

	if idempotencyKey == "" {
		return tbdom.CreateOperation{}, fmt.Errorf(
			"%w: idempotencyKey is required",
			tbdom.ErrInvalidCreateOperation,
		)
	}

	indexDocSnapshot, err :=
		r.idempotencyRef(
			idempotencyKey,
		).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return tbdom.CreateOperation{},
				tbdom.ErrCreateOperationNotFound
		}

		return tbdom.CreateOperation{}, err
	}

	indexDoc, err :=
		decodeTokenBlueprintCreateOperationIdempotencyDoc(
			indexDocSnapshot,
		)
	if err != nil {
		return tbdom.CreateOperation{}, err
	}

	// SHA-256 collisionは実用上無視できる可能性ですが、
	// index contractをhash衝突の仮定へ依存させないため、
	// original keyも必ず照合します。
	if indexDoc.IdempotencyKey !=
		idempotencyKey {
		return tbdom.CreateOperation{},
			tbdom.ErrCreateOperationIdempotencyConflict
	}

	return r.GetByID(
		ctx,
		indexDoc.OperationID,
	)
}

// ============================================================
// Update
// ============================================================

// Update persists the current CreateOperation state using
// optimistic concurrency control.
//
// expectedVersionは更新前に読み込んだVersionでなければなりません。
// 更新成功時はVersionを1増加させます。
//
// 作成時に確定した以下の情報は変更不可です。
// - ID
// - IdempotencyKey
// - TokenBlueprintID
// - CompanyID
// - ActorID
// - Icon expected metadata
// - Contents expected metadata
// - MaxRetries
// - CreatedAt
//
// Icon / Contentsのupload結果は変更可能です。
func (
	r *TokenBlueprintCreateOperationRepositoryFS,
) Update(
	ctx context.Context,
	operation tbdom.CreateOperation,
	expectedVersion int64,
) (tbdom.CreateOperation, error) {
	if err := r.validateClient(); err != nil {
		return tbdom.CreateOperation{}, err
	}

	normalizeTokenBlueprintCreateOperation(
		&operation,
	)

	if err := validateTokenBlueprintCreateOperationDocumentID(
		operation.ID,
	); err != nil {
		return tbdom.CreateOperation{}, err
	}

	if expectedVersion <= 0 {
		return tbdom.CreateOperation{}, fmt.Errorf(
			"%w: expectedVersion must be greater than zero",
			tbdom.ErrInvalidCreateOperation,
		)
	}

	if err := operation.Validate(); err != nil {
		return tbdom.CreateOperation{}, err
	}

	ref := r.operationRef(
		operation.ID,
	)

	var updated tbdom.CreateOperation

	err := r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *gfs.Transaction,
		) error {
			doc, err := tx.Get(
				ref,
			)
			if err != nil {
				if status.Code(err) ==
					codes.NotFound {
					return tbdom.
						ErrCreateOperationNotFound
				}

				return err
			}

			current, err :=
				decodeTokenBlueprintCreateOperationDoc(
					doc,
				)
			if err != nil {
				return err
			}

			if current.Version != expectedVersion {
				return fmt.Errorf(
					"%w: expected version %d, current version %d",
					tbdom.ErrCreateOperationConflict,
					expectedVersion,
					current.Version,
				)
			}

			if operation.ID != current.ID {
				return fmt.Errorf(
					"%w: operation id changed",
					tbdom.ErrCreateOperationConflict,
				)
			}

			currentFingerprint, err :=
				tokenBlueprintCreateOperationRequestFingerprint(
					current,
				)
			if err != nil {
				return err
			}

			candidateFingerprint, err :=
				tokenBlueprintCreateOperationRequestFingerprint(
					operation,
				)
			if err != nil {
				return err
			}

			if currentFingerprint !=
				candidateFingerprint {
				return fmt.Errorf(
					"%w: immutable create request changed",
					tbdom.ErrCreateOperationConflict,
				)
			}

			if normalizeTokenBlueprintCreateOperationTime(
				operation.UpdatedAt,
			).Before(
				normalizeTokenBlueprintCreateOperationTime(
					current.UpdatedAt,
				),
			) {
				return fmt.Errorf(
					"%w: updatedAt moved backwards",
					tbdom.ErrCreateOperationConflict,
				)
			}

			// 作成時固定値はFirestore上の値を正とします。
			operation.ID = current.ID
			operation.IdempotencyKey =
				current.IdempotencyKey

			operation.TokenBlueprintID =
				current.TokenBlueprintID

			operation.CompanyID =
				current.CompanyID

			operation.ActorID =
				current.ActorID

			operation.MaxRetries =
				current.MaxRetries

			operation.CreatedAt =
				current.CreatedAt

			operation.Version =
				current.Version + 1

			normalizeTokenBlueprintCreateOperation(
				&operation,
			)

			if err := operation.Validate(); err != nil {
				return err
			}

			if err := tx.Set(
				ref,
				encodeTokenBlueprintCreateOperationDoc(
					operation,
				),
			); err != nil {
				return err
			}

			updated = operation

			return nil
		},
	)
	if err != nil {
		return tbdom.CreateOperation{},
			mapTokenBlueprintCreateOperationRepositoryError(
				err,
			)
	}

	return updated, nil
}

// ============================================================
// Encode
// ============================================================

func encodeTokenBlueprintCreateOperationDoc(
	operation tbdom.CreateOperation,
) tokenBlueprintCreateOperationDoc {
	return tokenBlueprintCreateOperationDoc{
		ID:             operation.ID,
		IdempotencyKey: operation.IdempotencyKey,

		TokenBlueprintID: operation.TokenBlueprintID,
		CompanyID:        operation.CompanyID,
		ActorID:          operation.ActorID,

		Status: string(
			operation.Status,
		),

		ResumeStatus: string(
			operation.ResumeStatus,
		),

		Icon: encodeTokenBlueprintCreateOperationIcon(
			operation.Icon,
		),

		Contents: encodeTokenBlueprintCreateOperationContents(
			operation.Contents,
		),

		RetryCount: operation.RetryCount,
		MaxRetries: operation.MaxRetries,
		LastError:  operation.LastError,

		Version: operation.Version,

		CreatedAt: normalizeTokenBlueprintCreateOperationTime(
			operation.CreatedAt,
		),

		UpdatedAt: normalizeTokenBlueprintCreateOperationTime(
			operation.UpdatedAt,
		),

		FailedAt: normalizeTokenBlueprintCreateOperationTimePointer(
			operation.FailedAt,
		),

		CompletedAt: normalizeTokenBlueprintCreateOperationTimePointer(
			operation.CompletedAt,
		),
	}
}

func encodeTokenBlueprintCreateOperationIcon(
	icon *tbdom.CreateOperationIcon,
) *tokenBlueprintCreateOperationIconDoc {
	if icon == nil {
		return nil
	}

	return &tokenBlueprintCreateOperationIconDoc{
		FileName:    icon.FileName,
		ContentType: icon.ContentType,
		Size:        icon.Size,

		URL:        icon.URL,
		ObjectPath: icon.ObjectPath,

		Uploaded: icon.Uploaded,

		UploadedAt: normalizeTokenBlueprintCreateOperationTimePointer(
			icon.UploadedAt,
		),
	}
}

func encodeTokenBlueprintCreateOperationContents(
	contents []tbdom.CreateOperationContent,
) []tokenBlueprintCreateOperationContentDoc {
	if len(contents) == 0 {
		return []tokenBlueprintCreateOperationContentDoc{}
	}

	out := make(
		[]tokenBlueprintCreateOperationContentDoc,
		0,
		len(contents),
	)

	for _, content := range contents {
		out = append(
			out,
			tokenBlueprintCreateOperationContentDoc{
				ID:          content.ID,
				Name:        content.Name,
				Type:        string(content.Type),
				ContentType: content.ContentType,
				Size:        content.Size,

				URL:        content.URL,
				ObjectPath: content.ObjectPath,

				Uploaded: content.Uploaded,

				UploadedAt: normalizeTokenBlueprintCreateOperationTimePointer(
					content.UploadedAt,
				),
			},
		)
	}

	return out
}

// ============================================================
// Decode
// ============================================================

func decodeTokenBlueprintCreateOperationDoc(
	doc *gfs.DocumentSnapshot,
) (tbdom.CreateOperation, error) {
	if doc == nil ||
		doc.Ref == nil {
		return tbdom.CreateOperation{},
			errInvalidTokenBlueprintCreateOperationDocument
	}

	var raw tokenBlueprintCreateOperationDoc

	if err := doc.DataTo(
		&raw,
	); err != nil {
		return tbdom.CreateOperation{}, fmt.Errorf(
			"%w: %v",
			errInvalidTokenBlueprintCreateOperationDocument,
			err,
		)
	}

	operationID := raw.ID
	if operationID == "" {
		operationID = doc.Ref.ID
	}

	operation := tbdom.CreateOperation{
		ID:             operationID,
		IdempotencyKey: raw.IdempotencyKey,

		TokenBlueprintID: raw.TokenBlueprintID,
		CompanyID:        raw.CompanyID,
		ActorID:          raw.ActorID,

		Status: tbdom.CreateOperationStatus(
			raw.Status,
		),

		ResumeStatus: tbdom.CreateOperationStatus(
			raw.ResumeStatus,
		),

		Icon: decodeTokenBlueprintCreateOperationIcon(
			raw.Icon,
		),

		Contents: decodeTokenBlueprintCreateOperationContents(
			raw.Contents,
		),

		RetryCount: raw.RetryCount,
		MaxRetries: raw.MaxRetries,
		LastError:  raw.LastError,

		Version: raw.Version,

		CreatedAt: normalizeTokenBlueprintCreateOperationTime(
			raw.CreatedAt,
		),

		UpdatedAt: normalizeTokenBlueprintCreateOperationTime(
			raw.UpdatedAt,
		),

		FailedAt: normalizeTokenBlueprintCreateOperationTimePointer(
			raw.FailedAt,
		),

		CompletedAt: normalizeTokenBlueprintCreateOperationTimePointer(
			raw.CompletedAt,
		),
	}

	normalizeTokenBlueprintCreateOperation(
		&operation,
	)

	if err := operation.Validate(); err != nil {
		return tbdom.CreateOperation{}, fmt.Errorf(
			"%w: %v",
			errInvalidTokenBlueprintCreateOperationDocument,
			err,
		)
	}

	return operation, nil
}

func decodeTokenBlueprintCreateOperationIcon(
	raw *tokenBlueprintCreateOperationIconDoc,
) *tbdom.CreateOperationIcon {
	if raw == nil {
		return nil
	}

	return &tbdom.CreateOperationIcon{
		FileName:    raw.FileName,
		ContentType: raw.ContentType,
		Size:        raw.Size,

		URL:        raw.URL,
		ObjectPath: raw.ObjectPath,

		Uploaded: raw.Uploaded,

		UploadedAt: normalizeTokenBlueprintCreateOperationTimePointer(
			raw.UploadedAt,
		),
	}
}

func decodeTokenBlueprintCreateOperationContents(
	raw []tokenBlueprintCreateOperationContentDoc,
) []tbdom.CreateOperationContent {
	if len(raw) == 0 {
		return []tbdom.CreateOperationContent{}
	}

	out := make(
		[]tbdom.CreateOperationContent,
		0,
		len(raw),
	)

	for _, content := range raw {
		out = append(
			out,
			tbdom.CreateOperationContent{
				ID:   content.ID,
				Name: content.Name,

				Type: tbdom.ContentFileType(
					content.Type,
				),

				ContentType: content.ContentType,
				Size:        content.Size,

				URL:        content.URL,
				ObjectPath: content.ObjectPath,

				Uploaded: content.Uploaded,

				UploadedAt: normalizeTokenBlueprintCreateOperationTimePointer(
					content.UploadedAt,
				),
			},
		)
	}

	return out
}

func decodeTokenBlueprintCreateOperationIdempotencyDoc(
	doc *gfs.DocumentSnapshot,
) (
	tokenBlueprintCreateOperationIdempotencyDoc,
	error,
) {
	if doc == nil ||
		doc.Ref == nil {
		return tokenBlueprintCreateOperationIdempotencyDoc{},
			errInvalidTokenBlueprintCreateOperationDocument
	}

	var raw tokenBlueprintCreateOperationIdempotencyDoc

	if err := doc.DataTo(
		&raw,
	); err != nil {
		return tokenBlueprintCreateOperationIdempotencyDoc{},
			fmt.Errorf(
				"%w: %v",
				errInvalidTokenBlueprintCreateOperationDocument,
				err,
			)
	}

	if raw.OperationID == "" ||
		raw.IdempotencyKey == "" ||
		raw.RequestFingerprint == "" {
		return tokenBlueprintCreateOperationIdempotencyDoc{},
			errInvalidTokenBlueprintCreateOperationDocument
	}

	raw.CreatedAt =
		normalizeTokenBlueprintCreateOperationTime(
			raw.CreatedAt,
		)

	return raw, nil
}

// ============================================================
// Validation and normalization
// ============================================================

func (
	r *TokenBlueprintCreateOperationRepositoryFS,
) validateClient() error {
	if r == nil ||
		r.Client == nil {
		return errors.New(
			"firestore client is nil",
		)
	}

	return nil
}

func validateTokenBlueprintCreateOperationDocumentID(
	operationID string,
) error {
	operationID = strings.TrimSpace(
		operationID,
	)

	if operationID == "" {
		return fmt.Errorf(
			"%w: operationId is required",
			tbdom.ErrInvalidCreateOperation,
		)
	}

	if len(operationID) > 512 {
		return fmt.Errorf(
			"%w: operationId must not exceed 512 characters",
			tbdom.ErrInvalidCreateOperation,
		)
	}

	if strings.Contains(
		operationID,
		"/",
	) ||
		strings.Contains(
			operationID,
			"://",
		) ||
		strings.ContainsAny(
			operationID,
			"\r\n\x00",
		) {
		return fmt.Errorf(
			"%w: invalid operationId",
			tbdom.ErrInvalidCreateOperation,
		)
	}

	return nil
}

func normalizeTokenBlueprintCreateOperation(
	operation *tbdom.CreateOperation,
) {
	if operation == nil {
		return
	}

	operation.CreatedAt =
		normalizeTokenBlueprintCreateOperationTime(
			operation.CreatedAt,
		)

	operation.UpdatedAt =
		normalizeTokenBlueprintCreateOperationTime(
			operation.UpdatedAt,
		)

	operation.FailedAt =
		normalizeTokenBlueprintCreateOperationTimePointer(
			operation.FailedAt,
		)

	operation.CompletedAt =
		normalizeTokenBlueprintCreateOperationTimePointer(
			operation.CompletedAt,
		)

	if operation.Icon != nil {
		operation.Icon.UploadedAt =
			normalizeTokenBlueprintCreateOperationTimePointer(
				operation.Icon.UploadedAt,
			)
	}

	for i := range operation.Contents {
		operation.Contents[i].UploadedAt =
			normalizeTokenBlueprintCreateOperationTimePointer(
				operation.Contents[i].UploadedAt,
			)
	}
}

func normalizeTokenBlueprintCreateOperationTime(
	value time.Time,
) time.Time {
	if value.IsZero() {
		return value
	}

	return value.
		UTC().
		Truncate(
			time.Microsecond,
		)
}

func normalizeTokenBlueprintCreateOperationTimePointer(
	value *time.Time,
) *time.Time {
	if value == nil {
		return nil
	}

	normalized :=
		normalizeTokenBlueprintCreateOperationTime(
			*value,
		)

	return &normalized
}

// ============================================================
// Request fingerprint
// ============================================================

// tokenBlueprintCreateOperationRequestFingerprint calculates a fingerprint
// only from the immutable create request.
//
// Firebase Storage uploadによって変化する以下は含めません。
// - URL
// - ObjectPath
// - Uploaded
// - UploadedAt
//
// このためwaiting_upload中に一部upload済みとなったOperationに対して
// 同一IdempotencyKeyでStart要求が再送されても、元の作成要求と同一か
// 正しく判定できます。
func tokenBlueprintCreateOperationRequestFingerprint(
	operation tbdom.CreateOperation,
) (string, error) {
	type plannedIcon struct {
		FileName    string `json:"fileName"`
		ContentType string `json:"contentType"`
		Size        int64  `json:"size"`
	}

	type plannedContent struct {
		ID          string                `json:"id"`
		Name        string                `json:"name"`
		Type        tbdom.ContentFileType `json:"type"`
		ContentType string                `json:"contentType"`
		Size        int64                 `json:"size"`
	}

	var icon *plannedIcon

	if operation.Icon != nil {
		icon = &plannedIcon{
			FileName: operation.Icon.FileName,

			ContentType: operation.Icon.ContentType,

			Size: operation.Icon.Size,
		}
	}

	contents := make(
		[]plannedContent,
		0,
		len(operation.Contents),
	)

	for _, content := range operation.Contents {
		contents = append(
			contents,
			plannedContent{
				ID:   content.ID,
				Name: content.Name,

				Type: content.Type,

				ContentType: content.ContentType,

				Size: content.Size,
			},
		)
	}

	request := struct {
		TokenBlueprintID string `json:"tokenBlueprintId"`
		CompanyID        string `json:"companyId"`
		ActorID          string `json:"actorId"`

		Icon *plannedIcon `json:"icon,omitempty"`

		Contents []plannedContent `json:"contents"`

		MaxRetries int `json:"maxRetries"`
	}{
		TokenBlueprintID: operation.TokenBlueprintID,
		CompanyID:        operation.CompanyID,
		ActorID:          operation.ActorID,

		Icon: icon,

		Contents: contents,

		MaxRetries: operation.MaxRetries,
	}

	encoded, err := json.Marshal(
		request,
	)
	if err != nil {
		return "", fmt.Errorf(
			"encode token blueprint create operation fingerprint: %w",
			err,
		)
	}

	sum := sha256.Sum256(
		encoded,
	)

	return hex.EncodeToString(
		sum[:],
	), nil
}

func hashTokenBlueprintCreateOperationIdempotencyKey(
	idempotencyKey string,
) string {
	sum := sha256.Sum256(
		[]byte(
			idempotencyKey,
		),
	)

	return hex.EncodeToString(
		sum[:],
	)
}

// ============================================================
// Error mapping
// ============================================================

func mapTokenBlueprintCreateOperationRepositoryError(
	err error,
) error {
	switch {
	case err == nil:
		return nil

	case errors.Is(
		err,
		tbdom.ErrCreateOperationNotFound,
	):
		return tbdom.ErrCreateOperationNotFound

	case errors.Is(
		err,
		tbdom.ErrCreateOperationConflict,
	):
		return err

	case errors.Is(
		err,
		tbdom.ErrCreateOperationIdempotencyConflict,
	):
		return tbdom.
			ErrCreateOperationIdempotencyConflict
	}

	switch status.Code(err) {
	case codes.NotFound:
		return tbdom.ErrCreateOperationNotFound

	case codes.AlreadyExists,
		codes.Aborted,
		codes.FailedPrecondition:
		return fmt.Errorf(
			"%w: firestore transaction failed: %v",
			tbdom.ErrCreateOperationConflict,
			err,
		)

	default:
		return err
	}
}
