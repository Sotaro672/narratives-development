package usecase

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	listdom "narratives/internal/domain/list"
)

type ListReader interface {
	GetByID(ctx context.Context, id string) (listdom.List, error)
}

type ListLister interface {
	List(
		ctx context.Context,
		filter listdom.Filter,
		sort listdom.Sort,
		page listdom.Page,
	) (listdom.PageResult[listdom.List], error)

	Count(ctx context.Context, filter listdom.Filter) (int, error)
}

type ListCreator interface {
	Create(ctx context.Context, item listdom.List) (listdom.List, error)
}

type ListUpdater interface {
	Update(ctx context.Context, item listdom.List) (listdom.List, error)
}

type ListPatchUpdater interface {
	Update(ctx context.Context, id string, patch listdom.ListPatch) (listdom.List, error)
}

type ListDeleter interface {
	Delete(ctx context.Context, id string) error
}

type ListPrimaryImageSetter interface {
	SetPrimaryImageID(ctx context.Context, listID string, imageID string, now time.Time) error
	SetPrimaryImageIfEmpty(ctx context.Context, listID string, imageID string, now time.Time) error
}

type ListImageReader interface {
	ListByListID(ctx context.Context, listID string) ([]listdom.ListImage, error)
}

type ListImageRecordRepository interface {
	Upsert(ctx context.Context, img listdom.ListImage) (listdom.ListImage, error)
	ListByListID(ctx context.Context, listID string) ([]listdom.ListImage, error)
	GetByListIDAndID(ctx context.Context, listID string, imageID string) (listdom.ListImage, error)
	Delete(ctx context.Context, listID string, imageID string) error
}

type ListImageRecordByIDReader interface {
	GetByListIDAndID(ctx context.Context, listID string, imageID string) (listdom.ListImage, error)
}

type ListImageRecordDeleter interface {
	Delete(ctx context.Context, listID string, imageID string) error
}

type ListAggregate struct {
	List   listdom.List        `json:"list"`
	Images []listdom.ListImage `json:"images"`
}

type ListUsecase struct {
	listReader       ListReader
	listLister       ListLister
	listCreator      ListCreator
	listUpdater      ListUpdater
	listPatchUpdater ListPatchUpdater
	listDeleter      ListDeleter

	imageReader ListImageReader

	listImageRecordRepo    ListImageRecordRepository
	listPrimaryImageSetter ListPrimaryImageSetter
}

func NewListUsecase(
	listReader ListReader,
	listCreator ListCreator,
	listPatchUpdater ListPatchUpdater,
	imageReader ListImageReader,
	imageByIDReader ListImageRecordByIDReader,
) *ListUsecase {
	uc := &ListUsecase{
		listReader:             listReader,
		listLister:             nil,
		listCreator:            listCreator,
		listUpdater:            nil,
		listPatchUpdater:       listPatchUpdater,
		listDeleter:            nil,
		imageReader:            imageReader,
		listImageRecordRepo:    nil,
		listPrimaryImageSetter: nil,
	}

	if listReader != nil {
		if lister, ok := any(listReader).(ListLister); ok {
			uc.listLister = lister
		}

		if updater, ok := any(listReader).(ListUpdater); ok {
			uc.listUpdater = updater
		}

		if deleter, ok := any(listReader).(ListDeleter); ok {
			uc.listDeleter = deleter
		}

		if setter, ok := any(listReader).(ListPrimaryImageSetter); ok {
			uc.listPrimaryImageSetter = setter
		}
	}

	if uc.listLister == nil && listCreator != nil {
		if lister, ok := any(listCreator).(ListLister); ok {
			uc.listLister = lister
		}
	}

	if uc.listUpdater == nil && listCreator != nil {
		if updater, ok := any(listCreator).(ListUpdater); ok {
			uc.listUpdater = updater
		}
	}

	if uc.listDeleter == nil && listCreator != nil {
		if deleter, ok := any(listCreator).(ListDeleter); ok {
			uc.listDeleter = deleter
		}
	}

	if uc.listPrimaryImageSetter == nil && listPatchUpdater != nil {
		if setter, ok := any(listPatchUpdater).(ListPrimaryImageSetter); ok {
			uc.listPrimaryImageSetter = setter
		}
	}

	if uc.listDeleter == nil && listPatchUpdater != nil {
		if deleter, ok := any(listPatchUpdater).(ListDeleter); ok {
			uc.listDeleter = deleter
		}
	}

	if imageReader != nil {
		if repo, ok := any(imageReader).(ListImageRecordRepository); ok {
			uc.listImageRecordRepo = repo
		}
	}

	if uc.listImageRecordRepo == nil && imageByIDReader != nil {
		if repo, ok := any(imageByIDReader).(ListImageRecordRepository); ok {
			uc.listImageRecordRepo = repo
		}
	}

	return uc
}

