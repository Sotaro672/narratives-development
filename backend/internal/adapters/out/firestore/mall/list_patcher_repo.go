// backend/internal/adapters/out/firestore/mall/list_patcher_repo.go
package mall

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	outfs "narratives/internal/adapters/out/firestore"
	listdom "narratives/internal/domain/list"
)

// ListPatcherRepoForMall is a Firestore-backed adapter that satisfies
// usecase.ListPatcher (UpdateImageID etc.) AND usecase/list.ListPrimaryImageSetter
// for mall container.
type ListPatcherRepoForMall struct {
	// Keep the concrete FS repo to avoid duplicating Firestore mapping logic.
	// ✅ Added GetByID so we can implement SetPrimaryImageIfEmpty safely.
	repo interface {
		GetByID(ctx context.Context, id string) (listdom.List, error)
		Update(ctx context.Context, id string, patch listdom.ListPatch) (listdom.List, error)
	}
}

func NewListPatcherRepo(fsClient *firestore.Client) *ListPatcherRepoForMall {
	// We intentionally construct the canonical FS repo here.
	listRepo := outfs.NewListRepositoryFS(fsClient)
	return &ListPatcherRepoForMall{repo: listRepo}
}

// NewListPatcherRepoForMallWithRepo allows reusing an already-constructed repository
// (useful if container.go already has listRepoFS and you want to avoid duplicate instances).
func NewListPatcherRepoForMallWithRepo(
	repo interface {
		GetByID(ctx context.Context, id string) (listdom.List, error)
		Update(ctx context.Context, id string, patch listdom.ListPatch) (listdom.List, error)
	},
) *ListPatcherRepoForMall {
	return &ListPatcherRepoForMall{repo: repo}
}

// UpdateImageID updates list.imageId (and audit fields) and returns the updated List.
//
// Policy A:
// - list.image_id stores "primary imageId (Firestore docID)", NOT URL.
func (r *ListPatcherRepoForMall) UpdateImageID(
	ctx context.Context,
	listID string,
	imageID string,
	now time.Time,
	updatedBy *string,
) (listdom.List, error) {
	if r == nil || r.repo == nil {
		return listdom.List{}, errors.New("firestore.mall.ListPatcherRepoForMall: repo is nil")
	}

	listID = strings.TrimSpace(listID)
	imageID = strings.TrimSpace(imageID)

	if listID == "" {
		return listdom.List{}, listdom.ErrNotFound
	}
	if imageID == "" {
		return listdom.List{}, listdom.ErrEmptyImageID
	}

	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	patch := listdom.ListPatch{
		ImageID:   &imageID,
		UpdatedAt: &now,
		UpdatedBy: updatedBy,
		// DeletedAt/DeletedBy are untouched
		// ReadableID/Prices etc. untouched
	}

	return r.repo.Update(ctx, listID, patch)
}

// ✅ SetPrimaryImageID implements the newer usecase/list.ListPrimaryImageSetter contract.
//
// Policy A:
// - list.image_id stores "primary imageId (Firestore docID)".
func (r *ListPatcherRepoForMall) SetPrimaryImageID(
	ctx context.Context,
	listID string,
	imageID string,
	now time.Time,
) error {
	if r == nil || r.repo == nil {
		return errors.New("firestore.mall.ListPatcherRepoForMall: repo is nil")
	}

	listID = strings.TrimSpace(listID)
	imageID = strings.TrimSpace(imageID)

	if listID == "" {
		return listdom.ErrNotFound
	}
	if imageID == "" {
		return listdom.ErrEmptyImageID
	}

	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	_, err := r.UpdateImageID(ctx, listID, imageID, now, nil)
	return err
}

// ✅ SetPrimaryImageIfEmpty implements usecase/list.ListPrimaryImageSetter.
// It sets list.imageId only when current imageId is empty.
// (best-effort; does not overwrite existing primary)
//
// Policy A:
// - list.image_id stores "primary imageId (Firestore docID)".
func (r *ListPatcherRepoForMall) SetPrimaryImageIfEmpty(
	ctx context.Context,
	listID string,
	imageID string,
	now time.Time,
) error {
	if r == nil || r.repo == nil {
		return errors.New("firestore.mall.ListPatcherRepoForMall: repo is nil")
	}

	listID = strings.TrimSpace(listID)
	imageID = strings.TrimSpace(imageID)

	if listID == "" {
		return listdom.ErrNotFound
	}
	if imageID == "" {
		return listdom.ErrEmptyImageID
	}

	cur, err := r.repo.GetByID(ctx, listID)
	if err != nil {
		return err
	}

	// already set -> do nothing
	if strings.TrimSpace(cur.ImageID) != "" {
		return nil
	}

	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	// updatedBy はこのユースケースでは不要（nil）でOK
	_, err = r.UpdateImageID(ctx, listID, imageID, now, nil)
	return err
}
