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
// usecase.ListPatcher (UpdateImageID etc.) for mall container.
//
// Motivation:
//   - In DI, we previously used an inline adapter (listPatcherAdapter) that depended on
//     "Update(ctx, id, patch) (listdom.List, error)" signature.
//   - This file makes it an explicit out-adapter, so mall/adapter.go can stay lean.
type ListPatcherRepoForMall struct {
	// Keep the concrete FS repo to avoid duplicating Firestore mapping logic.
	repo interface {
		Update(ctx context.Context, id string, patch listdom.ListPatch) (listdom.List, error)
	}
}

func NewListPatcherRepo(fsClient *firestore.Client) *ListPatcherRepoForMall {
	// We intentionally construct the canonical FS repo here.
	// If your mall container already constructs ListRepositoryFS and wants to reuse it,
	// you can also add another constructor that accepts the repo directly.
	listRepo := outfs.NewListRepositoryFS(fsClient)
	return &ListPatcherRepoForMall{repo: listRepo}
}

// NewListPatcherRepoForMallWithRepo allows reusing an already-constructed repository
// (useful if container.go already has listRepoFS and you want to avoid duplicate instances).
func NewListPatcherRepoForMallWithRepo(
	repo interface {
		Update(ctx context.Context, id string, patch listdom.ListPatch) (listdom.List, error)
	},
) *ListPatcherRepoForMall {
	return &ListPatcherRepoForMall{repo: repo}
}

// UpdateImageID updates list.imageId (and audit fields) and returns the updated List.
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

	patch := listdom.ListPatch{
		ImageID:   &imageID,
		UpdatedAt: &now,
		UpdatedBy: updatedBy,
	}

	return r.repo.Update(ctx, listID, patch)
}
