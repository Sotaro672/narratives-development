// backend/internal/application/usecase/news_usecase.go
package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	common "narratives/internal/domain/common"
	newsdom "narratives/internal/domain/news"
)

// ============================================================
// Errors
// ============================================================

var (
	ErrNewsRepositoryNotConfigured     = errors.New("news_usecase: news repository is not configured")
	ErrNewsReadRepositoryNotConfigured = errors.New("news_usecase: news read repository is not configured")
)

// ============================================================
// NewsUsecase
// ============================================================

// NewsUsecase coordinates AMOL system-wide News and recipient read states.
//
// News:
// - Admin creates one News document.
// - The News is visible to all Console members and Mall avatars.
// - News creation never fans out recipient-specific documents.
//
// NewsRead:
// - Recipient identity is used only to derive the deterministic NewsRead ID.
// - RecipientType / RecipientID do not need to be persisted in Firestore.
// - A NewsRead document is created only when the recipient actually reads News.
type NewsUsecase struct {
	newsRepo     newsdom.Repository
	newsReadRepo newsdom.ReadRepository
	now          func() time.Time
}

func NewNewsUsecase(newsRepo newsdom.Repository, newsReadRepo newsdom.ReadRepository) *NewsUsecase {
	return &NewsUsecase{
		newsRepo:     newsRepo,
		newsReadRepo: newsReadRepo,
		now:          time.Now,
	}
}

func (u *NewsUsecase) WithNow(now func() time.Time) *NewsUsecase {
	if u != nil && now != nil {
		u.now = now
	}
	return u
}

// ============================================================
// Create
// ============================================================

type CreateNewsInput struct {
	// ID is optional.
	// When empty, the usecase generates a random ID.
	ID string

	Title string
	Body  string
	Image *newsdom.NewsImage

	// CreatedBy must identify the authenticated Admin.
	CreatedBy string
}

// CreateNews creates and immediately publishes system-wide News.
//
// There is intentionally no draft state.
// CreatedAt and PublishedAt are set to the same timestamp.
func (u *NewsUsecase) CreateNews(ctx context.Context, input CreateNewsInput) (newsdom.News, error) {
	if err := u.ensureNewsRepository(); err != nil {
		return newsdom.News{}, err
	}

	id := input.ID
	if id == "" {
		generatedID, err := newNewsID()
		if err != nil {
			return newsdom.News{}, err
		}
		id = generatedID
	}

	now := u.currentTime()

	entity, err := newsdom.New(
		newsdom.NewsID(id),
		input.Title,
		input.Body,
		input.Image,
		now,
		now,
		input.CreatedBy,
	)
	if err != nil {
		return newsdom.News{}, err
	}

	return u.newsRepo.Create(ctx, entity)
}

// ============================================================
// Admin list
// ============================================================

// ListNews returns system-wide News for Admin.
//
// If sort is omitted, newest published News is returned first.
func (u *NewsUsecase) ListNews(
	ctx context.Context,
	filter newsdom.Filter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[newsdom.News], error) {
	if err := u.ensureNewsRepository(); err != nil {
		return common.PageResult[newsdom.News]{}, err
	}

	if sort.Column == "" {
		sort.Column = "publishedAt"
	}
	if sort.Order == "" {
		sort.Order = common.SortDesc
	}

	return u.newsRepo.List(ctx, filter, sort, page)
}

// ============================================================
// Recipient list result
// ============================================================

// NewsRecipientItem combines globally shared News with the current reader's
// read state.
//
// NewsRead does not exist for unread News, so unread state is represented by:
// - IsRead = false
// - ReadAt = nil
type NewsRecipientItem struct {
	News newsdom.News `json:"news"`

	IsRead bool       `json:"isRead"`
	ReadAt *time.Time `json:"readAt,omitempty"`
}

// ============================================================
// Console / MEMBER
// ============================================================

// ListNewsForMember returns system-wide News with the current Console member's
// read state.
func (u *NewsUsecase) ListNewsForMember(
	ctx context.Context,
	memberID string,
	page common.Page,
) (common.PageResult[NewsRecipientItem], error) {
	return u.listNewsForRecipient(ctx, newsdom.RecipientTypeMember, memberID, page)
}

