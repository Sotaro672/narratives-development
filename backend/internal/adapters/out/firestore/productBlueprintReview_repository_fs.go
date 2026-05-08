// backend/internal/adapters/out/firestore/productBlueprintReview_repository_fs.go
package firestore

import (
	"context"
	"math"
	"strings"
	"time"

	domcommon "narratives/internal/domain/common"
	pbr "narratives/internal/domain/productBlueprintReview"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ✅ コレクション構成（期待値）
// - 集計ドキュメント: productBlueprintReviewAggregates/{productBlueprintId}
// - レビュー本体:      productBlueprintReviewAggregates/{productBlueprintId}/productBlueprintReviews/{reviewId}
const (
	defaultProductBlueprintReviewAggregateCollection = "productBlueprintReviewAggregates"
	defaultProductBlueprintReviewSubCollection       = "productBlueprintReviews"
)

// ProductBlueprintReviewRepositoryFS implements productBlueprintReview.Repository using Firestore.
//
// ✅ IMPORTANT:
//   - review 本体: productBlueprintReviewAggregates/{productBlueprintId}/productBlueprintReviews/{reviewId}
//   - 集計初期化: productBlueprintReviewAggregates/{productBlueprintId}
//     -> ProductBlueprintUsecase の reviewInit port としても利用する（InitForProductBlueprint）
//
// ✅ 集計反映の期待値:
//   - サブコレクションへ review を起票
//   - 集計結果は productBlueprintReviewAggregates/{productBlueprintId} に反映される（本実装では Create/Update/Delete 時に transaction で反映）
type ProductBlueprintReviewRepositoryFS struct {
	client     *firestore.Client
	collection string
	now        func() time.Time
}

func NewProductBlueprintReviewRepositoryFS(client *firestore.Client) *ProductBlueprintReviewRepositoryFS {
	return &ProductBlueprintReviewRepositoryFS{
		client:     client,
		collection: defaultProductBlueprintReviewSubCollection, // サブコレクション名（WithCollectionで変更可）
		now:        time.Now,
	}
}

func (r *ProductBlueprintReviewRepositoryFS) WithCollection(name string) *ProductBlueprintReviewRepositoryFS {
	if r != nil && strings.TrimSpace(name) != "" {
		r.collection = strings.TrimSpace(name)
	}
	return r
}

func (r *ProductBlueprintReviewRepositoryFS) WithNow(f func() time.Time) *ProductBlueprintReviewRepositoryFS {
	if r != nil && f != nil {
		r.now = f
	}
	return r
}

func (r *ProductBlueprintReviewRepositoryFS) aggregateDoc(productBlueprintID string) *firestore.DocumentRef {
	return r.client.Collection(defaultProductBlueprintReviewAggregateCollection).Doc(productBlueprintID)
}

func (r *ProductBlueprintReviewRepositoryFS) reviewsCol(productBlueprintID string) *firestore.CollectionRef {
	return r.aggregateDoc(productBlueprintID).Collection(r.collection)
}

func (r *ProductBlueprintReviewRepositoryFS) reviewDoc(productBlueprintID, reviewID string) *firestore.DocumentRef {
	return r.reviewsCol(productBlueprintID).Doc(reviewID)
}

// ============================================================
// ✅ Initializer (for ProductBlueprintUsecase)
// ============================================================
//
// productBlueprint 起票時に「口コミの集計ドキュメント」を作成する。
// - review 本体は投稿時に作られるため、ここでは空レビューは作らない（validationに抵触する）
// - 既に存在する場合はOK（idempotent）
func (r *ProductBlueprintReviewRepositoryFS) InitForProductBlueprint(
	ctx context.Context,
	productBlueprintID string,
	companyID string,
	createdAt time.Time,
	createdBy *string,
) error {
	if r == nil || r.client == nil {
		return pbr.ErrInternal
	}
	pbID := strings.TrimSpace(productBlueprintID)
	if pbID == "" {
		return pbr.ErrInvalid
	}

	now := createdAt
	if now.IsZero() {
		now = r.now().UTC()
	} else {
		now = now.UTC()
	}

	doc := r.aggregateDoc(pbID)

	payload := map[string]any{
		"productBlueprintId": pbID,
		"companyId":          companyID,

		"totalCount":    0,
		"averageRating": 0.0,

		"rating5Count": 0,
		"rating4Count": 0,
		"rating3Count": 0,
		"rating2Count": 0,
		"rating1Count": 0,

		"createdAt": now,
		"updatedAt": now,
	}

	if createdBy != nil && strings.TrimSpace(*createdBy) != "" {
		payload["createdBy"] = *createdBy
		payload["updatedBy"] = *createdBy
	}

	_, err := doc.Create(ctx, payload)
	if err == nil {
		return nil
	}
	if status.Code(err) == codes.AlreadyExists {
		// idempotent
		return nil
	}
	return err
}

// ============================================================
// Common CRUD (domcommon.Repository)
// ============================================================

func (r *ProductBlueprintReviewRepositoryFS) GetByID(ctx context.Context, id string) (pbr.Review, error) {
	if r == nil || r.client == nil {
		return pbr.Review{}, pbr.ErrInternal
	}
	reviewID := strings.TrimSpace(id)
	if reviewID == "" {
		return pbr.Review{}, pbr.ErrInvalid
	}

	// このドメインのID単体Getは、Firestoreの構造上 productBlueprintID がないと特定できないため禁止
	// 必要なら adapter 層で別途インデックス（reviewId -> productBlueprintId）を持つ設計にしてください。
	return pbr.Review{}, pbr.ErrInvalid
}

func (r *ProductBlueprintReviewRepositoryFS) Create(ctx context.Context, entity pbr.Review) (pbr.Review, error) {
	if r == nil || r.client == nil {
		return pbr.Review{}, pbr.ErrInternal
	}

	pbID := strings.TrimSpace(entity.ProductBlueprintID)
	id := strings.TrimSpace(string(entity.ID))
	if pbID == "" || id == "" {
		return pbr.Review{}, pbr.ErrInvalid
	}

	reviewDoc := r.reviewDoc(pbID, id)
	aggDoc := r.aggregateDoc(pbID)

	err := r.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// review: Create (fail if exists)
		if err := tx.Create(reviewDoc, encodeReviewDoc(entity)); err != nil {
			if isAlreadyExists(err) {
				return pbr.ErrConflict
			}
			return err
		}

		// aggregate: reflect (only when published)
		if entity.Status == pbr.ReviewStatusPublished {
			now := r.now().UTC()
			updates := buildAggregateDeltaOnCreatePublished(entity.Rating, now)
			return tx.Set(aggDoc, updates, firestore.MergeAll)
		}
		return nil
	})
	if err != nil {
		return pbr.Review{}, err
	}

	return entity, nil
}

