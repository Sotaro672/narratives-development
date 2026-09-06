// backend/internal/adapters/out/firestore/report_repository_fs.go
package firestore

import (
	"context"
	"fmt"
	"math"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	common "narratives/internal/domain/common"
	reportdom "narratives/internal/domain/report"
)

const (
	defaultReportCaseCollection = "reportCases"
	defaultReportSubCollection  = "reports"
)

type ReportRepositoryFS struct {
	client           *firestore.Client
	caseCollection   string
	reportCollection string
}

func NewReportRepositoryFS(client *firestore.Client) *ReportRepositoryFS {
	return &ReportRepositoryFS{
		client:           client,
		caseCollection:   defaultReportCaseCollection,
		reportCollection: defaultReportSubCollection,
	}
}

func (r *ReportRepositoryFS) WithCaseCollection(name string) *ReportRepositoryFS {
	if r != nil && name != "" {
		r.caseCollection = name
	}
	return r
}

func (r *ReportRepositoryFS) WithReportCollection(name string) *ReportRepositoryFS {
	if r != nil && name != "" {
		r.reportCollection = name
	}
	return r
}

func (r *ReportRepositoryFS) caseDoc(caseID reportdom.CaseID) *firestore.DocumentRef {
	return r.client.Collection(r.caseCollection).Doc(string(caseID))
}

func (r *ReportRepositoryFS) reportsCol(caseID reportdom.CaseID) *firestore.CollectionRef {
	return r.caseDoc(caseID).Collection(r.reportCollection)
}

func (r *ReportRepositoryFS) reportDoc(caseID reportdom.CaseID, reportID reportdom.ReportID) *firestore.DocumentRef {
	return r.reportsCol(caseID).Doc(string(reportID))
}

// ============================================================
// CaseRepository
// ============================================================

func (r *ReportRepositoryFS) GetCase(ctx context.Context, caseID reportdom.CaseID) (reportdom.ReportCase, error) {
	if r == nil || r.client == nil {
		return reportdom.ReportCase{}, fmt.Errorf("report repository is not configured")
	}
	if caseID == "" {
		return reportdom.ReportCase{}, reportdom.ErrInvalidCaseID
	}

	snap, err := r.caseDoc(caseID).Get(ctx)
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	return decodeReportCase(snap.Ref.ID, snap.Data())
}

// IsAvatarResaleSuspended reports whether the avatar is currently blocked from
// using the resale service by an Admin report decision.
//
// The AVATAR report case ID is deterministic (avatar_{avatarId}), so the
// check is a direct document lookup rather than a collection query. A missing
// case means the avatar has no resale suspension. Only REMOVED is treated as
// suspended; PENDING and KEPT remain allowed.
func (r *ReportRepositoryFS) IsAvatarResaleSuspended(ctx context.Context, avatarID string) (bool, error) {
	if r == nil || r.client == nil {
		return false, fmt.Errorf("report repository is not configured")
	}
	if avatarID == "" {
		return false, reportdom.ErrInvalidTargetID
	}

	caseID, err := reportdom.BuildCaseID(reportdom.TargetTypeAvatar, avatarID)
	if err != nil {
		return false, err
	}

	reportCase, err := r.GetCase(ctx, caseID)
	if err != nil {
		if reportIsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	if reportCase.TargetType != reportdom.TargetTypeAvatar {
		return false, reportdom.ErrInvalidTargetType
	}
	if reportCase.TargetID != avatarID {
		return false, reportdom.ErrInvalidTargetID
	}

	return reportCase.Status == reportdom.CaseStatusRemoved, nil
}

func (r *ReportRepositoryFS) ListCases(
	ctx context.Context,
	filter reportdom.CaseFilter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[reportdom.ReportCase], error) {
	if r == nil || r.client == nil {
		return common.PageResult[reportdom.ReportCase]{}, fmt.Errorf("report repository is not configured")
	}
	if filter.SearchQuery != "" {
		return common.PageResult[reportdom.ReportCase]{}, fmt.Errorf("report: searchQuery is not supported")
	}

	q := r.client.Collection(r.caseCollection).Query

	if filter.TargetType != nil {
		q = q.Where("targetType", "==", string(*filter.TargetType))
	}
	if filter.TargetID != "" {
		q = q.Where("targetId", "==", filter.TargetID)
	}
	if filter.TargetParentID != "" {
		q = q.Where("targetParentId", "==", filter.TargetParentID)
	}
	if filter.TargetAuthorID != "" {
		q = q.Where("targetAuthorId", "==", filter.TargetAuthorID)
	}
	if filter.TargetAuthorType != nil {
		q = q.Where("targetAuthorType", "==", string(*filter.TargetAuthorType))
	}
	if filter.Status != nil {
		q = q.Where("status", "==", string(*filter.Status))
	}

	createdRange := reportEffectiveTimeRange(filter.CreatedAt, filter.Created)
	updatedRange := reportEffectiveTimeRange(filter.UpdatedAt, filter.Updated)

	q = applyReportTimeRange(q, "createdAt", createdRange)
	q = applyReportTimeRange(q, "updatedAt", updatedRange)
	q = applyReportTimeRange(q, "decidedAt", filter.DecidedAt)

	sortColumn := sort.Column
	if sortColumn == "" {
		sortColumn = "updatedAt"
	}
	if _, ok := reportdom.AllowedCaseSortColumns[sortColumn]; !ok {
		return common.PageResult[reportdom.ReportCase]{}, fmt.Errorf("report: invalid case sort column: %s", sortColumn)
	}

	orderDirection := firestore.Desc
	if sort.Order == common.SortAsc {
		orderDirection = firestore.Asc
	}

	q = q.OrderBy(sortColumn, orderDirection)

	pageNumber, perPage := normalizeReportPage(page)
	offset := (pageNumber - 1) * perPage

	totalCount, err := reportCountQuery(ctx, q)
	if err != nil {
		return common.PageResult[reportdom.ReportCase]{}, err
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(perPage)))
	if totalPages == 0 {
		totalPages = 1
	}

	items := make([]reportdom.ReportCase, 0, perPage)
	iter := q.Offset(offset).Limit(perPage).Documents(ctx)
	defer iter.Stop()

	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.PageResult[reportdom.ReportCase]{}, err
		}

		item, err := decodeReportCase(snap.Ref.ID, snap.Data())
		if err != nil {
			return common.PageResult[reportdom.ReportCase]{}, err
		}

		items = append(items, item)
	}

	return common.PageResult[reportdom.ReportCase]{
		Items:      items,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       pageNumber,
		PerPage:    perPage,
	}, nil
}

