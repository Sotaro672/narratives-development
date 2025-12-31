// backend/internal/adapters/out/firestore/post_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	uc "narratives/internal/application/usecase"
)

var (
	ErrPostRepoFSInvalid = errors.New("firestore: post repository invalid")
)

// PostRepositoryFS implements usecase.PostRepo using Firestore.
type PostRepositoryFS struct {
	Client *firestore.Client
}

func NewPostRepositoryFS(client *firestore.Client) *PostRepositoryFS {
	return &PostRepositoryFS{Client: client}
}

func (r *PostRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("posts")
}

// ----------
// Firestore DTOs
// ----------

type postImageDoc struct {
	URL        string `firestore:"url,omitempty"`
	ObjectPath string `firestore:"objectPath,omitempty"`
	FileName   string `firestore:"fileName,omitempty"`
	Size       *int64 `firestore:"size,omitempty"`
}

type postDoc struct {
	AvatarID  string         `firestore:"avatarId"`
	Body      string         `firestore:"body"`
	Images    []postImageDoc `firestore:"images,omitempty"`
	CreatedAt time.Time      `firestore:"createdAt"`
	UpdatedAt time.Time      `firestore:"updatedAt"`
}

// ----------
// uc.PostRepo
// ----------

func (r *PostRepositoryFS) GetByID(ctx context.Context, id string) (uc.Post, error) {
	if r == nil || r.Client == nil {
		return uc.Post{}, ErrPostRepoFSInvalid
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return uc.Post{}, uc.ErrPostIDEmpty
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		return uc.Post{}, err
	}

	var d postDoc
	if err := snap.DataTo(&d); err != nil {
		return uc.Post{}, err
	}

	return toUCPost(snap.Ref.ID, d), nil
}

func (r *PostRepositoryFS) Create(ctx context.Context, p uc.Post) (uc.Post, error) {
	if r == nil || r.Client == nil {
		return uc.Post{}, ErrPostRepoFSInvalid
	}

	avatarID := strings.TrimSpace(p.AvatarID)
	body := strings.TrimSpace(p.Body)
	if avatarID == "" {
		return uc.Post{}, uc.ErrPostAvatarIDEmpty
	}
	if body == "" {
		return uc.Post{}, uc.ErrPostBodyEmpty
	}

	now := time.Now().UTC()
	createdAt := p.CreatedAt
	updatedAt := p.UpdatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	if updatedAt.IsZero() {
		updatedAt = now
	}

	doc := postDoc{
		AvatarID:  avatarID,
		Body:      body,
		Images:    toDocImages(p.Images),
		CreatedAt: createdAt.UTC(),
		UpdatedAt: updatedAt.UTC(),
	}

	// ID が空なら採番
	var docRef *firestore.DocumentRef
	if strings.TrimSpace(p.ID) == "" {
		docRef = r.col().NewDoc()
	} else {
		docRef = r.col().Doc(strings.TrimSpace(p.ID))
	}

	if _, err := docRef.Set(ctx, doc); err != nil {
		return uc.Post{}, err
	}

	return toUCPost(docRef.ID, doc), nil
}

func (r *PostRepositoryFS) Update(ctx context.Context, id string, patch uc.PostPatch) (uc.Post, error) {
	if r == nil || r.Client == nil {
		return uc.Post{}, ErrPostRepoFSInvalid
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return uc.Post{}, uc.ErrPostIDEmpty
	}

	now := time.Now().UTC()

	var ups []firestore.Update
	if patch.Body != nil {
		ups = append(ups, firestore.Update{
			Path:  "body",
			Value: strings.TrimSpace(*patch.Body),
		})
	}
	if patch.Images != nil {
		ups = append(ups, firestore.Update{
			Path:  "images",
			Value: toDocImages(*patch.Images),
		})
	}
	ups = append(ups, firestore.Update{
		Path:  "updatedAt",
		Value: now,
	})

	if _, err := r.col().Doc(id).Update(ctx, ups); err != nil {
		return uc.Post{}, err
	}
	return r.GetByID(ctx, id)
}

func (r *PostRepositoryFS) Delete(ctx context.Context, id string) error {
	if r == nil || r.Client == nil {
		return ErrPostRepoFSInvalid
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return uc.ErrPostIDEmpty
	}
	_, err := r.col().Doc(id).Delete(ctx)
	return err
}

func (r *PostRepositoryFS) ListByAvatarID(ctx context.Context, avatarID string, limit int) ([]uc.Post, error) {
	if r == nil || r.Client == nil {
		return nil, ErrPostRepoFSInvalid
	}
	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return nil, uc.ErrPostAvatarIDEmpty
	}
	if limit <= 0 {
		return nil, uc.ErrPostListLimitInvalid
	}

	it := r.col().
		Where("avatarId", "==", avatarID).
		OrderBy("createdAt", firestore.Desc).
		Limit(limit).
		Documents(ctx)
	defer it.Stop()

	var out []uc.Post
	for {
		snap, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var d postDoc
		if err := snap.DataTo(&d); err != nil {
			return nil, err
		}
		out = append(out, toUCPost(snap.Ref.ID, d))
	}
	return out, nil
}

// ----------
// mapping helpers
// ----------

func toDocImages(src []uc.PostImage) []postImageDoc {
	if len(src) == 0 {
		return nil
	}
	dst := make([]postImageDoc, 0, len(src))
	for _, it := range src {
		u := strings.TrimSpace(it.URL)
		p := strings.TrimSpace(it.ObjectPath)
		f := strings.TrimSpace(it.FileName)
		if u == "" && p == "" {
			continue
		}
		dst = append(dst, postImageDoc{
			URL:        u,
			ObjectPath: p,
			FileName:   f,
			Size:       it.Size,
		})
	}
	if len(dst) == 0 {
		return nil
	}
	return dst
}

func toUCImages(src []postImageDoc) []uc.PostImage {
	if len(src) == 0 {
		return nil
	}
	dst := make([]uc.PostImage, 0, len(src))
	for _, it := range src {
		u := strings.TrimSpace(it.URL)
		p := strings.TrimSpace(it.ObjectPath)
		f := strings.TrimSpace(it.FileName)
		if u == "" && p == "" {
			continue
		}
		dst = append(dst, uc.PostImage{
			URL:        u,
			ObjectPath: p,
			FileName:   f,
			Size:       it.Size,
		})
	}
	if len(dst) == 0 {
		return nil
	}
	return dst
}

func toUCPost(id string, d postDoc) uc.Post {
	return uc.Post{
		ID:       strings.TrimSpace(id),
		AvatarID: strings.TrimSpace(d.AvatarID),
		Body:     strings.TrimSpace(d.Body),
		Images:   toUCImages(d.Images),

		CreatedAt: d.CreatedAt.UTC(),
		UpdatedAt: d.UpdatedAt.UTC(),
	}
}
