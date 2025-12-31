// backend/internal/application/usecase/post_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidPostInput       = errors.New("post: invalid input")
	ErrPostRepoNotConfigured  = errors.New("post: repo not configured")
	ErrPostImageNotSupported  = errors.New("post: image repo not configured")
	ErrPostIDEmpty            = errors.New("post: id is empty")
	ErrPostAvatarIDEmpty      = errors.New("post: avatarId is empty")
	ErrPostBodyEmpty          = errors.New("post: body is empty")
	ErrPostBodyTooLong        = errors.New("post: body too long")
	ErrPostListLimitInvalid   = errors.New("post: list limit invalid")
	ErrPostImageFileNameEmpty = errors.New("post: image filename is empty")
)

// Post is a simple usecase-level model.
// (ドメインモデルが後で導入される場合は、この型を domain/post に移し、ここはポート経由に差し替えてOKです)
type Post struct {
	ID       string
	AvatarID string

	Body   string
	Images []PostImage

	CreatedAt time.Time
	UpdatedAt time.Time
}

type PostImage struct {
	URL        string
	ObjectPath string
	FileName   string
	Size       *int64
}

// PostPatch is a partial update payload.
type PostPatch struct {
	Body   *string
	Images *[]PostImage
}

// PostRepo is the persistence port for posts.
type PostRepo interface {
	GetByID(ctx context.Context, id string) (Post, error)
	Create(ctx context.Context, p Post) (Post, error)
	Update(ctx context.Context, id string, patch PostPatch) (Post, error)
	Delete(ctx context.Context, id string) error

	// optional list APIs (repo 側に無ければ未実装でOK)
	ListByAvatarID(ctx context.Context, avatarID string, limit int) ([]Post, error)
}

// PostImageRepo is an optional port for issuing signed upload URLs, etc.
// 画像アップロードを使わない段階でも、repo が nil でも動くように usecase を設計しています。
type PostImageRepo interface {
	// IssueSignedUploadURL returns (uploadURL, publicURL, objectPath).
	IssueSignedUploadURL(ctx context.Context, avatarID, fileName, contentType string, expiresIn time.Duration) (string, string, string, error)
}

type PostUsecase struct {
	repo      PostRepo
	imageRepo PostImageRepo
	now       func() time.Time
}

func NewPostUsecase(repo PostRepo, imageRepo PostImageRepo) *PostUsecase {
	return &PostUsecase{
		repo:      repo,
		imageRepo: imageRepo,
		now:       time.Now,
	}
}

func (u *PostUsecase) WithNow(now func() time.Time) *PostUsecase {
	u.now = now
	return u
}

// -----------------------
// Queries
// -----------------------

func (u *PostUsecase) GetByID(ctx context.Context, id string) (Post, error) {
	if u.repo == nil {
		return Post{}, ErrPostRepoNotConfigured
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return Post{}, ErrPostIDEmpty
	}
	return u.repo.GetByID(ctx, id)
}

func (u *PostUsecase) ListByAvatarID(ctx context.Context, avatarID string, limit int) ([]Post, error) {
	if u.repo == nil {
		return nil, ErrPostRepoNotConfigured
	}
	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return nil, ErrPostAvatarIDEmpty
	}
	if limit <= 0 {
		return nil, ErrPostListLimitInvalid
	}
	return u.repo.ListByAvatarID(ctx, avatarID, limit)
}

// -----------------------
// Commands
// -----------------------

type CreatePostInput struct {
	AvatarID string `json:"avatarId"`
	Body     string `json:"body"`
	Images   []PostImage
}