func (r *ReportRepositoryFS) UpdateCase(
	ctx context.Context,
	caseID reportdom.CaseID,
	patch reportdom.CasePatch,
) (reportdom.ReportCase, error) {
	if r == nil || r.client == nil {
		return reportdom.ReportCase{}, fmt.Errorf("report repository is not configured")
	}
	if caseID == "" {
		return reportdom.ReportCase{}, reportdom.ErrInvalidCaseID
	}

	doc := r.caseDoc(caseID)
	var updated reportdom.ReportCase

	err := r.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(doc)
		if err != nil {
			return err
		}

		entity, err := decodeReportCase(snap.Ref.ID, snap.Data())
		if err != nil {
			return err
		}

		if patch.ReportCount != nil {
			entity.ReportCount = *patch.ReportCount
		}
		if patch.Status != nil {
			entity.Status = *patch.Status
			if entity.Status == reportdom.CaseStatusPending {
				entity.DecidedAt = nil
				entity.DecidedBy = ""
				entity.DecisionReason = ""
			}
		}
		if patch.UpdatedAt != nil {
			entity.UpdatedAt = patch.UpdatedAt.UTC()
		}
		if patch.DecidedAt != nil {
			decidedAt := patch.DecidedAt.UTC()
			entity.DecidedAt = &decidedAt
		}
		if patch.DecidedBy != nil {
			entity.DecidedBy = *patch.DecidedBy
		}
		if patch.DecisionReason != nil {
			entity.DecisionReason = *patch.DecisionReason
		}

		if err := entity.Validate(); err != nil {
			return err
		}

		data := encodeReportCase(entity)
		if entity.DecidedAt == nil {
			data["decidedAt"] = firestore.Delete
		}

		if err := tx.Set(doc, data, firestore.MergeAll); err != nil {
			return err
		}

		updated = entity
		return nil
	})
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	return updated, nil
}

// ============================================================
// ReportRepository
// ============================================================

func (r *ReportRepositoryFS) GetReport(
	ctx context.Context,
	caseID reportdom.CaseID,
	reportID reportdom.ReportID,
) (reportdom.Report, error) {
	if r == nil || r.client == nil {
		return reportdom.Report{}, fmt.Errorf("report repository is not configured")
	}
	if caseID == "" {
		return reportdom.Report{}, reportdom.ErrInvalidCaseID
	}
	if reportID == "" {
		return reportdom.Report{}, reportdom.ErrInvalidReportID
	}

	snap, err := r.reportDoc(caseID, reportID).Get(ctx)
	if err != nil {
		return reportdom.Report{}, err
	}

	return decodeReport(snap.Ref.ID, caseID, snap.Data())
}

