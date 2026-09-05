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
	return &ProductBlueprintReviewRepositoryFS{client: client, collection: defaultProductBlueprintReviewSubCollection, now: time.Now}
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
		"totalCount":         0,
		"averageRating":      0.0,
		"rating5Count":       0,
		"rating4Count":       0,
		"rating3Count":       0,
		"rating2Count":       0,
		"rating1Count":       0,
		"createdAt":          now,
		"updatedAt":          now,
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

func (r *ProductBlueprintReviewRepositoryFS) GetByProductBlueprintID(
	ctx context.Context,
	productBlueprintID string,
	reviewID string,
) (pbr.Review, error) {
	if r == nil || r.client == nil {
		return pbr.Review{}, pbr.ErrInternal
	}

	pbID := strings.TrimSpace(productBlueprintID)
	id := strings.TrimSpace(reviewID)
	if pbID == "" || id == "" {
		return pbr.Review{}, pbr.ErrInvalid
	}

	snap, err := r.reviewDoc(pbID, id).Get(ctx)
	if err != nil {
		if isNotFound(err) {
			return pbr.Review{}, pbr.ErrNotFound
		}
		return pbr.Review{}, err
	}

	review, err := decodeReviewDoc(snap.Ref.ID, snap.Data())
	if err != nil {
		return pbr.Review{}, err
	}
	if review.ProductBlueprintID != pbID {
		return pbr.Review{}, pbr.ErrInvalid
	}
	return review, nil
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
		if err := tx.Create(reviewDoc, encodeReviewDoc(entity)); err != nil {
			if isAlreadyExists(err) {
				return pbr.ErrConflict
			}
			return err
		}

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
	return pbr.Review{}, pbr.ErrInvalid
}

