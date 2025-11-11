// backend/internal/adapters/out/firestore/mintRequest_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	mrdom "narratives/internal/domain/mintRequest"
)

// MintRequestRepositoryFS is a Firestore-based implementation of the MintRequest repository.
type MintRequestRepositoryFS struct {
	Client *firestore.Client
}

func NewMintRequestRepositoryFS(client *firestore.Client) *MintRequestRepositoryFS {
	return &MintRequestRepositoryFS{Client: client}
}

func (r *MintRequestRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("mint_requests")
}

// ============================================================
// Core (MintRequestRepo required methods)
// ============================================================

// GetByID returns a MintRequest by its ID.
func (r *MintRequestRepositoryFS) GetByID(ctx context.Context, id string) (mrdom.MintRequest, error) {
	if r.Client == nil {
		return mrdom.MintRequest{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return mrdom.MintRequest{}, mrdom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return mrdom.MintRequest{}, mrdom.ErrNotFound
		}
		return mrdom.MintRequest{}, err
	}

	return docToMintRequest(snap)
}

// Exists reports whether a MintRequest with the given ID exists.
func (r *MintRequestRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}

	_, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Create inserts a new MintRequest.
// Firestore版では ID は呼び出し側で指定してもらう前提（空ならエラー）。
func (r *MintRequestRepositoryFS) Create(ctx context.Context, v mrdom.MintRequest) (mrdom.MintRequest, error) {
	if r.Client == nil {
		return mrdom.MintRequest{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(v.ID)
	if id == "" {
		return mrdom.MintRequest{}, errors.New("missing id")
	}

	now := time.Now().UTC()

	// created/updated 系はここで確定させる（PG 実装準拠）
	v.ID = id

	if v.CreatedAt.IsZero() {
		v.CreatedAt = now
	}
	if v.UpdatedAt.IsZero() {
		v.UpdatedAt = now
	}
	// CreatedBy/UpdatedBy は渡された値を優先（空ならそのまま空）

	docRef := r.col().Doc(id)
	data := mintRequestToDoc(v)

	_, err := docRef.Create(ctx, data)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return mrdom.MintRequest{}, mrdom.ErrConflict
		}
		return mrdom.MintRequest{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return mrdom.MintRequest{}, err
	}
	return docToMintRequest(snap)
}

// Save upserts a MintRequest by ID.
// PG 実装の ON CONFLICT 相当: 基本的に新しい値で上書きしつつ createdAt/createdBy はできるだけ保持。
func (r *MintRequestRepositoryFS) Save(ctx context.Context, v mrdom.MintRequest) (mrdom.MintRequest, error) {
	if r.Client == nil {
		return mrdom.MintRequest{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(v.ID)
	if id == "" {
		return mrdom.MintRequest{}, errors.New("missing id")
	}
	v.ID = id

	now := time.Now().UTC()
	if v.CreatedAt.IsZero() {
		v.CreatedAt = now
	}
	if v.UpdatedAt.IsZero() {
		v.UpdatedAt = now
	}

	docRef := r.col().Doc(id)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		existingSnap, err := tx.Get(docRef)
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}

		// Start from new value
		out := v

		// If exists, preserve earliest createdAt/createdBy when appropriate
		if existingSnap.Exists() {
			existing, err := docToMintRequest(existingSnap)
			if err != nil {
				return err
			}

			// createdAt: earliest
			if !existing.CreatedAt.IsZero() && existing.CreatedAt.Before(out.CreatedAt) {
				out.CreatedAt = existing.CreatedAt
			}
			// createdBy: keep existing if non-empty, else use new
			if strings.TrimSpace(existing.CreatedBy) != "" {
				out.CreatedBy = existing.CreatedBy
			}

			// deletedAt / deletedBy / requestedAt などは「新しい値で上書き」方針とする。
		}

		data := mintRequestToDoc(out)
		if err := tx.Set(docRef, data, firestore.MergeAll); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return mrdom.MintRequest{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return mrdom.MintRequest{}, err
	}
	return docToMintRequest(snap)
}

// Delete removes a MintRequest by ID.
func (r *MintRequestRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return mrdom.ErrNotFound
	}

	_, err := r.col().Doc(id).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return mrdom.ErrNotFound
		}
		return err
	}
	return nil
}

