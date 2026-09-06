// backend/internal/domain/reviewReport/notification.go
package reviewReport

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

// ============================================================
// Errors
// ============================================================

var (
	ErrInvalidDecisionNotificationID = errors.New(
		"reviewReport: invalid decision notification id",
	)
	ErrDecisionNotificationCaseMismatch = errors.New(
		"reviewReport: decision notification case mismatch",
	)
	ErrDecisionNotificationCaseNotDecided = errors.New(
		"reviewReport: decision notification case is not decided",
	)
	ErrInvalidDecisionNotificationCreatedAt = errors.New(
		"reviewReport: invalid decision notification created at",
	)
	ErrInvalidDecisionNotificationUpdatedAt = errors.New(
		"reviewReport: invalid decision notification updated at",
	)
	ErrInvalidDecisionNotificationReadAt = errors.New(
		"reviewReport: invalid decision notification read at",
	)
)

// ============================================================
// ID
// ============================================================

type DecisionNotificationID string

// BuildDecisionNotificationID は、1回の裁定結果と1件の通報者を一意に結び付ける。
// caseId + reportId + decidedAt を利用することで、同一裁定の再実行では重複せず、
// KEPT 後にケースが再度 PENDING となり再裁定された場合は別通知として扱える。
func BuildDecisionNotificationID(
	caseID CaseID,
	reportID ReportID,
	decidedAt time.Time,
) (DecisionNotificationID, error) {
	if caseID == "" {
		return "", ErrInvalidCaseID
	}
	if reportID == "" {
		return "", ErrInvalidReportID
	}
	if decidedAt.IsZero() {
		return "", ErrDecisionNotificationCaseNotDecided
	}

	source := string(caseID) +
		"|" +
		string(reportID) +
		"|" +
		decidedAt.UTC().Format(time.RFC3339Nano)

	sum := sha256.Sum256([]byte(source))

	return DecisionNotificationID(
		"review_report_decision_" + hex.EncodeToString(sum[:]),
	), nil
}

// ============================================================
// Entity
// ============================================================

// DecisionNotification は、Admin が通報ケースを裁定した結果を
// そのケースを通報した AVATAR / BRAND へ通知するためのドメインエンティティ。
//
// Admin の内部識別子 DecidedBy は通知先へ公開しない。
// 対象の商品名・トークン名などの表示名は保存せず、Query/BFF 層で解決する。
type DecisionNotification struct {
	ID DecisionNotificationID

	CaseID   CaseID
	ReportID ReportID

	RecipientType ActorType
	RecipientID   string
	CompanyID     string

	TargetType     TargetType
	TargetID       string
	TargetParentID string

	ReportReason ReportReason
	ReportDetail string

	DecisionStatus CaseStatus
	DecisionReason string
	DecidedAt      time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
	ReadAt    *time.Time
}

// ============================================================
// Constructor
// ============================================================

// NewDecisionNotification は裁定済み ReportCase と、そのケースに紐づく
// 1件の Report から通報者向け通知を生成する。
func NewDecisionNotification(
	reportCase ReportCase,
	report Report,
	createdAt time.Time,
) (DecisionNotification, error) {
	if err := reportCase.Validate(); err != nil {
		return DecisionNotification{}, err
	}
	if err := report.Validate(); err != nil {
		return DecisionNotification{}, err
	}
	if report.CaseID != reportCase.ID {
		return DecisionNotification{}, ErrDecisionNotificationCaseMismatch
	}
	if reportCase.Status != CaseStatusKept &&
		reportCase.Status != CaseStatusRemoved {
		return DecisionNotification{}, ErrDecisionNotificationCaseNotDecided
	}
	if reportCase.DecidedAt == nil || reportCase.DecidedAt.IsZero() {
		return DecisionNotification{}, ErrDecisionNotificationCaseNotDecided
	}

	decidedAt := reportCase.DecidedAt.UTC()
	createdAt = createdAt.UTC()
	if createdAt.IsZero() || createdAt.Before(decidedAt) {
		return DecisionNotification{}, ErrInvalidDecisionNotificationCreatedAt
	}

	id, err := BuildDecisionNotificationID(
		reportCase.ID,
		report.ID,
		decidedAt,
	)
	if err != nil {
		return DecisionNotification{}, err
	}

	notification := DecisionNotification{
		ID:             id,
		CaseID:         reportCase.ID,
		ReportID:       report.ID,
		RecipientType:  report.ReporterType,
		RecipientID:    report.ReporterID,
		CompanyID:      report.CompanyID,
		TargetType:     reportCase.TargetType,
		TargetID:       reportCase.TargetID,
		TargetParentID: reportCase.TargetParentID,
		ReportReason:   report.Reason,
		ReportDetail:   report.Detail,
		DecisionStatus: reportCase.Status,
		DecisionReason: reportCase.DecisionReason,
		DecidedAt:      decidedAt,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
		ReadAt:         nil,
	}

	if err := notification.Validate(); err != nil {
		return DecisionNotification{}, err
	}

	return notification, nil
}

