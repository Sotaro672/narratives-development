// backend/internal/domain/reviewReport/entity.go
package reviewReport

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ============================================================
// Errors
// ============================================================

var (
	ErrInvalidCaseID             = errors.New("reviewReport: invalid case id")
	ErrInvalidReportID           = errors.New("reviewReport: invalid report id")
	ErrInvalidTargetType         = errors.New("reviewReport: invalid target type")
	ErrInvalidTargetID           = errors.New("reviewReport: invalid target id")
	ErrInvalidTargetParentID     = errors.New("reviewReport: invalid target parent id")
	ErrInvalidActorType          = errors.New("reviewReport: invalid actor type")
	ErrInvalidTargetAuthorID     = errors.New("reviewReport: invalid target author id")
	ErrInvalidReporterID         = errors.New("reviewReport: invalid reporter id")
	ErrInvalidCompanyID          = errors.New("reviewReport: invalid company id")
	ErrInvalidReason             = errors.New("reviewReport: invalid reason")
	ErrReportDetailRequired      = errors.New("reviewReport: report detail is required")
	ErrInvalidStatus             = errors.New("reviewReport: invalid status")
	ErrInvalidReportCount        = errors.New("reviewReport: invalid report count")
	ErrInvalidSnapshotRating     = errors.New("reviewReport: invalid snapshot rating")
	ErrInvalidCreatedAt          = errors.New("reviewReport: invalid created at")
	ErrInvalidUpdatedAt          = errors.New("reviewReport: invalid updated at")
	ErrDecisionReasonRequired    = errors.New("reviewReport: decision reason is required")
	ErrDecidedByRequired         = errors.New("reviewReport: decided by is required")
	ErrCaseAlreadyRemoved        = errors.New("reviewReport: case is already removed")
	ErrCannotReportRemovedTarget = errors.New("reviewReport: cannot report removed target")
)

func IsInvalid(err error) bool {
	return errors.Is(err, ErrInvalidCaseID) ||
		errors.Is(err, ErrInvalidReportID) ||
		errors.Is(err, ErrInvalidTargetType) ||
		errors.Is(err, ErrInvalidTargetID) ||
		errors.Is(err, ErrInvalidTargetParentID) ||
		errors.Is(err, ErrInvalidActorType) ||
		errors.Is(err, ErrInvalidTargetAuthorID) ||
		errors.Is(err, ErrInvalidReporterID) ||
		errors.Is(err, ErrInvalidCompanyID) ||
		errors.Is(err, ErrInvalidReason) ||
		errors.Is(err, ErrReportDetailRequired) ||
		errors.Is(err, ErrInvalidStatus) ||
		errors.Is(err, ErrInvalidReportCount) ||
		errors.Is(err, ErrInvalidSnapshotRating) ||
		errors.Is(err, ErrInvalidCreatedAt) ||
		errors.Is(err, ErrInvalidUpdatedAt) ||
		errors.Is(err, ErrDecisionReasonRequired) ||
		errors.Is(err, ErrDecidedByRequired)
}

// ============================================================
// ID
// ============================================================

type CaseID string
type ReportID string

func BuildCaseID(targetType TargetType, targetID string) (CaseID, error) {
	if err := targetType.Validate(); err != nil {
		return "", err
	}
	if !isValidDocumentIDPart(targetID) {
		return "", ErrInvalidTargetID
	}

	var prefix string
	switch targetType {
	case TargetTypeProductBlueprintReview:
		prefix = "productBlueprintReview"
	case TargetTypeTokenBlueprintComment:
		prefix = "tokenBlueprintComment"
	default:
		return "", ErrInvalidTargetType
	}

	return CaseID(prefix + "_" + targetID), nil
}

func BuildReporterKey(actorType ActorType, actorID string) (ReportID, error) {
	if err := actorType.Validate(); err != nil {
		return "", err
	}
	if !isValidDocumentIDPart(actorID) {
		return "", ErrInvalidReporterID
	}

	var prefix string
	switch actorType {
	case ActorTypeAvatar:
		prefix = "avatar"
	case ActorTypeBrand:
		prefix = "brand"
	default:
		return "", ErrInvalidActorType
	}

	return ReportID(prefix + "_" + actorID), nil
}

