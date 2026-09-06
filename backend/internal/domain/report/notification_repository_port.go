// backend/internal/domain/report/notification_repository_port.go
package report

import (
	"context"
	"time"

	common "narratives/internal/domain/common"
)

// ============================================================
// Filter
// ============================================================

type DecisionNotificationFilter struct {
	common.FilterCommon `json:",inline"`

	RecipientType  *ActorType  `json:"recipientType"`
	RecipientID    string      `json:"recipientId"`
	CompanyID      string      `json:"companyId"`
	TargetType     *TargetType `json:"targetType"`
	TargetID       string      `json:"targetId"`
	TargetParentID string      `json:"targetParentId"`
	DecisionStatus *CaseStatus `json:"decisionStatus"`
	IsRead         *bool       `json:"isRead"`

	CreatedAt common.TimeRange `json:"createdAt"`
	DecidedAt common.TimeRange `json:"decidedAt"`
}

// ============================================================
// Sort
// ============================================================

var AllowedDecisionNotificationSortColumns = map[string]struct{}{
	"createdAt":      {},
	"updatedAt":      {},
	"decidedAt":      {},
	"decisionStatus": {},
}

// ============================================================
// Create result
// ============================================================

// CreateDecisionNotificationResult represents the result of an idempotent
// notification creation.
//
// Created is true only when the notification did not already exist.
// If the deterministic notification ID already exists, implementations must
// return the existing entity with Created=false and must not overwrite its
// mutable read state.
type CreateDecisionNotificationResult struct {
	Notification DecisionNotification
	Created      bool
}

// ============================================================
// DecisionNotificationRepository
// ============================================================

// DecisionNotificationRepository persists report decision notifications.
//
// DecisionNotification.ID is deterministic for one ReportCase decision and
// one Report. CreateIfAbsent must therefore be idempotent.
//
// Existing notifications must never be recreated or reset to unread by a
// repeated Admin decision request or retry.
type DecisionNotificationRepository interface {
	CreateIfAbsent(
		ctx context.Context,
		notification DecisionNotification,
	) (CreateDecisionNotificationResult, error)

	GetByID(
		ctx context.Context,
		notificationID DecisionNotificationID,
	) (DecisionNotification, error)

	List(
		ctx context.Context,
		filter DecisionNotificationFilter,
		sort common.Sort,
		page common.Page,
	) (common.PageResult[DecisionNotification], error)

	MarkRead(
		ctx context.Context,
		notificationID DecisionNotificationID,
		recipientType ActorType,
		recipientID string,
		readAt time.Time,
	) (DecisionNotification, error)
}
