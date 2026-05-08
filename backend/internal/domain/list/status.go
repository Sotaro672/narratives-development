// backend/internal/domain/list/status.go
package list

// ListStatus mirrors TS: 'listing' | 'suspended' | 'deleted'
type ListStatus string

const (
	StatusListing   ListStatus = "listing"
	StatusSuspended ListStatus = "suspended"
	StatusDeleted   ListStatus = "deleted"
)

func IsValidStatus(s ListStatus) bool {
	switch s {
	case StatusListing, StatusSuspended, StatusDeleted:
		return true
	default:
		return false
	}
}