// ============================================================
// TargetType
// ============================================================

type TargetType string

const (
	TargetTypeProductBlueprintReview TargetType = "PRODUCT_BLUEPRINT_REVIEW"
	TargetTypeTokenBlueprintComment  TargetType = "TOKEN_BLUEPRINT_COMMENT"
)

func (t TargetType) Validate() error {
	switch t {
	case TargetTypeProductBlueprintReview, TargetTypeTokenBlueprintComment:
		return nil
	default:
		return ErrInvalidTargetType
	}
}

// ============================================================
// ActorType
// ============================================================

type ActorType string

const (
	ActorTypeAvatar ActorType = "AVATAR"
	ActorTypeBrand  ActorType = "BRAND"
)

func (t ActorType) Validate() error {
	switch t {
	case ActorTypeAvatar, ActorTypeBrand:
		return nil
	default:
		return ErrInvalidActorType
	}
}

// ============================================================
// ReportReason
// ============================================================

type ReportReason string

const (
	ReportReasonSpam             ReportReason = "SPAM"
	ReportReasonHarassment       ReportReason = "HARASSMENT"
	ReportReasonInappropriate    ReportReason = "INAPPROPRIATE"
	ReportReasonFalseInformation ReportReason = "FALSE_INFORMATION"
	ReportReasonOther            ReportReason = "OTHER"
)

func (r ReportReason) Validate() error {
	switch r {
	case ReportReasonSpam, ReportReasonHarassment, ReportReasonInappropriate, ReportReasonFalseInformation, ReportReasonOther:
		return nil
	default:
		return ErrInvalidReason
	}
}

// ============================================================
// CaseStatus
// ============================================================

type CaseStatus string

const (
	CaseStatusPending CaseStatus = "PENDING"
	CaseStatusKept    CaseStatus = "KEPT"
	CaseStatusRemoved CaseStatus = "REMOVED"
)

func (s CaseStatus) Validate() error {
	switch s {
	case CaseStatusPending, CaseStatusKept, CaseStatusRemoved:
		return nil
	default:
		return ErrInvalidStatus
	}
}

// ============================================================
// ReportCase
// ============================================================

type ReportCase struct {
	ID CaseID

	TargetType     TargetType
	TargetID       string
	TargetParentID string

	TargetAuthorID   string
	TargetAuthorType ActorType

	SnapshotTitle  string
	SnapshotBody   string
	SnapshotRating *int

	ReportCount int
	Status      CaseStatus

	CreatedAt time.Time
	UpdatedAt time.Time

	DecidedAt      *time.Time
	DecidedBy      string
	DecisionReason string
}

type NewReportCaseParams struct {
	TargetType     TargetType
	TargetID       string
	TargetParentID string

	TargetAuthorID   string
	TargetAuthorType ActorType

	SnapshotTitle  string
	SnapshotBody   string
	SnapshotRating *int

	CreatedAt time.Time
}

