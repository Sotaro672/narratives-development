// backend/internal/application/usecase/list_save_operation_usecase.go
package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	listdom "narratives/internal/domain/list"
	"strings"
	"time"
)

type ListSaveOperationStorage interface {
	Exists(ctx context.Context, storagePath string) (bool, error)
	Delete(ctx context.Context, storagePath string) error
}
type ListSaveOperationRetryQueue interface {
	EnqueueRetry(ctx context.Context, operationID string, scheduledAt time.Time) error
}
type ListSaveOperationUsecase struct {
	listRepo         listdom.Repository
	imageRepo        listdom.ImageRepository
	operationRepo    listdom.SaveOperationRepository
	storage          ListSaveOperationStorage
	retryQueue       ListSaveOperationRetryQueue
	now              func() time.Time
	isRetryableError func(error) bool
}
type NewListSaveOperationUsecaseParams struct {
	ListRepository      listdom.Repository
	ImageRepository     listdom.ImageRepository
	OperationRepository listdom.SaveOperationRepository
	Storage             ListSaveOperationStorage
	RetryQueue          ListSaveOperationRetryQueue
	Now                 func() time.Time
	IsRetryableError    func(error) bool
}
type StartListSaveOperationInput struct {
	OperationID    string
	IdempotencyKey string
	ListID         string
	Type           listdom.SaveOperationType
	TargetList     listdom.List
	NewImages      []listdom.SaveOperationImage
	DeleteImageIDs []string
	PrimaryImageID *string
	MaxRetries     int
}

