package productBlueprint

import (
	"context"
	"strings"
)

// brandId → brandName 解決用ヘルパ
func (h *Handler) getBrandNameByID(ctx context.Context, brandID string) string {
	brandID = strings.TrimSpace(brandID)
	if brandID == "" {
		return ""
	}
	if h.brandSvc == nil {
		return brandID
	}

	name, err := h.brandSvc.GetNameByID(ctx, brandID)
	if err != nil {
		return brandID
	}
	return strings.TrimSpace(name)
}

// assigneeId → assigneeName 解決用ヘルパ
func (h *Handler) getAssigneeNameByID(ctx context.Context, memberID string) string {
	memberID = strings.TrimSpace(memberID)
	if memberID == "" {
		return ""
	}
	if h.memberSvc == nil {
		return memberID
	}

	name, err := h.memberSvc.GetNameLastFirstByID(ctx, memberID)
	if err != nil {
		return memberID
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return memberID
	}
	return name
}
