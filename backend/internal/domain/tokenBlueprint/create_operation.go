// backend/internal/domain/tokenBlueprint/create_operation.go
package tokenBlueprint

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const DefaultCreateOperationMaxRetries = 3

// CreateOperationStatus represents the persisted state of a
// TokenBlueprint create operation.
//
// Firebase Storage upload is performed by the frontend.
// After all expected assets have been uploaded and registered,
// the operation is committed and subsequent processing is delegated
// to Cloud Tasks.
type CreateOperationStatus string

const (
	CreateOperationStatusWaitingUpload CreateOperationStatus = "waiting_upload"

	CreateOperationStatusQueued     CreateOperationStatus = "queued"
	CreateOperationStatusProcessing CreateOperationStatus = "processing"

	CreateOperationStatusCompleted CreateOperationStatus = "completed"

	CreateOperationStatusFailedRetryable CreateOperationStatus = "failed_retryable"
	CreateOperationStatusFailedFatal     CreateOperationStatus = "failed_fatal"
)

// CreateOperationIcon represents the icon expected by a create operation.
//
// Before upload:
// - FileName / ContentType / Size describe the selected local file.
// - URL / ObjectPath / UploadedAt must be empty.
//
// After upload:
// - URL is the Firebase Storage download URL.
// - ObjectPath is the Firebase Storage object path.
// - Uploaded is true.
// - UploadedAt is populated.
type CreateOperationIcon struct {
	FileName    string
	ContentType string
	Size        int64

	URL        string
	ObjectPath string

	Uploaded   bool
	UploadedAt *time.Time
}

// CreateOperationContent represents one content file expected by
// a create operation.
//
// ID is generated before the upload starts so the same content can be
// identified across frontend upload, operation registration, retry,
// and final TokenBlueprint persistence.
type CreateOperationContent struct {
	ID          string
	Name        string
	Type        ContentFileType
	ContentType string
	Size        int64

	URL        string
	ObjectPath string

	Uploaded   bool
	UploadedAt *time.Time
}

// CreateOperation is the persisted state machine for TokenBlueprint creation.
//
// The TokenBlueprint document itself is created before this operation starts.
// This provides TokenBlueprintID before Firebase Storage upload begins.
//
// The frontend is responsible only for:
// - uploading local File objects to Firebase Storage;
// - registering each successful upload on this operation;
// - committing the operation after every expected upload is complete.
//
// After commit, Cloud Tasks owns the remaining processing and the browser is
// no longer required for completion.
//
// Version is used by the repository adapter for optimistic concurrency control.
// Repository updates should increment Version.
type CreateOperation struct {
	ID             string
	IdempotencyKey string

	TokenBlueprintID string
	CompanyID        string
	ActorID          string

	Status       CreateOperationStatus
	ResumeStatus CreateOperationStatus

	Icon     *CreateOperationIcon
	Contents []CreateOperationContent

	RetryCount int
	MaxRetries int
	LastError  string

	Version int64

	CreatedAt   time.Time
	UpdatedAt   time.Time
	FailedAt    *time.Time
	CompletedAt *time.Time
}

// NewCreateOperationInput contains the values required to create
// a waiting_upload operation.
//
// Icon is nil when no icon was selected.
//
// Contents must contain the complete list of files that the frontend intends
// to upload. This allows Commit to determine whether every expected file was
// successfully uploaded before Cloud Tasks is invoked.
type NewCreateOperationInput struct {
	ID             string
	IdempotencyKey string

	TokenBlueprintID string
	CompanyID        string
	ActorID          string

	Icon     *CreateOperationIcon
	Contents []CreateOperationContent

	MaxRetries int
}

// RegisterCreateOperationIconUploadInput contains the confirmed Firebase
// Storage result for the expected icon.
type RegisterCreateOperationIconUploadInput struct {
	URL         string
	ObjectPath  string
	FileName    string
	ContentType string
	Size        int64
}

// RegisterCreateOperationContentUploadInput contains the confirmed Firebase
// Storage result for one expected content file.
type RegisterCreateOperationContentUploadInput struct {
	ContentID string

	URL         string
	ObjectPath  string
	Name        string
	ContentType string
	Size        int64
}

var (
	ErrInvalidCreateOperation = errors.New(
		"tokenBlueprint create operation: invalid",
	)

	ErrInvalidCreateOperationTransition = errors.New(
		"tokenBlueprint create operation: invalid status transition",
	)

	ErrCreateOperationUploadIncomplete = errors.New(
		"tokenBlueprint create operation: upload incomplete",
	)

	ErrCreateOperationAssetNotFound = errors.New(
		"tokenBlueprint create operation: asset not found",
	)

	ErrCreateOperationAssetConflict = errors.New(
		"tokenBlueprint create operation: asset conflict",
	)

	ErrCreateOperationNotRetryable = errors.New(
		"tokenBlueprint create operation: not retryable",
	)

	ErrCreateOperationRetryExhausted = errors.New(
		"tokenBlueprint create operation: retry limit exhausted",
	)
)

