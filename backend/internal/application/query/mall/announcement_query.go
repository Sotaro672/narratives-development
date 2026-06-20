// backend/internal/application/query/mall/announcement_query.go
package mall

import (
	"context"
	"time"

	ann "narratives/internal/domain/announcement"
	common "narratives/internal/domain/common"
	tokenblueprint "narratives/internal/domain/tokenBlueprint"
)

type AnnouncementQueryService struct {
	announcementRepo       ann.Repository
	announcementAvatarRepo ann.AvatarRepository
	tokenBlueprintRepo     tokenblueprint.RepositoryPort
}

func NewAnnouncementQueryService(
	announcementRepo ann.Repository,
	announcementAvatarRepo ann.AvatarRepository,
	tokenBlueprintRepo tokenblueprint.RepositoryPort,
) *AnnouncementQueryService {
	return &AnnouncementQueryService{
		announcementRepo:       announcementRepo,
		announcementAvatarRepo: announcementAvatarRepo,
		tokenBlueprintRepo:     tokenBlueprintRepo,
	}
}

type AnnouncementListResult struct {
	Items      []AnnouncementListItem `json:"items"`
	TotalCount int                    `json:"totalCount"`
	Page       int                    `json:"page"`
	PerPage    int                    `json:"perPage"`
}

type AnnouncementListItem struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`

	TargetToken string `json:"targetToken"`
	TokenName   string `json:"tokenName"`

	Published   bool       `json:"published"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`

	IsRead bool       `json:"isRead"`
	ReadAt *time.Time `json:"readAt,omitempty"`

	Attachments []string `json:"attachments,omitempty"`

	CreatedAt time.Time  `json:"createdAt"`
	CreatedBy string     `json:"createdBy"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy *string    `json:"updatedBy,omitempty"`
}

func (s *AnnouncementQueryService) ListByTargetAvatar(
	ctx context.Context,
	avatarID string,
	page common.Page,
) (AnnouncementListResult, error) {
	if s == nil || s.announcementRepo == nil {
		return AnnouncementListResult{}, ann.ErrNotFound
	}

	if avatarID == "" {
		return AnnouncementListResult{}, ann.ErrInvalidAvatarID
	}

	result, err := s.announcementRepo.ListByTargetAvatar(ctx, avatarID, page)
	if err != nil {
		return AnnouncementListResult{}, err
	}

	items := make([]AnnouncementListItem, 0, len(result.Items))

	for _, announcement := range result.Items {
		item, err := s.toListItem(ctx, announcement, avatarID)
		if err != nil {
			return AnnouncementListResult{}, err
		}

		items = append(items, item)
	}

	return AnnouncementListResult{
		Items:      items,
		TotalCount: result.TotalCount,
		Page:       result.Page,
		PerPage:    result.PerPage,
	}, nil
}

func (s *AnnouncementQueryService) toListItem(
	ctx context.Context,
	a ann.Announcement,
	avatarID string,
) (AnnouncementListItem, error) {
	targetToken := ""
	if a.TargetToken != nil {
		targetToken = *a.TargetToken
	}

	tokenName, err := s.resolveTokenName(ctx, targetToken)
	if err != nil {
		return AnnouncementListItem{}, err
	}

	isRead, readAt, err := s.resolveReadState(ctx, a.ID, avatarID)
	if err != nil {
		return AnnouncementListItem{}, err
	}

	return AnnouncementListItem{
		ID:          a.ID,
		Title:       a.Title,
		Content:     a.Content,
		TargetToken: targetToken,
		TokenName:   tokenName,

		Published:   a.Published,
		PublishedAt: a.PublishedAt,

		IsRead: isRead,
		ReadAt: readAt,

		Attachments: a.Attachments,

		CreatedAt: a.CreatedAt,
		CreatedBy: a.CreatedBy,
		UpdatedAt: a.UpdatedAt,
		UpdatedBy: a.UpdatedBy,
	}, nil
}

func (s *AnnouncementQueryService) resolveReadState(
	ctx context.Context,
	announcementID string,
	avatarID string,
) (bool, *time.Time, error) {
	if announcementID == "" {
		return false, nil, ann.ErrInvalidAnnouncementID
	}
	if avatarID == "" {
		return false, nil, ann.ErrInvalidAvatarID
	}

	if s == nil || s.announcementAvatarRepo == nil {
		return false, nil, nil
	}

	avatars, err := s.announcementAvatarRepo.ListByAnnouncementID(
		ctx,
		announcementID,
		ann.AnnouncementAvatarFilter{
			AvatarIDs: []string{avatarID},
		},
	)
	if err != nil {
		return false, nil, err
	}

	if len(avatars) == 0 {
		return false, nil, nil
	}

	return avatars[0].IsRead, avatars[0].ReadAt, nil
}

func (s *AnnouncementQueryService) resolveTokenName(
	ctx context.Context,
	targetToken string,
) (string, error) {
	if targetToken == "" {
		return "", nil
	}

	if s == nil || s.tokenBlueprintRepo == nil {
		return targetToken, nil
	}

	tb, err := s.tokenBlueprintRepo.GetByID(ctx, targetToken)
	if err != nil {
		return "", err
	}
	if tb == nil {
		return targetToken, nil
	}

	if tb.Name != "" {
		return tb.Name, nil
	}

	return targetToken, nil
}