func (r *ProductBlueprintReviewRepositoryFS) Update(ctx context.Context, id string, patch pbr.Patch) (pbr.Review, error) {
	if r == nil || r.client == nil {
		return pbr.Review{}, pbr.ErrInternal
	}
	reviewID := strings.TrimSpace(id)
	if reviewID == "" {
		return pbr.Review{}, pbr.ErrInvalid
	}

	// このadapterは subcollection 構造のため、Update には productBlueprintId が必要
	// ここでは patch からは取れないので、reviewID単体Updateは不可
	// 必要なら「UpdateByProductBlueprintID(ctx, productBlueprintID, reviewID, patch)」を別途定義してください。
	return pbr.Review{}, pbr.ErrInvalid
}

func (r *ProductBlueprintReviewRepositoryFS) Delete(ctx context.Context, id string) error {
	if r == nil || r.client == nil {
		return pbr.ErrInternal
	}
	reviewID := strings.TrimSpace(id)
	if reviewID == "" {
		return pbr.ErrInvalid
	}

	// subcollection 構造のため、reviewID単体Deleteは不可（productBlueprintIdが必要）
	return pbr.ErrInvalid
}

// ============================================================
// List (domcommon.RepositoryList)
// ============================================================

func (r *ProductBlueprintReviewRepositoryFS) List(
	ctx context.Context,
	filter pbr.Filter,
	sort domcommon.Sort,
	page domcommon.Page,
) (domcommon.PageResult[pbr.Review], error) {

	if r == nil || r.client == nil {
		return domcommon.PageResult[pbr.Review]{}, pbr.ErrInternal
	}

	// Firestoreで部分一致検索は基本不可
	if strings.TrimSpace(filter.SearchQuery) != "" {
		return domcommon.PageResult[pbr.Review]{}, pbr.ErrInvalid
	}

	// ✅ subcollection 前提のため ProductBlueprintID 必須
	if filter.ProductBlueprintID == nil || strings.TrimSpace(*filter.ProductBlueprintID) == "" {
		return domcommon.PageResult[pbr.Review]{}, pbr.ErrInvalid
	}
	pbID := strings.TrimSpace(*filter.ProductBlueprintID)

	q := r.reviewsCol(pbID).Query

	// equality filters
	if filter.AvatarID != nil && strings.TrimSpace(*filter.AvatarID) != "" {
		q = q.Where("avatarId", "==", strings.TrimSpace(*filter.AvatarID))
	}
	if filter.Status != nil {
		q = q.Where("status", "==", string(*filter.Status))
	}
	if filter.Rating != nil {
		q = q.Where("rating", "==", int(*filter.Rating))
	}
	if filter.RatingMin != nil {
		q = q.Where("rating", ">=", int(*filter.RatingMin))
	}
	if filter.RatingMax != nil {
		q = q.Where("rating", "<=", int(*filter.RatingMax))
	}

	// reviewedAt range
	if filter.Reviewed.From != nil {
		q = q.Where("reviewedAt", ">=", filter.Reviewed.From.UTC())
	}
	if filter.Reviewed.To != nil {
		q = q.Where("reviewedAt", "<=", filter.Reviewed.To.UTC())
	}

	// created/updated range（共通）
	if filter.Created.From != nil {
		q = q.Where("createdAt", ">=", filter.Created.From.UTC())
	}
	if filter.Created.To != nil {
		q = q.Where("createdAt", "<=", filter.Created.To.UTC())
	}
	if filter.Updated.From != nil {
		q = q.Where("updatedAt", ">=", filter.Updated.From.UTC())
	}
	if filter.Updated.To != nil {
		q = q.Where("updatedAt", "<=", filter.Updated.To.UTC())
	}

	// sort
	sortCol := strings.TrimSpace(sort.Column)
	if sortCol == "" {
		sortCol = "reviewedAt"
	}
	if _, ok := pbr.AllowedSortColumns[sortCol]; !ok {
		return domcommon.PageResult[pbr.Review]{}, pbr.ErrInvalid
	}

	orderDir := firestore.Desc
	if sort.Order == domcommon.SortAsc {
		orderDir = firestore.Asc
	}

	q = q.OrderBy(mapSortField(sortCol), orderDir)

	// paging（Offsetページング）
	pn := page.Number
	pp := page.PerPage
	if pn <= 0 {
		pn = 1
	}
	if pp <= 0 {
		pp = 20
	}
	offset := (pn - 1) * pp

	totalCount, err := countQuery(ctx, q)
	if err != nil {
		return domcommon.PageResult[pbr.Review]{}, err
	}
	totalPages := int(math.Ceil(float64(totalCount) / float64(pp)))
	if totalPages == 0 {
		totalPages = 1
	}

	items := make([]pbr.Review, 0, pp)
	iter := q.Offset(offset).Limit(pp).Documents(ctx)
	defer iter.Stop()

	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return domcommon.PageResult[pbr.Review]{}, err
		}
		review, derr := decodeReviewDoc(snap.Ref.ID, snap.Data())
		if derr != nil {
			return domcommon.PageResult[pbr.Review]{}, derr
		}
		items = append(items, review)
	}

	return domcommon.PageResult[pbr.Review]{
		Items:      items,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       pn,
		PerPage:    pp,
	}, nil
}

