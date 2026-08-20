// backend/internal/application/query/console/transportation_detail_query.go
package query

import (
	"context"
	"errors"

	memberdom "narratives/internal/domain/member"
	transportationdom "narratives/internal/domain/transportation"
)

type TransportationDetailQuery struct {
	transportationRepo transportationdom.RepositoryPort
	memberRepo         memberdom.Repository
}

func NewTransportationDetailQuery(
	transportationRepo transportationdom.RepositoryPort,
	memberRepo memberdom.Repository,
) *TransportationDetailQuery {
	return &TransportationDetailQuery{
		transportationRepo: transportationRepo,
		memberRepo:         memberRepo,
	}
}

type TransportationDetailResult struct {
	Transportation transportationdom.TransportationFeeSetting
	CreatedByName  string
	UpdatedByName  string
}

func (q *TransportationDetailQuery) GetByID(
	ctx context.Context,
	companyID string,
	transportationID string,
) (TransportationDetailResult, error) {
	if q == nil {
		return TransportationDetailResult{}, errors.New("transportation detail query is nil")
	}
	if q.transportationRepo == nil {
		return TransportationDetailResult{}, errors.New("transportation repository is nil")
	}
	if companyID == "" || len([]rune(companyID)) > transportationdom.MaxCompanyIDLength {
		return TransportationDetailResult{}, transportationdom.ErrInvalidCompanyID
	}
	if transportationID == "" || len([]rune(transportationID)) > transportationdom.MaxTransportationIDLength {
		return TransportationDetailResult{}, transportationdom.ErrInvalidID
	}

	setting, err := q.transportationRepo.GetByID(
		ctx,
		transportationID,
	)
	if err != nil {
		return TransportationDetailResult{}, err
	}
	if setting == nil {
		return TransportationDetailResult{}, transportationdom.ErrNotFound
	}
	if setting.CompanyID != companyID {
		return TransportationDetailResult{}, transportationdom.ErrNotFound
	}

	nameByMemberID := resolveMemberNamesByID(
		ctx,
		q.memberRepo,
		[]string{
			setting.CreatedBy,
			setting.UpdatedBy,
		},
	)

	return TransportationDetailResult{
		Transportation: *setting,
		CreatedByName:  nameByMemberID[setting.CreatedBy],
		UpdatedByName:  nameByMemberID[setting.UpdatedBy],
	}, nil
}
