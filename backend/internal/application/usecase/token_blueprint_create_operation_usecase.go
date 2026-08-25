// backend/internal/application/usecase/token_blueprint_create_operation_usecase.go
package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	tbdom "narratives/internal/domain/tokenBlueprint"
)

// TokenBlueprintCreateOperationQueue schedules backend finalization.
//
// Task payload側にはoperationIDのみを渡し、
// Firebase StorageのURLやファイル本体は渡さない。
type TokenBlueprintCreateOperationQueue interface {
	Enqueue(
		ctx context.Context,
		operationID string,
		scheduledAt time.Time,
	) error
}

type TokenBlueprintCreateOperationUsecase struct {
	tokenBlueprintUC *TokenBlueprintUsecase
	operationRepo    tbdom.CreateOperationRepository
	storage          TokenBlueprintAssetStorage
	queue            TokenBlueprintCreateOperationQueue
	now              func() time.Time
	isRetryableError func(error) bool
}

type NewTokenBlueprintCreateOperationUsecaseParams struct {
	TokenBlueprintUsecase *TokenBlueprintUsecase
	OperationRepository   tbdom.CreateOperationRepository
	Storage               TokenBlueprintAssetStorage
	Queue                 TokenBlueprintCreateOperationQueue
	Now                   func() time.Time
	IsRetryableError      func(error) bool
}

// StartTokenBlueprintCreateOperationInput contains the complete request
// required before frontend Firebase Storage upload starts.
//
// Icon / Contents contain only the expected local-file metadata.
// URL / ObjectPath / Uploaded / UploadedAt must still be empty.
//
// TokenBlueprint本体は先に作成し、そのIDをStorage pathと
// CreateOperationのTokenBlueprintIDとして使用する。
type StartTokenBlueprintCreateOperationInput struct {
	OperationID    string
	IdempotencyKey string

	Name        string
	Symbol      string
	BrandID     string
	CompanyID   string
	Description string
	AssigneeID  string
	ActorID     string

	Icon     *tbdom.CreateOperationIcon
	Contents []tbdom.CreateOperationContent

	MaxRetries int
}

func NewTokenBlueprintCreateOperationUsecase(
	p NewTokenBlueprintCreateOperationUsecaseParams,
) *TokenBlueprintCreateOperationUsecase {
	now := p.Now
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}

	isRetryableError := p.IsRetryableError
	if isRetryableError == nil {
		isRetryableError =
			defaultTokenBlueprintCreateOperationRetryableError
	}

	return &TokenBlueprintCreateOperationUsecase{
		tokenBlueprintUC: p.TokenBlueprintUsecase,
		operationRepo:    p.OperationRepository,
		storage:          p.Storage,
		queue:            p.Queue,
		now:              now,
		isRetryableError: isRetryableError,
	}
}