func NewListSaveOperationUsecase(p NewListSaveOperationUsecaseParams) *ListSaveOperationUsecase {
	now := p.Now
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}
	isRetryableError := p.IsRetryableError
	if isRetryableError == nil {
		isRetryableError = defaultListSaveOperationRetryableError
	}
	return &ListSaveOperationUsecase{
		listRepo:         p.ListRepository,
		imageRepo:        p.ImageRepository,
		operationRepo:    p.OperationRepository,
		storage:          p.Storage,
		retryQueue:       p.RetryQueue,
		now:              now,
		isRetryableError: isRetryableError,
	}
}
func (uc *ListSaveOperationUsecase) Start(ctx context.Context, input StartListSaveOperationInput) (listdom.SaveOperation, error) {
	if err := uc.validateDependencies(); err != nil {
		return listdom.SaveOperation{}, err
	}
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.IdempotencyKey == "" {
		return listdom.SaveOperation{}, fmt.Errorf("%w: idempotencyKey is required", listdom.ErrInvalidSaveOperation)
	}
	existing, err := uc.operationRepo.GetByIdempotencyKey(ctx, input.IdempotencyKey)
	if err == nil {
		return uc.executeLoaded(ctx, existing)
	}
	if !errors.Is(err, listdom.ErrSaveOperationNotFound) {
		return listdom.SaveOperation{}, err
	}
	now := uc.currentTime()
	operationID := strings.TrimSpace(input.OperationID)
	if operationID == "" {
		operationID, err = generateSaveOperationID("lso")
		if err != nil {
			return listdom.SaveOperation{}, err
		}
	}
	listID := strings.TrimSpace(input.ListID)
	if listID == "" {
		listID = strings.TrimSpace(input.TargetList.ID)
	}
	if listID == "" && input.Type == listdom.SaveOperationTypeCreate {
		listID, err = generateSaveOperationID("list")
		if err != nil {
			return listdom.SaveOperation{}, err
		}
	}
	if listID == "" {
		return listdom.SaveOperation{}, fmt.Errorf("%w: listId is required", listdom.ErrInvalidSaveOperation)
	}
	target, previousList, previousImages, previousPrimaryImageID, err := uc.preparePayloadState(ctx, input, listID, now)
	if err != nil {
		return listdom.SaveOperation{}, err
	}
	newImages, err := normalizeSaveOperationImages(input.NewImages)
	if err != nil {
		return listdom.SaveOperation{}, err
	}
	deleteImageIDs := normalizeSaveOperationImageIDs(input.DeleteImageIDs)
	primaryImageID := previousPrimaryImageID
	if input.PrimaryImageID != nil {
		primaryImageID = strings.TrimSpace(*input.PrimaryImageID)
	}
	payload := listdom.SaveOperationPayload{
		TargetList:             target,
		PreviousList:           previousList,
		NewImages:              newImages,
		DeleteImageIDs:         deleteImageIDs,
		PreviousImages:         previousImages,
		PrimaryImageID:         primaryImageID,
		PreviousPrimaryImageID: previousPrimaryImageID,
	}
	operation, err := listdom.NewSaveOperation(listdom.NewSaveOperationInput{
		ID:             operationID,
		IdempotencyKey: input.IdempotencyKey,
		ListID:         listID,
		Type:           input.Type,
		Payload:        payload,
		MaxRetries:     input.MaxRetries,
	}, now)
	if err != nil {
		return listdom.SaveOperation{}, err
	}
	created, err := uc.operationRepo.Create(ctx, operation)
	if err != nil {
		if errors.Is(err, listdom.ErrSaveOperationIdempotencyConflict) {
			existing, getErr := uc.operationRepo.GetByIdempotencyKey(ctx, input.IdempotencyKey)
			if getErr == nil {
				return uc.executeLoaded(ctx, existing)
			}
			return listdom.SaveOperation{}, errors.Join(err, getErr)
		}
		return listdom.SaveOperation{}, err
	}
	return uc.executeLoaded(ctx, created)
}
func (uc *ListSaveOperationUsecase) Get(ctx context.Context, operationID string) (listdom.SaveOperation, error) {
	if uc == nil || uc.operationRepo == nil {
		return listdom.SaveOperation{}, errors.New("list save operation repository is nil")
	}
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return listdom.SaveOperation{}, fmt.Errorf("%w: operationId is required", listdom.ErrInvalidSaveOperation)
	}
	return uc.operationRepo.GetByID(ctx, operationID)
}
func (uc *ListSaveOperationUsecase) Execute(ctx context.Context, operationID string) (listdom.SaveOperation, error) {
	if err := uc.validateDependencies(); err != nil {
		return listdom.SaveOperation{}, err
	}
	operation, err := uc.Get(ctx, operationID)
	if err != nil {
		return listdom.SaveOperation{}, err
	}
	return uc.executeLoaded(ctx, operation)
}
func (uc *ListSaveOperationUsecase) Retry(ctx context.Context, operationID string) (listdom.SaveOperation, error) {
	if err := uc.validateDependencies(); err != nil {
		return listdom.SaveOperation{}, err
	}
	operation, err := uc.Get(ctx, operationID)
	if err != nil {
		return listdom.SaveOperation{}, err
	}
	if !operation.CanRetry() {
		return operation, listdom.ErrSaveOperationNotRetryable
	}
	operation, err = uc.persistMutation(ctx, operation, func(value *listdom.SaveOperation) error {
		return value.StartRetry(uc.currentTime())
	})
	if err != nil {
		return operation, err
	}
	return uc.executeLoaded(ctx, operation)
}
func (uc *ListSaveOperationUsecase) Compensate(ctx context.Context, operationID string) (listdom.SaveOperation, error) {
	if err := uc.validateDependencies(); err != nil {
		return listdom.SaveOperation{}, err
	}
	operation, err := uc.Get(ctx, operationID)
	if err != nil {
		return listdom.SaveOperation{}, err
	}
	return uc.compensateLoaded(ctx, operation)
}
func (uc *ListSaveOperationUsecase) executeLoaded(ctx context.Context, operation listdom.SaveOperation) (listdom.SaveOperation, error) {
	switch operation.Status {
	case listdom.SaveOperationStatusCompleted, listdom.SaveOperationStatusCompensated, listdom.SaveOperationStatusFailedFatal, listdom.SaveOperationStatusFailedRetryable:
		return operation, nil
	case listdom.SaveOperationStatusCompensating:
		return uc.compensateLoaded(ctx, operation)
	}
	for {
		switch operation.Status {
		case listdom.SaveOperationStatusPending:
			var err error
			if len(operation.Payload.NewImages) > 0 {
				operation, err = uc.persistMutation(ctx, operation, func(value *listdom.SaveOperation) error {
					return value.StartUploading(uc.currentTime())
				})
			} else if len(operation.Payload.DeleteImageIDs) > 0 {
				operation, err = uc.persistMutation(ctx, operation, func(value *listdom.SaveOperation) error {
					return value.StartDeletingImages(uc.currentTime())
				})
			} else {
				operation, err = uc.persistMutation(ctx, operation, func(value *listdom.SaveOperation) error {
					return value.StartUpdatingList(uc.currentTime())
				})
			}
			if err != nil {
				return operation, err
			}
		case listdom.SaveOperationStatusUploading:
			var err error
			operation, err = uc.acknowledgeUploadedImages(ctx, operation)
			if err != nil {
				return operation, err
			}
			operation, err = uc.persistMutation(ctx, operation, func(value *listdom.SaveOperation) error {
				return value.StartRegisteringImages(uc.currentTime())
			})
			if err != nil {
				return operation, err
			}
		case listdom.SaveOperationStatusRegisteringImages:
			var err error
			operation, err = uc.registerImages(ctx, operation)
			if err != nil {
				return operation, err
			}
			if len(operation.Payload.DeleteImageIDs) > 0 {
				operation, err = uc.persistMutation(ctx, operation, func(value *listdom.SaveOperation) error {
					return value.StartDeletingImages(uc.currentTime())
				})
			} else {
				operation, err = uc.persistMutation(ctx, operation, func(value *listdom.SaveOperation) error {
					return value.StartUpdatingList(uc.currentTime())
				})
			}
			if err != nil {
				return operation, err
			}
		case listdom.SaveOperationStatusDeletingImages:
			var err error
			operation, err = uc.deleteImages(ctx, operation)
			if err != nil {
				return operation, err
			}
			operation, err = uc.persistMutation(ctx, operation, func(value *listdom.SaveOperation) error {
				return value.StartUpdatingList(uc.currentTime())
			})
			if err != nil {
				return operation, err
			}
		case listdom.SaveOperationStatusUpdatingList:
			if !operation.Progress.ListUpdated {
				if err := uc.applyTargetList(ctx, operation); err != nil {
					return uc.failExecution(ctx, operation, err)
				}
				var err error
				operation, err = uc.persistMutation(ctx, operation, func(value *listdom.SaveOperation) error {
					return value.MarkListUpdated(uc.currentTime())
				})
				if err != nil {
					return operation, err
				}
			}
			var err error
			operation, err = uc.persistMutation(ctx, operation, func(value *listdom.SaveOperation) error {
				return value.StartSettingPrimary(uc.currentTime())
			})
			if err != nil {
				return operation, err
			}
		case listdom.SaveOperationStatusSettingPrimary:
			if !operation.Progress.PrimaryImageUpdated {
				if err := uc.applyPrimaryImage(ctx, operation); err != nil {
					return uc.failExecution(ctx, operation, err)
				}
				var err error
				operation, err = uc.persistMutation(ctx, operation, func(value *listdom.SaveOperation) error {
					return value.MarkPrimaryImageUpdated(uc.currentTime())
				})
				if err != nil {
					return operation, err
				}
			}
			var err error
			operation, err = uc.persistMutation(ctx, operation, func(value *listdom.SaveOperation) error {
				return value.Complete(uc.currentTime())
			})
			return operation, err
		case listdom.SaveOperationStatusCompensating:
			return uc.compensateLoaded(ctx, operation)
		case listdom.SaveOperationStatusCompleted, listdom.SaveOperationStatusCompensated, listdom.SaveOperationStatusFailedRetryable, listdom.SaveOperationStatusFailedFatal:
			return operation, nil
		default:
			return operation, fmt.Errorf("%w: unsupported status %q", listdom.ErrInvalidSaveOperation, operation.Status)
		}
	}
}
func (uc *ListSaveOperationUsecase) acknowledgeUploadedImages(ctx context.Context, operation listdom.SaveOperation) (listdom.SaveOperation, error) {
	if uc.storage == nil {
		return uc.failExecution(ctx, operation, errors.New("list save operation storage is nil"))
	}
	for _, image := range operation.Payload.NewImages {
		if operation.IsImageUploaded(image.ImageID) {
			continue
		}
		exists, err := uc.storage.Exists(ctx, image.StoragePath)
		if err != nil {
			return uc.failExecution(ctx, operation, err)
		}
		if !exists {
			return uc.failExecution(ctx, operation, fmt.Errorf("list save operation storage object not found: %s", image.StoragePath))
		}
		operation, err = uc.persistMutation(ctx, operation, func(value *listdom.SaveOperation) error {
			return value.MarkImageUploaded(image.ImageID, uc.currentTime())
		})
		if err != nil {
			return operation, err
		}
	}
	return operation, nil
}
func (uc *ListSaveOperationUsecase) registerImages(ctx context.Context, operation listdom.SaveOperation) (listdom.SaveOperation, error) {
	createdBy := saveOperationActor(operation.Payload.TargetList)
	if createdBy == "" {
		return uc.failExecution(ctx, operation, listdom.ErrInvalidListImageCreatedBy)
	}
	for _, image := range operation.Payload.NewImages {
		if operation.IsImageRegistered(image.ImageID) {
			continue
		}
		_, err := uc.imageRepo.Create(ctx, listdom.ListImage{
			ID:           image.ImageID,
			ListID:       operation.ListID,
			URL:          image.URL,
			DisplayOrder: image.DisplayOrder,
			CreatedAt:    operation.CreatedAt,
			CreatedBy:    createdBy,
		})
		if err != nil {
			return uc.failExecution(ctx, operation, err)
		}
		operation, err = uc.persistMutation(ctx, operation, func(value *listdom.SaveOperation) error {
			return value.MarkImageRegistered(image.ImageID, uc.currentTime())
		})
		if err != nil {
			return operation, err
		}
	}
	return operation, nil
}
func (uc *ListSaveOperationUsecase) deleteImages(ctx context.Context, operation listdom.SaveOperation) (listdom.SaveOperation, error) {
	for _, imageID := range operation.Payload.DeleteImageIDs {
		if operation.IsImageDeleted(imageID) {
			continue
		}
		err := uc.imageRepo.Delete(ctx, operation.ListID, imageID)
		if err != nil && !errors.Is(err, listdom.ErrNotFound) {
			return uc.failExecution(ctx, operation, err)
		}
		operation, err = uc.persistMutation(ctx, operation, func(value *listdom.SaveOperation) error {
			return value.MarkImageDeleted(imageID, uc.currentTime())
		})
		if err != nil {
			return operation, err
		}
	}
	return operation, nil
}
func (uc *ListSaveOperationUsecase) applyTargetList(ctx context.Context, operation listdom.SaveOperation) error {
	target := operation.Payload.TargetList
	target.ID = operation.ListID
	target.ImageID = operation.Payload.PreviousPrimaryImageID
	now := uc.currentTime()
	target.UpdatedAt = &now
	if operation.Type == listdom.SaveOperationTypeUpdate {
		_, err := uc.listRepo.Update(ctx, operation.ListID, target)
		return err
	}
	existing, err := uc.listRepo.GetByID(ctx, operation.ListID)
	if err == nil {
		if !sameCreatedListIdentity(existing, target) {
			return listdom.ErrConflict
		}
		_, err = uc.listRepo.Update(ctx, operation.ListID, target)
		return err
	}
	if !errors.Is(err, listdom.ErrNotFound) {
		return err
	}
	_, err = uc.listRepo.Create(ctx, target)
	if err == nil {
		return nil
	}
	if !errors.Is(err, listdom.ErrConflict) {
		return err
	}
	existing, getErr := uc.listRepo.GetByID(ctx, operation.ListID)
	if getErr != nil {
		return err
	}
	if !sameCreatedListIdentity(existing, target) {
		return listdom.ErrConflict
	}
	_, updateErr := uc.listRepo.Update(ctx, operation.ListID, target)
	return updateErr
}
func (uc *ListSaveOperationUsecase) applyPrimaryImage(ctx context.Context, operation listdom.SaveOperation) error {
	primaryImageID := strings.TrimSpace(operation.Payload.PrimaryImageID)
	if primaryImageID != "" {
		image, err := uc.imageRepo.GetByID(ctx, operation.ListID, primaryImageID)
		if err != nil {
			return err
		}
		if image.ListID != operation.ListID {
			return errors.New("list: primary image belongs to another list")
		}
	}
	item, err := uc.listRepo.GetByID(ctx, operation.ListID)
	if err != nil {
		return err
	}
	item.ImageID = primaryImageID
	now := uc.currentTime()
	item.UpdatedAt = &now
	if operation.Payload.TargetList.UpdatedBy != nil {
		updatedBy := strings.TrimSpace(*operation.Payload.TargetList.UpdatedBy)
		if updatedBy != "" {
			item.UpdatedBy = &updatedBy
		}
	}
	_, err = uc.listRepo.Update(ctx, operation.ListID, item)
	return err
}
func (uc *ListSaveOperationUsecase) compensateLoaded(ctx context.Context, operation listdom.SaveOperation) (listdom.SaveOperation, error) {
	if operation.Status == listdom.SaveOperationStatusCompensated {
		return operation, nil
	}
	if operation.Status != listdom.SaveOperationStatusCompensating {
		if !operation.CanCompensate() {
			return operation, listdom.ErrSaveOperationNotCompensatable
		}
		var err error
		operation, err = uc.persistMutation(ctx, operation, func(value *listdom.SaveOperation) error {
			return value.BeginCompensation(uc.currentTime())
		})
		if err != nil {
			return operation, err
		}
	}
	for _, image := range operation.Payload.NewImages {
		if !operation.IsImageRegistered(image.ImageID) {
			continue
		}
		err := uc.imageRepo.Delete(ctx, operation.ListID, image.ImageID)
		if err != nil && !errors.Is(err, listdom.ErrNotFound) {
			return uc.failCompensation(ctx, operation, err)
		}
	}
	for _, image := range operation.Payload.PreviousImages {
		if !operation.IsImageDeleted(image.ID) {
			continue
		}
		_, err := uc.imageRepo.Create(ctx, image)
		if err != nil {
			return uc.failCompensation(ctx, operation, err)
		}
	}
	if operation.Progress.ListUpdated {
		if operation.Type == listdom.SaveOperationTypeCreate {
			err := uc.listRepo.Delete(ctx, operation.ListID)
			if err != nil && !errors.Is(err, listdom.ErrNotFound) {
				return uc.failCompensation(ctx, operation, err)
			}
		} else if operation.Payload.PreviousList != nil {
			previous := *operation.Payload.PreviousList
			_, err := uc.listRepo.Update(ctx, operation.ListID, previous)
			if err != nil {
				return uc.failCompensation(ctx, operation, err)
			}
		}
	}
	for _, image := range operation.Payload.NewImages {
		if !operation.IsImageUploaded(image.ImageID) {
			continue
		}
		storagePath := strings.TrimSpace(image.StoragePath)
		if containsSaveOperationString(operation.Progress.CompensatedStoragePaths, storagePath) {
			continue
		}
		if uc.storage == nil {
			return uc.failCompensation(ctx, operation, errors.New("list save operation storage is nil"))
		}
		if err := uc.storage.Delete(ctx, storagePath); err != nil {
			return uc.failCompensation(ctx, operation, err)
		}
		var err error
		operation, err = uc.persistMutation(ctx, operation, func(value *listdom.SaveOperation) error {
			return value.MarkStoragePathCompensated(storagePath, uc.currentTime())
		})
		if err != nil {
			return operation, err
		}
	}
	var err error
	operation, err = uc.persistMutation(ctx, operation, func(value *listdom.SaveOperation) error {
		return value.CompleteCompensation(uc.currentTime())
	})
	return operation, err
}
func (uc *ListSaveOperationUsecase) failExecution(ctx context.Context, operation listdom.SaveOperation, cause error) (listdom.SaveOperation, error) {
	retryable := uc.isRetryableError(cause)
	updated, persistErr := uc.persistMutation(ctx, operation, func(value *listdom.SaveOperation) error {
		if retryable {
			return value.FailRetryable(cause, uc.currentTime())
		}
		return value.FailFatal(cause, uc.currentTime())
	})
	if persistErr != nil {
		return operation, errors.Join(cause, persistErr)
	}
	if updated.Status == listdom.SaveOperationStatusFailedRetryable {
		if enqueueErr := uc.enqueueRetry(ctx, updated); enqueueErr != nil {
			return updated, errors.Join(cause, enqueueErr)
		}
	}
	if updated.Status == listdom.SaveOperationStatusFailedFatal && updated.CanCompensate() {
		compensated, compensationErr := uc.compensateLoaded(ctx, updated)
		if compensationErr != nil {
			return compensated, errors.Join(cause, compensationErr)
		}
		return compensated, cause
	}
	return updated, cause
}
func (uc *ListSaveOperationUsecase) failCompensation(ctx context.Context, operation listdom.SaveOperation, cause error) (listdom.SaveOperation, error) {
	updated, persistErr := uc.persistMutation(ctx, operation, func(value *listdom.SaveOperation) error {
		return value.FailRetryable(cause, uc.currentTime())
	})
	if persistErr != nil {
		return operation, errors.Join(cause, persistErr)
	}
	if updated.Status == listdom.SaveOperationStatusFailedRetryable {
		if enqueueErr := uc.enqueueRetry(ctx, updated); enqueueErr != nil {
			return updated, errors.Join(cause, enqueueErr)
		}
	}
	return updated, cause
}
func (uc *ListSaveOperationUsecase) enqueueRetry(ctx context.Context, operation listdom.SaveOperation) error {
	if operation.Status != listdom.SaveOperationStatusFailedRetryable || uc.retryQueue == nil {
		return nil
	}
	scheduledAt := uc.currentTime().Add(listSaveOperationRetryDelay(operation.RetryCount))
	return uc.retryQueue.EnqueueRetry(ctx, operation.ID, scheduledAt)
}
func (uc *ListSaveOperationUsecase) persistMutation(ctx context.Context, operation listdom.SaveOperation, mutate func(*listdom.SaveOperation) error) (listdom.SaveOperation, error) {
	expectedVersion := operation.Version
	if err := mutate(&operation); err != nil {
		return operation, err
	}
	updated, err := uc.operationRepo.Update(ctx, operation, expectedVersion)
	if err != nil {
		return operation, err
	}
	return updated, nil
}
func (uc *ListSaveOperationUsecase) preparePayloadState(ctx context.Context, input StartListSaveOperationInput, listID string, now time.Time) (listdom.List, *listdom.List, []listdom.ListImage, string, error) {
	target := input.TargetList
	target.ID = listID
	switch input.Type {
	case listdom.SaveOperationTypeCreate:
		if target.CreatedAt.IsZero() {
			target.CreatedAt = now
		} else {
			target.CreatedAt = target.CreatedAt.UTC()
		}
		target.CreatedBy = strings.TrimSpace(target.CreatedBy)
		if target.CreatedBy == "" {
			return listdom.List{}, nil, nil, "", listdom.ErrInvalidListImageCreatedBy
		}
		if target.ReadableID == "" {
			target.ReadableID = generateReadableID(listID, target.CreatedAt)
		}
		target.ImageID = ""
		target.UpdatedAt = &now
		return target, nil, []listdom.ListImage{}, "", nil
	case listdom.SaveOperationTypeUpdate:
		current, err := uc.listRepo.GetByID(ctx, listID)
		if err != nil {
			return listdom.List{}, nil, nil, "", err
		}
		images, err := uc.imageRepo.ListByListID(ctx, listID)
		if err != nil {
			return listdom.List{}, nil, nil, "", err
		}
		target.ID = listID
		target.InventoryID = current.InventoryID
		target.CreatedAt = current.CreatedAt
		target.CreatedBy = current.CreatedBy
		if target.ReadableID == "" {
			target.ReadableID = current.ReadableID
		}
		target.ImageID = current.ImageID
		target.UpdatedAt = &now
		previous := current
		return target, &previous, images, current.ImageID, nil
	default:
		return listdom.List{}, nil, nil, "", fmt.Errorf("%w: invalid operation type %q", listdom.ErrInvalidSaveOperation, input.Type)
	}
}
func (uc *ListSaveOperationUsecase) validateDependencies() error {
	if uc == nil {
		return errors.New("list save operation usecase is nil")
	}
	if uc.listRepo == nil {
		return errors.New("list repository is nil")
	}
	if uc.imageRepo == nil {
		return errors.New("list image repository is nil")
	}
	if uc.operationRepo == nil {
		return errors.New("list save operation repository is nil")
	}
	return nil
}
func (uc *ListSaveOperationUsecase) currentTime() time.Time {
	if uc == nil || uc.now == nil {
		return time.Now().UTC()
	}
	return uc.now().UTC()
}
func normalizeSaveOperationImages(images []listdom.SaveOperationImage) ([]listdom.SaveOperationImage, error) {
	out := make([]listdom.SaveOperationImage, 0, len(images))
	for index, image := range images {
		image.ImageID = strings.TrimSpace(image.ImageID)
		image.URL = strings.TrimSpace(image.URL)
		image.StoragePath = strings.TrimSpace(image.StoragePath)
		if image.ImageID == "" {
			return nil, fmt.Errorf("%w: newImages[%d].imageId is required", listdom.ErrInvalidSaveOperation, index)
		}
		if strings.Contains(image.ImageID, "/") || strings.Contains(image.ImageID, "://") {
			return nil, fmt.Errorf("%w: newImages[%d].imageId is invalid", listdom.ErrInvalidSaveOperation, index)
		}
		if image.URL == "" {
			return nil, fmt.Errorf("%w: newImages[%d].url is required", listdom.ErrInvalidSaveOperation, index)
		}
		if image.DisplayOrder < 0 {
			return nil, fmt.Errorf("%w: newImages[%d].displayOrder must be >= 0", listdom.ErrInvalidSaveOperation, index)
		}
		out = append(out, image)
	}
	return out, nil
}
func normalizeSaveOperationImageIDs(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, strings.TrimSpace(value))
	}
	return out
}
func saveOperationActor(item listdom.List) string {
	if item.UpdatedBy != nil {
		if value := strings.TrimSpace(*item.UpdatedBy); value != "" {
			return value
		}
	}
	return strings.TrimSpace(item.CreatedBy)
}
func sameCreatedListIdentity(existing listdom.List, target listdom.List) bool {
	return existing.ID == target.ID && existing.InventoryID == target.InventoryID && existing.CreatedBy == target.CreatedBy
}
func containsSaveOperationString(values []string, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}
func generateSaveOperationID(prefix string) (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate save operation id: %w", err)
	}
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return hex.EncodeToString(value), nil
	}
	return prefix + "_" + hex.EncodeToString(value), nil
}
func listSaveOperationRetryDelay(retryCount int) time.Duration {
	switch {
	case retryCount <= 0:
		return 30 * time.Second
	case retryCount == 1:
		return 2 * time.Minute
	default:
		return 10 * time.Minute
	}
}
func defaultListSaveOperationRetryableError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || errors.Is(err, listdom.ErrSaveOperationConflict)
}
