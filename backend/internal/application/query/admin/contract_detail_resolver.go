// backend/internal/application/query/admin/contract_detail_resolver.go
package query

import (
	"context"

	memberdom "narratives/internal/domain/member"
)

func (q *ContractDetailQuery) resolveBrandName(
	ctx context.Context,
	brandID string,
	cache map[string]string,
) string {
	if brandID == "" {
		return ""
	}

	if cached, ok := cache[brandID]; ok {
		return cached
	}

	name := brandID

	brand, err := q.brandRepo.GetByID(
		ctx,
		brandID,
	)
	if err == nil && brand.Name != "" {
		name = brand.Name
	}

	cache[brandID] = name
	return name
}

func (q *ContractDetailQuery) resolveMemberName(
	ctx context.Context,
	memberID string,
	cache map[string]string,
) string {
	if memberID == "" {
		return ""
	}

	if cached, ok := cache[memberID]; ok {
		return cached
	}

	name := memberID

	if record, err := q.memberRepo.GetByID(
		ctx,
		memberID,
	); err == nil {
		if resolved := memberdom.FormatLastFirst(
			record.Member.LastName,
			record.Member.FirstName,
		); resolved != "" {
			name = resolved
		}
	} else if record, uidErr := q.memberRepo.GetByUID(
		ctx,
		memberID,
	); uidErr == nil {
		if resolved := memberdom.FormatLastFirst(
			record.Member.LastName,
			record.Member.FirstName,
		); resolved != "" {
			name = resolved
		}
	}

	cache[memberID] = name
	return name
}
