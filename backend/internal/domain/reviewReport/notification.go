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
	ErrInvalidDecisionNotificationKind = errors.New(
		"reviewReport: invalid decision notification kind",
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
	ErrTargetDecisionNotificationReportData = errors.New(
		"reviewReport: target decision notification must not contain report data",
	)
)

// ============================================================
// NotificationKind
// ============================================================

type NotificationKind string

const (
	// NotificationKindReporterDecision は、通報者へ裁定結果を通知する。
	NotificationKindReporterDecision NotificationKind = "REPORTER_DECISION"

	// NotificationKindTargetEnforcement は、REMOVE裁定によって実際に
	// 削除・利用制限等の措置を受けた対象者へ通知する。
	NotificationKindTargetEnforcement NotificationKind = "TARGET_ENFORCEMENT"
)

func (k NotificationKind) Validate() error {
	switch k {
	case NotificationKindReporterDecision, NotificationKindTargetEnforcement:
		return nil
	default:
		return ErrInvalidDecisionNotificationKind
	}
}

func normalizeDecisionNotificationKind(kind NotificationKind) NotificationKind {
	// notificationKind 導入前に保存された既存通知は、
	// すべて通報者向け裁定通知として扱う。
	if kind == "" {
		return NotificationKindReporterDecision
	}
	return kind
}

// ============================================================
// ID
// ============================================================

type DecisionNotificationID string

// BuildDecisionNotificationID は、1回の裁定結果と1件の通報者を一意に結び付ける。
// caseId + reportId + decidedAt を利用することで、同一裁定の再実行では重複せず、
// KEPT 後にケースが再度 PENDING となり再裁定された場合は別通知として扱える。
//
// 既存通知とのID互換性を維持するため、この生成規則は変更しない。
//
// Firestore Timestamp はマイクロ秒精度で永続化されるため、
// ID生成に使用する decidedAt もマイクロ秒精度へ正規化する。
// これにより保存前とFirestore復元後で同じIDを再計算できる。
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

	canonicalDecidedAt := decidedAt.UTC().Truncate(time.Microsecond)

	source := string(caseID) +
		"|" +
		string(reportID) +
		"|" +
		canonicalDecidedAt.Format(time.RFC3339Nano)

	sum := sha256.Sum256([]byte(source))

	return DecisionNotificationID(
		"review_report_decision_" + hex.EncodeToString(sum[:]),
	), nil
}

// BuildTargetDecisionNotificationID は、1回のREMOVE裁定と1人の裁定対象者を
// 一意に結び付ける。
//
// caseId + recipientType + recipientId + decidedAt を利用するため、同一ケースに
// 複数の通報が存在しても裁定対象者への通知は1件だけになる。
// 同一裁定を再実行した場合も同じIDとなるため、CreateIfAbsentで冪等化できる。
func BuildTargetDecisionNotificationID(
	caseID CaseID,
	recipientType ActorType,
	recipientID string,
	decidedAt time.Time,
) (DecisionNotificationID, error) {
	if caseID == "" {
		return "", ErrInvalidCaseID
	}
	if err := recipientType.Validate(); err != nil {
		return "", err
	}
	if !isValidDocumentIDPart(recipientID) {
		return "", ErrInvalidReporterID
	}
	if decidedAt.IsZero() {
		return "", ErrDecisionNotificationCaseNotDecided
	}

	canonicalDecidedAt := decidedAt.UTC().Truncate(time.Microsecond)

	source := string(caseID) +
		"|" +
		string(recipientType) +
		"|" +
		recipientID +
		"|" +
		canonicalDecidedAt.Format(time.RFC3339Nano)

	sum := sha256.Sum256([]byte(source))

	return DecisionNotificationID(
		"review_report_target_enforcement_" + hex.EncodeToString(sum[:]),
	), nil
}

// ============================================================
// Entity
// ============================================================

