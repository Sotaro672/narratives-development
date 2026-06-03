// backend/internal/domain/invitation/entity.go
package invitation

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvitationTokenNotFound = errors.New("invitation: token not found")
)

// InvitationToken is the Firestore document model for invitation tokens.
type InvitationToken struct {
	Token            string   `firestore:"token"`
	MemberID         string   `firestore:"memberId"`
	CompanyID        string   `firestore:"companyId"`
	AssignedBrandIDs []string `firestore:"assignedBrands"`
	Permissions      []string `firestore:"permissions"`
	Email            string   `firestore:"email"`

	CreatedAt time.Time  `firestore:"createdAt"`
	ExpiresAt *time.Time `firestore:"expiresAt,omitempty"`
	UsedAt    *time.Time `firestore:"usedAt,omitempty"`
	UpdatedAt *time.Time `firestore:"updatedAt,omitempty"`
}

// InvitationInfo is the read model used by invitation validate / complete flow.
type InvitationInfo struct {
	MemberID         string   `json:"memberId"`
	CompanyID        string   `json:"companyId"`
	AssignedBrandIDs []string `json:"assignedBrandIds"`
	Permissions      []string `json:"permissions"`
	Email            string   `json:"email"`
}

type Repository interface {
	CreateInvitationToken(ctx context.Context, info InvitationInfo) (string, error)
	ResolveInvitationInfoByToken(ctx context.Context, token string) (InvitationInfo, error)
	ConsumeInvitationToken(ctx context.Context, token string) error
}
