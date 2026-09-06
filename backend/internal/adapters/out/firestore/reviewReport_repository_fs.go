// backend/internal/adapters/out/firestore/reviewReport_repository_fs.go
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
	reviewreport "narratives/internal/domain/reviewReport"
)

const (
	defaultReviewReportCaseCollection = "reviewReportCases"
	defaultReviewReportSubCollection  = "reports"
)

type ReviewReportRepositoryFS struct {
	client           *firestore.Client
	caseCollection   string
	reportCollection string
}

func NewReviewReportRepositoryFS(client *firestore.Client) *ReviewReportRepositoryFS {
	return &ReviewReportRepositoryFS{
		client:           client,
		caseCollection:   defaultReviewReportCaseCollection,
		reportCollection: defaultReviewReportSubCollection,
	}
}

func (r *ReviewReportRepositoryFS) WithCaseCollection(name string) *ReviewReportRepositoryFS {
	if r != nil && name != "" {
		r.caseCollection = name
	}
	return r
}

func (r *ReviewReportRepositoryFS) WithReportCollection(name string) *ReviewReportRepositoryFS {
	if r != nil && name != "" {
		r.reportCollection = name
	}
	return r
}

func (r *ReviewReportRepositoryFS) caseDoc(caseID reviewreport.CaseID) *firestore.DocumentRef {
	return r.client.Collection(r.caseCollection).Doc(string(caseID))
}

func (r *ReviewReportRepositoryFS) reportsCol(caseID reviewreport.CaseID) *firestore.CollectionRef {
	return r.caseDoc(caseID).Collection(r.reportCollection)
}

func (r *ReviewReportRepositoryFS) reportDoc(caseID reviewreport.CaseID, reportID reviewreport.ReportID) *firestore.DocumentRef {
	return r.reportsCol(caseID).Doc(string(reportID))
}

// ============================================================
// CaseRepository
// ============================================================

func (r *ReviewReportRepositoryFS) GetCase(ctx context.Context, caseID reviewreport.CaseID) (reviewreport.ReportCase, error) {
	if r == nil || r.client == nil {
		return reviewreport.ReportCase{}, fmt.Errorf("reviewReport repository is not configured")
	}
	if caseID == "" {
		return reviewreport.ReportCase{}, reviewreport.ErrInvalidCaseID
	}

	snap, err := r.caseDoc(caseID).Get(ctx)
	if err != nil {
		return reviewreport.ReportCase{}, err
	}

	return decodeReviewReportCase(snap.Ref.ID, snap.Data())
}