// DecisionNotification は、Admin が通報ケースを裁定した結果を通知する
// ドメインエンティティ。
//
// REPORTER_DECISION:
//   - そのケースを通報した AVATAR / BRAND 向け。
//   - ReportID / ReportReason / ReportDetail を保持する。
//
// TARGET_ENFORCEMENT:
//   - REMOVE裁定によって実際に措置を受けた対象者向け。
//   - 同一裁定につき対象者へ1件だけ生成する。
//   - 通報者の情報・通報理由・通報詳細は保持しない。
//
// Admin の内部識別子 DecidedBy は通知先へ公開しない。
// 対象の商品名・トークン名などの表示名は保存せず、Query/BFF 層で解決する。
type DecisionNotification struct {
	ID               DecisionNotificationID
	NotificationKind NotificationKind

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

// Kind は通知種別を返す。
// notificationKind 導入前の既存通知は REPORTER_DECISION として扱う。
func (n DecisionNotification) Kind() NotificationKind {
	return normalizeDecisionNotificationKind(n.NotificationKind)
}

// ============================================================
// Constructor
// ============================================================

// NewDecisionNotification は裁定済み ReportCase と、そのケースに紐づく
// 1件の Report から通報者向け通知を生成する。
//
// 既存呼び出しとの互換性を維持するため関数名は変更しない。
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

	decidedAt := reportCase.DecidedAt.UTC().Truncate(time.Microsecond)
	createdAt = createdAt.UTC().Truncate(time.Microsecond)
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
		ID:               id,
		NotificationKind: NotificationKindReporterDecision,
		CaseID:           reportCase.ID,
		ReportID:         report.ID,
		RecipientType:    report.ReporterType,
		RecipientID:      report.ReporterID,
		CompanyID:        report.CompanyID,
		TargetType:       reportCase.TargetType,
		TargetID:         reportCase.TargetID,
		TargetParentID:   reportCase.TargetParentID,
		ReportReason:     report.Reason,
		ReportDetail:     report.Detail,
		DecisionStatus:   reportCase.Status,
		DecisionReason:   reportCase.DecisionReason,
		DecidedAt:        decidedAt,
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
		ReadAt:           nil,
	}

	if err := notification.Validate(); err != nil {
		return DecisionNotification{}, err
	}

	return notification, nil
}

