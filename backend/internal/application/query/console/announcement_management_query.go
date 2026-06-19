// backend/internal/application/query/console/announcement_management_query.go
package query

import (
	"context"
	"errors"
	"time"

	announcementdom "narratives/internal/domain/announcement"
	common "narratives/internal/domain/common"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

type AnnouncementManagementTokenBlueprint struct {
	TokenBlueprintID string `json:"tokenBlueprintId"`
	TokenName        string `json:"tokenName"`
	BrandID          string `json:"brandId"`
}

type AnnouncementManagementAnnouncement struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	Content       string     `json:"content"`
	TargetToken   *string    `json:"targetToken,omitempty"`
	TargetAvatars []string   `json:"targetAvatars,omitempty"`
	Published     bool       `json:"published"`
	PublishedAt   *time.Time `json:"publishedAt,omitempty"`
	Attachments   []string   `json:"attachments,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	CreatedBy     string     `json:"createdBy"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy     *string    `json:"updatedBy,omitempty"`
}

type AnnouncementManagementRow struct {
	TokenBlueprint AnnouncementManagementTokenBlueprint `json:"tokenBlueprint"`
	Announcements  []AnnouncementManagementAnnouncement `json:"announcements"`
}

type AnnouncementManagementQueryResult struct {
	CompanyID string                      `json:"companyId"`
	Rows      []AnnouncementManagementRow `json:"rows"`
}

type AnnouncementManagementQuery struct {
	tokenBlueprintRepo tokenblueprintdom.RepositoryPort
	announcementRepo   announcementdom.Repository
}

func NewAnnouncementManagementQuery(
	tokenBlueprintRepo tokenblueprintdom.RepositoryPort,
	announcementRepo announcementdom.Repository,
) *AnnouncementManagementQuery {
	return &AnnouncementManagementQuery{
		tokenBlueprintRepo: tokenBlueprintRepo,
		announcementRepo:   announcementRepo,
	}
}

func (q *AnnouncementManagementQuery) ListByCompanyID(
	ctx context.Context,
	companyID string,
) (AnnouncementManagementQueryResult, error) {
	if q == nil {
		return AnnouncementManagementQueryResult{}, errors.New("announcement management query is nil")
	}
	if q.tokenBlueprintRepo == nil {
		return AnnouncementManagementQueryResult{}, errors.New("tokenBlueprintRepo is nil")
	}
	if q.announcementRepo == nil {
		return AnnouncementManagementQueryResult{}, errors.New("announcementRepo is nil")
	}
	if companyID == "" {
		return AnnouncementManagementQueryResult{}, errors.New("companyID is empty")
	}

	page := common.Page{
		Number:  1,
		PerPage: 1000,
	}

	tokenBlueprints, err := q.tokenBlueprintRepo.ListByCompanyID(ctx, companyID, page)
	if err != nil {
		return AnnouncementManagementQueryResult{}, err
	}

	rows := make([]AnnouncementManagementRow, 0, len(tokenBlueprints.Items))

	for _, tb := range tokenBlueprints.Items {
		if tb.ID == "" {
			continue
		}

		announcementResult, err := q.announcementRepo.ListByTargetToken(ctx, tb.ID, page)
		if err != nil {
			return AnnouncementManagementQueryResult{}, err
		}

		announcements := toAnnouncementManagementAnnouncements(announcementResult.Items)
		if len(announcements) == 0 {
			continue
		}

		rows = append(rows, AnnouncementManagementRow{
			TokenBlueprint: AnnouncementManagementTokenBlueprint{
				TokenBlueprintID: tb.ID,
				TokenName:        tb.Name,
				BrandID:          tb.BrandID,
			},
			Announcements: announcements,
		})
	}

	return AnnouncementManagementQueryResult{
		CompanyID: companyID,
		Rows:      rows,
	}, nil
}

func (q *AnnouncementManagementQuery) GetByID(
	ctx context.Context,
	announcementID string,
) (AnnouncementManagementAnnouncement, error) {
	if q == nil {
		return AnnouncementManagementAnnouncement{}, errors.New("announcement management query is nil")
	}
	if q.announcementRepo == nil {
		return AnnouncementManagementAnnouncement{}, errors.New("announcementRepo is nil")
	}
	if announcementID == "" {
		return AnnouncementManagementAnnouncement{}, announcementdom.ErrInvalidID
	}

	a, err := q.announcementRepo.GetByID(ctx, announcementID)
	if err != nil {
		return AnnouncementManagementAnnouncement{}, err
	}

	return toAnnouncementManagementAnnouncement(a), nil
}

func toAnnouncementManagementAnnouncements(
	items []announcementdom.Announcement,
) []AnnouncementManagementAnnouncement {
	if len(items) == 0 {
		return []AnnouncementManagementAnnouncement{}
	}

	result := make([]AnnouncementManagementAnnouncement, 0, len(items))

	for _, a := range items {
		result = append(result, toAnnouncementManagementAnnouncement(a))
	}

	return result
}

func toAnnouncementManagementAnnouncement(
	a announcementdom.Announcement,
) AnnouncementManagementAnnouncement {
	return AnnouncementManagementAnnouncement{
		ID:            a.ID,
		Title:         a.Title,
		Content:       a.Content,
		TargetToken:   a.TargetToken,
		TargetAvatars: a.TargetAvatars,
		Published:     a.Published,
		PublishedAt:   a.PublishedAt,
		Attachments:   a.Attachments,
		CreatedAt:     a.CreatedAt,
		CreatedBy:     a.CreatedBy,
		UpdatedAt:     a.UpdatedAt,
		UpdatedBy:     a.UpdatedBy,
	}
}
