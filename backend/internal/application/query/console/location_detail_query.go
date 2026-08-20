// backend/internal/application/query/console/location_detail_query.go
package query

import (
	"context"
	"errors"

	memberdom "narratives/internal/domain/member"
	shippingaddressdom "narratives/internal/domain/shippingAddress"
)

type LocationDetailQuery struct {
	locationRepo shippingaddressdom.RepositoryPort
	memberRepo   memberdom.Repository
}

func NewLocationDetailQuery(
	locationRepo shippingaddressdom.RepositoryPort,
	memberRepo memberdom.Repository,
) *LocationDetailQuery {
	return &LocationDetailQuery{
		locationRepo: locationRepo,
		memberRepo:   memberRepo,
	}
}

type LocationDetailResult struct {
	Location  shippingaddressdom.ShippingAddress
	CreatedBy string
	UpdatedBy string
}

func (q *LocationDetailQuery) GetByID(
	ctx context.Context,
	companyID string,
	locationID string,
) (LocationDetailResult, error) {
	if q == nil {
		return LocationDetailResult{}, errors.New("location detail query is nil")
	}
	if q.locationRepo == nil {
		return LocationDetailResult{}, errors.New("shippingAddress repository is nil")
	}
	if companyID == "" ||
		len([]rune(companyID)) > shippingaddressdom.MaxCompanyIDLength {
		return LocationDetailResult{}, shippingaddressdom.ErrInvalidCompanyID
	}
	if locationID == "" {
		return LocationDetailResult{}, shippingaddressdom.ErrInvalidID
	}

	location, err := q.locationRepo.GetByCompany(
		ctx,
		locationID,
		companyID,
	)
	if err != nil {
		return LocationDetailResult{}, err
	}
	if location == nil {
		return LocationDetailResult{}, shippingaddressdom.ErrNotFound
	}
	if location.CompanyID != companyID {
		return LocationDetailResult{}, shippingaddressdom.ErrNotFound
	}

	nameByMemberID := make(map[string]string)

	resolveMemberName := func(
		memberID string,
	) string {
		if memberID == "" {
			return ""
		}

		if name, ok := nameByMemberID[memberID]; ok {
			return name
		}

		if q.memberRepo == nil {
			nameByMemberID[memberID] = ""
			return ""
		}

		record, err := q.memberRepo.GetByID(
			ctx,
			memberID,
		)
		if err != nil {
			nameByMemberID[memberID] = ""
			return ""
		}

		name := memberdom.FormatLastFirst(
			record.Member.LastName,
			record.Member.FirstName,
		)

		nameByMemberID[memberID] = name
		return name
	}

	return LocationDetailResult{
		Location: *location,
		CreatedBy: resolveMemberName(
			location.CreatedBy,
		),
		UpdatedBy: resolveMemberName(
			location.UpdatedBy,
		),
	}, nil
}