// NewCreateOperation creates a waiting_upload operation.
//
// ID and IdempotencyKey must be generated before calling this constructor.
// The repository is responsible for enforcing IdempotencyKey uniqueness.
func NewCreateOperation(
	input NewCreateOperationInput,
	now time.Time,
) (CreateOperation, error) {
	id := strings.TrimSpace(input.ID)
	idempotencyKey := strings.TrimSpace(input.IdempotencyKey)
	tokenBlueprintID := strings.TrimSpace(input.TokenBlueprintID)
	companyID := strings.TrimSpace(input.CompanyID)
	actorID := strings.TrimSpace(input.ActorID)

	if err := validateCreateOperationID(
		"id",
		id,
	); err != nil {
		return CreateOperation{}, err
	}

	if err := validateCreateOperationIdempotencyKey(
		idempotencyKey,
	); err != nil {
		return CreateOperation{}, err
	}

	if err := validateCreateOperationID(
		"tokenBlueprintId",
		tokenBlueprintID,
	); err != nil {
		return CreateOperation{}, err
	}

	if err := validateCreateOperationID(
		"companyId",
		companyID,
	); err != nil {
		return CreateOperation{}, err
	}

	if err := validateCreateOperationID(
		"actorId",
		actorID,
	); err != nil {
		return CreateOperation{}, err
	}

	if err := validateNewCreateOperationIcon(
		input.Icon,
	); err != nil {
		return CreateOperation{}, err
	}

	if err := validateNewCreateOperationContents(
		input.Contents,
	); err != nil {
		return CreateOperation{}, err
	}

	maxRetries := input.MaxRetries
	if maxRetries <= 0 {
		maxRetries = DefaultCreateOperationMaxRetries
	}

	now = now.UTC()

	operation := CreateOperation{
		ID:             id,
		IdempotencyKey: idempotencyKey,

		TokenBlueprintID: tokenBlueprintID,
		CompanyID:        companyID,
		ActorID:          actorID,

		Status:       CreateOperationStatusWaitingUpload,
		ResumeStatus: "",

		Icon:     cloneCreateOperationIcon(input.Icon),
		Contents: cloneCreateOperationContents(input.Contents),

		RetryCount: 0,
		MaxRetries: maxRetries,
		LastError:  "",

		Version: 1,

		CreatedAt:   now,
		UpdatedAt:   now,
		FailedAt:    nil,
		CompletedAt: nil,
	}

	if err := operation.Validate(); err != nil {
		return CreateOperation{}, err
	}

	return operation, nil
}

// IsValid reports whether the status belongs to the create operation
// state machine.
func (s CreateOperationStatus) IsValid() bool {
	switch s {
	case CreateOperationStatusWaitingUpload,
		CreateOperationStatusQueued,
		CreateOperationStatusProcessing,
		CreateOperationStatusCompleted,
		CreateOperationStatusFailedRetryable,
		CreateOperationStatusFailedFatal:
		return true

	default:
		return false
	}
}

// IsTerminal reports whether normal processing must not continue.
func (s CreateOperationStatus) IsTerminal() bool {
	switch s {
	case CreateOperationStatusCompleted,
		CreateOperationStatusFailedFatal:
		return true

	default:
		return false
	}
}

// IsExecutionStatus reports whether the backend/Cloud Tasks portion of
// the operation is currently executable.
func (s CreateOperationStatus) IsExecutionStatus() bool {
	switch s {
	case CreateOperationStatusQueued,
		CreateOperationStatusProcessing:
		return true

	default:
		return false
	}
}

// IsResumable reports whether retry may resume from this status.
func (s CreateOperationStatus) IsResumable() bool {
	return s.IsExecutionStatus()
}