// ============================================================
// Domain extra methods (Repository)
// ============================================================

func (r *ProductBlueprintReviewRepositoryFS) ListByProductBlueprintID(
	ctx context.Context,
	productBlueprintID string,
	status pbr.ReviewStatus,
	page domcommon.Page,
) (domcommon.PageResult[pbr.Review], error) {
	pbID := strings.TrimSpace(productBlueprintID)
	if pbID == "" {
		return domcommon.PageResult[pbr.Review]{}, pbr.ErrInvalid
	}
	f := pbr.Filter{
		ProductBlueprintID: &pbID,
		Status:             &status,
	}
	return r.List(ctx, f, domcommon.Sort{Column: "reviewedAt", Order: domcommon.SortDesc}, page)
}

func (r *ProductBlueprintReviewRepositoryFS) GetProductSummary(
	ctx context.Context,
	productBlueprintID string,
	status pbr.ReviewStatus,
) (pbr.ProductReviewSummary, error) {

	if r == nil || r.client == nil {
		return pbr.ProductReviewSummary{}, pbr.ErrInternal
	}

	pbID := strings.TrimSpace(productBlueprintID)
	if pbID == "" {
		return pbr.ProductReviewSummary{}, pbr.ErrInvalid
	}

	q := r.reviewsCol(pbID).
		Where("status", "==", string(status))

	iter := q.Documents(ctx)
	defer iter.Stop()

	total := 0
	sum := 0

	var c1, c2, c3, c4, c5 int

	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return pbr.ProductReviewSummary{}, err
		}

		data := snap.Data()
		rv := getIntFromAny(data["rating"])
		if rv < int(pbr.RatingMin) || rv > int(pbr.RatingMax) {
			continue
		}

		total++
		sum += rv

		switch rv {
		case 5:
			c5++
		case 4:
			c4++
		case 3:
			c3++
		case 2:
			c2++
		case 1:
			c1++
		}
	}

	avg := 0.0
	if total > 0 {
		avg = float64(sum) / float64(total)
	}

	return pbr.ProductReviewSummary{
		ProductBlueprintID: pbID,
		Status:             string(status),
		TotalCount:         total,
		AverageRating:      avg,
		Rating5Count:       c5,
		Rating4Count:       c4,
		Rating3Count:       c3,
		Rating2Count:       c2,
		Rating1Count:       c1,
	}, nil
}