// ============================================================
// Convenience / extra query methods (List / Update / Count / Reset)
// ============================================================

// List returns paginated MintRequests applying filter/sort in a Firestore-friendly way.
func (r *MintRequestRepositoryFS) List(
	ctx context.Context,
	filter mrdom.Filter,
	sort mrdom.Sort,
	page mrdom.Page,
) (mrdom.PageResult, error) {
	if r.Client == nil {
		return mrdom.PageResult{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	q := r.col().Query
	q = applyMintRequestSort(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []mrdom.MintRequest
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return mrdom.PageResult{}, err
		}
		mr, err := docToMintRequest(doc)
		if err != nil {
			return mrdom.PageResult{}, err
		}
		if matchMintRequestFilter(mr, filter) {
			all = append(all, mr)
		}
	}

	total := len(all)
	if total == 0 {
		return mrdom.PageResult{
			Items:      []mrdom.MintRequest{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}
	items := all[offset:end]

	return mrdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// Update is a convenience partial update (not necessarily part of the core port).
func (r *MintRequestRepositoryFS) Update(
	ctx context.Context,
	id string,
	patch mrdom.UpdateMintRequest,
) (mrdom.MintRequest, error) {
	if r.Client == nil {
		return mrdom.MintRequest{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return mrdom.MintRequest{}, mrdom.ErrNotFound
	}

	docRef := r.col().Doc(id)
	var updates []firestore.Update

	setStr := func(path string, p *string) {
		if p != nil {
			v := strings.TrimSpace(*p)
			if v == "" {
				updates = append(updates, firestore.Update{
					Path:  path,
					Value: firestore.Delete,
				})
			} else {
				updates = append(updates, firestore.Update{
					Path:  path,
					Value: v,
				})
			}
		}
	}
	setInt := func(path string, p *int) {
		if p != nil {
			updates = append(updates, firestore.Update{
				Path:  path,
				Value: *p,
			})
		}
	}
	setStatus := func(p *mrdom.MintRequestStatus) {
		if p != nil {
			updates = append(updates, firestore.Update{
				Path:  "status",
				Value: string(*p),
			})
		}
	}
	setTime := func(path string, p *time.Time) {
		if p != nil {
			if p.IsZero() {
				updates = append(updates, firestore.Update{
					Path:  path,
					Value: firestore.Delete,
				})
			} else {
				updates = append(updates, firestore.Update{
					Path:  path,
					Value: p.UTC(),
				})
			}
		}
	}

	// Business fields
	setStatus(patch.Status)
	setStr("tokenBlueprintId", patch.TokenBlueprintID)
	setInt("mintQuantity", patch.MintQuantity)
	setTime("burnDate", patch.BurnDate)

	setStr("requestedBy", patch.RequestedBy)
	setTime("requestedAt", patch.RequestedAt)
	setTime("mintedAt", patch.MintedAt)

	// Soft delete-ish
	setTime("deletedAt", patch.DeletedAt)
	setStr("deletedBy", patch.DeletedBy)

	// Audit: updatedAt / updatedBy（PG版同様、必ず更新）
	now := time.Now().UTC()
	updates = append(updates,
		firestore.Update{
			Path:  "updatedAt",
			Value: now,
		},
		firestore.Update{
			Path:  "updatedBy",
			Value: strings.TrimSpace(patch.UpdatedBy),
		},
	)

	if len(updates) == 0 {
		return r.GetByID(ctx, id)
	}

	_, err := docRef.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return mrdom.MintRequest{}, mrdom.ErrNotFound
		}
		return mrdom.MintRequest{}, err
	}

	return r.GetByID(ctx, id)
}

// Count returns total number of MintRequests matching filter.
func (r *MintRequestRepositoryFS) Count(ctx context.Context, filter mrdom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	total := 0
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return 0, err
		}
		mr, err := docToMintRequest(doc)
		if err != nil {
			return 0, err
		}
		if matchMintRequestFilter(mr, filter) {
			total++
		}
	}
	return total, nil
}

// Reset deletes all MintRequests (mainly for admin/testing usage) using Transactions instead of WriteBatch.
func (r *MintRequestRepositoryFS) Reset(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	var refs []*firestore.DocumentRef
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		refs = append(refs, doc.Ref)
	}

	if len(refs) == 0 {
		return nil
	}

	const chunkSize = 400

	for start := 0; start < len(refs); start += chunkSize {
		end := start + chunkSize
		if end > len(refs) {
			end = len(refs)
		}
		chunk := refs[start:end]

		err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			for _, ref := range chunk {
				if err := tx.Delete(ref); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// ============================================================
// Helpers: encode/decode & filter/sort
// ============================================================

func mintRequestToDoc(v mrdom.MintRequest) map[string]any {
	m := map[string]any{
		"tokenBlueprintId": strings.TrimSpace(v.TokenBlueprintID),
		"productionId":     strings.TrimSpace(v.ProductionID),
		"mintQuantity":     v.MintQuantity,
		"status":           string(v.Status),
		"createdAt":        v.CreatedAt.UTC(),
		"createdBy":        strings.TrimSpace(v.CreatedBy),
		"updatedAt":        v.UpdatedAt.UTC(),
		"updatedBy":        strings.TrimSpace(v.UpdatedBy),
	}

	if v.BurnDate != nil && !v.BurnDate.IsZero() {
		m["burnDate"] = v.BurnDate.UTC()
	}
	if v.RequestedBy != nil {
		if s := strings.TrimSpace(*v.RequestedBy); s != "" {
			m["requestedBy"] = s
		}
	}
	if v.RequestedAt != nil && !v.RequestedAt.IsZero() {
		m["requestedAt"] = v.RequestedAt.UTC()
	}
	if v.MintedAt != nil && !v.MintedAt.IsZero() {
		m["mintedAt"] = v.MintedAt.UTC()
	}
	if v.DeletedAt != nil && !v.DeletedAt.IsZero() {
		m["deletedAt"] = v.DeletedAt.UTC()
	}
	if v.DeletedBy != nil {
		if s := strings.TrimSpace(*v.DeletedBy); s != "" {
			m["deletedBy"] = s
		}
	}

	return m
}

func docToMintRequest(doc *firestore.DocumentSnapshot) (mrdom.MintRequest, error) {
	data := doc.Data()
	if data == nil {
		return mrdom.MintRequest{}, fmt.Errorf("empty mint_request document: %s", doc.Ref.ID)
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	getStrPtr := func(keys ...string) *string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				s := strings.TrimSpace(v)
				if s != "" {
					return &s
				}
			}
		}
		return nil
	}
	getInt := func(keys ...string) int {
		for _, k := range keys {
			if v, ok := data[k]; ok {
				switch n := v.(type) {
				case int:
					return n
				case int32:
					return int(n)
				case int64:
					return int(n)
				case float64:
					return int(n)
				}
			}
		}
		return 0
	}
	getTimePtr := func(keys ...string) *time.Time {
		for _, k := range keys {
			if v, ok := data[k].(time.Time); ok && !v.IsZero() {
				t := v.UTC()
				return &t
			}
		}
		return nil
	}
	getStatus := func(keys ...string) mrdom.MintRequestStatus {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				return mrdom.MintRequestStatus(strings.TrimSpace(v))
			}
		}
		return ""
	}

	var mr mrdom.MintRequest

	mr.ID = strings.TrimSpace(doc.Ref.ID)
	mr.TokenBlueprintID = getStr("tokenBlueprintId", "token_blueprint_id")
	mr.ProductionID = getStr("productionId", "production_id")
	mr.MintQuantity = getInt("mintQuantity", "mint_quantity")
	mr.BurnDate = getTimePtr("burnDate", "burn_date")

	mr.Status = getStatus("status")
	mr.RequestedBy = getStrPtr("requestedBy", "requested_by")
	mr.RequestedAt = getTimePtr("requestedAt", "requested_at")
	mr.MintedAt = getTimePtr("mintedAt", "minted_at")

	if t := getTimePtr("createdAt", "created_at"); t != nil {
		mr.CreatedAt = *t
	}
	mr.CreatedBy = getStr("createdBy", "created_by")
	if t := getTimePtr("updatedAt", "updated_at"); t != nil {
		mr.UpdatedAt = *t
	}
	mr.UpdatedBy = getStr("updatedBy", "updated_by")
	mr.DeletedAt = getTimePtr("deletedAt", "deleted_at")
	mr.DeletedBy = getStrPtr("deletedBy", "deleted_by")

	return mr, nil
}