// Validate validates invariants for a persisted create operation.
func (o CreateOperation) Validate() error {
	if err := validateCreateOperationID(
		"id",
		o.ID,
	); err != nil {
		return err
	}

	if err := validateCreateOperationIdempotencyKey(
		o.IdempotencyKey,
	); err != nil {
		return err
	}

	if err := validateCreateOperationID(
		"tokenBlueprintId",
		o.TokenBlueprintID,
	); err != nil {
		return err
	}

	if err := validateCreateOperationID(
		"companyId",
		o.CompanyID,
	); err != nil {
		return err
	}

	if err := validateCreateOperationID(
		"actorId",
		o.ActorID,
	); err != nil {
		return err
	}

	if !o.Status.IsValid() {
		return fmt.Errorf(
			"%w: invalid status %q",
			ErrInvalidCreateOperation,
			o.Status,
		)
	}

	if o.MaxRetries < 0 {
		return fmt.Errorf(
			"%w: maxRetries must be >= 0",
			ErrInvalidCreateOperation,
		)
	}

	if o.RetryCount < 0 {
		return fmt.Errorf(
			"%w: retryCount must be >= 0",
			ErrInvalidCreateOperation,
		)
	}

	if o.RetryCount > o.MaxRetries {
		return fmt.Errorf(
			"%w: retryCount exceeds maxRetries",
			ErrInvalidCreateOperation,
		)
	}

	if o.Version <= 0 {
		return fmt.Errorf(
			"%w: version must be greater than zero",
			ErrInvalidCreateOperation,
		)
	}

	if o.CreatedAt.IsZero() {
		return fmt.Errorf(
			"%w: createdAt is required",
			ErrInvalidCreateOperation,
		)
	}

	if o.UpdatedAt.IsZero() {
		return fmt.Errorf(
			"%w: updatedAt is required",
			ErrInvalidCreateOperation,
		)
	}

	if o.UpdatedAt.Before(o.CreatedAt) {
		return fmt.Errorf(
			"%w: updatedAt must not precede createdAt",
			ErrInvalidCreateOperation,
		)
	}

	if err := validateCreateOperationIcon(
		o.Icon,
	); err != nil {
		return err
	}

	if err := validateCreateOperationContents(
		o.Contents,
	); err != nil {
		return err
	}

	if o.Status == CreateOperationStatusFailedRetryable {
		if !o.ResumeStatus.IsResumable() {
			return fmt.Errorf(
				"%w: failed_retryable requires a resumable status",
				ErrInvalidCreateOperation,
			)
		}
	} else if o.ResumeStatus != "" {
		return fmt.Errorf(
			"%w: resumeStatus is allowed only for failed_retryable",
			ErrInvalidCreateOperation,
		)
	}

	switch o.Status {
	case CreateOperationStatusFailedRetryable,
		CreateOperationStatusFailedFatal:
		if o.FailedAt == nil {
			return fmt.Errorf(
				"%w: failedAt is required for failed status",
				ErrInvalidCreateOperation,
			)
		}

		if strings.TrimSpace(o.LastError) == "" {
			return fmt.Errorf(
				"%w: lastError is required for failed status",
				ErrInvalidCreateOperation,
			)
		}

	default:
		if o.FailedAt != nil {
			return fmt.Errorf(
				"%w: failedAt is allowed only for failed status",
				ErrInvalidCreateOperation,
			)
		}

		if o.LastError != "" {
			return fmt.Errorf(
				"%w: lastError is allowed only for failed status",
				ErrInvalidCreateOperation,
			)
		}
	}

	if o.Status == CreateOperationStatusCompleted {
		if o.CompletedAt == nil {
			return fmt.Errorf(
				"%w: completedAt is required for completed status",
				ErrInvalidCreateOperation,
			)
		}
	} else if o.CompletedAt != nil {
		return fmt.Errorf(
			"%w: completedAt is allowed only for completed status",
			ErrInvalidCreateOperation,
		)
	}

	// Once the frontend has committed the operation and the backend owns
	// processing, every planned asset must already be in Firebase Storage.
	if o.Status != CreateOperationStatusWaitingUpload &&
		!o.AllUploadsCompleted() {
		return fmt.Errorf(
			"%w: status %q requires every expected upload to be completed",
			ErrCreateOperationUploadIncomplete,
			o.Status,
		)
	}

	return nil
}

// ExpectedUploadCount returns the number of assets that the frontend
// is expected to upload.
func (o CreateOperation) ExpectedUploadCount() int {
	count := len(o.Contents)

	if o.Icon != nil {
		count++
	}

	return count
}

// CompletedUploadCount returns the number of expected assets whose upload
// result has already been registered.
func (o CreateOperation) CompletedUploadCount() int {
	count := 0

	if o.Icon != nil && o.Icon.Uploaded {
		count++
	}

	for _, content := range o.Contents {
		if content.Uploaded {
			count++
		}
	}

	return count
}

// AllUploadsCompleted reports whether every expected frontend upload has
// been registered on the operation.
//
// An operation with no icon and no contents is immediately upload-complete.
func (o CreateOperation) AllUploadsCompleted() bool {
	if o.Icon != nil && !o.Icon.Uploaded {
		return false
	}

	for _, content := range o.Contents {
		if !content.Uploaded {
			return false
		}
	}

	return true
}

