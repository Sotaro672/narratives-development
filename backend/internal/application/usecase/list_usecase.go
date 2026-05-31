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

type ListAggregate struct {
	List   listdom.List        `json:"list"`
	Images []listdom.ListImage `json:"images"`
}

type ListUsecase struct {
	listRepo  listdom.Repository
	imageRepo listdom.ImageRepository
}

func NewListUsecase(
	listRepo listdom.Repository,
	imageRepo listdom.ImageRepository,
) *ListUsecase {
	return &ListUsecase{
		listRepo:  listRepo,
		imageRepo: imageRepo,
	}
}

func (uc *ListUsecase) WithImageRepository(repo listdom.ImageRepository) *ListUsecase {
	if uc == nil {
		return nil
	}

	uc.imageRepo = repo
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
	if uc == nil || uc.listRepo == nil {
		return listdom.PageResult[listdom.List]{}, ErrNotSupported("List.List")
	}

	return uc.listRepo.List(ctx, filter, sort, page)
}

func (uc *ListUsecase) Count(
	ctx context.Context,
	filter listdom.Filter,
) (int, error) {
	if uc == nil || uc.listRepo == nil {
		return 0, ErrNotSupported("List.Count")
	}

	return uc.listRepo.Count(ctx, filter)
}

func (uc *ListUsecase) Create(
	ctx context.Context,
	item listdom.List,
) (listdom.List, error) {
	if uc == nil || uc.listRepo == nil {
		return listdom.List{}, ErrNotSupported("List.Create")
	}

	created, err := uc.listRepo.Create(ctx, item)
	if err != nil {
		return listdom.List{}, err
	}

	if created.ReadableID == "" {
		rid := generateReadableID(created.ID, created.CreatedAt)
		now := time.Now().UTC()

		created.ReadableID = rid
		created.UpdatedAt = &now

		updated, err := uc.listRepo.Update(ctx, created.ID, created)
		if err == nil {
			created = updated
		}
	}

	return created, nil
}

func (uc *ListUsecase) Update(
	ctx context.Context,
	item listdom.List,
) (listdom.List, error) {
	if uc == nil || uc.listRepo == nil {
		return listdom.List{}, ErrNotSupported("List.Update")
	}

	id := strings.TrimSpace(item.ID)
	if id == "" {
		return listdom.List{}, listdom.ErrInvalidID
	}

	item.ID = id

	return uc.listRepo.Update(ctx, id, item)
}

func (uc *ListUsecase) Delete(ctx context.Context, id string) error {
	if uc == nil || uc.listRepo == nil {
		return ErrNotSupported("List.Delete")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return listdom.ErrInvalidID
	}

	return uc.listRepo.Delete(ctx, id)
}

func (uc *ListUsecase) GetByID(
	ctx context.Context,
	id string,
) (listdom.List, error) {
	if uc == nil || uc.listRepo == nil {
		return listdom.List{}, ErrNotSupported("List.GetByID")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return listdom.List{}, listdom.ErrInvalidID
	}

	return uc.listRepo.GetByID(ctx, id)
}

