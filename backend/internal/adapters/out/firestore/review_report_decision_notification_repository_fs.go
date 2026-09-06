// backend/internal/adapters/out/firestore/review_report_decision_notification_repository_fs.go
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
	reviewreport "narratives/internal/domain/reviewReport"
)

const defaultReviewReportDecisionNotificationCollection = "reviewReportDecisionNotifications"

type ReviewReportDecisionNotificationRepositoryFS struct {
	client     *firestore.Client
	collection string
}

func NewReviewReportDecisionNotificationRepositoryFS(
	client *firestore.Client,
) *ReviewReportDecisionNotificationRepositoryFS {
	return &ReviewReportDecisionNotificationRepositoryFS{
		client:     client,
		collection: defaultReviewReportDecisionNotificationCollection,
	}
}

func (r *ReviewReportDecisionNotificationRepositoryFS) WithCollection(
	name string,
) *ReviewReportDecisionNotificationRepositoryFS {
	if r != nil && name != "" {
		r.collection = name
	}
	return r
}

func (r *ReviewReportDecisionNotificationRepositoryFS) collectionRef() *firestore.CollectionRef {
	return r.client.Collection(r.collection)
}

func (r *ReviewReportDecisionNotificationRepositoryFS) notificationDoc(
	notificationID reviewreport.DecisionNotificationID,
) *firestore.DocumentRef {
	return r.collectionRef().Doc(string(notificationID))
}

// ============================================================
// CreateIfAbsent
// ============================================================

func (r *ReviewReportDecisionNotificationRepositoryFS) CreateIfAbsent(
	ctx context.Context,
	notification reviewreport.DecisionNotification,
) (reviewreport.CreateDecisionNotificationResult, error) {
	if r == nil || r.client == nil {
		return reviewreport.CreateDecisionNotificationResult{},
			fmt.Errorf("reviewReport decision notification repository is not configured")
	}
	if err := notification.Validate(); err != nil {
		return reviewreport.CreateDecisionNotificationResult{}, err
	}

	doc := r.notificationDoc(notification.ID)
	var result reviewreport.CreateDecisionNotificationResult

	err := r.client.RunTransaction(
		ctx,
		func(ctx context.Context, tx *firestore.Transaction) error {
			snap, err := tx.Get(doc)
			if err == nil {
				existing, decodeErr := decodeReviewReportDecisionNotification(
					snap.Ref.ID,
					snap.Data(),
				)
				if decodeErr != nil {
					return decodeErr
				}

				result = reviewreport.CreateDecisionNotificationResult{
					Notification: existing,
					Created:      false,
				}
				return nil
			}
			if !reviewReportIsNotFound(err) {
				return err
			}

			if err := tx.Create(
				doc,
				encodeReviewReportDecisionNotification(notification),
			); err != nil {
				return err
			}

			result = reviewreport.CreateDecisionNotificationResult{
				Notification: notification,
				Created:      true,
			}
			return nil
		},
	)
	if err != nil {
		return reviewreport.CreateDecisionNotificationResult{}, err
	}

	return result, nil
}

// ============================================================
// GetByID
// ============================================================

func (r *ReviewReportDecisionNotificationRepositoryFS) GetByID(
	ctx context.Context,
	notificationID reviewreport.DecisionNotificationID,
) (reviewreport.DecisionNotification, error) {
	if r == nil || r.client == nil {
		return reviewreport.DecisionNotification{},
			fmt.Errorf("reviewReport decision notification repository is not configured")
	}
	if notificationID == "" {
		return reviewreport.DecisionNotification{},
			reviewreport.ErrInvalidDecisionNotificationID
	}

	snap, err := r.notificationDoc(notificationID).Get(ctx)
	if err != nil {
		return reviewreport.DecisionNotification{}, err
	}

	return decodeReviewReportDecisionNotification(
		snap.Ref.ID,
		snap.Data(),
	)
}

// ============================================================
// List
// ============================================================

