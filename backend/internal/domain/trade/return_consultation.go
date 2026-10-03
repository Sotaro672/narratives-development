// backend/internal/domain/trade/return_consultation.go

package trade

import (
	"errors"
	"strings"
	"time"
)

type ReturnConsultationReason string

const (
	ReturnConsultationReasonNotAsDescribed ReturnConsultationReason = "not_as_described"
	ReturnConsultationReasonDamaged        ReturnConsultationReason = "damaged"
	ReturnConsultationReasonWrongItem      ReturnConsultationReason = "wrong_item"
	ReturnConsultationReasonOther          ReturnConsultationReason = "other"
)

const (
	MaxReturnConsultationDetailLength = 5000
)

var (
	ErrInvalidReturnConsultationID = errors.New(
		"trade: invalid return consultation id",
	)
	ErrInvalidReturnConsultationReason = errors.New(
		"trade: invalid return consultation reason",
	)
	ErrInvalidReturnConsultationDetail = errors.New(
		"trade: invalid return consultation detail",
	)
	ErrInvalidReturnConsultationCreatedAt = errors.New(
		"trade: invalid return consultation createdAt",
	)
)

// ReturnConsultation is the buyer's initial reason for starting a return
// discussion.
type ReturnConsultation struct {
	ID        string                   `json:"id"`
	Reason    ReturnConsultationReason `json:"reason"`
	Detail    string                   `json:"detail"`
	CreatedAt time.Time                `json:"createdAt"`
}

func (c ReturnConsultation) ValidateForCreate() error {
	if c.ID != "" && !isValidReferenceID(c.ID) {
		return ErrInvalidReturnConsultationID
	}
	if !IsValidReturnConsultationReason(c.Reason) {
		return ErrInvalidReturnConsultationReason
	}

	detail := strings.TrimSpace(c.Detail)
	if detail == "" ||
		len([]rune(detail)) > MaxReturnConsultationDetailLength {
		return ErrInvalidReturnConsultationDetail
	}

	return nil
}

func (c ReturnConsultation) ValidateForPersist() error {
	if !isValidReferenceID(c.ID) {
		return ErrInvalidReturnConsultationID
	}
	if err := c.ValidateForCreate(); err != nil {
		return err
	}
	if c.CreatedAt.IsZero() {
		return ErrInvalidReturnConsultationCreatedAt
	}

	return nil
}

func IsValidReturnConsultationReason(
	reason ReturnConsultationReason,
) bool {
	switch reason {
	case ReturnConsultationReasonNotAsDescribed,
		ReturnConsultationReasonDamaged,
		ReturnConsultationReasonWrongItem,
		ReturnConsultationReasonOther:
		return true
	default:
		return false
	}
}
