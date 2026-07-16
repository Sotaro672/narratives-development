package query

import (
	"context"
	"errors"
	"time"

	announcementdom "narratives/internal/domain/announcement"
	memberdom "narratives/internal/domain/member"
)

type AnnouncementDetailAttachmentFile struct {
	AnnouncementID string `json:"announcementId"`
	ID             string `json:"id"`
	FileName       string `json:"fileName"`
	FileURL        string `json:"fileUrl"`
	FileSize       int64  `json:"fileSize"`
	MimeType       string `json:"mimeType"`
	ObjectPath     string `json:"objectPath"`
}

type AnnouncementDetail struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	Content       string     `json:"content"`
	TargetToken   *string    `json:"targetToken,omitempty"`
	TargetAvatars []string   `json:"targetAvatars,omitempty"`
	Published     bool       `json:"published"`
	PublishedAt   *time.Time `json:"publishedAt,omitempty"`

	// Attachments is kept as the announcement attachment ID list.
	Attachments []string `json:"attachments,omitempty"`

	// AttachmentFiles contains attachment metadata stored under:
	// announcements/{announcementId}/attachments/{attachmentId}
	AttachmentFiles []AnnouncementDetailAttachmentFile `json:"attachmentFiles,omitempty"`

	CreatedAt     time.Time  `json:"createdAt"`
	CreatedBy     string     `json:"createdBy"`
	CreatedByName string     `json:"createdByName"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy     *string    `json:"updatedBy,omitempty"`
	UpdatedByName *string    `json:"updatedByName,omitempty"`
}

type AnnouncementDetailQuery struct {
	announcementRepo announcementdom.Repository
	attachmentRepo   announcementdom.AttachmentRepository
	memberRepo       memberdom.Repository
}

func NewAnnouncementDetailQuery(
	announcementRepo announcementdom.Repository,
	attachmentRepo announcementdom.AttachmentRepository,
	memberRepo memberdom.Repository,
) *AnnouncementDetailQuery {
	return &AnnouncementDetailQuery{
		announcementRepo: announcementRepo,
		attachmentRepo:   attachmentRepo,
		memberRepo:       memberRepo,
	}
}

func (q *AnnouncementDetailQuery) GetByID(
	ctx context.Context,
	announcementID string,
) (AnnouncementDetail, error) {
	if q == nil {
		return AnnouncementDetail{}, errors.New(
			"announcement detail query is nil",
		)
	}
	if q.announcementRepo == nil {
		return AnnouncementDetail{}, errors.New(
			"announcementRepo is nil",
		)
	}
	if q.attachmentRepo == nil {
		return AnnouncementDetail{}, errors.New(
			"attachmentRepo is nil",
		)
	}
	if q.memberRepo == nil {
		return AnnouncementDetail{}, errors.New(
			"memberRepo is nil",
		)
	}

	if announcementID == "" {
		return AnnouncementDetail{}, announcementdom.ErrInvalidID
	}

	a, err := q.announcementRepo.GetByID(
		ctx,
		announcementID,
	)
	if err != nil {
		return AnnouncementDetail{}, err
	}

	return q.toAnnouncementDetail(ctx, a)
}

func (q *AnnouncementDetailQuery) toAnnouncementDetail(
	ctx context.Context,
	a announcementdom.Announcement,
) (AnnouncementDetail, error) {
	createdByName, err := q.resolveMemberNameByID(
		ctx,
		a.CreatedBy,
	)
	if err != nil {
		return AnnouncementDetail{}, err
	}

	var updatedByName *string
	if a.UpdatedBy != nil {
		updatedByID := *a.UpdatedBy
		if updatedByID != "" {
			name, err := q.resolveMemberNameByID(
				ctx,
				updatedByID,
			)
			if err != nil {
				return AnnouncementDetail{}, err
			}

			updatedByName = &name
		}
	}

	attachmentFiles, err := q.resolveAttachmentFiles(
		ctx,
		a.ID,
	)
	if err != nil {
		return AnnouncementDetail{}, err
	}

	return AnnouncementDetail{
		ID:              a.ID,
		Title:           a.Title,
		Content:         a.Content,
		TargetToken:     a.TargetToken,
		TargetAvatars:   normalizeStringSlice(a.TargetAvatars),
		Published:       a.Published,
		PublishedAt:     a.PublishedAt,
		Attachments:     normalizeStringSlice(a.Attachments),
		AttachmentFiles: attachmentFiles,
		CreatedAt:       a.CreatedAt,
		CreatedBy:       a.CreatedBy,
		CreatedByName:   createdByName,
		UpdatedAt:       a.UpdatedAt,
		UpdatedBy:       a.UpdatedBy,
		UpdatedByName:   updatedByName,
	}, nil
}

func (q *AnnouncementDetailQuery) resolveMemberNameByID(
	ctx context.Context,
	memberID string,
) (string, error) {
	if memberID == "" {
		return "", nil
	}
	if q == nil {
		return "", errors.New(
			"announcement detail query is nil",
		)
	}
	if q.memberRepo == nil {
		return "", errors.New(
			"memberRepo is nil",
		)
	}

	rec, err := q.memberRepo.GetByID(
		ctx,
		memberID,
	)
	if err != nil {
		return "", err
	}

	name := memberdom.FormatLastFirst(
		rec.Member.LastName,
		rec.Member.FirstName,
	)
	if name != "" {
		return name, nil
	}

	return memberID, nil
}

func (q *AnnouncementDetailQuery) resolveAttachmentFiles(
	ctx context.Context,
	announcementID string,
) ([]AnnouncementDetailAttachmentFile, error) {
	if announcementID == "" {
		return []AnnouncementDetailAttachmentFile{}, nil
	}
	if q == nil {
		return nil, errors.New(
			"announcement detail query is nil",
		)
	}
	if q.attachmentRepo == nil {
		return nil, errors.New(
			"attachmentRepo is nil",
		)
	}

	files, err := q.attachmentRepo.ListByAnnouncementID(
		ctx,
		announcementID,
	)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return []AnnouncementDetailAttachmentFile{}, nil
	}

	result := make(
		[]AnnouncementDetailAttachmentFile,
		0,
		len(files),
	)

	for _, file := range files {
		id := file.ID
		fileName := file.FileName
		fileURL := file.FileURL
		mimeType := file.MimeType
		objectPath := file.ObjectPath

		if id == "" &&
			fileName == "" &&
			fileURL == "" &&
			objectPath == "" {
			continue
		}

		result = append(
			result,
			AnnouncementDetailAttachmentFile{
				AnnouncementID: file.AnnouncementID,
				ID:             id,
				FileName:       fileName,
				FileURL:        fileURL,
				FileSize:       file.FileSize,
				MimeType:       mimeType,
				ObjectPath:     objectPath,
			},
		)
	}

	return result, nil
}