// Start creates the base TokenBlueprint and a durable waiting_upload
// CreateOperation.
//
// The browser still owns local File objects while status=waiting_upload.
// Cloud Tasks does not start until Commit succeeds.
func (uc *TokenBlueprintCreateOperationUsecase) Start(
	ctx context.Context,
	input StartTokenBlueprintCreateOperationInput,
) (tbdom.CreateOperation, error) {
	if err := uc.validateDependencies(); err != nil {
		return tbdom.CreateOperation{}, err
	}

	input.IdempotencyKey =
		strings.TrimSpace(input.IdempotencyKey)

	if input.IdempotencyKey == "" {
		return tbdom.CreateOperation{}, fmt.Errorf(
			"%w: idempotencyKey is required",
			tbdom.ErrInvalidCreateOperation,
		)
	}

	// 通常の再送ではTokenBlueprintを再作成しない。
	existing, err :=
		uc.operationRepo.GetByIdempotencyKey(
			ctx,
			input.IdempotencyKey,
		)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(
		err,
		tbdom.ErrCreateOperationNotFound,
	) {
		return tbdom.CreateOperation{}, err
	}

	operationID :=
		strings.TrimSpace(input.OperationID)
	if operationID == "" {
		operationID, err =
			generateTokenBlueprintCreateOperationID(
				"tbco",
			)
		if err != nil {
			return tbdom.CreateOperation{}, err
		}
	}

	actorID := strings.TrimSpace(input.ActorID)
	if actorID == "" {
		return tbdom.CreateOperation{}, fmt.Errorf(
			"%w: actorId is required",
			tbdom.ErrInvalidCreateOperation,
		)
	}

	companyID := strings.TrimSpace(input.CompanyID)
	if companyID == "" {
		return tbdom.CreateOperation{}, fmt.Errorf(
			"%w: companyId is required",
			tbdom.ErrInvalidCreateOperation,
		)
	}

	icon, err :=
		normalizeTokenBlueprintCreateOperationIcon(
			input.Icon,
		)
	if err != nil {
		return tbdom.CreateOperation{}, err
	}

	contents, err :=
		normalizeTokenBlueprintCreateOperationContents(
			input.Contents,
		)
	if err != nil {
		return tbdom.CreateOperation{}, err
	}

	// Storage assetはまだTokenBlueprint本体へ保存しない。
	//
	// frontend uploadが完了しCloud Tasks側のfinalizeへ到達した時点で、
	// Icon / ContentFilesをTokenBlueprintへ反映する。
	created, err := uc.tokenBlueprintUC.Create(
		ctx,
		CreateBlueprintRequest{
			Name: strings.TrimSpace(
				input.Name,
			),
			Symbol: strings.TrimSpace(
				input.Symbol,
			),
			BrandID: strings.TrimSpace(
				input.BrandID,
			),
			CompanyID:   companyID,
			Description: input.Description,

			IconURL:         "",
			IconObjectPath:  "",
			IconFileName:    "",
			IconContentType: "",
			IconSize:        0,

			ContentFiles: []tbdom.ContentFile{},

			AssigneeID: strings.TrimSpace(
				input.AssigneeID,
			),
			CreatedBy: actorID,
		},
	)
	if err != nil {
		return tbdom.CreateOperation{}, err
	}

	if created == nil ||
		strings.TrimSpace(created.ID) == "" {
		return tbdom.CreateOperation{}, errors.New(
			"token blueprint create operation: base token blueprint returned empty id",
		)
	}

	now := uc.currentTime()

	operation, err :=
		tbdom.NewCreateOperation(
			tbdom.NewCreateOperationInput{
				ID:             operationID,
				IdempotencyKey: input.IdempotencyKey,

				TokenBlueprintID: created.ID,
				CompanyID:        created.CompanyID,
				ActorID:          actorID,

				Icon:     icon,
				Contents: contents,

				MaxRetries: input.MaxRetries,
			},
			now,
		)
	if err != nil {
		// Operationが永続化される前なので、この時点では
		// base TokenBlueprintを安全にcleanupできる。
		cleanupErr :=
			uc.cleanupUncommittedTokenBlueprint(
				ctx,
				created.ID,
			)

		return tbdom.CreateOperation{},
			errors.Join(
				err,
				cleanupErr,
			)
	}

	persisted, err :=
		uc.operationRepo.Create(
			ctx,
			operation,
		)
	if err == nil {
		return persisted, nil
	}

	// 同一IdempotencyKeyの並行Startが先に成功していた場合は、
	// 既存Operationを返す。
	if errors.Is(
		err,
		tbdom.ErrCreateOperationIdempotencyConflict,
	) {
		existing, getErr :=
			uc.operationRepo.GetByIdempotencyKey(
				ctx,
				input.IdempotencyKey,
			)
		if getErr == nil {
			// このStartが作ったbase TokenBlueprintは
			// 既存Operationとは別物なのでbest-effortで削除する。
			if existing.TokenBlueprintID !=
				created.ID {
				_ = uc.cleanupUncommittedTokenBlueprint(
					ctx,
					created.ID,
				)
			}

			return existing, nil
		}

		return tbdom.CreateOperation{},
			errors.Join(
				err,
				getErr,
			)
	}

	// Firestore transactionの結果が不明な通信エラーの場合、
	// 実際にはOperationがcommit済みの可能性がある。
	// そのため、ここではbase TokenBlueprintを削除しない。
	return tbdom.CreateOperation{}, err
}

