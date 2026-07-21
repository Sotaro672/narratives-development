// backend/internal/domain/list/save_operation.go
package list

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const DefaultSaveOperationMaxRetries = 3

// SaveOperationType identifies whether the operation creates or updates a List.
type SaveOperationType string

const (
	SaveOperationTypeCreate SaveOperationType = "create"
	SaveOperationTypeUpdate SaveOperationType = "update"
)

// SaveOperationStatus represents the current state of a List save operation.
//
// Firebase Storage and Firestore cannot participate in the same transaction.
// The operation therefore progresses through multiple persisted states.
type SaveOperationStatus string

const (
	SaveOperationStatusPending           SaveOperationStatus = "pending"
	SaveOperationStatusUploading         SaveOperationStatus = "uploading"
	SaveOperationStatusRegisteringImages SaveOperationStatus = "registering_images"
	SaveOperationStatusDeletingImages    SaveOperationStatus = "deleting_images"
	SaveOperationStatusUpdatingList      SaveOperationStatus = "updating_list"
	SaveOperationStatusSettingPrimary    SaveOperationStatus = "setting_primary"
	SaveOperationStatusCompleted         SaveOperationStatus = "completed"
	SaveOperationStatusFailedRetryable   SaveOperationStatus = "failed_retryable"
	SaveOperationStatusFailedFatal       SaveOperationStatus = "failed_fatal"
	SaveOperationStatusCompensating      SaveOperationStatus = "compensating"
	SaveOperationStatusCompensated       SaveOperationStatus = "compensated"
)

// SaveOperationImage represents an image handled by a save operation.
//
// StoragePath is stored only by the save operation so that an orphaned
// Firebase Storage object can be removed during compensation.
// ListImage itself continues to store only the download URL.
type SaveOperationImage struct {
	ImageID      string
	URL          string
	StoragePath  string
	DisplayOrder int
}

// SaveOperationPayload contains all information required to retry or compensate
// a save operation without relying on the original HTTP request.
//
// TargetList is the state that should exist when the operation completes.
//
// PreviousList and PreviousImages are snapshots used when compensation must
// restore the state that existed before the operation started.
type SaveOperationPayload struct {
	TargetList             List
	PreviousList           *List
	NewImages              []SaveOperationImage
	DeleteImageIDs         []string
	PreviousImages         []ListImage
	PrimaryImageID         string
	PreviousPrimaryImageID string
}

// SaveOperationProgress records side effects that have already completed.
//
// These values allow retries to skip completed work and allow compensation to
// identify which side effects must be reversed.
type SaveOperationProgress struct {
	UploadedImageIDs        []string
	RegisteredImageIDs      []string
	DeletedImageIDs         []string
	CompensatedStoragePaths []string
	ListUpdated             bool
	PrimaryImageUpdated     bool
}

// HasSideEffects reports whether this operation has already changed either
// Firebase Storage or Firestore.
func (p SaveOperationProgress) HasSideEffects() bool {
	return len(p.UploadedImageIDs) > 0 ||
		len(p.RegisteredImageIDs) > 0 ||
		len(p.DeletedImageIDs) > 0 ||
		len(p.CompensatedStoragePaths) > 0 ||
		p.ListUpdated ||
		p.PrimaryImageUpdated
}

// SaveOperation is the persisted state machine for a List save.
//
// Version is used by the repository adapter for optimistic concurrency control.
// Each successful repository update should increment Version.
type SaveOperation struct {
	ID             string
	IdempotencyKey string
	ListID         string
	Type           SaveOperationType
	Status         SaveOperationStatus
	// ResumeStatus records the state from which a retry should resume.
	// It is populated only while Status is failed_retryable.
	ResumeStatus  SaveOperationStatus
	Payload       SaveOperationPayload
	Progress      SaveOperationProgress
	RetryCount    int
	MaxRetries    int
	LastError     string
	Version       int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	FailedAt      *time.Time
	CompletedAt   *time.Time
	CompensatedAt *time.Time
}

// NewSaveOperationInput contains the values required to start an operation.
type NewSaveOperationInput struct {
	ID             string
	IdempotencyKey string
	ListID         string
	Type           SaveOperationType
	Payload        SaveOperationPayload
	MaxRetries     int
}

var (
	ErrInvalidSaveOperation           = errors.New("list save operation: invalid")
	ErrInvalidSaveOperationTransition = errors.New(
		"list save operation: invalid status transition",
	)
	ErrSaveOperationNotRetryable = errors.New(
		"list save operation: not retryable",
	)
	ErrSaveOperationRetryExhausted = errors.New(
		"list save operation: retry limit exhausted",
	)
	ErrSaveOperationNotCompensatable = errors.New(
		"list save operation: not compensatable",
	)
)