func (r *ReportRepositoryFS) ListReports(
	ctx context.Context,
	caseID reportdom.CaseID,
	filter reportdom.ReportFilter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[reportdom.Report], error) {
	if r == nil || r.client == nil {
		return common.PageResult[reportdom.Report]{}, fmt.Errorf("report repository is not configured")
	}
	if caseID == "" {
		return common.PageResult[reportdom.Report]{}, reportdom.ErrInvalidCaseID
	}
	if filter.CaseID != "" && filter.CaseID != caseID {
		return common.PageResult[reportdom.Report]{}, reportdom.ErrInvalidCaseID
	}
	if filter.SearchQuery != "" {
		return common.PageResult[reportdom.Report]{}, fmt.Errorf("report: searchQuery is not supported")
	}

	q := r.reportsCol(caseID).Query

	if filter.ReporterType != nil {
		q = q.Where("reporterType", "==", string(*filter.ReporterType))
	}
	if filter.ReporterID != "" {
		q = q.Where("reporterId", "==", filter.ReporterID)
	}
	if filter.CompanyID != "" {
		q = q.Where("companyId", "==", filter.CompanyID)
	}
	if filter.Reason != nil {
		q = q.Where("reason", "==", string(*filter.Reason))
	}

	createdRange := reportEffectiveTimeRange(filter.CreatedAt, filter.Created)
	q = applyReportTimeRange(q, "createdAt", createdRange)

	sortColumn := sort.Column
	if sortColumn == "" {
		sortColumn = "createdAt"
	}
	if _, ok := reportdom.AllowedReportSortColumns[sortColumn]; !ok {
		return common.PageResult[reportdom.Report]{}, fmt.Errorf("report: invalid report sort column: %s", sortColumn)
	}

	orderDirection := firestore.Desc
	if sort.Order == common.SortAsc {
		orderDirection = firestore.Asc
	}

	q = q.OrderBy(sortColumn, orderDirection)

	pageNumber, perPage := normalizeReportPage(page)
	offset := (pageNumber - 1) * perPage

	totalCount, err := reportCountQuery(ctx, q)
	if err != nil {
		return common.PageResult[reportdom.Report]{}, err
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(perPage)))
	if totalPages == 0 {
		totalPages = 1
	}

	items := make([]reportdom.Report, 0, perPage)
	iter := q.Offset(offset).Limit(perPage).Documents(ctx)
	defer iter.Stop()

	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.PageResult[reportdom.Report]{}, err
		}

		item, err := decodeReport(snap.Ref.ID, caseID, snap.Data())
		if err != nil {
			return common.PageResult[reportdom.Report]{}, err
		}

		items = append(items, item)
	}

	return common.PageResult[reportdom.Report]{
		Items:      items,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       pageNumber,
		PerPage:    perPage,
	}, nil
}

// ============================================================
// MutationRepository
// ============================================================