// Get returns a CreateOperation by operation ID.
func (uc *TokenBlueprintCreateOperationUsecase) Get(
	ctx context.Context,
	operationID string,
) (tbdom.CreateOperation, error) {
	if uc == nil ||
		uc.operationRepo == nil {
		return tbdom.CreateOperation{},
			errors.New(
				"token blueprint create operation repository is nil",
			)
	}

	operationID =
		strings.TrimSpace(operationID)

	if operationID == "" {
		return tbdom.CreateOperation{}, fmt.Errorf(
			"%w: operationId is required",
			tbdom.ErrInvalidCreateOperation,
		)
	}

	return uc.operationRepo.GetByID(
		ctx,
		operationID,
	)
}

// RegisterUploadedIcon persists one completed frontend icon upload.
//
// The Storage upload itself is not performed here.
// frontend uploads directly to Firebase Storage and then registers
// the resulting URL/ObjectPath on the operation.
func (
	uc *TokenBlueprintCreateOperationUsecase,
) RegisterUploadedIcon(
	ctx context.Context,
	operationID string,
	input tbdom.RegisterCreateOperationIconUploadInput,
) (tbdom.CreateOperation, error) {
	if err := uc.validateOperationRepository(); err != nil {
		return tbdom.CreateOperation{}, err
	}

	operation, err := uc.Get(
		ctx,
		operationID,
	)
	if err != nil {
		return tbdom.CreateOperation{}, err
	}

	return uc.persistMutation(
		ctx,
		operation,
		func(
			value *tbdom.CreateOperation,
		) error {
			return value.RegisterIconUpload(
				input,
				uc.currentTime(),
			)
		},
	)
}

// RegisterUploadedContent persists one completed frontend content upload.
//
// Each file is registered immediately after its Storage upload completes.
// This prevents a browser reload after several uploads from losing all
// references to already-uploaded Storage objects.
func (
	uc *TokenBlueprintCreateOperationUsecase,
) RegisterUploadedContent(
	ctx context.Context,
	operationID string,
	input tbdom.RegisterCreateOperationContentUploadInput,
) (tbdom.CreateOperation, error) {
	if err := uc.validateOperationRepository(); err != nil {
		return tbdom.CreateOperation{}, err
	}

	operation, err := uc.Get(
		ctx,
		operationID,
	)
	if err != nil {
		return tbdom.CreateOperation{}, err
	}

	return uc.persistMutation(
		ctx,
		operation,
		func(
			value *tbdom.CreateOperation,
		) error {
			return value.RegisterContentUpload(
				input,
				uc.currentTime(),
			)
		},
	)
}

// Commit completes the browser-dependent portion of the operation.
//
// Commit:
// 1. verifies every planned upload has been registered;
// 2. verifies each ObjectPath actually exists in Firebase Storage;
// 3. transitions waiting_upload -> queued;
// 4. enqueues the Cloud Task.
//
// Once queued is persisted, the browser can safely navigate away.
func (uc *TokenBlueprintCreateOperationUsecase) Commit(
	ctx context.Context,
	operationID string,
) (tbdom.CreateOperation, error) {
	if err := uc.validateDependencies(); err != nil {
		return tbdom.CreateOperation{}, err
	}

	operation, err := uc.Get(
		ctx,
		operationID,
	)
	if err != nil {
		return tbdom.CreateOperation{}, err
	}

	switch operation.Status {
	case tbdom.CreateOperationStatusCompleted,
		tbdom.CreateOperationStatusProcessing,
		tbdom.CreateOperationStatusFailedRetryable,
		tbdom.CreateOperationStatusFailedFatal:
		return operation, nil

	case tbdom.CreateOperationStatusQueued:
		// 前回CommitでFirestore更新後、Cloud Tasks enqueue応答だけ
		// 失敗したケースを回復する。
		if err := uc.enqueueOperation(
			ctx,
			operation,
		); err != nil {
			return operation, err
		}

		return operation, nil

	case tbdom.CreateOperationStatusWaitingUpload:

	default:
		return operation, fmt.Errorf(
			"%w: unsupported status %q",
			tbdom.ErrInvalidCreateOperation,
			operation.Status,
		)
	}

	if !operation.AllUploadsCompleted() {
		return operation, fmt.Errorf(
			"%w: %d of %d uploads completed",
			tbdom.ErrCreateOperationUploadIncomplete,
			operation.CompletedUploadCount(),
			operation.ExpectedUploadCount(),
		)
	}

	if err := uc.verifyUploadedAssets(
		ctx,
		operation,
	); err != nil {
		return operation, err
	}

	operation, err = uc.persistMutation(
		ctx,
		operation,
		func(
			value *tbdom.CreateOperation,
		) error {
			return value.MarkQueued(
				uc.currentTime(),
			)
		},
	)
	if err != nil {
		return operation, err
	}

	if err := uc.enqueueOperation(
		ctx,
		operation,
	); err != nil {
		return operation, err
	}

	return operation, nil
}

