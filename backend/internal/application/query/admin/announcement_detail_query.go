// backend/internal/application/query/admin/announcement_detail_query.go
package query

import (
	"context"
	"errors"

	announcementdom "narratives/internal/domain/announcement"
	companydom "narratives/internal/domain/company"
	memberdom "narratives/internal/domain/member"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

var ErrContractAnnouncementDetailQueryNotConfigured = errors.New(
	"contract announcement detail query is not configured",
)

type contractAnnouncementDetailCompanyReader interface {
	GetByID(ctx context.Context, id string) (companydom.Company, error)
}

type contractAnnouncementDetailReader interface {
	GetByID(ctx context.Context, id string) (announcementdom.Announcement, error)
}

type contractAnnouncementDetailAttachmentReader interface {
	ListByAnnouncementID(
		ctx context.Context,
		announcementID string,
	) ([]announcementdom.AttachmentFile, error)
}

type contractAnnouncementDetailTokenBlueprintReader interface {
	GetByID(
		ctx context.Context,
		id string,
	) (*tokenblueprintdom.TokenBlueprint, error)
}

type contractAnnouncementDetailMemberReader interface {
	GetByID(ctx context.Context, id string) (memberdom.Record, error)
	GetByUID(ctx context.Context, uid string) (memberdom.Record, error)
}

type ContractAnnouncementDetailQuery struct {
	companyRepo        contractAnnouncementDetailCompanyReader
	announcementRepo   contractAnnouncementDetailReader
	attachmentRepo     contractAnnouncementDetailAttachmentReader
	tokenBlueprintRepo contractAnnouncementDetailTokenBlueprintReader
	memberRepo         contractAnnouncementDetailMemberReader
}

func NewContractAnnouncementDetailQuery(
	companyRepo contractAnnouncementDetailCompanyReader,
	announcementRepo contractAnnouncementDetailReader,
	attachmentRepo contractAnnouncementDetailAttachmentReader,
	tokenBlueprintRepo contractAnnouncementDetailTokenBlueprintReader,
	memberRepo contractAnnouncementDetailMemberReader,
) *ContractAnnouncementDetailQuery {
	return &ContractAnnouncementDetailQuery{
		companyRepo:        companyRepo,
		announcementRepo:   announcementRepo,
		attachmentRepo:     attachmentRepo,
		tokenBlueprintRepo: tokenBlueprintRepo,
		memberRepo:         memberRepo,
	}
}

type ContractAnnouncementDetailResult struct {
	ID              string                                     `json:"id"`
	Title           string                                     `json:"title"`
	Content         string                                     `json:"content"`
	TargetToken     *string                                    `json:"targetToken,omitempty"`
	TokenName       string                                     `json:"tokenName"`
	TargetAvatars   []string                                   `json:"targetAvatars,omitempty"`
	Published       bool                                       `json:"published"`
	PublishedAt     string                                     `json:"publishedAt,omitempty"`
	Attachments     []string                                   `json:"attachments,omitempty"`
	AttachmentFiles []ContractAnnouncementDetailAttachmentFile `json:"attachmentFiles,omitempty"`
	CreatedAt       string                                     `json:"createdAt"`
	CreatedBy       string                                     `json:"createdBy"`
	CreatedByName   string                                     `json:"createdByName"`
	UpdatedAt       string                                     `json:"updatedAt,omitempty"`
	UpdatedBy       *string                                    `json:"updatedBy,omitempty"`
	UpdatedByName   *string                                    `json:"updatedByName,omitempty"`
}

type ContractAnnouncementDetailAttachmentFile struct {
	AnnouncementID string `json:"announcementId"`
	ID             string `json:"id"`
	FileName       string `json:"fileName"`
	FileURL        string `json:"fileUrl"`
	FileSize       int64  `json:"fileSize"`
	MimeType       string `json:"mimeType"`
	ObjectPath     string `json:"objectPath"`
}

func (q *ContractAnnouncementDetailQuery) Get(
	ctx context.Context,
	companyID string,
	announcementID string,
) (ContractAnnouncementDetailResult, error) {
	if q == nil ||
		q.companyRepo == nil ||
		q.announcementRepo == nil ||
		q.attachmentRepo == nil ||
		q.tokenBlueprintRepo == nil ||
		q.memberRepo == nil {
		return ContractAnnouncementDetailResult{},
			ErrContractAnnouncementDetailQueryNotConfigured
	}

	if companyID == "" {
		return ContractAnnouncementDetailResult{}, companydom.ErrInvalidID
	}
	if announcementID == "" {
		return ContractAnnouncementDetailResult{}, announcementdom.ErrInvalidID
	}

	if _, err := q.companyRepo.GetByID(ctx, companyID); err != nil {
		return ContractAnnouncementDetailResult{}, err
	}

	announcement, err := q.announcementRepo.GetByID(ctx, announcementID)
	if err != nil {
		return ContractAnnouncementDetailResult{}, err
	}
	if announcement.ID == "" || announcement.ID != announcementID {
		return ContractAnnouncementDetailResult{}, announcementdom.ErrNotFound
	}

	if announcement.TargetToken == nil || *announcement.TargetToken == "" {
		return ContractAnnouncementDetailResult{}, announcementdom.ErrNotFound
	}

	tokenBlueprint, err := q.tokenBlueprintRepo.GetByID(
		ctx,
		*announcement.TargetToken,
	)
	if err != nil {
		if errors.Is(err, tokenblueprintdom.ErrNotFound) {
			return ContractAnnouncementDetailResult{},
				announcementdom.ErrNotFound
		}
		return ContractAnnouncementDetailResult{}, err
	}
	if tokenBlueprint == nil ||
		tokenBlueprint.ID != *announcement.TargetToken ||
		tokenBlueprint.CompanyID != companyID {
		return ContractAnnouncementDetailResult{},
			announcementdom.ErrNotFound
	}

	attachmentFiles, err := q.attachmentRepo.ListByAnnouncementID(
		ctx,
		announcement.ID,
	)
	if err != nil {
		return ContractAnnouncementDetailResult{}, err
	}

	attachments := make(
		[]ContractAnnouncementDetailAttachmentFile,
		0,
		len(attachmentFiles),
	)
	for _, file := range attachmentFiles {
		if file.ID == "" && file.FileURL == "" && file.ObjectPath == "" {
			continue
		}

		attachments = append(
			attachments,
			ContractAnnouncementDetailAttachmentFile{
				AnnouncementID: file.AnnouncementID,
				ID:             file.ID,
				FileName:       file.FileName,
				FileURL:        file.FileURL,
				FileSize:       file.FileSize,
				MimeType:       file.MimeType,
				ObjectPath:     file.ObjectPath,
			},
		)
	}

	createdByName := q.resolveMemberName(
		ctx,
		announcement.CreatedBy,
	)

	var updatedByName *string
	if announcement.UpdatedBy != nil && *announcement.UpdatedBy != "" {
		name := q.resolveMemberName(
			ctx,
			*announcement.UpdatedBy,
		)
		updatedByName = &name
	}

	return ContractAnnouncementDetailResult{
		ID:              announcement.ID,
		Title:           announcement.Title,
		Content:         announcement.Content,
		TargetToken:     announcement.TargetToken,
		TokenName:       tokenBlueprint.Name,
		TargetAvatars:   cloneAnnouncementDetailStrings(announcement.TargetAvatars),
		Published:       announcement.Published,
		PublishedAt:     formatOptionalContractDetailTime(announcement.PublishedAt),
		Attachments:     cloneAnnouncementDetailStrings(announcement.Attachments),
		AttachmentFiles: attachments,
		CreatedAt:       formatContractDetailTime(announcement.CreatedAt),
		CreatedBy:       announcement.CreatedBy,
		CreatedByName:   createdByName,
		UpdatedAt:       formatOptionalContractDetailTime(announcement.UpdatedAt),
		UpdatedBy:       announcement.UpdatedBy,
		UpdatedByName:   updatedByName,
	}, nil
}

func (q *ContractAnnouncementDetailQuery) resolveMemberName(
	ctx context.Context,
	memberID string,
) string {
	if memberID == "" {
		return "-"
	}

	record, err := q.memberRepo.GetByID(ctx, memberID)
	if err == nil {
		name := memberdom.FormatLastFirst(
			record.Member.LastName,
			record.Member.FirstName,
		)
		if name != "" {
			return name
		}
	}

	record, err = q.memberRepo.GetByUID(ctx, memberID)
	if err == nil {
		name := memberdom.FormatLastFirst(
			record.Member.LastName,
			record.Member.FirstName,
		)
		if name != "" {
			return name
		}
	}

	return memberID
}

func cloneAnnouncementDetailStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		result = append(result, value)
	}

	return result
}
