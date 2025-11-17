// backend/internal/domain/member/invitation.go
package member

type InvitationInfo struct {
	MemberID         string
	CompanyID        string
	AssignedBrandIDs []string
	Permissions      []string
}
