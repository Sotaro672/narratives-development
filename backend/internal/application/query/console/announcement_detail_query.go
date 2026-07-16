package query

import (
	"context"
	"errors"
	"strings"
	"time"

	appresolver "narratives/internal/application/resolver"
	announcementdom "narratives/internal/domain/announcement"
	memberdom "narratives/internal/domain/member"
	tokendom "narratives/internal/domain/token"
)

type AnnouncementDetailProductBlueprint struct {
	ProductBlueprintID string `json:"productBlueprintId"`
	ProductName        string `json:"productName"`
}

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
	ID                string                               `json:"id"`
	Title             string                               `json:"title"`
	Content           string                               `json:"content"`
	TargetToken       *string                              `json:"targetToken,omitempty"`
	TargetAvatars     []string                             `json:"targetAvatars,omitempty"`
	MintAddresses     []string                             `json:"mintAddresses,omitempty"`
	ModelIDs          []string                             `json:"modelIds,omitempty"`
	ProductBlueprints []AnnouncementDetailProductBlueprint `json:"productBlueprints,omitempty"`
	Published         bool                                 `json:"published"`
	PublishedAt       *time.Time                           `json:"publishedAt,omitempty"`

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

type announcementDetailMintReader interface {
	ListMintAddressesByTokenBlueprintID(
		ctx context.Context,
		tokenBlueprintID string,
	) (tokendom.ListMintAddressesByTokenBlueprintIDResult, error)
}

type announcementDetailMintProductBlueprintResolver interface {
	ResolveByMintAddresses(
		ctx context.Context,
		mintAddresses []string,
	) (appresolver.MintProductBlueprintResolveResult, error)
}

type AnnouncementDetailQuery struct {
	announcementRepo             announcementdom.Repository
	attachmentRepo               announcementdom.AttachmentRepository
	memberRepo                   memberdom.Repository
	mintRepo                     announcementDetailMintReader
	mintProductBlueprintResolver announcementDetailMintProductBlueprintResolver
}

func NewAnnouncementDetailQuery(
	announcementRepo announcementdom.Repository,
	attachmentRepo announcementdom.AttachmentRepository,
	memberRepo memberdom.Repository,
	mintRepo announcementDetailMintReader,
	mintProductBlueprintResolver announcementDetailMintProductBlueprintResolver,
) *AnnouncementDetailQuery {
	return &AnnouncementDetailQuery{
		announcementRepo:             announcementRepo,
		attachmentRepo:               attachmentRepo,
		memberRepo:                   memberRepo,
		mintRepo:                     mintRepo,
		mintProductBlueprintResolver: mintProductBlueprintResolver,
	}
}

func (q *AnnouncementDetailQuery) GetByID(
	ctx context.Context,
	announcementID string,
) (AnnouncementDetail, error) {
	if q == nil {
		return AnnouncementDetail{}, errors.New("announcement detail query is nil")
	}
	if q.announcementRepo == nil {
		return AnnouncementDetail{}, errors.New("announcementRepo is nil")
	}
	if q.attachmentRepo == nil {
		return AnnouncementDetail{}, errors.New("attachmentRepo is nil")
	}
	if q.memberRepo == nil {
		return AnnouncementDetail{}, errors.New("memberRepo is nil")
	}
	if q.mintRepo == nil {
		return AnnouncementDetail{}, errors.New("mintRepo is nil")
	}
	if q.mintProductBlueprintResolver == nil {
		return AnnouncementDetail{}, errors.New("mintProductBlueprintResolver is nil")
	}

	announcementID = strings.TrimSpace(announcementID)
	if announcementID == "" {
		return AnnouncementDetail{}, announcementdom.ErrInvalidID
	}

	a, err := q.announcementRepo.GetByID(ctx, announcementID)
	if err != nil {
		return AnnouncementDetail{}, err
	}

	return q.toAnnouncementDetail(ctx, a)
}

