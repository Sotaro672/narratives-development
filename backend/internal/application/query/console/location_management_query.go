// backend/internal/application/query/console/location_management_query.go
package query

import (
	"context"
	"errors"

	memberdom "narratives/internal/domain/member"
	shippingaddressdom "narratives/internal/domain/shippingAddress"
)

type LocationMemberNames struct {
	CreatedBy string
	UpdatedBy string
}

type LocationWithMemberNames struct {
	Location    shippingaddressdom.ShippingAddress
	MemberNames LocationMemberNames
}

type LocationManagementQuery struct {
	locationRepo shippingaddressdom.RepositoryPort
	memberRepo   memberdom.Repository
}

func NewLocationManagementQuery(
	locationRepo shippingaddressdom.RepositoryPort,
	memberRepo memberdom.Repository,
) *LocationManagementQuery {
	return &LocationManagementQuery{
		locationRepo: locationRepo,
		memberRepo:   memberRepo,
	}
}

func (q *LocationManagementQuery) ListByCompanyID(
	ctx context.Context,
	companyID string,
) ([]LocationWithMemberNames, error) {
	if q == nil {
		return nil, errors.New("location management query is nil")
	}
	if q.locationRepo == nil {
		return nil, errors.New("shippingAddress repository is nil")
	}
	if companyID == "" {
		return nil, shippingaddressdom.ErrInvalidCompanyID
	}

	locations, err := q.locationRepo.ListByCompanyID(
		ctx,
		companyID,
	)
	if err != nil {
		return nil, err
	}
	if locations == nil {
		return []LocationWithMemberNames{}, nil
	}

	for i := range locations {
		if locations[i].CompanyID != companyID {
			return nil, errors.New(
				"shippingAddress repository returned location owned by another company",
			)
		}
	}

	return q.attachMemberNames(
		ctx,
		locations,
	), nil
}

func (q *LocationManagementQuery) attachMemberNames(
	ctx context.Context,
	locations []shippingaddressdom.ShippingAddress,
) []LocationWithMemberNames {
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

	items := make(
		[]LocationWithMemberNames,
		0,
		len(locations),
	)

	for i := range locations {
		location := locations[i]

		items = append(
			items,
			LocationWithMemberNames{
				Location: location,
				MemberNames: LocationMemberNames{
					CreatedBy: resolveMemberName(
						location.CreatedBy,
					),
					UpdatedBy: resolveMemberName(
						location.UpdatedBy,
					),
				},
			},
		)
	}

	return items
}