func (r *ProductBlueprintReviewRepositoryFS) IncrementHelpful(ctx context.Context, reviewID string) (pbr.Review, error) {
	// subcollection 構造のため reviewID単体では特定できない
	return pbr.Review{}, pbr.ErrInvalid
}

func (r *ProductBlueprintReviewRepositoryFS) IncrementNotHelpful(ctx context.Context, reviewID string) (pbr.Review, error) {
	// subcollection 構造のため reviewID単体では特定できない
	return pbr.Review{}, pbr.ErrInvalid
}

// ============================================================
// Encoding / Decoding
// ============================================================

func encodeReviewDoc(v pbr.Review) map[string]any {
	out := map[string]any{
		"productBlueprintId": v.ProductBlueprintID,
		"avatarId":           v.AvatarID,
		"rating":             int(v.Rating),
		"title":              v.Title,
		"body":               v.Body,
		"helpfulVotes":       v.HelpfulVotes,
		"totalVotes":         v.TotalVotes,
		"reviewedAt":         v.ReviewedAt.UTC(),
		"status":             string(v.Status),
		"createdAt":          v.CreatedAt.UTC(),
		"createdBy":          v.CreatedBy,
		"updatedAt":          v.UpdatedAt.UTC(),
		"updatedBy":          v.UpdatedBy,
	}

	if v.ModerationReason != nil {
		out["moderationReason"] = *v.ModerationReason
	}
	return out
}