// NewSaveOperation creates a pending operation.
//
// ID and IdempotencyKey must be generated before this function is called.
// The repository must enforce uniqueness of IdempotencyKey.
func NewSaveOperation(
	input NewSaveOperationInput,
	now time.Time,
) (SaveOperation, error) {
	id := strings.TrimSpace(input.ID)
	idempotencyKey := strings.TrimSpace(input.IdempotencyKey)
	listID := strings.TrimSpace(input.ListID)
	if id == "" {
		return SaveOperation{}, fmt.Errorf(
			"%w: id is required",
			ErrInvalidSaveOperation,
		)
	}
	if idempotencyKey == "" {
		return SaveOperation{}, fmt.Errorf(
			"%w: idempotencyKey is required",
			ErrInvalidSaveOperation,
		)
	}
	if listID == "" {
		return SaveOperation{}, fmt.Errorf(
			"%w: listId is required",
			ErrInvalidSaveOperation,
		)
	}
	if !input.Type.IsValid() {
		return SaveOperation{}, fmt.Errorf(
			"%w: invalid operation type %q",
			ErrInvalidSaveOperation,
			input.Type,
		)
	}
	maxRetries := input.MaxRetries
	if maxRetries <= 0 {
		maxRetries = DefaultSaveOperationMaxRetries
	}
	now = now.UTC()
	operation := SaveOperation{
		ID:             id,
		IdempotencyKey: idempotencyKey,
		ListID:         listID,
		Type:           input.Type,
		Status:         SaveOperationStatusPending,
		Payload:        cloneSaveOperationPayload(input.Payload),
		Progress:       SaveOperationProgress{},
		RetryCount:     0,
		MaxRetries:     maxRetries,
		Version:        1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := operation.Validate(); err != nil {
		return SaveOperation{}, err
	}
	return operation, nil
}

// IsValid reports whether the operation type is supported.
func (t SaveOperationType) IsValid() bool {
	switch t {
	case SaveOperationTypeCreate,
		SaveOperationTypeUpdate:
		return true
	default:
		return false
	}
}

// IsValid reports whether the status is part of the state machine.
func (s SaveOperationStatus) IsValid() bool {
	switch s {
	case SaveOperationStatusPending,
		SaveOperationStatusUploading,
		SaveOperationStatusRegisteringImages,
		SaveOperationStatusDeletingImages,
		SaveOperationStatusUpdatingList,
		SaveOperationStatusSettingPrimary,
		SaveOperationStatusCompleted,
		SaveOperationStatusFailedRetryable,
		SaveOperationStatusFailedFatal,
		SaveOperationStatusCompensating,
		SaveOperationStatusCompensated:
		return true
	default:
		return false
	}
}

// IsTerminal reports whether no additional normal processing should occur.
func (s SaveOperationStatus) IsTerminal() bool {
	switch s {
	case SaveOperationStatusCompleted,
		SaveOperationStatusFailedFatal,
		SaveOperationStatusCompensated:
		return true
	default:
		return false
	}
}

// IsExecutionStatus reports whether the operation is currently executing a
// normal save phase.
func (s SaveOperationStatus) IsExecutionStatus() bool {
	switch s {
	case SaveOperationStatusPending,
		SaveOperationStatusUploading,
		SaveOperationStatusRegisteringImages,
		SaveOperationStatusDeletingImages,
		SaveOperationStatusUpdatingList,
		SaveOperationStatusSettingPrimary:
		return true
	default:
		return false
	}
}

// IsResumable reports whether a failed operation can resume from this status.
func (s SaveOperationStatus) IsResumable() bool {
	return s.IsExecutionStatus() ||
		s == SaveOperationStatusCompensating
}

// Validate checks persisted operation invariants.
func (o SaveOperation) Validate() error {
	if err := validateSaveOperationID("id", o.ID); err != nil {
		return err
	}
	if strings.TrimSpace(o.IdempotencyKey) == "" {
		return fmt.Errorf("%w: idempotencyKey is required", ErrInvalidSaveOperation)
	}
	if strings.ContainsAny(o.IdempotencyKey, "\r\n\x00") {
		return fmt.Errorf("%w: idempotencyKey contains invalid characters", ErrInvalidSaveOperation)
	}
	if err := validateSaveOperationID("listId", o.ListID); err != nil {
		return err
	}
	if !o.Type.IsValid() {
		return fmt.Errorf("%w: invalid operation type %q", ErrInvalidSaveOperation, o.Type)
	}
	if !o.Status.IsValid() {
		return fmt.Errorf("%w: invalid status %q", ErrInvalidSaveOperation, o.Status)
	}
	if o.MaxRetries < 0 {
		return fmt.Errorf("%w: maxRetries must be >= 0", ErrInvalidSaveOperation)
	}
	if o.RetryCount < 0 {
		return fmt.Errorf("%w: retryCount must be >= 0", ErrInvalidSaveOperation)
	}
	if o.RetryCount > o.MaxRetries {
		return fmt.Errorf("%w: retryCount exceeds maxRetries", ErrInvalidSaveOperation)
	}
	if o.Version <= 0 {
		return fmt.Errorf("%w: version must be greater than zero", ErrInvalidSaveOperation)
	}
	if o.CreatedAt.IsZero() {
		return fmt.Errorf("%w: createdAt is required", ErrInvalidSaveOperation)
	}
	if o.UpdatedAt.IsZero() {
		return fmt.Errorf("%w: updatedAt is required", ErrInvalidSaveOperation)
	}
	if o.UpdatedAt.Before(o.CreatedAt) {
		return fmt.Errorf("%w: updatedAt must not precede createdAt", ErrInvalidSaveOperation)
	}
	if o.Status == SaveOperationStatusFailedRetryable {
		if !o.ResumeStatus.IsResumable() {
			return fmt.Errorf("%w: failed_retryable requires a resumable status", ErrInvalidSaveOperation)
		}
	} else if o.ResumeStatus != "" {
		return fmt.Errorf("%w: resumeStatus is allowed only for failed_retryable", ErrInvalidSaveOperation)
	}
	if o.Status == SaveOperationStatusFailedRetryable || o.Status == SaveOperationStatusFailedFatal {
		if o.FailedAt == nil {
			return fmt.Errorf("%w: failedAt is required for failed status", ErrInvalidSaveOperation)
		}
		if strings.TrimSpace(o.LastError) == "" {
			return fmt.Errorf("%w: lastError is required for failed status", ErrInvalidSaveOperation)
		}
	}
	if o.Status == SaveOperationStatusCompleted && o.CompletedAt == nil {
		return fmt.Errorf("%w: completedAt is required for completed status", ErrInvalidSaveOperation)
	}
	if o.Status != SaveOperationStatusCompleted && o.CompletedAt != nil {
		return fmt.Errorf("%w: completedAt is allowed only for completed status", ErrInvalidSaveOperation)
	}
	if o.Status == SaveOperationStatusCompensated && o.CompensatedAt == nil {
		return fmt.Errorf("%w: compensatedAt is required for compensated status", ErrInvalidSaveOperation)
	}
	if o.Status != SaveOperationStatusCompensated && o.CompensatedAt != nil {
		return fmt.Errorf("%w: compensatedAt is allowed only for compensated status", ErrInvalidSaveOperation)
	}
	if err := validateSaveOperationPayload(o.ListID, o.Payload); err != nil {
		return err
	}
	if err := validateSaveOperationProgress(o.Payload, o.Progress); err != nil {
		return err
	}
	if o.Status == SaveOperationStatusCompleted {
		if err := validateSaveOperationCompletion(o.Payload, o.Progress); err != nil {
			return err
		}
	}
	return nil
}

// StartUploading moves the operation into the Firebase Storage upload phase.
//
// This phase may be driven by the frontend. It remains part of the persisted
// operation so orphaned uploads can be detected and compensated.
func (o *SaveOperation) StartUploading(now time.Time) error {
	return o.transitionTo(
		SaveOperationStatusUploading,
		now,
	)
}

// StartRegisteringImages begins Firestore image metadata registration.
//
// An operation may move directly from pending to this phase when the files
// were already uploaded before the operation was created.
func (o *SaveOperation) StartRegisteringImages(
	now time.Time,
) error {
	return o.transitionTo(
		SaveOperationStatusRegisteringImages,
		now,
	)
}

// StartDeletingImages begins deletion of image metadata scheduled for removal.
func (o *SaveOperation) StartDeletingImages(
	now time.Time,
) error {
	return o.transitionTo(
		SaveOperationStatusDeletingImages,
		now,
	)
}

// StartUpdatingList begins the List document update phase.
func (o *SaveOperation) StartUpdatingList(
	now time.Time,
) error {
	return o.transitionTo(
		SaveOperationStatusUpdatingList,
		now,
	)
}

// StartSettingPrimary begins the primary image update phase.
func (o *SaveOperation) StartSettingPrimary(
	now time.Time,
) error {
	return o.transitionTo(
		SaveOperationStatusSettingPrimary,
		now,
	)
}

// MarkImageUploaded records a completed Firebase Storage upload.
func (o *SaveOperation) MarkImageUploaded(
	imageID string,
	now time.Time,
) error {
	if o == nil {
		return fmt.Errorf(
			"%w: operation is nil",
			ErrInvalidSaveOperation,
		)
	}
	if o.Status != SaveOperationStatusUploading {
		return fmt.Errorf(
			"%w: cannot mark image uploaded while status is %q",
			ErrInvalidSaveOperationTransition,
			o.Status,
		)
	}
	imageID = strings.TrimSpace(imageID)
	if !containsSaveOperationImageID(o.Payload.NewImages, imageID) {
		return fmt.Errorf("%w: imageId %q is not included in newImages", ErrInvalidSaveOperation, imageID)
	}
	o.Progress.UploadedImageIDs = appendUniqueString(
		o.Progress.UploadedImageIDs,
		imageID,
	)
	o.touch(now)
	return nil
}

// MarkImageRegistered records a completed ListImage metadata creation.
func (o *SaveOperation) MarkImageRegistered(
	imageID string,
	now time.Time,
) error {
	if o == nil {
		return fmt.Errorf(
			"%w: operation is nil",
			ErrInvalidSaveOperation,
		)
	}
	if o.Status != SaveOperationStatusRegisteringImages {
		return fmt.Errorf(
			"%w: cannot mark image registered while status is %q",
			ErrInvalidSaveOperationTransition,
			o.Status,
		)
	}
	imageID = strings.TrimSpace(imageID)
	if !containsSaveOperationImageID(o.Payload.NewImages, imageID) {
		return fmt.Errorf("%w: imageId %q is not included in newImages", ErrInvalidSaveOperation, imageID)
	}
	o.Progress.RegisteredImageIDs = appendUniqueString(
		o.Progress.RegisteredImageIDs,
		imageID,
	)
	o.touch(now)
	return nil
}

// MarkImageDeleted records a completed image metadata deletion.
func (o *SaveOperation) MarkImageDeleted(
	imageID string,
	now time.Time,
) error {
	if o == nil {
		return fmt.Errorf(
			"%w: operation is nil",
			ErrInvalidSaveOperation,
		)
	}
	if o.Status != SaveOperationStatusDeletingImages {
		return fmt.Errorf(
			"%w: cannot mark image deleted while status is %q",
			ErrInvalidSaveOperationTransition,
			o.Status,
		)
	}
	imageID = strings.TrimSpace(imageID)
	if !containsString(o.Payload.DeleteImageIDs, imageID) {
		return fmt.Errorf("%w: imageId %q is not included in deleteImageIds", ErrInvalidSaveOperation, imageID)
	}
	o.Progress.DeletedImageIDs = appendUniqueString(
		o.Progress.DeletedImageIDs,
		imageID,
	)
	o.touch(now)
	return nil
}

// MarkListUpdated records that the target List document was written.
func (o *SaveOperation) MarkListUpdated(
	now time.Time,
) error {
	if o == nil {
		return fmt.Errorf(
			"%w: operation is nil",
			ErrInvalidSaveOperation,
		)
	}
	if o.Status != SaveOperationStatusUpdatingList {
		return fmt.Errorf(
			"%w: cannot mark list updated while status is %q",
			ErrInvalidSaveOperationTransition,
			o.Status,
		)
	}
	o.Progress.ListUpdated = true
	o.touch(now)
	return nil
}

// MarkPrimaryImageUpdated records that the primary image was updated.
func (o *SaveOperation) MarkPrimaryImageUpdated(
	now time.Time,
) error {
	if o == nil {
		return fmt.Errorf(
			"%w: operation is nil",
			ErrInvalidSaveOperation,
		)
	}
	if o.Status != SaveOperationStatusSettingPrimary {
		return fmt.Errorf(
			"%w: cannot mark primary image updated while status is %q",
			ErrInvalidSaveOperationTransition,
			o.Status,
		)
	}
	o.Progress.PrimaryImageUpdated = true
	o.touch(now)
	return nil
}

// MarkStoragePathCompensated records removal of an orphaned Storage object.
func (o *SaveOperation) MarkStoragePathCompensated(
	storagePath string,
	now time.Time,
) error {
	if o == nil {
		return fmt.Errorf(
			"%w: operation is nil",
			ErrInvalidSaveOperation,
		)
	}
	if o.Status != SaveOperationStatusCompensating {
		return fmt.Errorf(
			"%w: cannot mark storage compensation while status is %q",
			ErrInvalidSaveOperationTransition,
			o.Status,
		)
	}
	storagePath = strings.TrimSpace(storagePath)
	image, exists := findSaveOperationImageByStoragePath(o.Payload.NewImages, storagePath)
	if !exists {
		return fmt.Errorf("%w: storagePath %q is not included in newImages", ErrInvalidSaveOperation, storagePath)
	}
	if !o.IsImageUploaded(image.ImageID) {
		return fmt.Errorf("%w: storagePath %q was not recorded as uploaded", ErrInvalidSaveOperationTransition, storagePath)
	}
	o.Progress.CompensatedStoragePaths = appendUniqueString(
		o.Progress.CompensatedStoragePaths,
		storagePath,
	)
	o.touch(now)
	return nil
}

// Complete marks the normal save flow as completed.
func (o *SaveOperation) Complete(now time.Time) error {
	if o == nil {
		return fmt.Errorf(
			"%w: operation is nil",
			ErrInvalidSaveOperation,
		)
	}
	if !o.Status.IsExecutionStatus() {
		return fmt.Errorf("%w: cannot complete from status %q", ErrInvalidSaveOperationTransition, o.Status)
	}
	if err := validateSaveOperationCompletion(o.Payload, o.Progress); err != nil {
		return err
	}
	now = now.UTC()
	o.Status = SaveOperationStatusCompleted
	o.ResumeStatus = ""
	o.LastError = ""
	o.FailedAt = nil
	o.CompletedAt = timePointer(now)
	o.CompensatedAt = nil
	o.UpdatedAt = now
	return nil
}

// FailRetryable records a failure that may be retried.
//
// RetryCount is incremented when StartRetry is called, not when the failure is
// recorded. This makes RetryCount represent actual retry attempts.
func (o *SaveOperation) FailRetryable(
	cause error,
	now time.Time,
) error {
	if o == nil {
		return fmt.Errorf(
			"%w: operation is nil",
			ErrInvalidSaveOperation,
		)
	}
	if !o.Status.IsResumable() {
		return fmt.Errorf(
			"%w: cannot fail retryably from status %q",
			ErrInvalidSaveOperationTransition,
			o.Status,
		)
	}
	if o.RetryCount >= o.MaxRetries {
		return o.FailFatal(
			fmt.Errorf(
				"%w: %v",
				ErrSaveOperationRetryExhausted,
				cause,
			),
			now,
		)
	}
	resumeStatus := o.Status
	now = now.UTC()
	o.Status = SaveOperationStatusFailedRetryable
	o.ResumeStatus = resumeStatus
	o.LastError = errorMessage(cause)
	o.FailedAt = timePointer(now)
	o.CompletedAt = nil
	o.CompensatedAt = nil
	o.UpdatedAt = now
	return nil
}

// FailFatal records a non-retryable failure.
func (o *SaveOperation) FailFatal(
	cause error,
	now time.Time,
) error {
	if o == nil {
		return fmt.Errorf(
			"%w: operation is nil",
			ErrInvalidSaveOperation,
		)
	}
	if o.Status.IsTerminal() {
		return fmt.Errorf(
			"%w: cannot fail terminal status %q",
			ErrInvalidSaveOperationTransition,
			o.Status,
		)
	}
	now = now.UTC()
	o.Status = SaveOperationStatusFailedFatal
	o.ResumeStatus = ""
	o.LastError = errorMessage(cause)
	o.FailedAt = timePointer(now)
	o.CompletedAt = nil
	o.CompensatedAt = nil
	o.UpdatedAt = now
	return nil
}

// CanRetry reports whether StartRetry may be called.
func (o SaveOperation) CanRetry() bool {
	return o.Status == SaveOperationStatusFailedRetryable &&
		o.ResumeStatus.IsResumable() &&
		o.RetryCount < o.MaxRetries
}

// StartRetry resumes processing from the status at which the operation failed.
func (o *SaveOperation) StartRetry(
	now time.Time,
) error {
	if o == nil {
		return fmt.Errorf(
			"%w: operation is nil",
			ErrInvalidSaveOperation,
		)
	}
	if o.Status != SaveOperationStatusFailedRetryable {
		return fmt.Errorf(
			"%w: current status is %q",
			ErrSaveOperationNotRetryable,
			o.Status,
		)
	}
	if o.RetryCount >= o.MaxRetries {
		return ErrSaveOperationRetryExhausted
	}
	if !o.ResumeStatus.IsResumable() {
		return fmt.Errorf(
			"%w: invalid resume status %q",
			ErrSaveOperationNotRetryable,
			o.ResumeStatus,
		)
	}
	resumeStatus := o.ResumeStatus
	o.RetryCount++
	o.Status = resumeStatus
	o.ResumeStatus = ""
	o.LastError = ""
	o.FailedAt = nil
	o.CompletedAt = nil
	o.CompensatedAt = nil
	o.touch(now)
	return nil
}

// CanCompensate reports whether side effects exist that may need reversal.
func (o SaveOperation) CanCompensate() bool {
	if o.Status == SaveOperationStatusCompleted ||
		o.Status == SaveOperationStatusCompensated ||
		o.Status == SaveOperationStatusCompensating {
		return false
	}
	return o.Progress.HasSideEffects()
}

// BeginCompensation starts reversal of completed side effects.
func (o *SaveOperation) BeginCompensation(
	now time.Time,
) error {
	if o == nil {
		return fmt.Errorf(
			"%w: operation is nil",
			ErrInvalidSaveOperation,
		)
	}
	if !o.CanCompensate() {
		return fmt.Errorf(
			"%w: status=%q",
			ErrSaveOperationNotCompensatable,
			o.Status,
		)
	}
	now = now.UTC()
	o.Status = SaveOperationStatusCompensating
	o.ResumeStatus = ""
	o.CompletedAt = nil
	o.CompensatedAt = nil
	o.UpdatedAt = now
	return nil
}

// CompleteCompensation marks all required compensation work as completed.
func (o *SaveOperation) CompleteCompensation(
	now time.Time,
) error {
	if o == nil {
		return fmt.Errorf(
			"%w: operation is nil",
			ErrInvalidSaveOperation,
		)
	}
	if o.Status != SaveOperationStatusCompensating {
		return fmt.Errorf(
			"%w: cannot complete compensation from status %q",
			ErrInvalidSaveOperationTransition,
			o.Status,
		)
	}
	if err := validateSaveOperationCompensation(o.Payload, o.Progress); err != nil {
		return err
	}
	now = now.UTC()
	o.Status = SaveOperationStatusCompensated
	o.ResumeStatus = ""
	o.CompensatedAt = timePointer(now)
	o.CompletedAt = nil
	o.UpdatedAt = now
	return nil
}

// IsImageUploaded reports whether the upload step already completed for an
// image. Retry processing can use this to avoid duplicate uploads.
func (o SaveOperation) IsImageUploaded(
	imageID string,
) bool {
	return containsString(
		o.Progress.UploadedImageIDs,
		strings.TrimSpace(imageID),
	)
}

// IsImageRegistered reports whether metadata registration already completed.
func (o SaveOperation) IsImageRegistered(
	imageID string,
) bool {
	return containsString(
		o.Progress.RegisteredImageIDs,
		strings.TrimSpace(imageID),
	)
}

// IsImageDeleted reports whether deletion already completed.
func (o SaveOperation) IsImageDeleted(
	imageID string,
) bool {
	return containsString(
		o.Progress.DeletedImageIDs,
		strings.TrimSpace(imageID),
	)
}
func (o *SaveOperation) transitionTo(
	next SaveOperationStatus,
	now time.Time,
) error {
	if o == nil {
		return fmt.Errorf(
			"%w: operation is nil",
			ErrInvalidSaveOperation,
		)
	}
	if !next.IsValid() {
		return fmt.Errorf(
			"%w: unknown next status %q",
			ErrInvalidSaveOperationTransition,
			next,
		)
	}
	if !canTransitionSaveOperation(o.Status, next) {
		return fmt.Errorf(
			"%w: %q -> %q",
			ErrInvalidSaveOperationTransition,
			o.Status,
			next,
		)
	}
	o.Status = next
	o.ResumeStatus = ""
	o.CompletedAt = nil
	o.CompensatedAt = nil
	o.touch(now)
	return nil
}
func canTransitionSaveOperation(
	current SaveOperationStatus,
	next SaveOperationStatus,
) bool {
	switch current {
	case SaveOperationStatusPending:
		switch next {
		case SaveOperationStatusUploading,
			SaveOperationStatusRegisteringImages,
			SaveOperationStatusDeletingImages,
			SaveOperationStatusUpdatingList,
			SaveOperationStatusSettingPrimary:
			return true
		}
	case SaveOperationStatusUploading:
		return next == SaveOperationStatusRegisteringImages
	case SaveOperationStatusRegisteringImages:
		switch next {
		case SaveOperationStatusDeletingImages,
			SaveOperationStatusUpdatingList,
			SaveOperationStatusSettingPrimary:
			return true
		}
	case SaveOperationStatusDeletingImages:
		switch next {
		case SaveOperationStatusUpdatingList,
			SaveOperationStatusSettingPrimary:
			return true
		}
	case SaveOperationStatusUpdatingList:
		return next == SaveOperationStatusSettingPrimary
	}
	return false
}
func validateSaveOperationPayload(listID string, payload SaveOperationPayload) error {
	targetListID := strings.TrimSpace(payload.TargetList.ID)
	if targetListID != "" && targetListID != listID {
		return fmt.Errorf("%w: target list id %q does not match operation list id %q", ErrInvalidSaveOperation, targetListID, listID)
	}
	if payload.PreviousList != nil {
		previousListID := strings.TrimSpace(payload.PreviousList.ID)
		if previousListID != listID {
			return fmt.Errorf("%w: previous list id %q does not match operation list id %q", ErrInvalidSaveOperation, previousListID, listID)
		}
	}
	imageIDs := make(map[string]struct{}, len(payload.NewImages))
	storagePaths := make(map[string]struct{}, len(payload.NewImages))
	for index, image := range payload.NewImages {
		imageID := strings.TrimSpace(image.ImageID)
		if err := validateSaveOperationID(fmt.Sprintf("newImages[%d].imageId", index), imageID); err != nil {
			return err
		}
		if strings.TrimSpace(image.URL) == "" {
			return fmt.Errorf("%w: newImages[%d].url is required", ErrInvalidSaveOperation, index)
		}
		storagePath := strings.TrimSpace(image.StoragePath)
		if err := validateSaveOperationStoragePath(listID, imageID, storagePath); err != nil {
			return fmt.Errorf("%w: newImages[%d]: %v", ErrInvalidSaveOperation, index, err)
		}
		if image.DisplayOrder < 0 {
			return fmt.Errorf("%w: newImages[%d].displayOrder must be >= 0", ErrInvalidSaveOperation, index)
		}
		if _, exists := imageIDs[imageID]; exists {
			return fmt.Errorf("%w: duplicate new image id %q", ErrInvalidSaveOperation, imageID)
		}
		if _, exists := storagePaths[storagePath]; exists {
			return fmt.Errorf("%w: duplicate storage path %q", ErrInvalidSaveOperation, storagePath)
		}
		imageIDs[imageID] = struct{}{}
		storagePaths[storagePath] = struct{}{}
	}
	previousImageIDs := make(map[string]struct{}, len(payload.PreviousImages))
	for index, image := range payload.PreviousImages {
		imageID := strings.TrimSpace(image.ID)
		if err := validateSaveOperationID(fmt.Sprintf("previousImages[%d].id", index), imageID); err != nil {
			return err
		}
		if strings.TrimSpace(image.ListID) != listID {
			return fmt.Errorf("%w: previousImages[%d].listId does not match operation list id", ErrInvalidSaveOperation, index)
		}
		if _, exists := previousImageIDs[imageID]; exists {
			return fmt.Errorf("%w: duplicate previous image id %q", ErrInvalidSaveOperation, imageID)
		}
		previousImageIDs[imageID] = struct{}{}
	}
	deleteIDs := make(map[string]struct{}, len(payload.DeleteImageIDs))
	for index, rawImageID := range payload.DeleteImageIDs {
		imageID := strings.TrimSpace(rawImageID)
		if err := validateSaveOperationID(fmt.Sprintf("deleteImageIds[%d]", index), imageID); err != nil {
			return err
		}
		if _, exists := deleteIDs[imageID]; exists {
			return fmt.Errorf("%w: duplicate delete image id %q", ErrInvalidSaveOperation, imageID)
		}
		if _, alsoCreated := imageIDs[imageID]; alsoCreated {
			return fmt.Errorf("%w: image %q cannot be created and deleted in the same operation", ErrInvalidSaveOperation, imageID)
		}
		if len(previousImageIDs) > 0 {
			if _, exists := previousImageIDs[imageID]; !exists {
				return fmt.Errorf("%w: delete image id %q is not included in previousImages", ErrInvalidSaveOperation, imageID)
			}
		}
		deleteIDs[imageID] = struct{}{}
	}
	primaryImageID := strings.TrimSpace(payload.PrimaryImageID)
	if primaryImageID != "" {
		if err := validateSaveOperationID("primaryImageId", primaryImageID); err != nil {
			return err
		}
		_, isNew := imageIDs[primaryImageID]
		_, isPrevious := previousImageIDs[primaryImageID]
		_, isDeleted := deleteIDs[primaryImageID]
		if !isNew && !isPrevious {
			return fmt.Errorf("%w: primaryImageId %q is not included in available images", ErrInvalidSaveOperation, primaryImageID)
		}
		if isDeleted {
			return fmt.Errorf("%w: primaryImageId %q is scheduled for deletion", ErrInvalidSaveOperation, primaryImageID)
		}
	}
	previousPrimaryImageID := strings.TrimSpace(payload.PreviousPrimaryImageID)
	if previousPrimaryImageID != "" {
		if err := validateSaveOperationID("previousPrimaryImageId", previousPrimaryImageID); err != nil {
			return err
		}
		if len(previousImageIDs) > 0 {
			if _, exists := previousImageIDs[previousPrimaryImageID]; !exists {
				return fmt.Errorf("%w: previousPrimaryImageId %q is not included in previousImages", ErrInvalidSaveOperation, previousPrimaryImageID)
			}
		}
	}
	return nil
}
func validateSaveOperationProgress(payload SaveOperationPayload, progress SaveOperationProgress) error {
	if err := validateProgressIDs("uploadedImageIds", progress.UploadedImageIDs, func(value string) bool {
		return containsSaveOperationImageID(payload.NewImages, value)
	}); err != nil {
		return err
	}
	if err := validateProgressIDs("registeredImageIds", progress.RegisteredImageIDs, func(value string) bool {
		return containsSaveOperationImageID(payload.NewImages, value)
	}); err != nil {
		return err
	}
	if err := validateProgressIDs("deletedImageIds", progress.DeletedImageIDs, func(value string) bool {
		return containsString(payload.DeleteImageIDs, value)
	}); err != nil {
		return err
	}
	if err := validateProgressIDs("compensatedStoragePaths", progress.CompensatedStoragePaths, func(value string) bool {
		_, exists := findSaveOperationImageByStoragePath(payload.NewImages, value)
		return exists
	}); err != nil {
		return err
	}
	return nil
}
func validateSaveOperationCompletion(payload SaveOperationPayload, progress SaveOperationProgress) error {
	for _, image := range payload.NewImages {
		if !containsString(progress.RegisteredImageIDs, strings.TrimSpace(image.ImageID)) {
			return fmt.Errorf("%w: image %q has not been registered", ErrInvalidSaveOperationTransition, image.ImageID)
		}
	}
	for _, imageID := range payload.DeleteImageIDs {
		if !containsString(progress.DeletedImageIDs, strings.TrimSpace(imageID)) {
			return fmt.Errorf("%w: image %q has not been deleted", ErrInvalidSaveOperationTransition, imageID)
		}
	}
	if !progress.ListUpdated {
		return fmt.Errorf("%w: list has not been updated", ErrInvalidSaveOperationTransition)
	}
	if strings.TrimSpace(payload.PrimaryImageID) != strings.TrimSpace(payload.PreviousPrimaryImageID) && !progress.PrimaryImageUpdated {
		return fmt.Errorf("%w: primary image has not been updated", ErrInvalidSaveOperationTransition)
	}
	return nil
}
func validateSaveOperationCompensation(payload SaveOperationPayload, progress SaveOperationProgress) error {
	for _, imageID := range progress.UploadedImageIDs {
		image, exists := findSaveOperationImageByID(payload.NewImages, imageID)
		if !exists {
			return fmt.Errorf("%w: uploaded image %q is not included in newImages", ErrInvalidSaveOperation, imageID)
		}
		if !containsString(progress.CompensatedStoragePaths, strings.TrimSpace(image.StoragePath)) {
			return fmt.Errorf("%w: storagePath %q has not been compensated", ErrInvalidSaveOperationTransition, image.StoragePath)
		}
	}
	return nil
}
func validateSaveOperationID(fieldName string, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%w: %s is required", ErrInvalidSaveOperation, fieldName)
	}
	if strings.Contains(value, "/") || strings.Contains(value, "://") || strings.ContainsAny(value, "\r\n\x00") {
		return fmt.Errorf("%w: %s is invalid", ErrInvalidSaveOperation, fieldName)
	}
	return nil
}
func validateSaveOperationStoragePath(listID string, imageID string, storagePath string) error {
	storagePath = strings.TrimSpace(storagePath)
	prefix := "lists/" + strings.TrimSpace(listID) + "/images/" + strings.TrimSpace(imageID) + "/"
	if !strings.HasPrefix(storagePath, prefix) {
		return fmt.Errorf("storagePath must start with %q", prefix)
	}
	fileName := strings.TrimPrefix(storagePath, prefix)
	if fileName == "" || strings.Contains(fileName, "/") || fileName == "." || fileName == ".." || strings.ContainsAny(fileName, "\r\n\x00") {
		return errors.New("storagePath file name is invalid")
	}
	return nil
}
func validateProgressIDs(fieldName string, values []string, allowed func(string) bool) error {
	seen := make(map[string]struct{}, len(values))
	for index, rawValue := range values {
		value := strings.TrimSpace(rawValue)
		if value == "" {
			return fmt.Errorf("%w: %s[%d] is empty", ErrInvalidSaveOperation, fieldName, index)
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("%w: duplicate %s value %q", ErrInvalidSaveOperation, fieldName, value)
		}
		if !allowed(value) {
			return fmt.Errorf("%w: %s value %q is not included in the payload", ErrInvalidSaveOperation, fieldName, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}
func containsSaveOperationImageID(images []SaveOperationImage, imageID string) bool {
	_, exists := findSaveOperationImageByID(images, imageID)
	return exists
}
func findSaveOperationImageByID(images []SaveOperationImage, imageID string) (SaveOperationImage, bool) {
	imageID = strings.TrimSpace(imageID)
	for _, image := range images {
		if strings.TrimSpace(image.ImageID) == imageID {
			return image, true
		}
	}
	return SaveOperationImage{}, false
}
func findSaveOperationImageByStoragePath(images []SaveOperationImage, storagePath string) (SaveOperationImage, bool) {
	storagePath = strings.TrimSpace(storagePath)
	for _, image := range images {
		if strings.TrimSpace(image.StoragePath) == storagePath {
			return image, true
		}
	}
	return SaveOperationImage{}, false
}
func cloneSaveOperationPayload(source SaveOperationPayload) SaveOperationPayload {
	target := source
	target.TargetList = cloneSaveOperationList(source.TargetList)
	if source.PreviousList != nil {
		previousList := cloneSaveOperationList(*source.PreviousList)
		target.PreviousList = &previousList
	}
	target.NewImages = append([]SaveOperationImage(nil), source.NewImages...)
	target.DeleteImageIDs = append([]string(nil), source.DeleteImageIDs...)
	target.PreviousImages = cloneSaveOperationListImages(source.PreviousImages)
	return target
}
func cloneSaveOperationList(source List) List {
	target := source
	target.Prices = append([]ListPriceRow(nil), source.Prices...)
	if source.UpdatedAt != nil {
		updatedAt := *source.UpdatedAt
		target.UpdatedAt = &updatedAt
	}
	if source.UpdatedBy != nil {
		updatedBy := *source.UpdatedBy
		target.UpdatedBy = &updatedBy
	}
	return target
}
func cloneSaveOperationListImages(source []ListImage) []ListImage {
	target := make([]ListImage, len(source))
	for index, image := range source {
		target[index] = image
		if image.UpdatedAt != nil {
			updatedAt := *image.UpdatedAt
			target[index].UpdatedAt = &updatedAt
		}
		if image.UpdatedBy != nil {
			updatedBy := *image.UpdatedBy
			target[index].UpdatedBy = &updatedBy
		}
	}
	return target
}
func (o *SaveOperation) touch(now time.Time) {
	o.UpdatedAt = now.UTC()
}
func appendUniqueString(
	values []string,
	value string,
) []string {
	if containsString(values, value) {
		return values
	}
	return append(values, value)
}
func containsString(
	values []string,
	target string,
) bool {
	if target == "" {
		return false
	}
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
func errorMessage(err error) string {
	if err == nil {
		return "unknown error"
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "unknown error"
	}
	return message
}
func timePointer(value time.Time) *time.Time {
	value = value.UTC()
	return &value
}