func (uc *ListUsecase) WithListImageRecordRepo(repo ListImageRecordRepository) *ListUsecase {
	if uc == nil {
		return nil
	}

	uc.listImageRecordRepo = repo
	return uc
}

func (uc *ListUsecase) WithListPrimaryImageSetter(setter ListPrimaryImageSetter) *ListUsecase {
	if uc == nil {
		return nil
	}

	uc.listPrimaryImageSetter = setter
	return uc
}

func (uc *ListUsecase) WithListDeleter(deleter ListDeleter) *ListUsecase {
	if uc == nil {
		return nil
	}

	uc.listDeleter = deleter
	return uc
}

func isImageURL(v string) bool {
	return strings.HasPrefix(v, "https://") ||
		strings.HasPrefix(v, "http://") ||
		strings.HasPrefix(v, "gs://")
}

func generateReadableID(listID string, createdAt time.Time) string {
	t := createdAt
	if t.IsZero() {
		t = time.Now().UTC()
	}

	date := t.UTC().Format("20060102")

	base := listID
	if base == "" {
		base = fmt.Sprintf("noid-%d", time.Now().UTC().UnixNano())
	}

	sum := sha1.Sum([]byte(base))
	hex6 := hex.EncodeToString(sum[:])
	if len(hex6) > 6 {
		hex6 = hex6[:6]
	}

	return fmt.Sprintf("L-%s-%s", date, hex6)
}

func (uc *ListUsecase) getPatchUpdater() ListPatchUpdater {
	if uc == nil {
		return nil
	}

	if uc.listReader != nil {
		if pu, ok := any(uc.listReader).(ListPatchUpdater); ok {
			return pu
		}
	}

	if uc.listCreator != nil {
		if pu, ok := any(uc.listCreator).(ListPatchUpdater); ok {
			return pu
		}
	}

	return nil
}

func buildPatchFromItem(item listdom.List) listdom.ListPatch {
	statusV := item.Status
	assigneeV := item.AssigneeID
	imageV := item.ImageID
	titleV := item.Title
	descV := item.Description

	var updatedByV *string
	if item.UpdatedBy != nil {
		v := *item.UpdatedBy
		if v != "" {
			updatedByV = &v
		}
	}

	now := time.Now().UTC()
	updatedAtV := now
	if item.UpdatedAt != nil && !item.UpdatedAt.IsZero() {
		updatedAtV = item.UpdatedAt.UTC()
	}

	var pricesPtr *[]listdom.ListPriceRow
	if item.Prices != nil {
		pv := item.Prices
		pricesPtr = &pv
	}

	var readableIDPtr *string
	if item.ReadableID != "" {
		v := item.ReadableID
		readableIDPtr = &v
	}

	return listdom.ListPatch{
		Status:      &statusV,
		AssigneeID:  &assigneeV,
		ImageID:     &imageV,
		Title:       &titleV,
		Description: &descV,
		ReadableID:  readableIDPtr,
		UpdatedBy:   updatedByV,
		UpdatedAt:   &updatedAtV,
		Prices:      pricesPtr,
	}
}

func ptrTime(t time.Time) *time.Time {
	tt := t
	return &tt
}

func (uc *ListUsecase) List(
	ctx context.Context,
	filter listdom.Filter,
	sort listdom.Sort,
	page listdom.Page,
) (listdom.PageResult[listdom.List], error) {
	if uc == nil || uc.listLister == nil {
		return listdom.PageResult[listdom.List]{}, ErrNotSupported("List.List")
	}

	return uc.listLister.List(ctx, filter, sort, page)
}

func (uc *ListUsecase) Count(
	ctx context.Context,
	filter listdom.Filter,
) (int, error) {
	if uc == nil || uc.listLister == nil {
		return 0, ErrNotSupported("List.Count")
	}

	return uc.listLister.Count(ctx, filter)
}