// Execute performs the browser-independent finalization.
//
// Cloud Tasks should call this method for queued/processing operations.
//
// Processing is idempotent:
// - completed is a no-op;
// - processing may safely resume;
// - TokenBlueprint icon/content update may be repeated;
// - Complete uses CreateOperation optimistic locking.
func (uc *TokenBlueprintCreateOperationUsecase) Execute(
	ctx context.Context,
	operationID string,
) (tbdom.CreateOperation, error) {
	if err := uc.validateDependencies(); err != nil {
		return tbdom.CreateOperation{}, err
	}

	operation, err := uc.Get(
		ctx,
		operationID,
	)
	if err != nil {
		return tbdom.CreateOperation{}, err
	}

	return uc.executeLoaded(
		ctx,
		operation,
	)
}

// Retry resumes a failed_retryable operation from its persisted ResumeStatus.
//
// This method is suitable for a Cloud Tasks retry handler:
// StartRetry restores queued/processing and processing continues in the
// same request.
func (uc *TokenBlueprintCreateOperationUsecase) Retry(
	ctx context.Context,
	operationID string,
) (tbdom.CreateOperation, error) {
	if err := uc.validateDependencies(); err != nil {
		return tbdom.CreateOperation{}, err
	}

	operation, err := uc.Get(
		ctx,
		operationID,
	)
	if err != nil {
		return tbdom.CreateOperation{}, err
	}

	if !operation.CanRetry() {
		return operation,
			tbdom.ErrCreateOperationNotRetryable
	}

	operation, err = uc.persistMutation(
		ctx,
		operation,
		func(
			value *tbdom.CreateOperation,
		) error {
			return value.StartRetry(
				uc.currentTime(),
			)
		},
	)
	if err != nil {
		return operation, err
	}

	return uc.executeLoaded(
		ctx,
		operation,
	)
}

func (
	uc *TokenBlueprintCreateOperationUsecase,
) executeLoaded(
	ctx context.Context,
	operation tbdom.CreateOperation,
) (tbdom.CreateOperation, error) {
	switch operation.Status {
	case tbdom.CreateOperationStatusCompleted,
		tbdom.CreateOperationStatusFailedFatal,
		tbdom.CreateOperationStatusFailedRetryable:
		return operation, nil

	case tbdom.CreateOperationStatusWaitingUpload:
		return operation, fmt.Errorf(
			"%w: waiting_upload operation cannot be executed",
			tbdom.ErrInvalidCreateOperationTransition,
		)

	case tbdom.CreateOperationStatusQueued:
		var err error

		operation, err = uc.persistMutation(
			ctx,
			operation,
			func(
				value *tbdom.CreateOperation,
			) error {
				return value.StartProcessing(
					uc.currentTime(),
				)
			},
		)
		if err != nil {
			return operation, err
		}

	case tbdom.CreateOperationStatusProcessing:

	default:
		return operation, fmt.Errorf(
			"%w: unsupported status %q",
			tbdom.ErrInvalidCreateOperation,
			operation.Status,
		)
	}

	if err := uc.verifyUploadedAssets(
		ctx,
		operation,
	); err != nil {
		return uc.failExecution(
			ctx,
			operation,
			err,
		)
	}

	if err := uc.finalizeTokenBlueprint(
		ctx,
		operation,
	); err != nil {
		return uc.failExecution(
			ctx,
			operation,
			err,
		)
	}

	operation, err := uc.persistMutation(
		ctx,
		operation,
		func(
			value *tbdom.CreateOperation,
		) error {
			return value.Complete(
				uc.currentTime(),
			)
		},
	)
	if err != nil {
		return operation, err
	}

	return operation, nil
}