// CountUnreadNewsForMember returns the number of unread News for one Console
// member.
func (u *NewsUsecase) CountUnreadNewsForMember(ctx context.Context, memberID string) (int, error) {
	return u.countUnreadNewsForRecipient(ctx, newsdom.RecipientTypeMember, memberID)
}

// MarkNewsReadForMember marks one News as read by one Console member.
//
// The operation is idempotent.
// An existing NewsRead keeps its original ReadAt.
func (u *NewsUsecase) MarkNewsReadForMember(
	ctx context.Context,
	newsID newsdom.NewsID,
	memberID string,
) (newsdom.NewsRead, error) {
	return u.markNewsReadForRecipient(ctx, newsID, newsdom.RecipientTypeMember, memberID)
}

// ============================================================
// Mall / AVATAR
// ============================================================

// ListNewsForAvatar returns system-wide News with the current Mall avatar's
// read state.
func (u *NewsUsecase) ListNewsForAvatar(
	ctx context.Context,
	avatarID string,
	page common.Page,
) (common.PageResult[NewsRecipientItem], error) {
	return u.listNewsForRecipient(ctx, newsdom.RecipientTypeAvatar, avatarID, page)
}

// CountUnreadNewsForAvatar returns the number of unread News for one Mall
// avatar.
func (u *NewsUsecase) CountUnreadNewsForAvatar(ctx context.Context, avatarID string) (int, error) {
	return u.countUnreadNewsForRecipient(ctx, newsdom.RecipientTypeAvatar, avatarID)
}

// MarkNewsReadForAvatar marks one News as read by one Mall avatar.
//
// The operation is idempotent.
// An existing NewsRead keeps its original ReadAt.
func (u *NewsUsecase) MarkNewsReadForAvatar(
	ctx context.Context,
	newsID newsdom.NewsID,
	avatarID string,
) (newsdom.NewsRead, error) {
	return u.markNewsReadForRecipient(ctx, newsID, newsdom.RecipientTypeAvatar, avatarID)
}

// ============================================================
// Recipient list
// ============================================================

func (u *NewsUsecase) listNewsForRecipient(
	ctx context.Context,
	recipientType newsdom.RecipientType,
	recipientID string,
	page common.Page,
) (common.PageResult[NewsRecipientItem], error) {
	if err := u.ensureNewsRepository(); err != nil {
		return common.PageResult[NewsRecipientItem]{}, err
	}
	if err := u.ensureNewsReadRepository(); err != nil {
		return common.PageResult[NewsRecipientItem]{}, err
	}
	if err := validateNewsRecipient(recipientType, recipientID); err != nil {
		return common.PageResult[NewsRecipientItem]{}, err
	}

	newsResult, err := u.newsRepo.List(
		ctx,
		newsdom.Filter{},
		common.Sort{
			Column: "publishedAt",
			Order:  common.SortDesc,
		},
		page,
	)
	if err != nil {
		return common.PageResult[NewsRecipientItem]{}, err
	}

	if len(newsResult.Items) == 0 {
		return common.PageResult[NewsRecipientItem]{
			Items:      []NewsRecipientItem{},
			TotalCount: newsResult.TotalCount,
			TotalPages: newsResult.TotalPages,
			Page:       newsResult.Page,
			PerPage:    newsResult.PerPage,
		}, nil
	}

	newsIDs := make([]newsdom.NewsID, 0, len(newsResult.Items))
	for _, item := range newsResult.Items {
		newsIDs = append(newsIDs, item.ID)
	}

	readResult, err := u.newsReadRepo.List(
		ctx,
		newsdom.ReadFilter{
			RecipientType: &recipientType,
			RecipientID:   recipientID,
			NewsIDs:       newsIDs,
		},
		common.Sort{
			Column: "readAt",
			Order:  common.SortDesc,
		},
		common.Page{
			Number:  1,
			PerPage: len(newsIDs),
		},
	)
	if err != nil {
		return common.PageResult[NewsRecipientItem]{}, err
	}

	readByNewsID := make(map[newsdom.NewsID]newsdom.NewsRead, len(readResult.Items))
	for _, read := range readResult.Items {
		readByNewsID[read.NewsID] = read
	}

	items := make([]NewsRecipientItem, 0, len(newsResult.Items))
	for _, entity := range newsResult.Items {
		item := NewsRecipientItem{
			News:   entity,
			IsRead: false,
			ReadAt: nil,
		}

		if read, ok := readByNewsID[entity.ID]; ok {
			readAt := read.ReadAt
			item.IsRead = true
			item.ReadAt = &readAt
		}

		items = append(items, item)
	}

	return common.PageResult[NewsRecipientItem]{
		Items:      items,
		TotalCount: newsResult.TotalCount,
		TotalPages: newsResult.TotalPages,
		Page:       newsResult.Page,
		PerPage:    newsResult.PerPage,
	}, nil
}