func NewReportCase(params NewReportCaseParams) (ReportCase, error) {
	if err := params.TargetType.Validate(); err != nil {
		return ReportCase{}, err
	}
	if !isValidDocumentIDPart(params.TargetID) {
		return ReportCase{}, ErrInvalidTargetID
	}
	if params.TargetParentID == "" {
		return ReportCase{}, ErrInvalidTargetParentID
	}
	if err := params.TargetAuthorType.Validate(); err != nil {
		return ReportCase{}, err
	}
	if params.TargetAuthorID == "" {
		return ReportCase{}, ErrInvalidTargetAuthorID
	}
	if params.SnapshotBody == "" {
		return ReportCase{}, fmt.Errorf("%w: snapshot body is empty", ErrInvalidTargetID)
	}
	if params.CreatedAt.IsZero() {
		return ReportCase{}, ErrInvalidCreatedAt
	}

	caseID, err := BuildCaseID(params.TargetType, params.TargetID)
	if err != nil {
		return ReportCase{}, err
	}

	snapshotRating, err := normalizeSnapshotRating(params.TargetType, params.SnapshotRating)
	if err != nil {
		return ReportCase{}, err
	}

	createdAt := params.CreatedAt.UTC()
	entity := ReportCase{
		ID:               caseID,
		TargetType:       params.TargetType,
		TargetID:         params.TargetID,
		TargetParentID:   params.TargetParentID,
		TargetAuthorID:   params.TargetAuthorID,
		TargetAuthorType: params.TargetAuthorType,
		SnapshotTitle:    params.SnapshotTitle,
		SnapshotBody:     params.SnapshotBody,
		SnapshotRating:   snapshotRating,
		ReportCount:      0,
		Status:           CaseStatusPending,
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
		DecidedAt:        nil,
		DecidedBy:        "",
		DecisionReason:   "",
	}

	if err := entity.Validate(); err != nil {
		return ReportCase{}, err
	}
	return entity, nil
}

func (c ReportCase) Validate() error {
	if c.ID == "" {
		return ErrInvalidCaseID
	}
	if err := c.TargetType.Validate(); err != nil {
		return err
	}
	if !isValidDocumentIDPart(c.TargetID) {
		return ErrInvalidTargetID
	}
	if c.TargetParentID == "" {
		return ErrInvalidTargetParentID
	}
	if err := c.TargetAuthorType.Validate(); err != nil {
		return err
	}
	if c.TargetAuthorID == "" {
		return ErrInvalidTargetAuthorID
	}
	if c.SnapshotBody == "" {
		return fmt.Errorf("%w: snapshot body is empty", ErrInvalidTargetID)
	}
	if _, err := normalizeSnapshotRating(c.TargetType, c.SnapshotRating); err != nil {
		return err
	}
	if c.ReportCount < 0 {
		return ErrInvalidReportCount
	}
	if err := c.Status.Validate(); err != nil {
		return err
	}
	if c.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if c.UpdatedAt.IsZero() {
		return ErrInvalidUpdatedAt
	}

	switch c.Status {
	case CaseStatusPending:
		if c.DecidedAt != nil || c.DecidedBy != "" || c.DecisionReason != "" {
			return ErrInvalidStatus
		}
	case CaseStatusKept, CaseStatusRemoved:
		if c.DecidedAt == nil || c.DecidedAt.IsZero() {
			return ErrInvalidUpdatedAt
		}
		if c.DecidedBy == "" {
			return ErrDecidedByRequired
		}
		if c.DecisionReason == "" {
			return ErrDecisionReasonRequired
		}
	}

	return nil
}

func (c *ReportCase) IncrementReportCount(now time.Time) error {
	if c == nil {
		return ErrInvalidCaseID
	}
	if c.Status == CaseStatusRemoved {
		return ErrCannotReportRemovedTarget
	}
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	if c.ReportCount < 0 {
		return ErrInvalidReportCount
	}

	c.ReportCount++
	c.UpdatedAt = now.UTC()
	return nil
}

func (c *ReportCase) Keep(reason string, now time.Time, decidedBy string) error {
	if c == nil {
		return ErrInvalidCaseID
	}
	if reason == "" {
		return ErrDecisionReasonRequired
	}
	if decidedBy == "" {
		return ErrDecidedByRequired
	}
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	if c.Status == CaseStatusRemoved {
		return ErrCaseAlreadyRemoved
	}
	if c.Status == CaseStatusKept {
		return nil
	}

	decidedAt := now.UTC()
	c.Status = CaseStatusKept
	c.DecidedAt = &decidedAt
	c.DecidedBy = decidedBy
	c.DecisionReason = reason
	c.UpdatedAt = decidedAt
	return nil
}