// verifyUploadedAssets verifies both persisted upload metadata and
// Firebase Storage existence.
//
// This is intentionally executed both:
// - before waiting_upload -> queued;
// - immediately before TokenBlueprint finalization.
func (
	uc *TokenBlueprintCreateOperationUsecase,
) verifyUploadedAssets(
	ctx context.Context,
	operation tbdom.CreateOperation,
) error {
	if uc.storage == nil {
		return errors.New(
			"token blueprint asset storage is nil",
		)
	}

	if !operation.AllUploadsCompleted() {
		return fmt.Errorf(
			"%w: %d of %d uploads completed",
			tbdom.ErrCreateOperationUploadIncomplete,
			operation.CompletedUploadCount(),
			operation.ExpectedUploadCount(),
		)
	}

	if operation.Icon != nil {
		objectPath :=
			strings.TrimSpace(
				operation.Icon.ObjectPath,
			)

		if objectPath == "" {
			return fmt.Errorf(
				"%w: icon objectPath is empty",
				tbdom.ErrCreateOperationUploadIncomplete,
			)
		}

		exists, err := uc.storage.Exists(
			ctx,
			objectPath,
		)
		if err != nil {
			return err
		}

		if !exists {
			return fmt.Errorf(
				"%w: icon objectPath %q",
				tbdom.ErrCreateOperationAssetNotFound,
				objectPath,
			)
		}
	}

	for _, content := range operation.Contents {
		objectPath :=
			strings.TrimSpace(
				content.ObjectPath,
			)

		if objectPath == "" {
			return fmt.Errorf(
				"%w: content %q objectPath is empty",
				tbdom.ErrCreateOperationUploadIncomplete,
				content.ID,
			)
		}

		exists, err := uc.storage.Exists(
			ctx,
			objectPath,
		)
		if err != nil {
			return err
		}

		if !exists {
			return fmt.Errorf(
				"%w: content %q objectPath %q",
				tbdom.ErrCreateOperationAssetNotFound,
				content.ID,
				objectPath,
			)
		}
	}

	return nil
}

// finalizeTokenBlueprint applies all registered Storage metadata to the
// already-created TokenBlueprint.
func (
	uc *TokenBlueprintCreateOperationUsecase,
) finalizeTokenBlueprint(
	ctx context.Context,
	operation tbdom.CreateOperation,
) error {
	if uc.tokenBlueprintUC == nil {
		return errors.New(
			"token blueprint usecase is nil",
		)
	}

	contentFiles, err :=
		createOperationContentFiles(
			operation,
		)
	if err != nil {
		return err
	}

	request := UpdateBlueprintRequest{
		ID: operation.TokenBlueprintID,

		ContentFiles: &contentFiles,

		UpdatedBy: operation.ActorID,
	}

	if operation.Icon != nil {
		iconURL := operation.Icon.URL
		iconObjectPath :=
			operation.Icon.ObjectPath
		iconFileName :=
			operation.Icon.FileName
		iconContentType :=
			operation.Icon.ContentType
		iconSize :=
			operation.Icon.Size

		request.IconURL =
			&iconURL
		request.IconObjectPath =
			&iconObjectPath
		request.IconFileName =
			&iconFileName
		request.IconContentType =
			&iconContentType
		request.IconSize =
			&iconSize
	}

	_, err = uc.tokenBlueprintUC.Update(
		ctx,
		request,
	)

	return err
}