func decodeReviewDoc(id string, data map[string]any) (pbr.Review, error) {
	getString := func(k string) string {
		if v, ok := data[k]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
	getTime := func(k string) time.Time {
		if v, ok := data[k]; ok {
			switch vv := v.(type) {
			case time.Time:
				return vv.UTC()
			}
		}
		return time.Time{}
	}

	var modReason *string
	if v, ok := data["moderationReason"]; ok {
		if s, ok := v.(string); ok {
			modReason = &s
		}
	}

	out := pbr.Review{
		ID:                 pbr.ReviewID(id),
		ProductBlueprintID: getString("productBlueprintId"),
		AvatarID:           getString("avatarId"),
		Rating:             pbr.Rating(getIntFromAny(data["rating"])),
		Title:              getString("title"),
		Body:               getString("body"),
		HelpfulVotes:       getIntFromAny(data["helpfulVotes"]),
		TotalVotes:         getIntFromAny(data["totalVotes"]),
		ReviewedAt:         getTime("reviewedAt"),
		Status:             pbr.ReviewStatus(getString("status")),
		CreatedAt:          getTime("createdAt"),
		CreatedBy:          getString("createdBy"),
		UpdatedAt:          getTime("updatedAt"),
		UpdatedBy:          getString("updatedBy"),
		ModerationReason:   modReason,
	}

	if out.ProductBlueprintID == "" {
		return pbr.Review{}, pbr.ErrInvalid
	}
	return out, nil
}

// ============================================================
// Aggregate helpers (transactional reflect)
// ============================================================

func buildAggregateDeltaOnCreatePublished(rating pbr.Rating, now time.Time) map[string]any {
	out := map[string]any{
		"totalCount": firestore.Increment(1),
		"updatedAt":  now,
	}
	switch int(rating) {
	case 5:
		out["rating5Count"] = firestore.Increment(1)
	case 4:
		out["rating4Count"] = firestore.Increment(1)
	case 3:
		out["rating3Count"] = firestore.Increment(1)
	case 2:
		out["rating2Count"] = firestore.Increment(1)
	case 1:
		out["rating1Count"] = firestore.Increment(1)
	}
	// averageRating はインクリメントだけでは正確に出せないため、別途再計算が必要。
	// ここでは「後段で再計算される」前提で updatedAt のみ確実に更新する。
	// （正確な平均を常に保つなら、sumRating を aggregate に持つ設計にしてください）
	return out
}

// ============================================================
// helpers
// ============================================================

func mapSortField(col string) string {
	switch col {
	case "createdAt":
		return "createdAt"
	case "updatedAt":
		return "updatedAt"
	case "reviewedAt":
		return "reviewedAt"
	case "rating":
		return "rating"
	case "helpfulVotes":
		return "helpfulVotes"
	case "totalVotes":
		return "totalVotes"
	default:
		return "reviewedAt"
	}
}

func countQuery(ctx context.Context, q firestore.Query) (int, error) {
	iter := q.Documents(ctx)
	defer iter.Stop()

	n := 0
	for {
		_, err := iter.Next()
		if err == iterator.Done {
			return n, nil
		}
		if err != nil {
			return 0, err
		}
		n++
	}
}

func getIntFromAny(v any) int {
	switch vv := v.(type) {
	case int:
		return vv
	case int64:
		return int(vv)
	case float64:
		return int(vv)
	default:
		return 0
	}
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	// Firestore not found is gRPC codes.NotFound
	if status.Code(err) == codes.NotFound {
		return true
	}
	// Some wrappers may embed NotFound text
	msg := err.Error()
	return strings.Contains(msg, "NotFound") || strings.Contains(msg, "not found")
}

func isAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	// Firestore create conflict is gRPC codes.AlreadyExists
	if status.Code(err) == codes.AlreadyExists {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "AlreadyExists") || strings.Contains(msg, "already exists")
}