// Create creates a post (text-first).
// - 画像は「既にアップロード済みの public URL / objectPath を受け取る」形を基本にしています。
func (u *PostUsecase) Create(ctx context.Context, in CreatePostInput) (Post, error) {
	if u.repo == nil {
		return Post{}, ErrPostRepoNotConfigured
	}

	avatarID := strings.TrimSpace(in.AvatarID)
	if avatarID == "" {
		return Post{}, ErrPostAvatarIDEmpty
	}

	body := strings.TrimSpace(in.Body)
	if body == "" {
		return Post{}, ErrPostBodyEmpty
	}
	// 必要なら上限調整
	if len(body) > 4000 {
		return Post{}, ErrPostBodyTooLong
	}

	now := u.now().UTC()
	p := Post{
		// ID is assigned by repo layer if empty
		ID:        "",
		AvatarID:  avatarID,
		Body:      body,
		Images:    normalizePostImages(in.Images),
		CreatedAt: now,
		UpdatedAt: now,
	}

	created, err := u.repo.Create(ctx, p)
	if err != nil {
		return Post{}, err
	}
	return created, nil
}

func (u *PostUsecase) Update(ctx context.Context, id string, patch PostPatch) (Post, error) {
	if u.repo == nil {
		return Post{}, ErrPostRepoNotConfigured
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return Post{}, ErrPostIDEmpty
	}

	if patch.Body != nil {
		v := strings.TrimSpace(*patch.Body)
		if v == "" {
			return Post{}, ErrPostBodyEmpty
		}
		if len(v) > 4000 {
			return Post{}, ErrPostBodyTooLong
		}
		patch.Body = &v
	}
	if patch.Images != nil {
		imgs := normalizePostImages(*patch.Images)
		patch.Images = &imgs
	}

	updated, err := u.repo.Update(ctx, id, patch)
	if err != nil {
		return Post{}, err
	}
	return updated, nil
}

func (u *PostUsecase) Delete(ctx context.Context, id string) error {
	if u.repo == nil {
		return ErrPostRepoNotConfigured
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrPostIDEmpty
	}
	return u.repo.Delete(ctx, id)
}

// -----------------------
// Optional: image upload helper
// -----------------------

type IssuePostImageUploadURLInput struct {
	AvatarID    string
	FileName    string
	ContentType string
	ExpiresIn   time.Duration
}

type IssuePostImageUploadURLResult struct {
	UploadURL  string `json:"uploadUrl"`
	PublicURL  string `json:"publicUrl"`
	ObjectPath string `json:"objectPath"`
}

// IssueUploadURL is an optional helper if your frontend uploads images directly to GCS.
// imageRepo が nil の場合は明示的にエラー。
func (u *PostUsecase) IssueUploadURL(ctx context.Context, in IssuePostImageUploadURLInput) (IssuePostImageUploadURLResult, error) {
	if u.imageRepo == nil {
		return IssuePostImageUploadURLResult{}, ErrPostImageNotSupported
	}
	avatarID := strings.TrimSpace(in.AvatarID)
	if avatarID == "" {
		return IssuePostImageUploadURLResult{}, ErrPostAvatarIDEmpty
	}
	fileName := strings.TrimSpace(in.FileName)
	if fileName == "" {
		return IssuePostImageUploadURLResult{}, ErrPostImageFileNameEmpty
	}
	ct := strings.TrimSpace(in.ContentType)
	if ct == "" {
		// best-effort: allow empty; adapter may default
	}
	exp := in.ExpiresIn
	if exp <= 0 {
		exp = 10 * time.Minute
	}

	up, pub, path, err := u.imageRepo.IssueSignedUploadURL(ctx, avatarID, fileName, ct, exp)
	if err != nil {
		return IssuePostImageUploadURLResult{}, err
	}
	return IssuePostImageUploadURLResult{
		UploadURL:  up,
		PublicURL:  pub,
		ObjectPath: path,
	}, nil
}

// -----------------------
// helpers
// -----------------------

func normalizePostImages(src []PostImage) []PostImage {
	if len(src) == 0 {
		return nil
	}
	dst := make([]PostImage, 0, len(src))
	for _, it := range src {
		u := strings.TrimSpace(it.URL)
		p := strings.TrimSpace(it.ObjectPath)
		f := strings.TrimSpace(it.FileName)

		// URL も ObjectPath も空は捨てる
		if u == "" && p == "" {
			continue
		}
		dst = append(dst, PostImage{
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
