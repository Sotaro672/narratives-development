// backend/internal/domain/member/invitation.go

package member

import "time"

type InvitationToken struct {
	Token            string   `firestore:"token"`
	MemberID         string   `firestore:"memberId"`
	CompanyID        string   `firestore:"companyId"`
	AssignedBrandIDs []string `firestore:"assignedBrands"`
	Permissions      []string `firestore:"permissions"`
	Email            string   `firestore:"email"` // ★ 新規追加

	CreatedAt time.Time  `firestore:"createdAt"`
	ExpiresAt *time.Time `firestore:"expiresAt,omitempty"`
	UsedAt    *time.Time `firestore:"usedAt,omitempty"`
	UpdatedAt *time.Time `firestore:"updatedAt,omitempty"`
}

// 招待リンク表示用
type InvitationInfo struct {
	MemberID         string   `json:"memberId"`
	CompanyID        string   `json:"companyId"`
	AssignedBrandIDs []string `json:"assignedBrandIds"`
	Permissions      []string `json:"permissions"`
	Email            string   `json:"email"` // ★ 追加
}