func (c *ReportCase) Remove(reason string, now time.Time, decidedBy string) error {
	if c == nil {
		return ErrInvalidCaseID
	}
	if reason == "" {
		return ErrDecisionReasonRequired
	}
	if decidedBy == "" {
		return ErrDecidedByRequired
	}
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	if c.Status == CaseStatusRemoved {
		return nil
	}

	decidedAt := now.UTC()
	c.Status = CaseStatusRemoved
	c.DecidedAt = &decidedAt
	c.DecidedBy = decidedBy
	c.DecisionReason = reason
	c.UpdatedAt = decidedAt
	return nil
}

func (c ReportCase) IsPending() bool {
	return c.Status == CaseStatusPending
}

func (c ReportCase) IsKept() bool {
	return c.Status == CaseStatusKept
}

func (c ReportCase) IsRemoved() bool {
	return c.Status == CaseStatusRemoved
}

// ============================================================
// Report
// ============================================================

type Report struct {
	ID     ReportID
	CaseID CaseID

	ReporterType ActorType
	ReporterID   string
	CompanyID    string

	Reason ReportReason
	Detail string

	CreatedAt time.Time
}

type NewReportParams struct {
	CaseID CaseID

	ReporterType ActorType
	ReporterID   string
	CompanyID    string

	Reason ReportReason
	Detail string

	CreatedAt time.Time
}

func NewReport(params NewReportParams) (Report, error) {
	if params.CaseID == "" {
		return Report{}, ErrInvalidCaseID
	}
	if err := params.ReporterType.Validate(); err != nil {
		return Report{}, err
	}
	if !isValidDocumentIDPart(params.ReporterID) {
		return Report{}, ErrInvalidReporterID
	}
	if params.ReporterType == ActorTypeBrand && params.CompanyID == "" {
		return Report{}, ErrInvalidCompanyID
	}
	if err := params.Reason.Validate(); err != nil {
		return Report{}, err
	}
	if params.Reason == ReportReasonOther && params.Detail == "" {
		return Report{}, ErrReportDetailRequired
	}
	if params.CreatedAt.IsZero() {
		return Report{}, ErrInvalidCreatedAt
	}

	reportID, err := BuildReporterKey(params.ReporterType, params.ReporterID)
	if err != nil {
		return Report{}, err
	}

	entity := Report{
		ID:           reportID,
		CaseID:       params.CaseID,
		ReporterType: params.ReporterType,
		ReporterID:   params.ReporterID,
		CompanyID:    params.CompanyID,
		Reason:       params.Reason,
		Detail:       params.Detail,
		CreatedAt:    params.CreatedAt.UTC(),
	}

	if err := entity.Validate(); err != nil {
		return Report{}, err
	}
	return entity, nil
}

func (r Report) Validate() error {
	if r.ID == "" {
		return ErrInvalidReportID
	}
	if r.CaseID == "" {
		return ErrInvalidCaseID
	}
	if err := r.ReporterType.Validate(); err != nil {
		return err
	}
	if !isValidDocumentIDPart(r.ReporterID) {
		return ErrInvalidReporterID
	}
	if r.ReporterType == ActorTypeBrand && r.CompanyID == "" {
		return ErrInvalidCompanyID
	}
	if err := r.Reason.Validate(); err != nil {
		return err
	}
	if r.Reason == ReportReasonOther && r.Detail == "" {
		return ErrReportDetailRequired
	}
	if r.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	return nil
}

// ============================================================
// Helpers
// ============================================================

func normalizeSnapshotRating(targetType TargetType, rating *int) (*int, error) {
	switch targetType {
	case TargetTypeProductBlueprintReview:
		if rating == nil || *rating < 1 || *rating > 5 {
			return nil, ErrInvalidSnapshotRating
		}
		value := *rating
		return &value, nil
	case TargetTypeTokenBlueprintComment:
		if rating != nil {
			return nil, ErrInvalidSnapshotRating
		}
		return nil, nil
	default:
		return nil, ErrInvalidTargetType
	}
}

func isValidDocumentIDPart(value string) bool {
	return value != "" && !strings.Contains(value, "/")
}