// matchMintRequestFilter mirrors buildMintRequestWhere but in-memory for Firestore.
func matchMintRequestFilter(m mrdom.MintRequest, f mrdom.Filter) bool {
	// ProductionID
	if v := strings.TrimSpace(f.ProductionID); v != "" {
		if m.ProductionID != v {
			return false
		}
	}
	// TokenBlueprintID
	if v := strings.TrimSpace(f.TokenBlueprintID); v != "" {
		if m.TokenBlueprintID != v {
			return false
		}
	}
	// Statuses
	if len(f.Statuses) > 0 {
		ok := false
		for _, st := range f.Statuses {
			if m.Status == st {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	// RequestedBy
	if v := strings.TrimSpace(f.RequestedBy); v != "" {
		if m.RequestedBy == nil || strings.TrimSpace(*m.RequestedBy) != v {
			return false
		}
	}

	// Helper for time ranges
	inRange := func(t *time.Time, from, to *time.Time, useZero bool) bool {
		if t == nil {
			// if !useZero { return false }
			// return true
			return useZero
		}
		tv := t.UTC()
		if from != nil && tv.Before(from.UTC()) {
			return false
		}
		if to != nil && !tv.Before(to.UTC()) {
			return false
		}
		return true
	}

	// CreatedAt
	if f.CreatedFrom != nil || f.CreatedTo != nil {
		t := m.CreatedAt
		if f.CreatedFrom != nil && t.Before(f.CreatedFrom.UTC()) {
			return false
		}
		if f.CreatedTo != nil && !t.Before(f.CreatedTo.UTC()) {
			return false
		}
	}

	// RequestedAt
	if f.RequestFrom != nil || f.RequestTo != nil {
		if !inRange(m.RequestedAt, f.RequestFrom, f.RequestTo, false) {
			return false
		}
	}
	// MintedAt
	if f.MintedFrom != nil || f.MintedTo != nil {
		if !inRange(m.MintedAt, f.MintedFrom, f.MintedTo, false) {
			return false
		}
	}
	// BurnDate
	if f.BurnFrom != nil || f.BurnTo != nil {
		if !inRange(m.BurnDate, f.BurnFrom, f.BurnTo, false) {
			return false
		}
	}

	// Deleted flag
	if f.Deleted != nil {
		if *f.Deleted {
			if m.DeletedAt == nil {
				return false
			}
		} else {
			if m.DeletedAt != nil {
				return false
			}
		}
	}

	return true
}

// applyMintRequestSort maps domain Sort to Firestore orderBy.
func applyMintRequestSort(q firestore.Query, sort mrdom.Sort) firestore.Query {
	col := strings.ToLower(strings.TrimSpace(string(sort.Column)))
	var field string

	switch col {
	case "createdat", "created_at":
		field = "createdAt"
	case "updatedat", "updated_at":
		field = "updatedAt"
	case "burndate", "burn_date":
		field = "burnDate"
	case "mintedat", "minted_at":
		field = "mintedAt"
	case "requestedat", "requested_at":
		field = "requestedAt"
	default:
		// default: createdAt DESC, id DESC
		return q.OrderBy("createdAt", firestore.Desc).OrderBy(firestore.DocumentID, firestore.Desc)
	}

	dir := firestore.Desc
	if strings.EqualFold(string(sort.Order), "asc") {
		dir = firestore.Asc
	}

	// Secondary sort by document ID for stability
	return q.OrderBy(field, dir).OrderBy(firestore.DocumentID, dir)
}