func createOperationContentFiles(
	operation tbdom.CreateOperation,
) ([]tbdom.ContentFile, error) {
	files := make(
		[]tbdom.ContentFile,
		0,
		len(operation.Contents),
	)

	for index, content := range operation.Contents {
		if !content.Uploaded ||
			content.UploadedAt == nil {
			return nil, fmt.Errorf(
				"%w: contents[%d] is not uploaded",
				tbdom.ErrCreateOperationUploadIncomplete,
				index,
			)
		}

		uploadedAt :=
			content.UploadedAt.UTC()

		file := tbdom.ContentFile{
			ID:          content.ID,
			Name:        content.Name,
			Type:        content.Type,
			ContentType: content.ContentType,
			URL:         content.URL,
			ObjectPath:  content.ObjectPath,
			IsPublic:    false,
			Size:        content.Size,

			CreatedAt: uploadedAt,
			CreatedBy: operation.ActorID,
			UpdatedAt: uploadedAt,
			UpdatedBy: operation.ActorID,
		}

		if err := file.Validate(); err != nil {
			return nil, fmt.Errorf(
				"token blueprint create operation content %q: %w",
				content.ID,
				err,
			)
		}

		files = append(
			files,
			file,
		)
	}

	if err := tbdom.ValidateContentFiles(
		files,
	); err != nil {
		return nil, err
	}

	return files, nil
}

func (
	uc *TokenBlueprintCreateOperationUsecase,
) failExecution(
	ctx context.Context,
	operation tbdom.CreateOperation,
	cause error,
) (tbdom.CreateOperation, error) {
	retryable :=
		uc.isRetryableError(cause)

	updated, persistErr :=
		uc.persistMutation(
			ctx,
			operation,
			func(
				value *tbdom.CreateOperation,
			) error {
				if retryable {
					return value.FailRetryable(
						cause,
						uc.currentTime(),
					)
				}

				return value.FailFatal(
					cause,
					uc.currentTime(),
				)
			},
		)
	if persistErr != nil {
		return operation,
			errors.Join(
				cause,
				persistErr,
			)
	}

	if updated.Status ==
		tbdom.CreateOperationStatusFailedRetryable {
		if enqueueErr :=
			uc.enqueueRetry(
				ctx,
				updated,
			); enqueueErr != nil {
			return updated,
				errors.Join(
					cause,
					enqueueErr,
				)
		}
	}

	return updated, cause
}

func (
	uc *TokenBlueprintCreateOperationUsecase,
) enqueueOperation(
	ctx context.Context,
	operation tbdom.CreateOperation,
) error {
	if uc.queue == nil {
		return errors.New(
			"token blueprint create operation queue is nil",
		)
	}

	if operation.Status !=
		tbdom.CreateOperationStatusQueued {
		return fmt.Errorf(
			"%w: cannot enqueue status %q",
			tbdom.ErrInvalidCreateOperationTransition,
			operation.Status,
		)
	}

	// UpdatedAtはMarkQueued時に固定されるため、
	// Commit再送時も同じscheduledAtとなる。
	// Queue adapter側のdeterministic task ID生成に利用できる。
	scheduledAt := operation.UpdatedAt
	if scheduledAt.IsZero() {
		scheduledAt = uc.currentTime()
	}

	return uc.queue.Enqueue(
		ctx,
		operation.ID,
		scheduledAt,
	)
}

func (
	uc *TokenBlueprintCreateOperationUsecase,
) enqueueRetry(
	ctx context.Context,
	operation tbdom.CreateOperation,
) error {
	if operation.Status !=
		tbdom.CreateOperationStatusFailedRetryable {
		return nil
	}

	if uc.queue == nil {
		return errors.New(
			"token blueprint create operation queue is nil",
		)
	}

	baseTime := operation.UpdatedAt
	if baseTime.IsZero() {
		baseTime = uc.currentTime()
	}

	scheduledAt :=
		baseTime.Add(
			tokenBlueprintCreateOperationRetryDelay(
				operation.RetryCount,
			),
		)

	return uc.queue.Enqueue(
		ctx,
		operation.ID,
		scheduledAt,
	)
}