// ============================================================
// Unread count
// ============================================================

// countUnreadNewsForRecipient calculates:
//
//	total News count - current reader's NewsRead count
//
// RecipientType / RecipientID are not persisted as fields.
// They are used only to derive deterministic NewsRead document IDs.
func (u *NewsUsecase) countUnreadNewsForRecipient(
	ctx context.Context,
	recipientType newsdom.RecipientType,
	recipientID string,
) (int, error) {
	if err := u.ensureNewsRepository(); err != nil {
		return 0, err
	}
	if err := u.ensureNewsReadRepository(); err != nil {
		return 0, err
	}
	if err := validateNewsRecipient(recipientType, recipientID); err != nil {
		return 0, err
	}

	newsResult, err := u.newsRepo.List(
		ctx,
		newsdom.Filter{},
		common.Sort{
			Column: "publishedAt",
			Order:  common.SortDesc,
		},
		common.Page{
			Number:  1,
			PerPage: 1,
		},
	)
	if err != nil {
		return 0, err
	}

	readResult, err := u.newsReadRepo.List(
		ctx,
		newsdom.ReadFilter{
			RecipientType: &recipientType,
			RecipientID:   recipientID,
		},
		common.Sort{
			Column: "readAt",
			Order:  common.SortDesc,
		},
		common.Page{
			Number:  1,
			PerPage: 1,
		},
	)
	if err != nil {
		return 0, err
	}

	unreadCount := newsResult.TotalCount - readResult.TotalCount
	if unreadCount < 0 {
		return 0, nil
	}
	return unreadCount, nil
}

// ============================================================
// Mark read
// ============================================================

func (u *NewsUsecase) markNewsReadForRecipient(
	ctx context.Context,
	newsID newsdom.NewsID,
	recipientType newsdom.RecipientType,
	recipientID string,
) (newsdom.NewsRead, error) {
	if err := u.ensureNewsRepository(); err != nil {
		return newsdom.NewsRead{}, err
	}
	if err := u.ensureNewsReadRepository(); err != nil {
		return newsdom.NewsRead{}, err
	}
	if err := newsID.Validate(); err != nil {
		return newsdom.NewsRead{}, err
	}
	if err := validateNewsRecipient(recipientType, recipientID); err != nil {
		return newsdom.NewsRead{}, err
	}

	// Do not create an orphan NewsRead for a non-existing News.
	if _, err := u.newsRepo.GetByID(ctx, newsID); err != nil {
		return newsdom.NewsRead{}, err
	}

	read, err := newsdom.NewRead(
		newsID,
		recipientType,
		recipientID,
		u.currentTime(),
	)
	if err != nil {
		return newsdom.NewsRead{}, err
	}

	result, err := u.newsReadRepo.CreateIfAbsent(ctx, read)
	if err != nil {
		return newsdom.NewsRead{}, err
	}

	return result.Read, nil
}

// ============================================================
// Validation
// ============================================================

func validateNewsRecipient(recipientType newsdom.RecipientType, recipientID string) error {
	if err := recipientType.Validate(); err != nil {
		return err
	}
	if recipientID == "" {
		return newsdom.ErrInvalidRecipientID
	}
	return nil
}

// ============================================================
// Repository guards
// ============================================================

func (u *NewsUsecase) ensureNewsRepository() error {
	if u == nil || u.newsRepo == nil {
		return ErrNewsRepositoryNotConfigured
	}
	return nil
}

func (u *NewsUsecase) ensureNewsReadRepository() error {
	if u == nil || u.newsReadRepo == nil {
		return ErrNewsReadRepositoryNotConfigured
	}
	return nil
}

// ============================================================
// Time
// ============================================================

func (u *NewsUsecase) currentTime() time.Time {
	if u == nil || u.now == nil {
		return time.Now().UTC().Truncate(time.Microsecond)
	}
	return u.now().UTC().Truncate(time.Microsecond)
}

// ============================================================
// ID
// ============================================================

func newNewsID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}