// RegisterIconUpload records the Firebase Storage result for the expected icon.
//
// The method is idempotent when the same upload result is submitted more than
// once. A different result for an already registered icon is treated as a
// conflict.
func (o *CreateOperation) RegisterIconUpload(
	input RegisterCreateOperationIconUploadInput,
	now time.Time,
) error {
	if o == nil {
		return fmt.Errorf(
			"%w: operation is nil",
			ErrInvalidCreateOperation,
		)
	}

	if o.Status != CreateOperationStatusWaitingUpload {
		return fmt.Errorf(
			"%w: cannot register icon while status is %q",
			ErrInvalidCreateOperationTransition,
			o.Status,
		)
	}

	if o.Icon == nil {
		return fmt.Errorf(
			"%w: icon is not expected",
			ErrCreateOperationAssetNotFound,
		)
	}

	normalized, err := normalizeCreateOperationIconUploadInput(
		input,
	)
	if err != nil {
		return err
	}

	if !sameExpectedIconMetadata(
		*o.Icon,
		normalized,
	) {
		return fmt.Errorf(
			"%w: uploaded icon metadata does not match expected icon",
			ErrCreateOperationAssetConflict,
		)
	}

	if o.Icon.Uploaded {
		if sameRegisteredIconUpload(
			*o.Icon,
			normalized,
		) {
			return nil
		}

		return fmt.Errorf(
			"%w: icon upload is already registered with different values",
			ErrCreateOperationAssetConflict,
		)
	}

	now = now.UTC()

	o.Icon.URL = normalized.URL
	o.Icon.ObjectPath = normalized.ObjectPath
	o.Icon.Uploaded = true
	o.Icon.UploadedAt = createOperationTimePointer(now)

	o.touch(now)

	return nil
}

// RegisterContentUpload records the Firebase Storage result for one expected
// content file.
//
// ContentID is the stable ID generated before upload begins.
// Re-registering the same result is idempotent.
func (o *CreateOperation) RegisterContentUpload(
	input RegisterCreateOperationContentUploadInput,
	now time.Time,
) error {
	if o == nil {
		return fmt.Errorf(
			"%w: operation is nil",
			ErrInvalidCreateOperation,
		)
	}

	if o.Status != CreateOperationStatusWaitingUpload {
		return fmt.Errorf(
			"%w: cannot register content while status is %q",
			ErrInvalidCreateOperationTransition,
			o.Status,
		)
	}

	contentID := strings.TrimSpace(
		input.ContentID,
	)
	if contentID == "" {
		return fmt.Errorf(
			"%w: contentId is required",
			ErrInvalidCreateOperation,
		)
	}

	contentIndex := -1

	for i := range o.Contents {
		if o.Contents[i].ID == contentID {
			contentIndex = i
			break
		}
	}

	if contentIndex < 0 {
		return fmt.Errorf(
			"%w: contentId %q",
			ErrCreateOperationAssetNotFound,
			contentID,
		)
	}

	normalized, err := normalizeCreateOperationContentUploadInput(
		input,
	)
	if err != nil {
		return err
	}

	content := &o.Contents[contentIndex]

	if !sameExpectedContentMetadata(
		*content,
		normalized,
	) {
		return fmt.Errorf(
			"%w: uploaded content metadata does not match expected content %q",
			ErrCreateOperationAssetConflict,
			contentID,
		)
	}

	if content.Uploaded {
		if sameRegisteredContentUpload(
			*content,
			normalized,
		) {
			return nil
		}

		return fmt.Errorf(
			"%w: content %q upload is already registered with different values",
			ErrCreateOperationAssetConflict,
			contentID,
		)
	}

	now = now.UTC()

	content.URL = normalized.URL
	content.ObjectPath = normalized.ObjectPath
	content.Uploaded = true
	content.UploadedAt = createOperationTimePointer(now)

	o.touch(now)

	return nil
}

// MarkQueued commits the frontend upload phase.
//
// After this transition all expected local File objects are no longer required.
// Cloud Tasks may safely own the remaining processing.
func (o *CreateOperation) MarkQueued(
	now time.Time,
) error {
	if o == nil {
		return fmt.Errorf(
			"%w: operation is nil",
			ErrInvalidCreateOperation,
		)
	}

	if o.Status != CreateOperationStatusWaitingUpload {
		return fmt.Errorf(
			"%w: cannot queue from status %q",
			ErrInvalidCreateOperationTransition,
			o.Status,
		)
	}

	if !o.AllUploadsCompleted() {
		return fmt.Errorf(
			"%w: %d of %d uploads completed",
			ErrCreateOperationUploadIncomplete,
			o.CompletedUploadCount(),
			o.ExpectedUploadCount(),
		)
	}

	now = now.UTC()

	o.Status = CreateOperationStatusQueued
	o.ResumeStatus = ""
	o.LastError = ""
	o.FailedAt = nil
	o.CompletedAt = nil
	o.UpdatedAt = now

	return nil
}

