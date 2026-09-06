// backend/internal/adapters/out/firestore/report_decision_notification_repository_fs.go
package firestore

import (
	"context"
	"fmt"
	"math"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	common "narratives/internal/domain/common"
	reportdom "narratives/internal/domain/report"
)

const defaultReportDecisionNotificationCollection = "reportDecisionNotifications"

type ReportDecisionNotificationRepositoryFS struct {
	client     *firestore.Client
	collection string
}

func NewReportDecisionNotificationRepositoryFS(
	client *firestore.Client,
) *ReportDecisionNotificationRepositoryFS {
	return &ReportDecisionNotificationRepositoryFS{
		client:     client,
		collection: defaultReportDecisionNotificationCollection,
	}
}

func (r *ReportDecisionNotificationRepositoryFS) WithCollection(
	name string,
) *ReportDecisionNotificationRepositoryFS {
	if r != nil && name != "" {
		r.collection = name
	}
	return r
}

func (r *ReportDecisionNotificationRepositoryFS) collectionRef() *firestore.CollectionRef {
	return r.client.Collection(r.collection)
}

func (r *ReportDecisionNotificationRepositoryFS) notificationDoc(
	notificationID reportdom.DecisionNotificationID,
) *firestore.DocumentRef {
	return r.collectionRef().Doc(string(notificationID))
}

// ============================================================
// CreateIfAbsent
// ============================================================

func (r *ReportDecisionNotificationRepositoryFS) CreateIfAbsent(
	ctx context.Context,
	notification reportdom.DecisionNotification,
) (reportdom.CreateDecisionNotificationResult, error) {
	if r == nil || r.client == nil {
		return reportdom.CreateDecisionNotificationResult{},
			fmt.Errorf("report decision notification repository is not configured")
	}
	if err := notification.Validate(); err != nil {
		return reportdom.CreateDecisionNotificationResult{}, err
	}

	doc := r.notificationDoc(notification.ID)
	var result reportdom.CreateDecisionNotificationResult

	err := r.client.RunTransaction(
		ctx,
		func(ctx context.Context, tx *firestore.Transaction) error {
			snap, err := tx.Get(doc)
			if err == nil {
				existing, decodeErr := decodeReportDecisionNotification(
					snap.Ref.ID,
					snap.Data(),
				)
				if decodeErr != nil {
					return decodeErr
				}

				result = reportdom.CreateDecisionNotificationResult{
					Notification: existing,
					Created:      false,
				}
				return nil
			}
			if !reportIsNotFound(err) {
				return err
			}

			if err := tx.Create(
				doc,
				encodeReportDecisionNotification(notification),
			); err != nil {
				return err
			}

			result = reportdom.CreateDecisionNotificationResult{
				Notification: notification,
				Created:      true,
			}
			return nil
		},
	)
	if err != nil {
		return reportdom.CreateDecisionNotificationResult{}, err
	}

	return result, nil
}

// ============================================================
// GetByID
// ============================================================

func (r *ReportDecisionNotificationRepositoryFS) GetByID(
	ctx context.Context,
	notificationID reportdom.DecisionNotificationID,
) (reportdom.DecisionNotification, error) {
	if r == nil || r.client == nil {
		return reportdom.DecisionNotification{},
			fmt.Errorf("report decision notification repository is not configured")
	}
	if notificationID == "" {
		return reportdom.DecisionNotification{},
			reportdom.ErrInvalidDecisionNotificationID
	}

	snap, err := r.notificationDoc(notificationID).Get(ctx)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}

	return decodeReportDecisionNotification(
		snap.Ref.ID,
		snap.Data(),
	)
}

// ============================================================
// List
// ============================================================