func (q *AnnouncementDetailQuery) toAnnouncementDetail(
	ctx context.Context,
	a announcementdom.Announcement,
) (AnnouncementDetail, error) {
	createdByName, err := q.resolveMemberNameByID(ctx, a.CreatedBy)
	if err != nil {
		return AnnouncementDetail{}, err
	}

	var updatedByName *string
	if a.UpdatedBy != nil {
		updatedByID := strings.TrimSpace(*a.UpdatedBy)
		if updatedByID != "" {
			name, err := q.resolveMemberNameByID(ctx, updatedByID)
			if err != nil {
				return AnnouncementDetail{}, err
			}
			updatedByName = &name
		}
	}

	mintAddresses, modelIDs, productBlueprints, err :=
		q.resolveProductBlueprintsByTargetToken(ctx, a.TargetToken)
	if err != nil {
		return AnnouncementDetail{}, err
	}

	attachmentFiles, err := q.resolveAttachmentFiles(ctx, a.ID)
	if err != nil {
		return AnnouncementDetail{}, err
	}

	return AnnouncementDetail{
		ID:                a.ID,
		Title:             a.Title,
		Content:           a.Content,
		TargetToken:       a.TargetToken,
		TargetAvatars:     normalizeStringSlice(a.TargetAvatars),
		MintAddresses:     mintAddresses,
		ModelIDs:          modelIDs,
		ProductBlueprints: productBlueprints,
		Published:         a.Published,
		PublishedAt:       a.PublishedAt,
		Attachments:       normalizeStringSlice(a.Attachments),
		AttachmentFiles:   attachmentFiles,
		CreatedAt:         a.CreatedAt,
		CreatedBy:         a.CreatedBy,
		CreatedByName:     createdByName,
		UpdatedAt:         a.UpdatedAt,
		UpdatedBy:         a.UpdatedBy,
		UpdatedByName:     updatedByName,
	}, nil
}

func (q *AnnouncementDetailQuery) resolveMemberNameByID(
	ctx context.Context,
	memberID string,
) (string, error) {
	memberID = strings.TrimSpace(memberID)
	if memberID == "" {
		return "", nil
	}
	if q == nil {
		return "", errors.New("announcement detail query is nil")
	}
	if q.memberRepo == nil {
		return "", errors.New("memberRepo is nil")
	}

	rec, err := q.memberRepo.GetByID(ctx, memberID)
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
	announcementID = strings.TrimSpace(announcementID)
	if announcementID == "" {
		return []AnnouncementDetailAttachmentFile{}, nil
	}
	if q == nil {
		return nil, errors.New("announcement detail query is nil")
	}
	if q.attachmentRepo == nil {
		return nil, errors.New("attachmentRepo is nil")
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
		id := strings.TrimSpace(file.ID)
		fileName := strings.TrimSpace(file.FileName)
		fileURL := strings.TrimSpace(file.FileURL)
		mimeType := strings.TrimSpace(file.MimeType)
		objectPath := strings.TrimSpace(file.ObjectPath)

		if id == "" &&
			fileName == "" &&
			fileURL == "" &&
			objectPath == "" {
			continue
		}

		result = append(
			result,
			AnnouncementDetailAttachmentFile{
				AnnouncementID: strings.TrimSpace(file.AnnouncementID),
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

func (q *AnnouncementDetailQuery) resolveProductBlueprintsByTargetToken(
	ctx context.Context,
	targetToken *string,
) (
	[]string,
	[]string,
	[]AnnouncementDetailProductBlueprint,
	error,
) {
	if targetToken == nil {
		return []string{},
			[]string{},
			[]AnnouncementDetailProductBlueprint{},
			nil
	}

	tokenBlueprintID := strings.TrimSpace(*targetToken)
	if tokenBlueprintID == "" {
		return []string{},
			[]string{},
			[]AnnouncementDetailProductBlueprint{},
			nil
	}
	if q == nil {
		return nil, nil, nil, errors.New(
			"announcement detail query is nil",
		)
	}
	if q.mintRepo == nil {
		return nil, nil, nil, errors.New("mintRepo is nil")
	}
	if q.mintProductBlueprintResolver == nil {
		return nil, nil, nil, errors.New(
			"mintProductBlueprintResolver is nil",
		)
	}

	mintResult, err := q.mintRepo.ListMintAddressesByTokenBlueprintID(
		ctx,
		tokenBlueprintID,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	mintAddresses := normalizeStringSlice(
		mintResult.MintAddresses,
	)

	resolved, err :=
		q.mintProductBlueprintResolver.ResolveByMintAddresses(
			ctx,
			mintAddresses,
		)
	if err != nil {
		return nil, nil, nil, err
	}

	productBlueprints := make(
		[]AnnouncementDetailProductBlueprint,
		0,
		len(resolved.ProductBlueprints),
	)

	for _, pb := range resolved.ProductBlueprints {
		productBlueprintID := strings.TrimSpace(
			pb.ProductBlueprintID,
		)
		productName := strings.TrimSpace(
			pb.ProductName,
		)

		if productBlueprintID == "" {
			continue
		}

		productBlueprints = append(
			productBlueprints,
			AnnouncementDetailProductBlueprint{
				ProductBlueprintID: productBlueprintID,
				ProductName:        productName,
			},
		)
	}

	return mintAddresses,
		normalizeStringSlice(resolved.ModelIDs),
		productBlueprints,
		nil
}