// StartProcessing marks that a Cloud Tasks worker has started finalization.
func (o *CreateOperation) StartProcessing(
	now time.Time,
) error {
	if o == nil {
		return fmt.Errorf(
			"%w: operation is nil",
			ErrInvalidCreateOperation,
		)
	}

	if o.Status != CreateOperationStatusQueued {
		return fmt.Errorf(
			"%w: cannot start processing from status %q",
			ErrInvalidCreateOperationTransition,
			o.Status,
		)
	}

	if !o.AllUploadsCompleted() {
		return ErrCreateOperationUploadIncomplete
	}

	now = now.UTC()

	o.Status = CreateOperationStatusProcessing
	o.ResumeStatus = ""
	o.LastError = ""
	o.FailedAt = nil
	o.CompletedAt = nil
	o.UpdatedAt = now

	return nil
}

// Complete marks TokenBlueprint finalization as completed.
func (o *CreateOperation) Complete(
	now time.Time,
) error {
	if o == nil {
		return fmt.Errorf(
			"%w: operation is nil",
			ErrInvalidCreateOperation,
		)
	}

	if o.Status != CreateOperationStatusProcessing {
		return fmt.Errorf(
			"%w: cannot complete from status %q",
			ErrInvalidCreateOperationTransition,
			o.Status,
		)
	}

	if !o.AllUploadsCompleted() {
		return ErrCreateOperationUploadIncomplete
	}

	now = now.UTC()

	o.Status = CreateOperationStatusCompleted
	o.ResumeStatus = ""
	o.LastError = ""
	o.FailedAt = nil
	o.CompletedAt = createOperationTimePointer(now)
	o.UpdatedAt = now

	return nil
}

// FailRetryable records a backend failure that may be retried.
//
// RetryCount is incremented by StartRetry rather than here, so RetryCount
// represents actual retry attempts.
func (o *CreateOperation) FailRetryable(
	cause error,
	now time.Time,
) error {
	if o == nil {
		return fmt.Errorf(
			"%w: operation is nil",
			ErrInvalidCreateOperation,
		)
	}

	if !o.Status.IsExecutionStatus() {
		return fmt.Errorf(
			"%w: cannot fail retryably from status %q",
			ErrInvalidCreateOperationTransition,
			o.Status,
		)
	}

	if o.RetryCount >= o.MaxRetries {
		return o.FailFatal(
			fmt.Errorf(
				"%w: %v",
				ErrCreateOperationRetryExhausted,
				cause,
			),
			now,
		)
	}

	resumeStatus := o.Status
	now = now.UTC()

	o.Status = CreateOperationStatusFailedRetryable
	o.ResumeStatus = resumeStatus
	o.LastError = createOperationErrorMessage(cause)
	o.FailedAt = createOperationTimePointer(now)
	o.CompletedAt = nil
	o.UpdatedAt = now

	return nil
}

// FailFatal records a non-retryable backend failure.
func (o *CreateOperation) FailFatal(
	cause error,
	now time.Time,
) error {
	if o == nil {
		return fmt.Errorf(
			"%w: operation is nil",
			ErrInvalidCreateOperation,
		)
	}

	switch o.Status {
	case CreateOperationStatusQueued,
		CreateOperationStatusProcessing,
		CreateOperationStatusFailedRetryable:

	default:
		return fmt.Errorf(
			"%w: cannot fail fatally from status %q",
			ErrInvalidCreateOperationTransition,
			o.Status,
		)
	}

	now = now.UTC()

	o.Status = CreateOperationStatusFailedFatal
	o.ResumeStatus = ""
	o.LastError = createOperationErrorMessage(cause)
	o.FailedAt = createOperationTimePointer(now)
	o.CompletedAt = nil
	o.UpdatedAt = now

	return nil
}

// CanRetry reports whether StartRetry can resume the operation.
func (o CreateOperation) CanRetry() bool {
	return o.Status == CreateOperationStatusFailedRetryable &&
		o.ResumeStatus.IsResumable() &&
		o.RetryCount < o.MaxRetries
}