func (r *ProductBlueprintReviewRepositoryFS) UpdateByProductBlueprintID(
	ctx context.Context,
	productBlueprintID string,
	reviewID string,
	patch pbr.Patch,
) (pbr.Review, error) {
	if r == nil || r.client == nil {
		return pbr.Review{}, pbr.ErrInternal
	}

	pbID := strings.TrimSpace(productBlueprintID)
	id := strings.TrimSpace(reviewID)
	if pbID == "" || id == "" {
		return pbr.Review{}, pbr.ErrInvalid
	}

	reviewDoc := r.reviewDoc(pbID, id)
	aggDoc := r.aggregateDoc(pbID)
	var updated pbr.Review

	err := r.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(reviewDoc)
		if err != nil {
			if isNotFound(err) {
				return pbr.ErrNotFound
			}
			return err
		}

		current, err := decodeReviewDoc(snap.Ref.ID, snap.Data())
		if err != nil {
			return err
		}
		if current.ProductBlueprintID != pbID {
			return pbr.ErrInvalid
		}

		next, err := applyReviewPatch(current, patch, r.now)
		if err != nil {
			return err
		}

		if err := tx.Set(reviewDoc, encodeReviewDoc(next)); err != nil {
			return err
		}

		aggUpdates := buildAggregateDeltaOnUpdate(current, next)
		if len(aggUpdates) > 0 {
			aggUpdates["updatedAt"] = next.UpdatedAt.UTC()
			if err := tx.Set(aggDoc, aggUpdates, firestore.MergeAll); err != nil {
				return err
			}
		}

		updated = next
		return nil
	})
	if err != nil {
		return pbr.Review{}, err
	}
	return updated, nil
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
	if strings.TrimSpace(filter.SearchQuery) != "" {
		return domcommon.PageResult[pbr.Review]{}, pbr.ErrInvalid
	}
	if filter.ProductBlueprintID == nil || strings.TrimSpace(*filter.ProductBlueprintID) == "" {
		return domcommon.PageResult[pbr.Review]{}, pbr.ErrInvalid
	}

	pbID := strings.TrimSpace(*filter.ProductBlueprintID)
	q := r.reviewsCol(pbID).Query

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
	if filter.Reviewed.From != nil {
		q = q.Where("reviewedAt", ">=", filter.Reviewed.From.UTC())
	}
	if filter.Reviewed.To != nil {
		q = q.Where("reviewedAt", "<=", filter.Reviewed.To.UTC())
	}
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

		review, decodeErr := decodeReviewDoc(snap.Ref.ID, snap.Data())
		if decodeErr != nil {
			return domcommon.PageResult[pbr.Review]{}, decodeErr
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

	f := pbr.Filter{ProductBlueprintID: &pbID, Status: &status}
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

	// published は集計ドキュメントから取得する。
	// rating1Count..rating5Count は review 作成・更新時に transaction で更新されるため、
	// review 本体を全件走査せずに件数と平均評価を算出できる。
	if status == pbr.ReviewStatusPublished {
		snap, err := r.aggregateDoc(pbID).Get(ctx)
		if err != nil {
			if isNotFound(err) {
				return emptyProductReviewSummary(pbID, status), nil
			}
			return pbr.ProductReviewSummary{}, err
		}

		data := snap.Data()
		c1 := getIntFromAny(data["rating1Count"])
		c2 := getIntFromAny(data["rating2Count"])
		c3 := getIntFromAny(data["rating3Count"])
		c4 := getIntFromAny(data["rating4Count"])
		c5 := getIntFromAny(data["rating5Count"])
		ratingCount := c1 + c2 + c3 + c4 + c5

		total := getIntFromAny(data["totalCount"])
		if total < ratingCount {
			total = ratingCount
		}

		avg := 0.0
		if ratingCount > 0 {
			weightedSum := c1 + c2*2 + c3*3 + c4*4 + c5*5
			avg = float64(weightedSum) / float64(ratingCount)
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

	// published 以外の status は aggregate に保持していないため、
	// 従来どおり対象 status の review 本体を走査する。
	q := r.reviewsCol(pbID).Where("status", "==", string(status))
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

		rv := getIntFromAny(snap.Data()["rating"])
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

func emptyProductReviewSummary(productBlueprintID string, status pbr.ReviewStatus) pbr.ProductReviewSummary {
	return pbr.ProductReviewSummary{ProductBlueprintID: productBlueprintID, Status: string(status)}
}

func (r *ProductBlueprintReviewRepositoryFS) IncrementHelpful(
	ctx context.Context,
	reviewID string,
) (pbr.Review, error) {
	// subcollection 構造のため reviewID単体では特定できない
	return pbr.Review{}, pbr.ErrInvalid
}

func (r *ProductBlueprintReviewRepositoryFS) IncrementNotHelpful(
	ctx context.Context,
	reviewID string,
) (pbr.Review, error) {
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
	getString := func(key string) string {
		if value, ok := data[key]; ok {
			if stringValue, ok := value.(string); ok {
				return stringValue
			}
		}
		return ""
	}

	getTime := func(key string) time.Time {
		if value, ok := data[key]; ok {
			if typedValue, ok := value.(time.Time); ok {
				return typedValue.UTC()
			}
		}
		return time.Time{}
	}

	var moderationReason *string
	if value, ok := data["moderationReason"]; ok {
		if stringValue, ok := value.(string); ok {
			moderationReason = &stringValue
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
		ModerationReason:   moderationReason,
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
	out := map[string]any{"totalCount": firestore.Increment(1), "updatedAt": now}

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

	// averageRating 自体はここでは更新しない。
	// GetProductSummary は rating1Count..rating5Count から平均値を算出する。
	return out
}

func buildAggregateDeltaOnUpdate(before, after pbr.Review) map[string]any {
	beforePublished := before.Status == pbr.ReviewStatusPublished
	afterPublished := after.Status == pbr.ReviewStatusPublished
	totalDelta := int64(0)
	ratingDeltas := map[string]int64{}

	if beforePublished {
		totalDelta--
		if field := ratingCountField(before.Rating); field != "" {
			ratingDeltas[field]--
		}
	}
	if afterPublished {
		totalDelta++
		if field := ratingCountField(after.Rating); field != "" {
			ratingDeltas[field]++
		}
	}

	updates := map[string]any{}
	if totalDelta != 0 {
		updates["totalCount"] = firestore.Increment(totalDelta)
	}
	for field, delta := range ratingDeltas {
		if delta != 0 {
			updates[field] = firestore.Increment(delta)
		}
	}
	return updates
}

func ratingCountField(rating pbr.Rating) string {
	switch rating {
	case 5:
		return "rating5Count"
	case 4:
		return "rating4Count"
	case 3:
		return "rating3Count"
	case 2:
		return "rating2Count"
	case 1:
		return "rating1Count"
	default:
		return ""
	}
}

// ============================================================
// helpers
// ============================================================

func applyReviewPatch(current pbr.Review, patch pbr.Patch, now func() time.Time) (pbr.Review, error) {
	next := current

	if patch.Title != nil {
		if *patch.Title == "" {
			return pbr.Review{}, pbr.ErrInvalid
		}
		next.Title = *patch.Title
	}
	if patch.Body != nil {
		if *patch.Body == "" {
			return pbr.Review{}, pbr.ErrInvalid
		}
		next.Body = *patch.Body
	}
	if patch.Rating != nil {
		if *patch.Rating < pbr.RatingMin || *patch.Rating > pbr.RatingMax {
			return pbr.Review{}, pbr.ErrInvalid
		}
		next.Rating = *patch.Rating
	}
	if patch.Status != nil {
		switch *patch.Status {
		case pbr.ReviewStatusPublished, pbr.ReviewStatusHidden, pbr.ReviewStatusRemoved:
			next.Status = *patch.Status
		default:
			return pbr.Review{}, pbr.ErrInvalid
		}
	}
	if patch.ModerationReason != nil {
		if *patch.ModerationReason == "" {
			next.ModerationReason = nil
		} else {
			reason := *patch.ModerationReason
			next.ModerationReason = &reason
		}
	}

	updatedAt := time.Now().UTC()
	if now != nil {
		updatedAt = now().UTC()
	}
	if patch.UpdatedAt != nil {
		if patch.UpdatedAt.IsZero() {
			return pbr.Review{}, pbr.ErrInvalid
		}
		updatedAt = patch.UpdatedAt.UTC()
	}
	next.UpdatedAt = updatedAt

	if patch.UpdatedBy != nil {
		if *patch.UpdatedBy == "" {
			return pbr.Review{}, pbr.ErrInvalid
		}
		next.UpdatedBy = *patch.UpdatedBy
	}

	if next.ProductBlueprintID == "" || next.AvatarID == "" || next.Title == "" || next.Body == "" ||
		next.Rating < pbr.RatingMin || next.Rating > pbr.RatingMax ||
		next.CreatedAt.IsZero() || next.CreatedBy == "" || next.UpdatedAt.IsZero() || next.UpdatedBy == "" ||
		next.HelpfulVotes < 0 || next.TotalVotes < 0 || next.HelpfulVotes > next.TotalVotes {
		return pbr.Review{}, pbr.ErrInvalid
	}

	return next, nil
}

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
	switch value := v.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	if status.Code(err) == codes.NotFound {
		return true
	}
	message := err.Error()
	return strings.Contains(message, "NotFound") || strings.Contains(message, "not found")
}

func isAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	if status.Code(err) == codes.AlreadyExists {
		return true
	}
	message := err.Error()
	return strings.Contains(message, "AlreadyExists") || strings.Contains(message, "already exists")
}