func (r *ReportRepositoryFS) AddReport(
	ctx context.Context,
	initialCase reportdom.ReportCase,
	report reportdom.Report,
) (reportdom.AddReportResult, error) {
	if r == nil || r.client == nil {
		return reportdom.AddReportResult{}, fmt.Errorf("report repository is not configured")
	}
	if err := initialCase.Validate(); err != nil {
		return reportdom.AddReportResult{}, err
	}
	if err := report.Validate(); err != nil {
		return reportdom.AddReportResult{}, err
	}
	if report.CaseID != initialCase.ID {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidCaseID
	}

	expectedCaseID, err := reportdom.BuildCaseID(initialCase.TargetType, initialCase.TargetID)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if expectedCaseID != initialCase.ID {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidCaseID
	}

	expectedReportID, err := reportdom.BuildReporterKey(report.ReporterType, report.ReporterID)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if expectedReportID != report.ID {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidReportID
	}

	caseRef := r.caseDoc(initialCase.ID)
	reportRef := r.reportDoc(initialCase.ID, report.ID)
	var result reportdom.AddReportResult

	err = r.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		caseExists := true
		var currentCase reportdom.ReportCase

		caseSnap, caseErr := tx.Get(caseRef)
		if caseErr != nil {
			if reportIsNotFound(caseErr) {
				caseExists = false
			} else {
				return caseErr
			}
		} else {
			decodedCase, decodeErr := decodeReportCase(caseSnap.Ref.ID, caseSnap.Data())
			if decodeErr != nil {
				return decodeErr
			}
			currentCase = decodedCase
		}

		reportSnap, reportErr := tx.Get(reportRef)
		if reportErr == nil {
			if !caseExists {
				return fmt.Errorf("report: report exists without parent case")
			}

			existingReport, decodeErr := decodeReport(
				reportSnap.Ref.ID,
				initialCase.ID,
				reportSnap.Data(),
			)
			if decodeErr != nil {
				return decodeErr
			}

			result = reportdom.AddReportResult{
				Case:          currentCase,
				Report:        existingReport,
				CaseCreated:   false,
				ReportCreated: false,
			}
			return nil
		}
		if !reportIsNotFound(reportErr) {
			return reportErr
		}

		if !caseExists {
			currentCase = initialCase
		}

		wasKept := caseExists && currentCase.Status == reportdom.CaseStatusKept

		if err := currentCase.IncrementReportCount(report.CreatedAt); err != nil {
			return err
		}
		if err := currentCase.Validate(); err != nil {
			return err
		}

		if !caseExists {
			if err := tx.Create(caseRef, encodeReportCase(currentCase)); err != nil {
				return err
			}
		} else {
			updates := []firestore.Update{
				{Path: "reportCount", Value: currentCase.ReportCount},
				{Path: "updatedAt", Value: currentCase.UpdatedAt.UTC()},
			}

			if wasKept {
				updates = append(
					updates,
					firestore.Update{
						Path:  "status",
						Value: string(currentCase.Status),
					},
					firestore.Update{
						Path:  "decidedAt",
						Value: firestore.Delete,
					},
					firestore.Update{
						Path:  "decidedBy",
						Value: currentCase.DecidedBy,
					},
					firestore.Update{
						Path:  "decisionReason",
						Value: currentCase.DecisionReason,
					},
				)
			}

			if err := tx.Update(caseRef, updates); err != nil {
				return err
			}
		}

		if err := tx.Create(reportRef, encodeReport(report)); err != nil {
			return err
		}

		result = reportdom.AddReportResult{
			Case:          currentCase,
			Report:        report,
			CaseCreated:   !caseExists,
			ReportCreated: true,
		}
		return nil
	})
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	return result, nil
}

// ============================================================
// Encode / Decode
// ============================================================

func encodeReportCase(entity reportdom.ReportCase) map[string]any {
	out := map[string]any{
		"id":               string(entity.ID),
		"targetType":       string(entity.TargetType),
		"targetId":         entity.TargetID,
		"targetParentId":   entity.TargetParentID,
		"targetAuthorId":   entity.TargetAuthorID,
		"targetAuthorType": string(entity.TargetAuthorType),
		"snapshotTitle":    entity.SnapshotTitle,
		"snapshotBody":     entity.SnapshotBody,
		"reportCount":      entity.ReportCount,
		"status":           string(entity.Status),
		"createdAt":        entity.CreatedAt.UTC(),
		"updatedAt":        entity.UpdatedAt.UTC(),
		"decidedBy":        entity.DecidedBy,
		"decisionReason":   entity.DecisionReason,
	}

	if entity.SnapshotRating != nil {
		out["snapshotRating"] = *entity.SnapshotRating
	}
	if entity.DecidedAt != nil {
		out["decidedAt"] = entity.DecidedAt.UTC()
	}

	return out
}

func decodeReportCase(id string, data map[string]any) (reportdom.ReportCase, error) {
	targetType, err := firestoreRequiredString(data, "targetType")
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	targetID, err := firestoreRequiredString(data, "targetId")
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	targetParentID, err := firestoreRequiredString(data, "targetParentId")
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	targetAuthorID, err := firestoreRequiredString(data, "targetAuthorId")
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	targetAuthorType, err := firestoreRequiredString(data, "targetAuthorType")
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	snapshotTitle, err := firestoreString(data, "snapshotTitle")
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	snapshotBody, err := firestoreString(data, "snapshotBody")
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	snapshotRating64, err := firestoreOptionalInt64(data, "snapshotRating")
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	var snapshotRating *int
	if snapshotRating64 != nil {
		value := int(*snapshotRating64)
		snapshotRating = &value
	}

	reportCount64, err := firestoreRequiredInt64(data, "reportCount")
	if err != nil {
		return reportdom.ReportCase{}, err
	}
	reportCount := int(reportCount64)

	caseStatus, err := firestoreRequiredString(data, "status")
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	createdAt, err := firestoreRequiredTime(data, "createdAt")
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	updatedAt, err := firestoreRequiredTime(data, "updatedAt")
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	decidedAt, err := firestoreOptionalTime(data, "decidedAt")
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	decidedBy, err := firestoreString(data, "decidedBy")
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	decisionReason, err := firestoreString(data, "decisionReason")
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	entity := reportdom.ReportCase{
		ID:               reportdom.CaseID(id),
		TargetType:       reportdom.TargetType(targetType),
		TargetID:         targetID,
		TargetParentID:   targetParentID,
		TargetAuthorID:   targetAuthorID,
		TargetAuthorType: reportdom.ActorType(targetAuthorType),
		SnapshotTitle:    snapshotTitle,
		SnapshotBody:     snapshotBody,
		SnapshotRating:   snapshotRating,
		ReportCount:      reportCount,
		Status:           reportdom.CaseStatus(caseStatus),
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		DecidedAt:        decidedAt,
		DecidedBy:        decidedBy,
		DecisionReason:   decisionReason,
	}

	if err := entity.Validate(); err != nil {
		return reportdom.ReportCase{}, err
	}

	return entity, nil
}