// NewTargetEnforcementNotification は、REMOVE裁定によって実際に措置を受けた
// 対象者向け通知を生成する。
//
// 現在の対象者通知は AVATAR への配信を対象とする。
// PRODUCT_BLUEPRINT_REVIEW の削除ではレビュー投稿者、AVATAR のREMOVEでは
// 対象Avatar自身が Recipient となる。
//
// 通報者の特定や通報内容の漏えいを避けるため、ReportID / ReportReason /
// ReportDetail は対象者通知へ保存しない。
func NewTargetEnforcementNotification(
	reportCase ReportCase,
	createdAt time.Time,
) (DecisionNotification, error) {
	if err := reportCase.Validate(); err != nil {
		return DecisionNotification{}, err
	}
	if reportCase.Status != CaseStatusRemoved {
		return DecisionNotification{}, ErrDecisionNotificationCaseNotDecided
	}
	if reportCase.DecidedAt == nil || reportCase.DecidedAt.IsZero() {
		return DecisionNotification{}, ErrDecisionNotificationCaseNotDecided
	}
	if reportCase.TargetAuthorType != ActorTypeAvatar {
		return DecisionNotification{}, ErrInvalidActorType
	}
	if !isValidDocumentIDPart(reportCase.TargetAuthorID) {
		return DecisionNotification{}, ErrInvalidTargetAuthorID
	}

	decidedAt := reportCase.DecidedAt.UTC().Truncate(time.Microsecond)
	createdAt = createdAt.UTC().Truncate(time.Microsecond)
	if createdAt.IsZero() || createdAt.Before(decidedAt) {
		return DecisionNotification{}, ErrInvalidDecisionNotificationCreatedAt
	}

	id, err := BuildTargetDecisionNotificationID(
		reportCase.ID,
		reportCase.TargetAuthorType,
		reportCase.TargetAuthorID,
		decidedAt,
	)
	if err != nil {
		return DecisionNotification{}, err
	}

	notification := DecisionNotification{
		ID:               id,
		NotificationKind: NotificationKindTargetEnforcement,
		CaseID:           reportCase.ID,
		ReportID:         "",
		RecipientType:    reportCase.TargetAuthorType,
		RecipientID:      reportCase.TargetAuthorID,
		CompanyID:        "",
		TargetType:       reportCase.TargetType,
		TargetID:         reportCase.TargetID,
		TargetParentID:   reportCase.TargetParentID,
		ReportReason:     "",
		ReportDetail:     "",
		DecisionStatus:   reportCase.Status,
		DecisionReason:   reportCase.DecisionReason,
		DecidedAt:        decidedAt,
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
		ReadAt:           nil,
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

	kind := n.Kind()
	if err := kind.Validate(); err != nil {
		return err
	}

	if n.CaseID == "" {
		return ErrInvalidCaseID
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

	switch kind {
	case NotificationKindReporterDecision:
		if n.ReportID == "" {
			return ErrInvalidReportID
		}
		if err := n.ReportReason.Validate(); err != nil {
			return err
		}
		if n.ReportReason == ReportReasonOther && n.ReportDetail == "" {
			return ErrReportDetailRequired
		}

	case NotificationKindTargetEnforcement:
		if n.ReportID != "" || n.ReportReason != "" || n.ReportDetail != "" {
			return ErrTargetDecisionNotificationReportData
		}
		if n.DecisionStatus != CaseStatusRemoved {
			return ErrDecisionNotificationCaseNotDecided
		}

	default:
		return ErrInvalidDecisionNotificationKind
	}

	switch n.DecisionStatus {
	case CaseStatusKept, CaseStatusRemoved:
	default:
		return ErrDecisionNotificationCaseNotDecided
	}

	if n.DecisionReason == "" {
		return ErrDecisionReasonRequired
	}

	n.DecidedAt = n.DecidedAt.UTC().Truncate(time.Microsecond)
	if n.DecidedAt.IsZero() {
		return ErrDecisionNotificationCaseNotDecided
	}

	var expectedID DecisionNotificationID
	var err error

	switch kind {
	case NotificationKindReporterDecision:
		expectedID, err = BuildDecisionNotificationID(
			n.CaseID,
			n.ReportID,
			n.DecidedAt,
		)
	case NotificationKindTargetEnforcement:
		expectedID, err = BuildTargetDecisionNotificationID(
			n.CaseID,
			n.RecipientType,
			n.RecipientID,
			n.DecidedAt,
		)
	default:
		return ErrInvalidDecisionNotificationKind
	}
	if err != nil {
		return err
	}
	if n.ID != expectedID {
		return ErrInvalidDecisionNotificationID
	}

	n.CreatedAt = n.CreatedAt.UTC().Truncate(time.Microsecond)
	if n.CreatedAt.IsZero() || n.CreatedAt.Before(n.DecidedAt) {
		return ErrInvalidDecisionNotificationCreatedAt
	}

	n.UpdatedAt = n.UpdatedAt.UTC().Truncate(time.Microsecond)
	if n.UpdatedAt.IsZero() || n.UpdatedAt.Before(n.CreatedAt) {
		return ErrInvalidDecisionNotificationUpdatedAt
	}

	if n.ReadAt != nil {
		readAt := n.ReadAt.UTC().Truncate(time.Microsecond)
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

	now = now.UTC().Truncate(time.Microsecond)
	if now.IsZero() || now.Before(n.CreatedAt.UTC().Truncate(time.Microsecond)) {
		return ErrInvalidDecisionNotificationReadAt
	}

	if n.ReadAt == nil {
		readAt := now
		n.ReadAt = &readAt
	}

	n.UpdatedAt = now
	return n.Validate()
}