// StartRetry resumes processing from the status at which the operation failed.
func (o *CreateOperation) StartRetry(
	now time.Time,
) error {
	if o == nil {
		return fmt.Errorf(
			"%w: operation is nil",
			ErrInvalidCreateOperation,
		)
	}

	if o.Status != CreateOperationStatusFailedRetryable {
		return fmt.Errorf(
			"%w: current status is %q",
			ErrCreateOperationNotRetryable,
			o.Status,
		)
	}

	if o.RetryCount >= o.MaxRetries {
		return ErrCreateOperationRetryExhausted
	}

	if !o.ResumeStatus.IsResumable() {
		return fmt.Errorf(
			"%w: invalid resume status %q",
			ErrCreateOperationNotRetryable,
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

	o.touch(now)

	return nil
}

func validateNewCreateOperationIcon(
	icon *CreateOperationIcon,
) error {
	if icon == nil {
		return nil
	}

	if err := validateCreateOperationIconBase(
		*icon,
	); err != nil {
		return err
	}

	if icon.Uploaded {
		return fmt.Errorf(
			"%w: new icon must not already be uploaded",
			ErrInvalidCreateOperation,
		)
	}

	if icon.URL != "" {
		return fmt.Errorf(
			"%w: new icon URL must be empty",
			ErrInvalidCreateOperation,
		)
	}

	if icon.ObjectPath != "" {
		return fmt.Errorf(
			"%w: new icon objectPath must be empty",
			ErrInvalidCreateOperation,
		)
	}

	if icon.UploadedAt != nil {
		return fmt.Errorf(
			"%w: new icon uploadedAt must be empty",
			ErrInvalidCreateOperation,
		)
	}

	return nil
}

func validateNewCreateOperationContents(
	contents []CreateOperationContent,
) error {
	seenIDs := make(
		map[string]struct{},
		len(contents),
	)

	for i := range contents {
		content := contents[i]

		if err := validateCreateOperationContentBase(
			content,
		); err != nil {
			return fmt.Errorf(
				"%w: contents[%d]: %v",
				ErrInvalidCreateOperation,
				i,
				err,
			)
		}

		if _, exists := seenIDs[content.ID]; exists {
			return fmt.Errorf(
				"%w: duplicate contentId %q",
				ErrCreateOperationAssetConflict,
				content.ID,
			)
		}

		seenIDs[content.ID] = struct{}{}

		if content.Uploaded {
			return fmt.Errorf(
				"%w: contents[%d] must not already be uploaded",
				ErrInvalidCreateOperation,
				i,
			)
		}

		if content.URL != "" {
			return fmt.Errorf(
				"%w: contents[%d].url must be empty",
				ErrInvalidCreateOperation,
				i,
			)
		}

		if content.ObjectPath != "" {
			return fmt.Errorf(
				"%w: contents[%d].objectPath must be empty",
				ErrInvalidCreateOperation,
				i,
			)
		}

		if content.UploadedAt != nil {
			return fmt.Errorf(
				"%w: contents[%d].uploadedAt must be empty",
				ErrInvalidCreateOperation,
				i,
			)
		}
	}

	return nil
}

func validateCreateOperationIcon(
	icon *CreateOperationIcon,
) error {
	if icon == nil {
		return nil
	}

	if err := validateCreateOperationIconBase(
		*icon,
	); err != nil {
		return err
	}

	if !icon.Uploaded {
		if icon.URL != "" ||
			icon.ObjectPath != "" ||
			icon.UploadedAt != nil {
			return fmt.Errorf(
				"%w: incomplete icon must not contain upload result",
				ErrInvalidCreateOperation,
			)
		}

		return nil
	}

	if strings.TrimSpace(icon.URL) == "" {
		return fmt.Errorf(
			"%w: uploaded icon URL is required",
			ErrInvalidCreateOperation,
		)
	}

	if strings.TrimSpace(icon.ObjectPath) == "" {
		return fmt.Errorf(
			"%w: uploaded icon objectPath is required",
			ErrInvalidCreateOperation,
		)
	}

	if icon.UploadedAt == nil ||
		icon.UploadedAt.IsZero() {
		return fmt.Errorf(
			"%w: uploaded icon uploadedAt is required",
			ErrInvalidCreateOperation,
		)
	}

	return nil
}

func validateCreateOperationIconBase(
	icon CreateOperationIcon,
) error {
	if strings.TrimSpace(icon.FileName) == "" {
		return fmt.Errorf(
			"%w: icon fileName is required",
			ErrInvalidCreateOperation,
		)
	}

	if strings.TrimSpace(icon.ContentType) == "" {
		return fmt.Errorf(
			"%w: icon contentType is required",
			ErrInvalidCreateOperation,
		)
	}

	if icon.Size < 0 {
		return fmt.Errorf(
			"%w: icon size must be >= 0",
			ErrInvalidCreateOperation,
		)
	}

	return nil
}

func validateCreateOperationContents(
	contents []CreateOperationContent,
) error {
	seenIDs := make(
		map[string]struct{},
		len(contents),
	)

	seenObjectPaths := make(
		map[string]struct{},
		len(contents),
	)

	for i := range contents {
		content := contents[i]

		if err := validateCreateOperationContentBase(
			content,
		); err != nil {
			return fmt.Errorf(
				"%w: contents[%d]: %v",
				ErrInvalidCreateOperation,
				i,
				err,
			)
		}

		if _, exists := seenIDs[content.ID]; exists {
			return fmt.Errorf(
				"%w: duplicate contentId %q",
				ErrCreateOperationAssetConflict,
				content.ID,
			)
		}

		seenIDs[content.ID] = struct{}{}

		if !content.Uploaded {
			if content.URL != "" ||
				content.ObjectPath != "" ||
				content.UploadedAt != nil {
				return fmt.Errorf(
					"%w: incomplete content %q must not contain upload result",
					ErrInvalidCreateOperation,
					content.ID,
				)
			}

			continue
		}

		if strings.TrimSpace(content.URL) == "" {
			return fmt.Errorf(
				"%w: contents[%d].url is required after upload",
				ErrInvalidCreateOperation,
				i,
			)
		}

		objectPath := strings.TrimSpace(
			content.ObjectPath,
		)
		if objectPath == "" {
			return fmt.Errorf(
				"%w: contents[%d].objectPath is required after upload",
				ErrInvalidCreateOperation,
				i,
			)
		}

		if _, exists := seenObjectPaths[objectPath]; exists {
			return fmt.Errorf(
				"%w: duplicate uploaded objectPath %q",
				ErrCreateOperationAssetConflict,
				objectPath,
			)
		}

		seenObjectPaths[objectPath] = struct{}{}

		if content.UploadedAt == nil ||
			content.UploadedAt.IsZero() {
			return fmt.Errorf(
				"%w: contents[%d].uploadedAt is required after upload",
				ErrInvalidCreateOperation,
				i,
			)
		}
	}

	return nil
}

func validateCreateOperationContentBase(
	content CreateOperationContent,
) error {
	if strings.TrimSpace(content.ID) == "" {
		return fmt.Errorf(
			"%w: contentId is required",
			ErrInvalidCreateOperation,
		)
	}

	if strings.TrimSpace(content.Name) == "" {
		return fmt.Errorf(
			"%w: content name is required",
			ErrInvalidCreateOperation,
		)
	}

	if !IsValidContentType(content.Type) {
		return fmt.Errorf(
			"%w: invalid content type %q",
			ErrInvalidCreateOperation,
			content.Type,
		)
	}

	if strings.TrimSpace(content.ContentType) == "" {
		return fmt.Errorf(
			"%w: contentType is required",
			ErrInvalidCreateOperation,
		)
	}

	if content.Size < 0 {
		return fmt.Errorf(
			"%w: content size must be >= 0",
			ErrInvalidCreateOperation,
		)
	}

	return nil
}

func normalizeCreateOperationIconUploadInput(
	input RegisterCreateOperationIconUploadInput,
) (
	RegisterCreateOperationIconUploadInput,
	error,
) {
	input.URL = strings.TrimSpace(
		input.URL,
	)
	input.ObjectPath = strings.TrimSpace(
		input.ObjectPath,
	)
	input.FileName = strings.TrimSpace(
		input.FileName,
	)
	input.ContentType = strings.TrimSpace(
		input.ContentType,
	)

	if input.URL == "" {
		return RegisterCreateOperationIconUploadInput{},
			fmt.Errorf(
				"%w: icon URL is required",
				ErrInvalidCreateOperation,
			)
	}

	if input.ObjectPath == "" {
		return RegisterCreateOperationIconUploadInput{},
			fmt.Errorf(
				"%w: icon objectPath is required",
				ErrInvalidCreateOperation,
			)
	}

	if input.FileName == "" {
		return RegisterCreateOperationIconUploadInput{},
			fmt.Errorf(
				"%w: icon fileName is required",
				ErrInvalidCreateOperation,
			)
	}

	if input.ContentType == "" {
		return RegisterCreateOperationIconUploadInput{},
			fmt.Errorf(
				"%w: icon contentType is required",
				ErrInvalidCreateOperation,
			)
	}

	if input.Size < 0 {
		return RegisterCreateOperationIconUploadInput{},
			fmt.Errorf(
				"%w: icon size must be >= 0",
				ErrInvalidCreateOperation,
			)
	}

	return input, nil
}

func normalizeCreateOperationContentUploadInput(
	input RegisterCreateOperationContentUploadInput,
) (
	RegisterCreateOperationContentUploadInput,
	error,
) {
	input.ContentID = strings.TrimSpace(
		input.ContentID,
	)
	input.URL = strings.TrimSpace(
		input.URL,
	)
	input.ObjectPath = strings.TrimSpace(
		input.ObjectPath,
	)
	input.Name = strings.TrimSpace(
		input.Name,
	)
	input.ContentType = strings.TrimSpace(
		input.ContentType,
	)

	if input.ContentID == "" {
		return RegisterCreateOperationContentUploadInput{},
			fmt.Errorf(
				"%w: contentId is required",
				ErrInvalidCreateOperation,
			)
	}

	if input.URL == "" {
		return RegisterCreateOperationContentUploadInput{},
			fmt.Errorf(
				"%w: content URL is required",
				ErrInvalidCreateOperation,
			)
	}

	if input.ObjectPath == "" {
		return RegisterCreateOperationContentUploadInput{},
			fmt.Errorf(
				"%w: content objectPath is required",
				ErrInvalidCreateOperation,
			)
	}

	if input.Name == "" {
		return RegisterCreateOperationContentUploadInput{},
			fmt.Errorf(
				"%w: content name is required",
				ErrInvalidCreateOperation,
			)
	}

	if input.ContentType == "" {
		return RegisterCreateOperationContentUploadInput{},
			fmt.Errorf(
				"%w: content contentType is required",
				ErrInvalidCreateOperation,
			)
	}

	if input.Size < 0 {
		return RegisterCreateOperationContentUploadInput{},
			fmt.Errorf(
				"%w: content size must be >= 0",
				ErrInvalidCreateOperation,
			)
	}

	return input, nil
}

func sameExpectedIconMetadata(
	expected CreateOperationIcon,
	actual RegisterCreateOperationIconUploadInput,
) bool {
	return expected.FileName == actual.FileName &&
		expected.ContentType == actual.ContentType &&
		expected.Size == actual.Size
}

func sameRegisteredIconUpload(
	existing CreateOperationIcon,
	actual RegisterCreateOperationIconUploadInput,
) bool {
	return sameExpectedIconMetadata(
		existing,
		actual,
	) &&
		existing.URL == actual.URL &&
		existing.ObjectPath == actual.ObjectPath
}

func sameExpectedContentMetadata(
	expected CreateOperationContent,
	actual RegisterCreateOperationContentUploadInput,
) bool {
	return expected.ID == actual.ContentID &&
		expected.Name == actual.Name &&
		expected.ContentType == actual.ContentType &&
		expected.Size == actual.Size
}

func sameRegisteredContentUpload(
	existing CreateOperationContent,
	actual RegisterCreateOperationContentUploadInput,
) bool {
	return sameExpectedContentMetadata(
		existing,
		actual,
	) &&
		existing.URL == actual.URL &&
		existing.ObjectPath == actual.ObjectPath
}

func validateCreateOperationID(
	fieldName string,
	value string,
) error {
	value = strings.TrimSpace(
		value,
	)

	if value == "" {
		return fmt.Errorf(
			"%w: %s is required",
			ErrInvalidCreateOperation,
			fieldName,
		)
	}

	if len(value) > 512 {
		return fmt.Errorf(
			"%w: %s must not exceed 512 characters",
			ErrInvalidCreateOperation,
			fieldName,
		)
	}

	if strings.Contains(
		value,
		"/",
	) ||
		strings.Contains(
			value,
			"://",
		) ||
		strings.ContainsAny(
			value,
			"\r\n\x00",
		) {
		return fmt.Errorf(
			"%w: %s is invalid",
			ErrInvalidCreateOperation,
			fieldName,
		)
	}

	return nil
}

func validateCreateOperationIdempotencyKey(
	value string,
) error {
	value = strings.TrimSpace(
		value,
	)

	if value == "" {
		return fmt.Errorf(
			"%w: idempotencyKey is required",
			ErrInvalidCreateOperation,
		)
	}

	if len(value) > 512 {
		return fmt.Errorf(
			"%w: idempotencyKey must not exceed 512 characters",
			ErrInvalidCreateOperation,
		)
	}

	if strings.ContainsAny(
		value,
		"\r\n\x00",
	) {
		return fmt.Errorf(
			"%w: idempotencyKey contains invalid characters",
			ErrInvalidCreateOperation,
		)
	}

	return nil
}

func cloneCreateOperationIcon(
	icon *CreateOperationIcon,
) *CreateOperationIcon {
	if icon == nil {
		return nil
	}

	cloned := *icon

	if icon.UploadedAt != nil {
		uploadedAt := icon.UploadedAt.UTC()
		cloned.UploadedAt = &uploadedAt
	}

	return &cloned
}

func cloneCreateOperationContents(
	contents []CreateOperationContent,
) []CreateOperationContent {
	if len(contents) == 0 {
		return []CreateOperationContent{}
	}

	cloned := make(
		[]CreateOperationContent,
		len(contents),
	)

	copy(
		cloned,
		contents,
	)

	for i := range cloned {
		if contents[i].UploadedAt == nil {
			continue
		}

		uploadedAt := contents[i].UploadedAt.UTC()
		cloned[i].UploadedAt = &uploadedAt
	}

	return cloned
}

func createOperationErrorMessage(
	err error,
) string {
	if err == nil {
		return "unknown error"
	}

	message := strings.TrimSpace(
		err.Error(),
	)
	if message == "" {
		return "unknown error"
	}

	return message
}

func createOperationTimePointer(
	value time.Time,
) *time.Time {
	value = value.UTC()
	return &value
}

func (o *CreateOperation) touch(
	now time.Time,
) {
	if o == nil {
		return
	}

	o.UpdatedAt = now.UTC()
}