func encodeReport(entity reportdom.Report) map[string]any {
	return map[string]any{
		"id":           string(entity.ID),
		"caseId":       string(entity.CaseID),
		"reporterType": string(entity.ReporterType),
		"reporterId":   entity.ReporterID,
		"companyId":    entity.CompanyID,
		"reason":       string(entity.Reason),
		"detail":       entity.Detail,
		"createdAt":    entity.CreatedAt.UTC(),
	}
}

func decodeReport(
	id string,
	caseID reportdom.CaseID,
	data map[string]any,
) (reportdom.Report, error) {
	storedCaseIDValue, err := firestoreString(data, "caseId")
	if err != nil {
		return reportdom.Report{}, err
	}

	storedCaseID := reportdom.CaseID(storedCaseIDValue)
	if storedCaseID == "" {
		storedCaseID = caseID
	}
	if storedCaseID != caseID {
		return reportdom.Report{}, reportdom.ErrInvalidCaseID
	}

	reporterType, err := firestoreRequiredString(data, "reporterType")
	if err != nil {
		return reportdom.Report{}, err
	}

	reporterID, err := firestoreRequiredString(data, "reporterId")
	if err != nil {
		return reportdom.Report{}, err
	}

	companyID, err := firestoreString(data, "companyId")
	if err != nil {
		return reportdom.Report{}, err
	}

	reason, err := firestoreRequiredString(data, "reason")
	if err != nil {
		return reportdom.Report{}, err
	}

	detail, err := firestoreString(data, "detail")
	if err != nil {
		return reportdom.Report{}, err
	}

	createdAt, err := firestoreRequiredTime(data, "createdAt")
	if err != nil {
		return reportdom.Report{}, err
	}

	entity := reportdom.Report{
		ID:           reportdom.ReportID(id),
		CaseID:       storedCaseID,
		ReporterType: reportdom.ActorType(reporterType),
		ReporterID:   reporterID,
		CompanyID:    companyID,
		Reason:       reportdom.ReportReason(reason),
		Detail:       detail,
		CreatedAt:    createdAt,
	}

	if err := entity.Validate(); err != nil {
		return reportdom.Report{}, err
	}

	return entity, nil
}

// ============================================================
// Helpers
// ============================================================

func normalizeReportPage(page common.Page) (int, int) {
	pageNumber := page.Number
	perPage := page.PerPage

	if pageNumber <= 0 {
		pageNumber = 1
	}
	if perPage <= 0 {
		perPage = 20
	}

	return pageNumber, perPage
}

func reportEffectiveTimeRange(primary, fallback common.TimeRange) common.TimeRange {
	if primary.From != nil || primary.To != nil {
		return primary
	}
	return fallback
}

func applyReportTimeRange(q firestore.Query, field string, timeRange common.TimeRange) firestore.Query {
	if timeRange.From != nil {
		q = q.Where(field, ">=", timeRange.From.UTC())
	}
	if timeRange.To != nil {
		q = q.Where(field, "<=", timeRange.To.UTC())
	}
	return q
}

func reportCountQuery(ctx context.Context, q firestore.Query) (int, error) {
	iter := q.Documents(ctx)
	defer iter.Stop()

	count := 0
	for {
		_, err := iter.Next()
		if err == iterator.Done {
			return count, nil
		}
		if err != nil {
			return 0, err
		}
		count++
	}
}

func reportIsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return status.Code(err) == codes.NotFound
}

var _ reportdom.RepositoryPort = (*ReportRepositoryFS)(nil)