// IsAvatarResaleSuspended reports whether the avatar is currently blocked from
// using the resale service by an Admin review-report decision.
//
// The AVATAR review-report case ID is deterministic (avatar_{avatarId}), so the
// check is a direct document lookup rather than a collection query. A missing
// case means the avatar has no resale suspension. Only REMOVED is treated as
// suspended; PENDING and KEPT remain allowed.
func (r *ReviewReportRepositoryFS) IsAvatarResaleSuspended(
	ctx context.Context,
	avatarID string,
) (bool, error) {
	if r == nil || r.client == nil {
		return false, fmt.Errorf("reviewReport repository is not configured")
	}
	if avatarID == "" {
		return false, reviewreport.ErrInvalidTargetID
	}

	caseID, err := reviewreport.BuildCaseID(
		reviewreport.TargetTypeAvatar,
		avatarID,
	)
	if err != nil {
		return false, err
	}

	reportCase, err := r.GetCase(ctx, caseID)
	if err != nil {
		if reviewReportIsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	if reportCase.TargetType != reviewreport.TargetTypeAvatar {
		return false, reviewreport.ErrInvalidTargetType
	}
	if reportCase.TargetID != avatarID {
		return false, reviewreport.ErrInvalidTargetID
	}

	return reportCase.Status == reviewreport.CaseStatusRemoved, nil
}

func (r *ReviewReportRepositoryFS) ListCases(ctx context.Context, filter reviewreport.CaseFilter, sort common.Sort, page common.Page) (common.PageResult[reviewreport.ReportCase], error) {
	if r == nil || r.client == nil {
		return common.PageResult[reviewreport.ReportCase]{}, fmt.Errorf("reviewReport repository is not configured")
	}
	if filter.SearchQuery != "" {
		return common.PageResult[reviewreport.ReportCase]{}, fmt.Errorf("reviewReport: searchQuery is not supported")
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

	createdRange := reviewReportEffectiveTimeRange(filter.CreatedAt, filter.Created)
	updatedRange := reviewReportEffectiveTimeRange(filter.UpdatedAt, filter.Updated)

	q = applyReviewReportTimeRange(q, "createdAt", createdRange)
	q = applyReviewReportTimeRange(q, "updatedAt", updatedRange)
	q = applyReviewReportTimeRange(q, "decidedAt", filter.DecidedAt)

	sortColumn := sort.Column
	if sortColumn == "" {
		sortColumn = "updatedAt"
	}
	if _, ok := reviewreport.AllowedCaseSortColumns[sortColumn]; !ok {
		return common.PageResult[reviewreport.ReportCase]{}, fmt.Errorf("reviewReport: invalid case sort column: %s", sortColumn)
	}

	orderDirection := firestore.Desc
	if sort.Order == common.SortAsc {
		orderDirection = firestore.Asc
	}

	q = q.OrderBy(sortColumn, orderDirection)

	pageNumber, perPage := normalizeReviewReportPage(page)
	offset := (pageNumber - 1) * perPage

	totalCount, err := reviewReportCountQuery(ctx, q)
	if err != nil {
		return common.PageResult[reviewreport.ReportCase]{}, err
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(perPage)))
	if totalPages == 0 {
		totalPages = 1
	}

	items := make([]reviewreport.ReportCase, 0, perPage)
	iter := q.Offset(offset).Limit(perPage).Documents(ctx)
	defer iter.Stop()

	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.PageResult[reviewreport.ReportCase]{}, err
		}

		item, err := decodeReviewReportCase(snap.Ref.ID, snap.Data())
		if err != nil {
			return common.PageResult[reviewreport.ReportCase]{}, err
		}

		items = append(items, item)
	}

	return common.PageResult[reviewreport.ReportCase]{
		Items:      items,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       pageNumber,
		PerPage:    perPage,
	}, nil
}

