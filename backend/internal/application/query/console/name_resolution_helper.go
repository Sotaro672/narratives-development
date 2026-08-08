// backend/internal/application/query/console/name_resolution_helper.go
package query

import (
	"context"

	branddom "narratives/internal/domain/brand"
	memberdom "narratives/internal/domain/member"
)

func uniqueNonEmptyIDs(ids []string) []string {
	if len(ids) == 0 {
		return []string{}
	}

	out := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))

	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}

		seen[id] = struct{}{}
		out = append(out, id)
	}

	return out
}

func resolveMemberNameByID(
	ctx context.Context,
	repo memberdom.Repository,
	memberID *string,
) string {
	if repo == nil {
		return ""
	}

	if memberID == nil || *memberID == "" {
		return ""
	}

	rec, err := repo.GetByID(ctx, *memberID)
	if err != nil {
		return ""
	}

	return memberdom.FormatLastFirst(
		rec.Member.LastName,
		rec.Member.FirstName,
	)
}

func resolveMemberNamesByID(
	ctx context.Context,
	repo memberdom.Repository,
	ids []string,
) map[string]string {
	uniq := uniqueNonEmptyIDs(ids)
	out := make(map[string]string, len(uniq))

	if repo == nil {
		for _, id := range uniq {
			out[id] = ""
		}

		return out
	}

	for _, id := range uniq {
		memberID := id

		out[id] = resolveMemberNameByID(
			ctx,
			repo,
			&memberID,
		)
	}

	return out
}

func resolveBrandNamesByID(
	ctx context.Context,
	repo branddom.Repository,
	ids []string,
) map[string]string {
	uniq := uniqueNonEmptyIDs(ids)
	out := make(map[string]string, len(uniq))

	if repo == nil {
		for _, id := range uniq {
			out[id] = ""
		}

		return out
	}

	for _, id := range uniq {
		brand, err := repo.GetByID(ctx, id)
		if err != nil {
			out[id] = ""
			continue
		}

		out[id] = brand.Name
	}

	return out
}