func (r *ReportDecisionNotificationRepositoryFS) List(
	ctx context.Context,
	filter reportdom.DecisionNotificationFilter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[reportdom.DecisionNotification], error) {
	if r == nil || r.client == nil {
		return common.PageResult[reportdom.DecisionNotification]{},
			fmt.Errorf("report decision notification repository is not configured")
	}
	if filter.SearchQuery != "" {
		return common.PageResult[reportdom.DecisionNotification]{},
			fmt.Errorf("report decision notification: searchQuery is not supported")
	}

	q := r.collectionRef().Query

	if filter.RecipientType != nil {
		q = q.Where(
			"recipientType",
			"==",
			string(*filter.RecipientType),
		)
	}
	if filter.RecipientID != "" {
		q = q.Where(
			"recipientId",
			"==",
			filter.RecipientID,
		)
	}
	if filter.CompanyID != "" {
		q = q.Where(
			"companyId",
			"==",
			filter.CompanyID,
		)
	}
	if filter.TargetType != nil {
		q = q.Where(
			"targetType",
			"==",
			string(*filter.TargetType),
		)
	}
	if filter.TargetID != "" {
		q = q.Where(
			"targetId",
			"==",
			filter.TargetID,
		)
	}
	if filter.TargetParentID != "" {
		q = q.Where(
			"targetParentId",
			"==",
			filter.TargetParentID,
		)
	}
	if filter.DecisionStatus != nil {
		q = q.Where(
			"decisionStatus",
			"==",
			string(*filter.DecisionStatus),
		)
	}
	if filter.IsRead != nil {
		q = q.Where(
			"isRead",
			"==",
			*filter.IsRead,
		)
	}

	createdRange := reportEffectiveTimeRange(
		filter.CreatedAt,
		filter.Created,
	)

	q = applyReportTimeRange(
		q,
		"createdAt",
		createdRange,
	)
	q = applyReportTimeRange(
		q,
		"updatedAt",
		filter.Updated,
	)
	q = applyReportTimeRange(
		q,
		"decidedAt",
		filter.DecidedAt,
	)

	sortColumn := sort.Column
	if sortColumn == "" {
		sortColumn = "createdAt"
	}
	if _, ok := reportdom.AllowedDecisionNotificationSortColumns[sortColumn]; !ok {
		return common.PageResult[reportdom.DecisionNotification]{},
			fmt.Errorf(
				"report decision notification: invalid sort column: %s",
				sortColumn,
			)
	}

	orderDirection := firestore.Desc
	if sort.Order == common.SortAsc {
		orderDirection = firestore.Asc
	}

	q = q.OrderBy(
		sortColumn,
		orderDirection,
	)

	pageNumber, perPage := normalizeReportPage(page)
	offset := (pageNumber - 1) * perPage

	totalCount, err := reportCountQuery(ctx, q)
	if err != nil {
		return common.PageResult[reportdom.DecisionNotification]{}, err
	}

	totalPages := int(
		math.Ceil(
			float64(totalCount) /
				float64(perPage),
		),
	)
	if totalPages == 0 {
		totalPages = 1
	}

	items := make(
		[]reportdom.DecisionNotification,
		0,
		perPage,
	)

	iter := q.
		Offset(offset).
		Limit(perPage).
		Documents(ctx)
	defer iter.Stop()

	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.PageResult[reportdom.DecisionNotification]{}, err
		}

		item, err := decodeReportDecisionNotification(
			snap.Ref.ID,
			snap.Data(),
		)
		if err != nil {
			return common.PageResult[reportdom.DecisionNotification]{}, err
		}

		items = append(items, item)
	}

	return common.PageResult[reportdom.DecisionNotification]{
		Items:      items,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       pageNumber,
		PerPage:    perPage,
	}, nil
}

// ============================================================
// MarkRead
// ============================================================

func (r *ReportDecisionNotificationRepositoryFS) MarkRead(
	ctx context.Context,
	notificationID reportdom.DecisionNotificationID,
	recipientType reportdom.ActorType,
	recipientID string,
	readAt time.Time,
) (reportdom.DecisionNotification, error) {
	if r == nil || r.client == nil {
		return reportdom.DecisionNotification{},
			fmt.Errorf("report decision notification repository is not configured")
	}
	if notificationID == "" {
		return reportdom.DecisionNotification{},
			reportdom.ErrInvalidDecisionNotificationID
	}
	if err := recipientType.Validate(); err != nil {
		return reportdom.DecisionNotification{}, err
	}
	if recipientID == "" {
		return reportdom.DecisionNotification{},
			reportdom.ErrInvalidReporterID
	}
	if readAt.IsZero() {
		return reportdom.DecisionNotification{},
			reportdom.ErrInvalidDecisionNotificationReadAt
	}

	doc := r.notificationDoc(notificationID)
	var updated reportdom.DecisionNotification

	err := r.client.RunTransaction(
		ctx,
		func(ctx context.Context, tx *firestore.Transaction) error {
			snap, err := tx.Get(doc)
			if err != nil {
				return err
			}

			entity, err := decodeReportDecisionNotification(
				snap.Ref.ID,
				snap.Data(),
			)
			if err != nil {
				return err
			}

			if entity.RecipientType != recipientType ||
				entity.RecipientID != recipientID {
				return status.Error(
					codes.NotFound,
					"report decision notification not found",
				)
			}

			if err := entity.MarkRead(readAt); err != nil {
				return err
			}

			updates := []firestore.Update{
				{
					Path:  "isRead",
					Value: entity.IsRead(),
				},
				{
					Path:  "updatedAt",
					Value: entity.UpdatedAt.UTC(),
				},
			}

			if entity.ReadAt != nil {
				updates = append(
					updates,
					firestore.Update{
						Path:  "readAt",
						Value: entity.ReadAt.UTC(),
					},
				)
			}

			if err := tx.Update(doc, updates); err != nil {
				return err
			}

			updated = entity
			return nil
		},
	)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}

	return updated, nil
}