func (
	uc *TokenBlueprintCreateOperationUsecase,
) persistMutation(
	ctx context.Context,
	operation tbdom.CreateOperation,
	mutate func(
		*tbdom.CreateOperation,
	) error,
) (tbdom.CreateOperation, error) {
	if mutate == nil {
		return operation, errors.New(
			"token blueprint create operation mutation is nil",
		)
	}

	expectedVersion :=
		operation.Version

	if err := mutate(
		&operation,
	); err != nil {
		return operation, err
	}

	updated, err :=
		uc.operationRepo.Update(
			ctx,
			operation,
			expectedVersion,
		)
	if err != nil {
		return operation, err
	}

	return updated, nil
}

func (
	uc *TokenBlueprintCreateOperationUsecase,
) cleanupUncommittedTokenBlueprint(
	ctx context.Context,
	tokenBlueprintID string,
) error {
	if uc == nil ||
		uc.tokenBlueprintUC == nil {
		return nil
	}

	tokenBlueprintID =
		strings.TrimSpace(
			tokenBlueprintID,
		)
	if tokenBlueprintID == "" {
		return nil
	}

	err := uc.tokenBlueprintUC.Delete(
		ctx,
		tokenBlueprintID,
	)
	if err == nil ||
		errors.Is(
			err,
			tbdom.ErrNotFound,
		) {
		return nil
	}

	return err
}

func (
	uc *TokenBlueprintCreateOperationUsecase,
) validateDependencies() error {
	if uc == nil {
		return errors.New(
			"token blueprint create operation usecase is nil",
		)
	}

	if uc.tokenBlueprintUC == nil {
		return errors.New(
			"token blueprint usecase is nil",
		)
	}

	if uc.operationRepo == nil {
		return errors.New(
			"token blueprint create operation repository is nil",
		)
	}

	if uc.storage == nil {
		return errors.New(
			"token blueprint asset storage is nil",
		)
	}

	if uc.queue == nil {
		return errors.New(
			"token blueprint create operation queue is nil",
		)
	}

	return nil
}

func (
	uc *TokenBlueprintCreateOperationUsecase,
) validateOperationRepository() error {
	if uc == nil {
		return errors.New(
			"token blueprint create operation usecase is nil",
		)
	}

	if uc.operationRepo == nil {
		return errors.New(
			"token blueprint create operation repository is nil",
		)
	}

	return nil
}

func (
	uc *TokenBlueprintCreateOperationUsecase,
) currentTime() time.Time {
	if uc == nil ||
		uc.now == nil {
		return time.Now().UTC()
	}

	return uc.now().UTC()
}

func normalizeTokenBlueprintCreateOperationIcon(
	icon *tbdom.CreateOperationIcon,
) (*tbdom.CreateOperationIcon, error) {
	if icon == nil {
		return nil, nil
	}

	normalized := *icon

	normalized.FileName =
		strings.TrimSpace(
			normalized.FileName,
		)

	normalized.ContentType =
		strings.TrimSpace(
			normalized.ContentType,
		)

	normalized.URL =
		strings.TrimSpace(
			normalized.URL,
		)

	normalized.ObjectPath =
		strings.TrimSpace(
			normalized.ObjectPath,
		)

	if normalized.FileName == "" {
		return nil, fmt.Errorf(
			"%w: icon fileName is required",
			tbdom.ErrInvalidCreateOperation,
		)
	}

	if normalized.ContentType == "" {
		return nil, fmt.Errorf(
			"%w: icon contentType is required",
			tbdom.ErrInvalidCreateOperation,
		)
	}

	if normalized.Size < 0 {
		return nil, fmt.Errorf(
			"%w: icon size must be >= 0",
			tbdom.ErrInvalidCreateOperation,
		)
	}

	if normalized.Uploaded ||
		normalized.URL != "" ||
		normalized.ObjectPath != "" ||
		normalized.UploadedAt != nil {
		return nil, fmt.Errorf(
			"%w: icon must not contain upload result before Start",
			tbdom.ErrInvalidCreateOperation,
		)
	}

	return &normalized, nil
}