func (r *ReviewReportDecisionNotificationRepositoryFS) List(
	ctx context.Context,
	filter reviewreport.DecisionNotificationFilter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[reviewreport.DecisionNotification], error) {
	if r == nil || r.client == nil {
		return common.PageResult[reviewreport.DecisionNotification]{},
			fmt.Errorf("reviewReport decision notification repository is not configured")
	}
	if filter.SearchQuery != "" {
		return common.PageResult[reviewreport.DecisionNotification]{},
			fmt.Errorf("reviewReport decision notification: searchQuery is not supported")
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

	createdRange := reviewReportEffectiveTimeRange(
		filter.CreatedAt,
		filter.Created,
	)

	q = applyReviewReportTimeRange(
		q,
		"createdAt",
		createdRange,
	)
	q = applyReviewReportTimeRange(
		q,
		"updatedAt",
		filter.Updated,
	)
	q = applyReviewReportTimeRange(
		q,
		"decidedAt",
		filter.DecidedAt,
	)

	sortColumn := sort.Column
	if sortColumn == "" {
		sortColumn = "createdAt"
	}
	if _, ok := reviewreport.AllowedDecisionNotificationSortColumns[sortColumn]; !ok {
		return common.PageResult[reviewreport.DecisionNotification]{},
			fmt.Errorf(
				"reviewReport decision notification: invalid sort column: %s",
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

	pageNumber, perPage := normalizeReviewReportPage(page)
	offset := (pageNumber - 1) * perPage

	totalCount, err := reviewReportCountQuery(ctx, q)
	if err != nil {
		return common.PageResult[reviewreport.DecisionNotification]{}, err
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
		[]reviewreport.DecisionNotification,
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
			return common.PageResult[reviewreport.DecisionNotification]{}, err
		}

		item, err := decodeReviewReportDecisionNotification(
			snap.Ref.ID,
			snap.Data(),
		)
		if err != nil {
			return common.PageResult[reviewreport.DecisionNotification]{}, err
		}

		items = append(items, item)
	}

	return common.PageResult[reviewreport.DecisionNotification]{
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

func (r *ReviewReportDecisionNotificationRepositoryFS) MarkRead(
	ctx context.Context,
	notificationID reviewreport.DecisionNotificationID,
	recipientType reviewreport.ActorType,
	recipientID string,
	readAt time.Time,
) (reviewreport.DecisionNotification, error) {
	if r == nil || r.client == nil {
		return reviewreport.DecisionNotification{},
			fmt.Errorf("reviewReport decision notification repository is not configured")
	}
	if notificationID == "" {
		return reviewreport.DecisionNotification{},
			reviewreport.ErrInvalidDecisionNotificationID
	}
	if err := recipientType.Validate(); err != nil {
		return reviewreport.DecisionNotification{}, err
	}
	if recipientID == "" {
		return reviewreport.DecisionNotification{},
			reviewreport.ErrInvalidReporterID
	}
	if readAt.IsZero() {
		return reviewreport.DecisionNotification{},
			reviewreport.ErrInvalidDecisionNotificationReadAt
	}

	doc := r.notificationDoc(notificationID)
	var updated reviewreport.DecisionNotification

	err := r.client.RunTransaction(
		ctx,
		func(ctx context.Context, tx *firestore.Transaction) error {
			snap, err := tx.Get(doc)
			if err != nil {
				return err
			}

			entity, err := decodeReviewReportDecisionNotification(
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
					"review report decision notification not found",
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
		return reviewreport.DecisionNotification{}, err
	}

	return updated, nil
}

// ============================================================
// Encode / Decode
// ============================================================

func encodeReviewReportDecisionNotification(
	entity reviewreport.DecisionNotification,
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

func decodeReviewReportDecisionNotification(
	id string,
	data map[string]any,
) (reviewreport.DecisionNotification, error) {
	// notificationKind導入前の既存通知にはフィールドが存在しないため、
	// 欠損時は従来どおりREPORTER_DECISIONとして復元する。
	notificationKind := reviewreport.NotificationKindReporterDecision
	notificationKindValue, err := firestoreOptionalString(
		data,
		"notificationKind",
	)
	if err != nil {
		return reviewreport.DecisionNotification{}, err
	}
	if notificationKindValue != nil {
		notificationKind = reviewreport.NotificationKind(*notificationKindValue)
		if err := notificationKind.Validate(); err != nil {
			return reviewreport.DecisionNotification{}, err
		}
	}

	caseIDValue, err := firestoreRequiredString(
		data,
		"caseId",
	)
	if err != nil {
		return reviewreport.DecisionNotification{}, err
	}

	// REPORTER_DECISIONではValidateで必須、
	// TARGET_ENFORCEMENTでは空文字を許可する。
	reportIDValue, err := firestoreString(
		data,
		"reportId",
	)
	if err != nil {
		return reviewreport.DecisionNotification{}, err
	}

	recipientTypeValue, err := firestoreRequiredString(
		data,
		"recipientType",
	)
	if err != nil {
		return reviewreport.DecisionNotification{}, err
	}

	recipientID, err := firestoreRequiredString(
		data,
		"recipientId",
	)
	if err != nil {
		return reviewreport.DecisionNotification{}, err
	}

	companyID, err := firestoreString(
		data,
		"companyId",
	)
	if err != nil {
		return reviewreport.DecisionNotification{}, err
	}

	targetTypeValue, err := firestoreRequiredString(
		data,
		"targetType",
	)
	if err != nil {
		return reviewreport.DecisionNotification{}, err
	}

	targetID, err := firestoreRequiredString(
		data,
		"targetId",
	)
	if err != nil {
		return reviewreport.DecisionNotification{}, err
	}

	targetParentID, err := firestoreRequiredString(
		data,
		"targetParentId",
	)
	if err != nil {
		return reviewreport.DecisionNotification{}, err
	}

	// REPORTER_DECISIONではValidateで有効なReportReasonが必須、
	// TARGET_ENFORCEMENTでは空文字を許可する。
	reportReasonValue, err := firestoreString(
		data,
		"reportReason",
	)
	if err != nil {
		return reviewreport.DecisionNotification{}, err
	}

	reportDetail, err := firestoreString(
		data,
		"reportDetail",
	)
	if err != nil {
		return reviewreport.DecisionNotification{}, err
	}

	decisionStatusValue, err := firestoreRequiredString(
		data,
		"decisionStatus",
	)
	if err != nil {
		return reviewreport.DecisionNotification{}, err
	}

	decisionReason, err := firestoreRequiredString(
		data,
		"decisionReason",
	)
	if err != nil {
		return reviewreport.DecisionNotification{}, err
	}

	decidedAt, err := firestoreRequiredTime(
		data,
		"decidedAt",
	)
	if err != nil {
		return reviewreport.DecisionNotification{}, err
	}

	createdAt, err := firestoreRequiredTime(
		data,
		"createdAt",
	)
	if err != nil {
		return reviewreport.DecisionNotification{}, err
	}

	updatedAt, err := firestoreRequiredTime(
		data,
		"updatedAt",
	)
	if err != nil {
		return reviewreport.DecisionNotification{}, err
	}

	readAt, err := firestoreOptionalTime(
		data,
		"readAt",
	)
	if err != nil {
		return reviewreport.DecisionNotification{}, err
	}

	entity := reviewreport.DecisionNotification{
		ID:               reviewreport.DecisionNotificationID(id),
		NotificationKind: notificationKind,

		CaseID:   reviewreport.CaseID(caseIDValue),
		ReportID: reviewreport.ReportID(reportIDValue),

		RecipientType: reviewreport.ActorType(recipientTypeValue),
		RecipientID:   recipientID,
		CompanyID:     companyID,

		TargetType:     reviewreport.TargetType(targetTypeValue),
		TargetID:       targetID,
		TargetParentID: targetParentID,

		ReportReason: reviewreport.ReportReason(reportReasonValue),
		ReportDetail: reportDetail,

		DecisionStatus: reviewreport.CaseStatus(decisionStatusValue),
		DecisionReason: decisionReason,
		DecidedAt:      decidedAt,

		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		ReadAt:    readAt,
	}

	if err := entity.Validate(); err != nil {
		return reviewreport.DecisionNotification{}, err
	}

	return entity, nil
}

var _ reviewreport.DecisionNotificationRepository = (*ReviewReportDecisionNotificationRepositoryFS)(nil)