func (uc *ListUsecase) Create(
	ctx context.Context,
	item listdom.List,
) (listdom.List, error) {
	if uc == nil || uc.listCreator == nil {
		return listdom.List{}, ErrNotSupported("List.Create")
	}

	created, err := uc.listCreator.Create(ctx, item)
	if err != nil {
		return listdom.List{}, err
	}

	if created.ReadableID == "" {
		rid := generateReadableID(created.ID, created.CreatedAt)
		created.ReadableID = rid

		if pu := uc.getPatchUpdater(); pu != nil {
			now := time.Now().UTC()
			patch := listdom.ListPatch{
				ReadableID: &rid,
				UpdatedAt:  &now,
			}

			_, _ = pu.Update(ctx, created.ID, patch)
		}
	}

	return created, nil
}

func (uc *ListUsecase) Update(
	ctx context.Context,
	item listdom.List,
) (listdom.List, error) {
	id := item.ID
	if id == "" {
		return listdom.List{}, listdom.ErrInvalidID
	}

	patch := buildPatchFromItem(item)

	if uc != nil && uc.listReader != nil {
		if pu, ok := any(uc.listReader).(ListPatchUpdater); ok {
			return pu.Update(ctx, id, patch)
		}
	}

	if uc != nil && uc.listCreator != nil {
		if pu, ok := any(uc.listCreator).(ListPatchUpdater); ok {
			return pu.Update(ctx, id, patch)
		}
	}

	if uc == nil || uc.listUpdater == nil {
		return listdom.List{}, ErrNotSupported("List.Update")
	}

	return uc.listUpdater.Update(ctx, item)
}

func (uc *ListUsecase) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return listdom.ErrInvalidID
	}

	if uc == nil || uc.listDeleter == nil {
		return ErrNotSupported("List.Delete")
	}

	return uc.listDeleter.Delete(ctx, id)
}

func (uc *ListUsecase) GetByID(
	ctx context.Context,
	id string,
) (listdom.List, error) {
	if uc == nil || uc.listReader == nil {
		return listdom.List{}, ErrNotSupported("List.GetByID")
	}

	return uc.listReader.GetByID(ctx, id)
}

func (uc *ListUsecase) GetImages(
	ctx context.Context,
	listID string,
) ([]listdom.ListImage, error) {
	if uc == nil || uc.imageReader == nil {
		return []listdom.ListImage{}, nil
	}

	items, err := uc.imageReader.ListByListID(ctx, listID)
	if err != nil {
		return nil, err
	}

	if items == nil {
		return []listdom.ListImage{}, nil
	}

	return items, nil
}

func (uc *ListUsecase) GetAggregate(
	ctx context.Context,
	id string,
) (ListAggregate, error) {
	if uc == nil || uc.listReader == nil {
		return ListAggregate{}, ErrNotSupported("List.GetAggregate")
	}

	li, err := uc.listReader.GetByID(ctx, id)
	if err != nil {
		return ListAggregate{}, err
	}

	images := []listdom.ListImage{}

	if uc.imageReader != nil {
		items, err := uc.imageReader.ListByListID(ctx, id)
		if err != nil {
			return ListAggregate{}, err
		}

		if items != nil {
			images = items
		}
	}

	return ListAggregate{
		List:   li,
		Images: images,
	}, nil
}

func (uc *ListUsecase) SaveImage(
	ctx context.Context,
	img listdom.ListImage,
) (listdom.ListImage, error) {
	if uc == nil {
		return listdom.ListImage{}, ErrNotSupported("List.SaveImage")
	}

	if uc.listImageRecordRepo == nil {
		return listdom.ListImage{}, ErrNotSupported("List.SaveImage.RecordRepo")
	}

	img.ID = strings.TrimSpace(img.ID)
	img.ListID = strings.TrimSpace(img.ListID)
	img.URL = strings.TrimSpace(img.URL)
	img.CreatedBy = strings.TrimSpace(img.CreatedBy)

	if img.ListID == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageListID
	}

	if img.ID == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageID
	}

	if strings.Contains(img.ID, "/") || strings.Contains(img.ID, "://") {
		return listdom.ListImage{}, ErrInvalidArgument("invalid_image_id")
	}

	if img.DisplayOrder < 0 {
		img.DisplayOrder = 0
	}

	if img.CreatedAt.IsZero() {
		img.CreatedAt = time.Now().UTC()
	} else {
		img.CreatedAt = img.CreatedAt.UTC()
	}

	if err := img.Validate(); err != nil {
		return listdom.ListImage{}, err
	}

	saved, err := uc.listImageRecordRepo.Upsert(ctx, img)
	if err != nil {
		return listdom.ListImage{}, err
	}

	return saved, nil
}