func normalizeTokenBlueprintCreateOperationContents(
	contents []tbdom.CreateOperationContent,
) ([]tbdom.CreateOperationContent, error) {
	if len(contents) == 0 {
		return []tbdom.CreateOperationContent{},
			nil
	}

	out := make(
		[]tbdom.CreateOperationContent,
		0,
		len(contents),
	)

	seenIDs := make(
		map[string]struct{},
		len(contents),
	)

	for index, content := range contents {
		content.ID =
			strings.TrimSpace(
				content.ID,
			)

		content.Name =
			strings.TrimSpace(
				content.Name,
			)

		content.ContentType =
			strings.TrimSpace(
				content.ContentType,
			)

		content.URL =
			strings.TrimSpace(
				content.URL,
			)

		content.ObjectPath =
			strings.TrimSpace(
				content.ObjectPath,
			)

		if content.ID == "" {
			return nil, fmt.Errorf(
				"%w: contents[%d].id is required",
				tbdom.ErrInvalidCreateOperation,
				index,
			)
		}

		if _, exists :=
			seenIDs[content.ID]; exists {
			return nil, fmt.Errorf(
				"%w: duplicate contentId %q",
				tbdom.ErrCreateOperationAssetConflict,
				content.ID,
			)
		}

		seenIDs[content.ID] = struct{}{}

		if content.Name == "" {
			return nil, fmt.Errorf(
				"%w: contents[%d].name is required",
				tbdom.ErrInvalidCreateOperation,
				index,
			)
		}

		if !tbdom.IsValidContentType(
			content.Type,
		) {
			return nil, fmt.Errorf(
				"%w: contents[%d].type %q is invalid",
				tbdom.ErrInvalidCreateOperation,
				index,
				content.Type,
			)
		}

		if content.ContentType == "" {
			return nil, fmt.Errorf(
				"%w: contents[%d].contentType is required",
				tbdom.ErrInvalidCreateOperation,
				index,
			)
		}

		if content.Size < 0 {
			return nil, fmt.Errorf(
				"%w: contents[%d].size must be >= 0",
				tbdom.ErrInvalidCreateOperation,
				index,
			)
		}

		if content.Uploaded ||
			content.URL != "" ||
			content.ObjectPath != "" ||
			content.UploadedAt != nil {
			return nil, fmt.Errorf(
				"%w: contents[%d] must not contain upload result before Start",
				tbdom.ErrInvalidCreateOperation,
				index,
			)
		}

		out = append(
			out,
			content,
		)
	}

	return out, nil
}

func generateTokenBlueprintCreateOperationID(
	prefix string,
) (string, error) {
	value := make(
		[]byte,
		16,
	)

	if _, err := rand.Read(
		value,
	); err != nil {
		return "", fmt.Errorf(
			"generate token blueprint create operation id: %w",
			err,
		)
	}

	prefix =
		strings.TrimSpace(
			prefix,
		)

	if prefix == "" {
		return hex.EncodeToString(
			value,
		), nil
	}

	return prefix +
			"_" +
			hex.EncodeToString(value),
		nil
}

func tokenBlueprintCreateOperationRetryDelay(
	retryCount int,
) time.Duration {
	switch {
	case retryCount <= 0:
		return 30 * time.Second

	case retryCount == 1:
		return 2 * time.Minute

	default:
		return 10 * time.Minute
	}
}

func defaultTokenBlueprintCreateOperationRetryableError(
	err error,
) bool {
	return errors.Is(
		err,
		context.DeadlineExceeded,
	) ||
		errors.Is(
			err,
			context.Canceled,
		) ||
		errors.Is(
			err,
			tbdom.ErrCreateOperationConflict,
		)
}