func (r *ReviewReportRepositoryFS) UpdateCase(ctx context.Context, caseID reviewreport.CaseID, patch reviewreport.CasePatch) (reviewreport.ReportCase, error) {
	if r == nil || r.client == nil {
		return reviewreport.ReportCase{}, fmt.Errorf("reviewReport repository is not configured")
	}
	if caseID == "" {
		return reviewreport.ReportCase{}, reviewreport.ErrInvalidCaseID
	}

	doc := r.caseDoc(caseID)
	var updated reviewreport.ReportCase

	err := r.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(doc)
		if err != nil {
			return err
		}

		entity, err := decodeReviewReportCase(snap.Ref.ID, snap.Data())
		if err != nil {
			return err
		}

		if patch.ReportCount != nil {
			entity.ReportCount = *patch.ReportCount
		}
		if patch.Status != nil {
			entity.Status = *patch.Status

			if entity.Status == reviewreport.CaseStatusPending {
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

		data := encodeReviewReportCase(entity)
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
		return reviewreport.ReportCase{}, err
	}

	return updated, nil
}

// ============================================================
// ReportRepository
// ============================================================

func (r *ReviewReportRepositoryFS) GetReport(ctx context.Context, caseID reviewreport.CaseID, reportID reviewreport.ReportID) (reviewreport.Report, error) {
	if r == nil || r.client == nil {
		return reviewreport.Report{}, fmt.Errorf("reviewReport repository is not configured")
	}
	if caseID == "" {
		return reviewreport.Report{}, reviewreport.ErrInvalidCaseID
	}
	if reportID == "" {
		return reviewreport.Report{}, reviewreport.ErrInvalidReportID
	}

	snap, err := r.reportDoc(caseID, reportID).Get(ctx)
	if err != nil {
		return reviewreport.Report{}, err
	}

	return decodeReviewReport(snap.Ref.ID, caseID, snap.Data())
}

func (r *ReviewReportRepositoryFS) ListReports(ctx context.Context, caseID reviewreport.CaseID, filter reviewreport.ReportFilter, sort common.Sort, page common.Page) (common.PageResult[reviewreport.Report], error) {
	if r == nil || r.client == nil {
		return common.PageResult[reviewreport.Report]{}, fmt.Errorf("reviewReport repository is not configured")
	}
	if caseID == "" {
		return common.PageResult[reviewreport.Report]{}, reviewreport.ErrInvalidCaseID
	}
	if filter.CaseID != "" && filter.CaseID != caseID {
		return common.PageResult[reviewreport.Report]{}, reviewreport.ErrInvalidCaseID
	}
	if filter.SearchQuery != "" {
		return common.PageResult[reviewreport.Report]{}, fmt.Errorf("reviewReport: searchQuery is not supported")
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

	createdRange := reviewReportEffectiveTimeRange(filter.CreatedAt, filter.Created)
	q = applyReviewReportTimeRange(q, "createdAt", createdRange)

	sortColumn := sort.Column
	if sortColumn == "" {
		sortColumn = "createdAt"
	}
	if _, ok := reviewreport.AllowedReportSortColumns[sortColumn]; !ok {
		return common.PageResult[reviewreport.Report]{}, fmt.Errorf("reviewReport: invalid report sort column: %s", sortColumn)
	}

	orderDirection := firestore.Desc
	if sort.Order == common.SortAsc {
		orderDirection = firestore.Asc
	}

	q = q.OrderBy(sortColumn, orderDirection)

	pageNumber, perPage := normalizeReviewReportPage(page)
	offset := (pageNumber - 1) * perPage

	totalCount, err := reviewReportCountQuery(ctx, q)
	if err != nil {
		return common.PageResult[reviewreport.Report]{}, err
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(perPage)))
	if totalPages == 0 {
		totalPages = 1
	}

	items := make([]reviewreport.Report, 0, perPage)
	iter := q.Offset(offset).Limit(perPage).Documents(ctx)
	defer iter.Stop()

	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.PageResult[reviewreport.Report]{}, err
		}

		item, err := decodeReviewReport(snap.Ref.ID, caseID, snap.Data())
		if err != nil {
			return common.PageResult[reviewreport.Report]{}, err
		}

		items = append(items, item)
	}

	return common.PageResult[reviewreport.Report]{
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

func (r *ReviewReportRepositoryFS) AddReport(ctx context.Context, initialCase reviewreport.ReportCase, report reviewreport.Report) (reviewreport.AddReportResult, error) {
	if r == nil || r.client == nil {
		return reviewreport.AddReportResult{}, fmt.Errorf("reviewReport repository is not configured")
	}
	if err := initialCase.Validate(); err != nil {
		return reviewreport.AddReportResult{}, err
	}
	if err := report.Validate(); err != nil {
		return reviewreport.AddReportResult{}, err
	}
	if report.CaseID != initialCase.ID {
		return reviewreport.AddReportResult{}, reviewreport.ErrInvalidCaseID
	}

	expectedCaseID, err := reviewreport.BuildCaseID(initialCase.TargetType, initialCase.TargetID)
	if err != nil {
		return reviewreport.AddReportResult{}, err
	}
	if expectedCaseID != initialCase.ID {
		return reviewreport.AddReportResult{}, reviewreport.ErrInvalidCaseID
	}

	expectedReportID, err := reviewreport.BuildReporterKey(report.ReporterType, report.ReporterID)
	if err != nil {
		return reviewreport.AddReportResult{}, err
	}
	if expectedReportID != report.ID {
		return reviewreport.AddReportResult{}, reviewreport.ErrInvalidReportID
	}

	caseRef := r.caseDoc(initialCase.ID)
	reportRef := r.reportDoc(initialCase.ID, report.ID)
	var result reviewreport.AddReportResult

	err = r.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		caseExists := true
		var currentCase reviewreport.ReportCase

		caseSnap, caseErr := tx.Get(caseRef)
		if caseErr != nil {
			if reviewReportIsNotFound(caseErr) {
				caseExists = false
			} else {
				return caseErr
			}
		} else {
			decodedCase, decodeErr := decodeReviewReportCase(caseSnap.Ref.ID, caseSnap.Data())
			if decodeErr != nil {
				return decodeErr
			}
			currentCase = decodedCase
		}

		reportSnap, reportErr := tx.Get(reportRef)
		if reportErr == nil {
			if !caseExists {
				return fmt.Errorf("reviewReport: report exists without parent case")
			}

			existingReport, decodeErr := decodeReviewReport(
				reportSnap.Ref.ID,
				initialCase.ID,
				reportSnap.Data(),
			)
			if decodeErr != nil {
				return decodeErr
			}

			result = reviewreport.AddReportResult{
				Case:          currentCase,
				Report:        existingReport,
				CaseCreated:   false,
				ReportCreated: false,
			}
			return nil
		}
		if !reviewReportIsNotFound(reportErr) {
			return reportErr
		}

		if !caseExists {
			currentCase = initialCase
		}

		wasKept := caseExists && currentCase.Status == reviewreport.CaseStatusKept

		if err := currentCase.IncrementReportCount(report.CreatedAt); err != nil {
			return err
		}
		if err := currentCase.Validate(); err != nil {
			return err
		}

		if !caseExists {
			if err := tx.Create(caseRef, encodeReviewReportCase(currentCase)); err != nil {
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

		if err := tx.Create(reportRef, encodeReviewReport(report)); err != nil {
			return err
		}

		result = reviewreport.AddReportResult{
			Case:          currentCase,
			Report:        report,
			CaseCreated:   !caseExists,
			ReportCreated: true,
		}
		return nil
	})
	if err != nil {
		return reviewreport.AddReportResult{}, err
	}

	return result, nil
}

// ============================================================
// Encode / Decode
// ============================================================

func encodeReviewReportCase(entity reviewreport.ReportCase) map[string]any {
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

func decodeReviewReportCase(id string, data map[string]any) (reviewreport.ReportCase, error) {
	targetType, err := firestoreRequiredString(data, "targetType")
	if err != nil {
		return reviewreport.ReportCase{}, err
	}

	targetID, err := firestoreRequiredString(data, "targetId")
	if err != nil {
		return reviewreport.ReportCase{}, err
	}

	targetParentID, err := firestoreRequiredString(data, "targetParentId")
	if err != nil {
		return reviewreport.ReportCase{}, err
	}

	targetAuthorID, err := firestoreRequiredString(data, "targetAuthorId")
	if err != nil {
		return reviewreport.ReportCase{}, err
	}

	targetAuthorType, err := firestoreRequiredString(data, "targetAuthorType")
	if err != nil {
		return reviewreport.ReportCase{}, err
	}

	snapshotTitle, err := firestoreString(data, "snapshotTitle")
	if err != nil {
		return reviewreport.ReportCase{}, err
	}

	snapshotBody, err := firestoreString(data, "snapshotBody")
	if err != nil {
		return reviewreport.ReportCase{}, err
	}

	snapshotRating64, err := firestoreOptionalInt64(data, "snapshotRating")
	if err != nil {
		return reviewreport.ReportCase{}, err
	}

	var snapshotRating *int
	if snapshotRating64 != nil {
		value := int(*snapshotRating64)
		snapshotRating = &value
	}

	reportCount64, err := firestoreRequiredInt64(data, "reportCount")
	if err != nil {
		return reviewreport.ReportCase{}, err
	}
	reportCount := int(reportCount64)

	caseStatus, err := firestoreRequiredString(data, "status")
	if err != nil {
		return reviewreport.ReportCase{}, err
	}

	createdAt, err := firestoreRequiredTime(data, "createdAt")
	if err != nil {
		return reviewreport.ReportCase{}, err
	}

	updatedAt, err := firestoreRequiredTime(data, "updatedAt")
	if err != nil {
		return reviewreport.ReportCase{}, err
	}

	decidedAt, err := firestoreOptionalTime(data, "decidedAt")
	if err != nil {
		return reviewreport.ReportCase{}, err
	}

	decidedBy, err := firestoreString(data, "decidedBy")
	if err != nil {
		return reviewreport.ReportCase{}, err
	}

	decisionReason, err := firestoreString(data, "decisionReason")
	if err != nil {
		return reviewreport.ReportCase{}, err
	}

	entity := reviewreport.ReportCase{
		ID:               reviewreport.CaseID(id),
		TargetType:       reviewreport.TargetType(targetType),
		TargetID:         targetID,
		TargetParentID:   targetParentID,
		TargetAuthorID:   targetAuthorID,
		TargetAuthorType: reviewreport.ActorType(targetAuthorType),
		SnapshotTitle:    snapshotTitle,
		SnapshotBody:     snapshotBody,
		SnapshotRating:   snapshotRating,
		ReportCount:      reportCount,
		Status:           reviewreport.CaseStatus(caseStatus),
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		DecidedAt:        decidedAt,
		DecidedBy:        decidedBy,
		DecisionReason:   decisionReason,
	}

	if err := entity.Validate(); err != nil {
		return reviewreport.ReportCase{}, err
	}

	return entity, nil
}

func encodeReviewReport(entity reviewreport.Report) map[string]any {
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

func decodeReviewReport(id string, caseID reviewreport.CaseID, data map[string]any) (reviewreport.Report, error) {
	storedCaseIDValue, err := firestoreString(data, "caseId")
	if err != nil {
		return reviewreport.Report{}, err
	}

	storedCaseID := reviewreport.CaseID(storedCaseIDValue)
	if storedCaseID == "" {
		storedCaseID = caseID
	}
	if storedCaseID != caseID {
		return reviewreport.Report{}, reviewreport.ErrInvalidCaseID
	}

	reporterType, err := firestoreRequiredString(data, "reporterType")
	if err != nil {
		return reviewreport.Report{}, err
	}

	reporterID, err := firestoreRequiredString(data, "reporterId")
	if err != nil {
		return reviewreport.Report{}, err
	}

	companyID, err := firestoreString(data, "companyId")
	if err != nil {
		return reviewreport.Report{}, err
	}

	reason, err := firestoreRequiredString(data, "reason")
	if err != nil {
		return reviewreport.Report{}, err
	}

	detail, err := firestoreString(data, "detail")
	if err != nil {
		return reviewreport.Report{}, err
	}

	createdAt, err := firestoreRequiredTime(data, "createdAt")
	if err != nil {
		return reviewreport.Report{}, err
	}

	entity := reviewreport.Report{
		ID:           reviewreport.ReportID(id),
		CaseID:       storedCaseID,
		ReporterType: reviewreport.ActorType(reporterType),
		ReporterID:   reporterID,
		CompanyID:    companyID,
		Reason:       reviewreport.ReportReason(reason),
		Detail:       detail,
		CreatedAt:    createdAt,
	}

	if err := entity.Validate(); err != nil {
		return reviewreport.Report{}, err
	}

	return entity, nil
}

// ============================================================
// Helpers
// ============================================================

func normalizeReviewReportPage(page common.Page) (int, int) {
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

func reviewReportEffectiveTimeRange(primary, fallback common.TimeRange) common.TimeRange {
	if primary.From != nil || primary.To != nil {
		return primary
	}
	return fallback
}

func applyReviewReportTimeRange(q firestore.Query, field string, timeRange common.TimeRange) firestore.Query {
	if timeRange.From != nil {
		q = q.Where(field, ">=", timeRange.From.UTC())
	}
	if timeRange.To != nil {
		q = q.Where(field, "<=", timeRange.To.UTC())
	}
	return q
}

func reviewReportCountQuery(ctx context.Context, q firestore.Query) (int, error) {
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

func reviewReportIsNotFound(err error) bool {
	if err == nil {
		return false
	}

	return status.Code(err) == codes.NotFound
}

var _ reviewreport.RepositoryPort = (*ReviewReportRepositoryFS)(nil)