// ============================================================
// Encode / Decode
// ============================================================

func encodeReportDecisionNotification(
	entity reportdom.DecisionNotification,
) map[string]any {
	out := map[string]any{
		"id":               string(entity.ID),
		"notificationKind": string(entity.Kind()),
		"caseId":           string(entity.CaseID),
		"reportId":         string(entity.ReportID),
		"recipientType":    string(entity.RecipientType),
		"recipientId":      entity.RecipientID,
		"companyId":        entity.CompanyID,
		"targetType":       string(entity.TargetType),
		"targetId":         entity.TargetID,
		"targetParentId":   entity.TargetParentID,
		"reportReason":     string(entity.ReportReason),
		"reportDetail":     entity.ReportDetail,
		"decisionStatus":   string(entity.DecisionStatus),
		"decisionReason":   entity.DecisionReason,
		"decidedAt":        entity.DecidedAt.UTC(),
		"createdAt":        entity.CreatedAt.UTC(),
		"updatedAt":        entity.UpdatedAt.UTC(),
		"isRead":           entity.IsRead(),
	}

	if entity.ReadAt != nil {
		out["readAt"] = entity.ReadAt.UTC()
	}

	return out
}

func decodeReportDecisionNotification(
	id string,
	data map[string]any,
) (reportdom.DecisionNotification, error) {
	notificationKindValue, err := firestoreRequiredString(
		data,
		"notificationKind",
	)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}

	notificationKind := reportdom.NotificationKind(notificationKindValue)
	if err := notificationKind.Validate(); err != nil {
		return reportdom.DecisionNotification{}, err
	}

	caseIDValue, err := firestoreRequiredString(
		data,
		"caseId",
	)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}

	// REPORTER_DECISION では Validate で必須、
	// TARGET_ENFORCEMENT では空文字を許可する。
	reportIDValue, err := firestoreString(
		data,
		"reportId",
	)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}

	recipientTypeValue, err := firestoreRequiredString(
		data,
		"recipientType",
	)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}

	recipientID, err := firestoreRequiredString(
		data,
		"recipientId",
	)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}

	companyID, err := firestoreString(
		data,
		"companyId",
	)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}

	targetTypeValue, err := firestoreRequiredString(
		data,
		"targetType",
	)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}

	targetID, err := firestoreRequiredString(
		data,
		"targetId",
	)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}

	targetParentID, err := firestoreRequiredString(
		data,
		"targetParentId",
	)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}

	// REPORTER_DECISION では Validate で有効な ReportReason が必須、
	// TARGET_ENFORCEMENT では空文字を許可する。
	reportReasonValue, err := firestoreString(
		data,
		"reportReason",
	)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}

	reportDetail, err := firestoreString(
		data,
		"reportDetail",
	)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}

	decisionStatusValue, err := firestoreRequiredString(
		data,
		"decisionStatus",
	)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}

	decisionReason, err := firestoreRequiredString(
		data,
		"decisionReason",
	)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}

	decidedAt, err := firestoreRequiredTime(
		data,
		"decidedAt",
	)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}

	createdAt, err := firestoreRequiredTime(
		data,
		"createdAt",
	)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}

	updatedAt, err := firestoreRequiredTime(
		data,
		"updatedAt",
	)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}

	readAt, err := firestoreOptionalTime(
		data,
		"readAt",
	)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}

	entity := reportdom.DecisionNotification{
		ID:               reportdom.DecisionNotificationID(id),
		NotificationKind: notificationKind,

		CaseID:   reportdom.CaseID(caseIDValue),
		ReportID: reportdom.ReportID(reportIDValue),

		RecipientType: reportdom.ActorType(recipientTypeValue),
		RecipientID:   recipientID,
		CompanyID:     companyID,

		TargetType:     reportdom.TargetType(targetTypeValue),
		TargetID:       targetID,
		TargetParentID: targetParentID,

		ReportReason: reportdom.ReportReason(reportReasonValue),
		ReportDetail: reportDetail,

		DecisionStatus: reportdom.CaseStatus(decisionStatusValue),
		DecisionReason: decisionReason,
		DecidedAt:      decidedAt,

		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		ReadAt:    readAt,
	}

	if err := entity.Validate(); err != nil {
		return reportdom.DecisionNotification{}, err
	}

	return entity, nil
}

var _ reportdom.DecisionNotificationRepository = (*ReportDecisionNotificationRepositoryFS)(nil)
