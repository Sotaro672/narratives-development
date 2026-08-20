// backend/internal/application/query/console/transportation_management_query.go
package query

import (
	"context"
	"errors"

	memberdom "narratives/internal/domain/member"
	transportationdom "narratives/internal/domain/transportation"
)

type TransportationMemberNames struct {
	CreatedByName string
	UpdatedByName string
}

type TransportationWithMemberNames struct {
	Transportation transportationdom.TransportationFeeSetting
	MemberNames    TransportationMemberNames
}

type TransportationManagementQuery struct {
	transportationRepo transportationdom.RepositoryPort
	memberRepo         memberdom.Repository
}

func NewTransportationManagementQuery(
	transportationRepo transportationdom.RepositoryPort,
	memberRepo memberdom.Repository,
) *TransportationManagementQuery {
	return &TransportationManagementQuery{
		transportationRepo: transportationRepo,
		memberRepo:         memberRepo,
	}
}

func (q *TransportationManagementQuery) ListByCompanyID(
	ctx context.Context,
	companyID string,
) ([]TransportationWithMemberNames, error) {
	if q == nil {
		return nil, errors.New("transportation management query is nil")
	}
	if q.transportationRepo == nil {
		return nil, errors.New("transportation repository is nil")
	}
	if companyID == "" {
		return nil, transportationdom.ErrInvalidCompanyID
	}

	settings, err := q.transportationRepo.ListByCompanyID(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		return []TransportationWithMemberNames{}, nil
	}

	for i := range settings {
		if settings[i].CompanyID != companyID {
			return nil, errors.New("transportation repository returned setting owned by another company")
		}
	}

	return q.attachResolvedNames(ctx, settings), nil
}

func (q *TransportationManagementQuery) attachResolvedNames(
	ctx context.Context,
	settings []transportationdom.TransportationFeeSetting,
) []TransportationWithMemberNames {
	memberIDs := make([]string, 0, len(settings)*2)

	for i := range settings {
		memberIDs = append(
			memberIDs,
			settings[i].CreatedBy,
			settings[i].UpdatedBy,
		)
	}

	nameByMemberID := resolveMemberNamesByID(
		ctx,
		q.memberRepo,
		memberIDs,
	)

	items := make([]TransportationWithMemberNames, 0, len(settings))

	for i := range settings {
		setting := settings[i]

		items = append(items, TransportationWithMemberNames{
			Transportation: setting,
			MemberNames: TransportationMemberNames{
				CreatedByName: nameByMemberID[setting.CreatedBy],
				UpdatedByName: nameByMemberID[setting.UpdatedBy],
			},
		})
	}

	return items
}