func (uc *ListUsecase) DeleteImage(ctx context.Context, listID string, imageID string) error {
	listID = strings.TrimSpace(listID)
	imageID = strings.TrimSpace(imageID)

	if listID == "" {
		return listdom.ErrInvalidListImageListID
	}

	if imageID == "" {
		return listdom.ErrInvalidListImageID
	}

	if strings.Contains(imageID, "/") || strings.Contains(imageID, "://") {
		return ErrInvalidArgument("invalid_image_id")
	}

	if uc == nil {
		return ErrNotSupported("List.DeleteImage")
	}

	if uc.listImageRecordRepo == nil {
		return ErrNotSupported("List.DeleteImage.RecordRepo")
	}

	if deleter, ok := any(uc.listImageRecordRepo).(ListImageRecordDeleter); ok {
		if err := deleter.Delete(ctx, listID, imageID); err != nil {
			if !errors.Is(err, listdom.ErrListImageNotFound) {
				return err
			}
		}
	} else {
		return ErrNotSupported("List.DeleteImage.RecordRepo.Delete")
	}

	if uc.listReader != nil && uc.listPatchUpdater != nil {
		l, err := uc.listReader.GetByID(ctx, listID)
		if err == nil && l.ImageID == imageID {
			now := time.Now().UTC()
			empty := ""

			patch := listdom.ListPatch{
				ImageID:   &empty,
				UpdatedAt: &now,
			}

			_, _ = uc.listPatchUpdater.Update(ctx, listID, patch)
		}
	}

	return nil
}

func (uc *ListUsecase) SetPrimaryImage(
	ctx context.Context,
	listID string,
	imageID string,
	now time.Time,
	updatedBy *string,
) (listdom.List, error) {
	if uc == nil {
		return listdom.List{}, ErrNotSupported("List.SetPrimaryImage")
	}

	if uc.listPatchUpdater == nil {
		return listdom.List{}, ErrNotSupported("List.SetPrimaryImage")
	}

	lid := strings.TrimSpace(listID)
	iid := strings.TrimSpace(imageID)

	if lid == "" {
		return listdom.List{}, listdom.ErrInvalidID
	}

	if iid == "" {
		return listdom.List{}, listdom.ErrEmptyImageID
	}

	if strings.Contains(iid, "/") || strings.Contains(iid, "://") {
		return listdom.List{}, listdom.ErrInvalidImageID
	}

	if uc.listImageRecordRepo == nil {
		return listdom.List{}, ErrNotSupported("List.SetPrimaryImage.RecordRepo")
	}

	reader, ok := any(uc.listImageRecordRepo).(ListImageRecordByIDReader)
	if !ok || reader == nil {
		return listdom.List{}, ErrNotSupported("List.SetPrimaryImage.RecordRepo.GetByListIDAndID")
	}

	img, err := reader.GetByListIDAndID(ctx, lid, iid)
	if err != nil {
		return listdom.List{}, err
	}

	if img.ID == "" {
		return listdom.List{}, listdom.ErrInvalidImageID
	}

	if img.ListID != "" && img.ListID != lid {
		return listdom.List{}, errors.New("list: image belongs to other list")
	}

	if strings.TrimSpace(img.URL) == "" {
		return listdom.List{}, listdom.ErrInvalidListImageURL
	}

	if now.IsZero() {
		now = time.Now().UTC()
	}

	patch := listdom.ListPatch{
		ImageID:   &iid,
		UpdatedAt: ptrTime(now.UTC()),
		UpdatedBy: updatedBy,
	}

	updated, err := uc.listPatchUpdater.Update(ctx, lid, patch)
	if err != nil {
		return listdom.List{}, err
	}

	return updated, nil
}