func (uc *ListUsecase) GetImages(
	ctx context.Context,
	listID string,
) ([]listdom.ListImage, error) {
	if uc == nil || uc.imageRepo == nil {
		return []listdom.ListImage{}, nil
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		return []listdom.ListImage{}, nil
	}

	items, err := uc.imageRepo.ListByListID(ctx, listID)
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
	if uc == nil || uc.listRepo == nil {
		return ListAggregate{}, ErrNotSupported("List.GetAggregate")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return ListAggregate{}, listdom.ErrInvalidID
	}

	li, err := uc.listRepo.GetByID(ctx, id)
	if err != nil {
		return ListAggregate{}, err
	}

	images := []listdom.ListImage{}

	if uc.imageRepo != nil {
		items, err := uc.imageRepo.ListByListID(ctx, id)
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

func (uc *ListUsecase) CreateImage(
	ctx context.Context,
	img listdom.ListImage,
) (listdom.ListImage, error) {
	if uc == nil {
		return listdom.ListImage{}, ErrNotSupported("List.CreateImage")
	}

	if uc.imageRepo == nil {
		return listdom.ListImage{}, ErrNotSupported("List.CreateImage.ImageRepo")
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

	created, err := uc.imageRepo.Create(ctx, img)
	if err != nil {
		return listdom.ListImage{}, err
	}

	return created, nil
}

func (uc *ListUsecase) UpdateImage(
	ctx context.Context,
	listID string,
	imageID string,
	patch listdom.ListImagePatch,
) (listdom.ListImage, error) {
	if uc == nil {
		return listdom.ListImage{}, ErrNotSupported("List.UpdateImage")
	}

	if uc.imageRepo == nil {
		return listdom.ListImage{}, ErrNotSupported("List.UpdateImage.ImageRepo")
	}

	listID = strings.TrimSpace(listID)
	imageID = strings.TrimSpace(imageID)

	if listID == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageListID
	}

	if imageID == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageID
	}

	if strings.Contains(imageID, "/") || strings.Contains(imageID, "://") {
		return listdom.ListImage{}, ErrInvalidArgument("invalid_image_id")
	}

	if patch.URL != nil {
		v := strings.TrimSpace(*patch.URL)
		patch.URL = &v
	}

	if patch.DisplayOrder != nil && *patch.DisplayOrder < 0 {
		return listdom.ListImage{}, listdom.ErrInvalidListImageDisplayOrder
	}

	if patch.UpdatedAt == nil {
		now := time.Now().UTC()
		patch.UpdatedAt = &now
	} else if !patch.UpdatedAt.IsZero() {
		t := patch.UpdatedAt.UTC()
		patch.UpdatedAt = &t
	}

	if patch.UpdatedBy != nil {
		v := strings.TrimSpace(*patch.UpdatedBy)
		patch.UpdatedBy = &v
	}

	updated, err := uc.imageRepo.Update(ctx, listID, imageID, patch)
	if err != nil {
		return listdom.ListImage{}, err
	}

	return updated, nil
}

func (uc *ListUsecase) DeleteImage(ctx context.Context, listID string, imageID string) error {
	if uc == nil {
		return ErrNotSupported("List.DeleteImage")
	}

	if uc.imageRepo == nil {
		return ErrNotSupported("List.DeleteImage.ImageRepo")
	}

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

	if err := uc.imageRepo.Delete(ctx, listID, imageID); err != nil {
		if !errors.Is(err, listdom.ErrListImageNotFound) {
			return err
		}
	}

	if uc.listRepo != nil {
		l, err := uc.listRepo.GetByID(ctx, listID)
		if err == nil && l.ImageID == imageID {
			now := time.Now().UTC()

			l.ImageID = ""
			l.UpdatedAt = &now

			_, _ = uc.listRepo.Update(ctx, listID, l)
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

	if uc.listRepo == nil {
		return listdom.List{}, ErrNotSupported("List.SetPrimaryImage.ListRepo")
	}

	if uc.imageRepo == nil {
		return listdom.List{}, ErrNotSupported("List.SetPrimaryImage.ImageRepo")
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

	img, err := uc.imageRepo.GetByListIDAndID(ctx, lid, iid)
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

	l, err := uc.listRepo.GetByID(ctx, lid)
	if err != nil {
		return listdom.List{}, err
	}

	l.ImageID = iid
	l.UpdatedAt = ptrTime(now.UTC())

	if updatedBy != nil {
		v := strings.TrimSpace(*updatedBy)
		l.UpdatedBy = &v
	}

	updated, err := uc.listRepo.Update(ctx, lid, l)
	if err != nil {
		return listdom.List{}, err
	}

	return updated, nil
}