// ============================================================
// Validation
// ============================================================

func (n DecisionNotification) Validate() error {
	if n.ID == "" {
		return ErrInvalidDecisionNotificationID
	}
	if n.CaseID == "" {
		return ErrInvalidCaseID
	}
	if n.ReportID == "" {
		return ErrInvalidReportID
	}
	if err := n.RecipientType.Validate(); err != nil {
		return err
	}
	if !isValidDocumentIDPart(n.RecipientID) {
		return ErrInvalidReporterID
	}
	if n.RecipientType == ActorTypeBrand && n.CompanyID == "" {
		return ErrInvalidCompanyID
	}
	if err := n.TargetType.Validate(); err != nil {
		return err
	}
	if !isValidDocumentIDPart(n.TargetID) {
		return ErrInvalidTargetID
	}
	if n.TargetParentID == "" {
		return ErrInvalidTargetParentID
	}
	if err := n.ReportReason.Validate(); err != nil {
		return err
	}
	if n.ReportReason == ReportReasonOther && n.ReportDetail == "" {
		return ErrReportDetailRequired
	}

	switch n.DecisionStatus {
	case CaseStatusKept, CaseStatusRemoved:
	default:
		return ErrDecisionNotificationCaseNotDecided
	}

	if n.DecisionReason == "" {
		return ErrDecisionReasonRequired
	}

	n.DecidedAt = n.DecidedAt.UTC()
	if n.DecidedAt.IsZero() {
		return ErrDecisionNotificationCaseNotDecided
	}

	expectedID, err := BuildDecisionNotificationID(
		n.CaseID,
		n.ReportID,
		n.DecidedAt,
	)
	if err != nil {
		return err
	}
	if n.ID != expectedID {
		return ErrInvalidDecisionNotificationID
	}

	n.CreatedAt = n.CreatedAt.UTC()
	if n.CreatedAt.IsZero() || n.CreatedAt.Before(n.DecidedAt) {
		return ErrInvalidDecisionNotificationCreatedAt
	}

	n.UpdatedAt = n.UpdatedAt.UTC()
	if n.UpdatedAt.IsZero() || n.UpdatedAt.Before(n.CreatedAt) {
		return ErrInvalidDecisionNotificationUpdatedAt
	}

	if n.ReadAt != nil {
		readAt := n.ReadAt.UTC()
		if readAt.IsZero() || readAt.Before(n.CreatedAt) {
			return ErrInvalidDecisionNotificationReadAt
		}
	}

	return nil
}

// ============================================================
// Read state
// ============================================================

func (n DecisionNotification) IsRead() bool {
	return n.ReadAt != nil && !n.ReadAt.IsZero()
}

// MarkRead は通知を既読化する。
// すでに既読の場合も成功扱いとし、既読日時は最初の値を維持する。
func (n *DecisionNotification) MarkRead(now time.Time) error {
	if n == nil {
		return ErrInvalidDecisionNotificationID
	}

	now = now.UTC()
	if now.IsZero() || now.Before(n.CreatedAt.UTC()) {
		return ErrInvalidDecisionNotificationReadAt
	}

	if n.ReadAt == nil {
		readAt := now
		n.ReadAt = &readAt
	}

	n.UpdatedAt = now
	return n.Validate()
}
