// backend/internal/domain/report/repository_port.go
package report

import (
	"context"
	"time"

	common "narratives/internal/domain/common"
)

// ============================================================
// Filter
// ============================================================

type CaseFilter struct {
	common.FilterCommon `json:",inline"`

	TargetType       *TargetType `json:"targetType"`
	TargetID         string      `json:"targetId"`
	TargetParentID   string      `json:"targetParentId"`
	TargetAuthorID   string      `json:"targetAuthorId"`
	TargetAuthorType *ActorType  `json:"targetAuthorType"`
	Status           *CaseStatus `json:"status"`

	CreatedAt common.TimeRange `json:"createdAt"`
	UpdatedAt common.TimeRange `json:"updatedAt"`
	DecidedAt common.TimeRange `json:"decidedAt"`
}

type ReportFilter struct {
	common.FilterCommon `json:",inline"`

	CaseID       CaseID        `json:"caseId"`
	ReporterType *ActorType    `json:"reporterType"`
	ReporterID   string        `json:"reporterId"`
	CompanyID    string        `json:"companyId"`
	Reason       *ReportReason `json:"reason"`

	CreatedAt common.TimeRange `json:"createdAt"`
}

// ============================================================
// Patch
// ============================================================

type CasePatch struct {
	ReportCount *int        `json:"reportCount"`
	Status      *CaseStatus `json:"status"`

	UpdatedAt *time.Time `json:"updatedAt"`

	DecidedAt      *time.Time `json:"decidedAt"`
	DecidedBy      *string    `json:"decidedBy"`
	DecisionReason *string    `json:"decisionReason"`
}

func NewCasePatchFromEntity(entity ReportCase) CasePatch {
	return CasePatch{
		ReportCount:    &entity.ReportCount,
		Status:         &entity.Status,
		UpdatedAt:      &entity.UpdatedAt,
		DecidedAt:      entity.DecidedAt,
		DecidedBy:      &entity.DecidedBy,
		DecisionReason: &entity.DecisionReason,
	}
}

// ============================================================
// Sort
// ============================================================

var AllowedCaseSortColumns = map[string]struct{}{
	"createdAt":   {},
	"updatedAt":   {},
	"decidedAt":   {},
	"reportCount": {},
	"status":      {},
}

var AllowedReportSortColumns = map[string]struct{}{
	"createdAt": {},
	"reason":    {},
}

// ============================================================
// AddReport
// ============================================================

// AddReportResult represents the result of the atomic report operation.
//
// CaseCreated:
//
//	true when this is the first report against the target.
//
// ReportCreated:
//
//	true when this reporter had not reported the target before.
//
// Duplicate reports from the same reporter should return the existing case
// with ReportCreated=false and must not increment ReportCount.
type AddReportResult struct {
	Case          ReportCase
	Report        Report
	CaseCreated   bool
	ReportCreated bool
}

// ============================================================
// CaseRepository
// ============================================================

type CaseRepository interface {
	GetCase(ctx context.Context, caseID CaseID) (ReportCase, error)

	ListCases(
		ctx context.Context,
		filter CaseFilter,
		sort common.Sort,
		page common.Page,
	) (common.PageResult[ReportCase], error)

	UpdateCase(
		ctx context.Context,
		caseID CaseID,
		patch CasePatch,
	) (ReportCase, error)
}

// ============================================================
// ReportRepository
// ============================================================

type ReportRepository interface {
	GetReport(
		ctx context.Context,
		caseID CaseID,
		reportID ReportID,
	) (Report, error)

	ListReports(
		ctx context.Context,
		caseID CaseID,
		filter ReportFilter,
		sort common.Sort,
		page common.Page,
	) (common.PageResult[Report], error)
}

// ============================================================
// MutationRepository
// ============================================================

// MutationRepository owns operations that must be atomic.
//
// AddReport must execute the following in one storage transaction:
//
//  1. Read the existing ReportCase.
//  2. Create initialCase when the case does not exist.
//  3. Read the existing Report for the reporter.
//  4. If the reporter already exists, return ReportCreated=false.
//  5. Otherwise create the Report.
//  6. Increment ReportCase.ReportCount exactly once.
//  7. Update ReportCase.UpdatedAt.
//
// When the case already exists, its target metadata and snapshot must remain
// unchanged. initialCase is only used when creating the case for the first
// report.
type MutationRepository interface {
	AddReport(
		ctx context.Context,
		initialCase ReportCase,
		report Report,
	) (AddReportResult, error)
}

// ============================================================
// RepositoryPort
// ============================================================

type RepositoryPort interface {
	CaseRepository
	ReportRepository
	MutationRepository
}
