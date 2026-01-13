// backend/internal/adapters/out/firestore/transfer_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	common "narratives/internal/domain/common"
	transferdom "narratives/internal/domain/transfer"
)

/*
責任と機能:
- transfer.RepositoryPort の Firestore 実装。
- コレクション設計（推奨）:
  - transfers/{productId}                      : メタ（latestAttempt など、最新状態の参照用）
  - transfers/{productId}/attempts/{attemptId} : 試行履歴（attemptId は "att_000001" のような文字列）
- CreateAttempt は Transaction で latestAttempt をインクリメントし、
  attempts に新規 attempt ドキュメントを作成して、meta を更新する（排他・一意性担保）。
- Patch/Save は attempts/{attemptId} を更新し、必要なら meta の最新状態も更新する。

注意:
- helper_repository_fs.go に asInt が既にある前提で、このファイルでは asInt を再定義しない。
- asInt のシグネチャは「int を1つだけ返す」前提で使う（2戻り値では受けない）。
*/

type TransferRepositoryFS struct {
	Client *firestore.Client
}

func NewTransferRepositoryFS(client *firestore.Client) *TransferRepositoryFS {
	return &TransferRepositoryFS{Client: client}
}

func (r *TransferRepositoryFS) transfersCol() *firestore.CollectionRef {
	return r.Client.Collection("transfers")
}

func (r *TransferRepositoryFS) transferDoc(productID string) *firestore.DocumentRef {
	return r.transfersCol().Doc(productID)
}

func (r *TransferRepositoryFS) attemptsCol(productID string) *firestore.CollectionRef {
	return r.transferDoc(productID).Collection("attempts")
}

func (r *TransferRepositoryFS) attemptDoc(productID string, attempt int) *firestore.DocumentRef {
	return r.attemptsCol(productID).Doc(attemptDocID(attempt))
}

var (
	errTransferNotFound = errors.New("transfer: not found")
)

// ============================================================
// RepositoryPort impl
// ============================================================

func (r *TransferRepositoryFS) GetLatestByProductID(ctx context.Context, productID string) (*transferdom.Transfer, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, errTransferNotFound
	}

	// meta を見て latestAttempt が取れれば、それを優先
	metaSnap, err := r.transferDoc(productID).Get(ctx)
	if err == nil && metaSnap != nil && metaSnap.Exists() {
		la := asInt(metaSnap.Data()["latestAttempt"])
		if la > 0 {
			return r.GetByProductIDAndAttempt(ctx, productID, la)
		}
	}

	// meta が無い/壊れている場合は attempts を createdAt desc で 1 件取る
	it := r.attemptsCol(productID).
		OrderBy("createdAt", firestore.Desc).
		Limit(1).
		Documents(ctx)
	defer it.Stop()

	doc, nerr := it.Next()
	if nerr == iterator.Done {
		return nil, errTransferNotFound
	}
	if nerr != nil {
		return nil, nerr
	}

	t, derr := docToTransfer(doc, productID)
	if derr != nil {
		return nil, derr
	}
	return &t, nil
}

func (r *TransferRepositoryFS) GetByProductIDAndAttempt(ctx context.Context, productID string, attempt int) (*transferdom.Transfer, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" || attempt <= 0 {
		return nil, errTransferNotFound
	}

	snap, err := r.attemptDoc(productID, attempt).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, errTransferNotFound
		}
		return nil, err
	}

	t, err := docToTransfer(snap, productID)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TransferRepositoryFS) ListByProductID(ctx context.Context, productID string) ([]transferdom.Transfer, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return []transferdom.Transfer{}, nil
	}

	it := r.attemptsCol(productID).
		OrderBy("attempt", firestore.Asc).
		Documents(ctx)
	defer it.Stop()

	out := make([]transferdom.Transfer, 0, 8)
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		t, derr := docToTransfer(doc, productID)
		if derr != nil {
			return nil, derr
		}
		out = append(out, t)
	}
	return out, nil
}

func (r *TransferRepositoryFS) List(ctx context.Context, filter transferdom.Filter, sort transferdom.Sort, page common.Page) (common.PageResult[transferdom.Transfer], error) {
	if r == nil || r.Client == nil {
		return common.PageResult[transferdom.Transfer]{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	// cross-product list は collectionGroup("attempts") を使う
	q := r.Client.CollectionGroup("attempts").Query

	// ---- filter ----
	if filter.ID != nil && strings.TrimSpace(*filter.ID) != "" {
		q = q.Where("productId", "==", strings.TrimSpace(*filter.ID))
	}
	if filter.ProductID != nil && strings.TrimSpace(*filter.ProductID) != "" {
		q = q.Where("productId", "==", strings.TrimSpace(*filter.ProductID))
	}
	if filter.OrderID != nil && strings.TrimSpace(*filter.OrderID) != "" {
		q = q.Where("orderId", "==", strings.TrimSpace(*filter.OrderID))
	}
	if filter.AvatarID != nil && strings.TrimSpace(*filter.AvatarID) != "" {
		q = q.Where("avatarId", "==", strings.TrimSpace(*filter.AvatarID))
	}
	if filter.Status != nil && strings.TrimSpace(string(*filter.Status)) != "" {
		q = q.Where("status", "==", strings.TrimSpace(string(*filter.Status)))
	}
	if filter.ErrorType != nil && strings.TrimSpace(string(*filter.ErrorType)) != "" {
		q = q.Where("errorType", "==", strings.TrimSpace(string(*filter.ErrorType)))
	}

	// ---- sort ----
	field := strings.TrimSpace(sort.Field)
	if field == "" {
		field = "createdAt"
	}
	dir := firestore.Asc
	if sort.Desc {
		dir = firestore.Desc
	}
	q = q.OrderBy(field, dir).OrderBy("productId", firestore.Asc).OrderBy("attempt", firestore.Asc)

	// ---- page ----
	q = q.Offset(offset).Limit(perPage)

	it := q.Documents(ctx)
	defer it.Stop()

	items := make([]transferdom.Transfer, 0, perPage)
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.PageResult[transferdom.Transfer]{}, err
		}

		pid := ""
		if v := doc.Data()["productId"]; v != nil {
			if s, ok := v.(string); ok {
				pid = strings.TrimSpace(s)
			}
		}
		t, derr := docToTransfer(doc, pid)
		if derr != nil {
			return common.PageResult[transferdom.Transfer]{}, derr
		}
		items = append(items, t)
	}

	total, cerr := r.Count(ctx, filter)
	if cerr != nil {
		return common.PageResult[transferdom.Transfer]{}, cerr
	}

	return common.PageResult[transferdom.Transfer]{
		Items:      items,
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *TransferRepositoryFS) Count(ctx context.Context, filter transferdom.Filter) (int, error) {
	if r == nil || r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	q := r.Client.CollectionGroup("attempts").Query

	if filter.ID != nil && strings.TrimSpace(*filter.ID) != "" {
		q = q.Where("productId", "==", strings.TrimSpace(*filter.ID))
	}
	if filter.ProductID != nil && strings.TrimSpace(*filter.ProductID) != "" {
		q = q.Where("productId", "==", strings.TrimSpace(*filter.ProductID))
	}
	if filter.OrderID != nil && strings.TrimSpace(*filter.OrderID) != "" {
		q = q.Where("orderId", "==", strings.TrimSpace(*filter.OrderID))
	}
	if filter.AvatarID != nil && strings.TrimSpace(*filter.AvatarID) != "" {
		q = q.Where("avatarId", "==", strings.TrimSpace(*filter.AvatarID))
	}
	if filter.Status != nil && strings.TrimSpace(string(*filter.Status)) != "" {
		q = q.Where("status", "==", strings.TrimSpace(string(*filter.Status)))
	}
	if filter.ErrorType != nil && strings.TrimSpace(string(*filter.ErrorType)) != "" {
		q = q.Where("errorType", "==", strings.TrimSpace(string(*filter.ErrorType)))
	}

	it := q.Documents(ctx)
	defer it.Stop()

	total := 0
	for {
		_, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		total++
	}
	return total, nil
}

func (r *TransferRepositoryFS) CreateAttempt(ctx context.Context, t transferdom.Transfer) (*transferdom.Transfer, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	productID := strings.TrimSpace(t.ProductID)
	if productID == "" {
		return nil, transferdom.ErrInvalidProductID
	}

	// 推奨: ID == productId
	if strings.TrimSpace(t.ID) == "" {
		t.ID = productID
	}

	now := time.Now().UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}

	metaRef := r.transferDoc(productID)

	var created transferdom.Transfer

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		latestAttempt := 0
		metaSnap, merr := tx.Get(metaRef)
		if merr == nil && metaSnap != nil && metaSnap.Exists() {
			latestAttempt = asInt(metaSnap.Data()["latestAttempt"])
		} else if merr != nil && status.Code(merr) != codes.NotFound {
			return merr
		}

		next := latestAttempt + 1
		if next <= 0 {
			next = 1
		}

		t.Attempt = next
		u := now
		t.UpdatedAt = &u

		attemptRef := r.attemptDoc(productID, next)

		if err := tx.Create(attemptRef, transferToDoc(t)); err != nil {
			if status.Code(err) == codes.AlreadyExists {
				return fmt.Errorf("transfer attempt already exists productId=%s attempt=%d", productID, next)
			}
			return err
		}

		meta := map[string]any{
			"productId":     productID,
			"latestAttempt": next,
			"latestStatus":  string(t.Status),
			"latestErrorType": func() any {
				if t.ErrorType == nil {
					return nil
				}
				return string(*t.ErrorType)
			}(),
			"latestUpdatedAt": now,
			"createdAt": func() time.Time {
				if metaSnap != nil && metaSnap.Exists() {
					if v := metaSnap.Data()["createdAt"]; v != nil {
						if tt, ok := v.(time.Time); ok && !tt.IsZero() {
							return tt.UTC()
						}
					}
				}
				return t.CreatedAt.UTC()
			}(),
		}

		if err := tx.Set(metaRef, meta, firestore.MergeAll); err != nil {
			return err
		}

		created = t
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &created, nil
}

func (r *TransferRepositoryFS) Save(ctx context.Context, t transferdom.Transfer, _ *common.SaveOptions) (*transferdom.Transfer, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	productID := strings.TrimSpace(t.ProductID)
	if productID == "" {
		return nil, transferdom.ErrInvalidProductID
	}
	if t.Attempt <= 0 {
		return nil, errors.New("transfer: invalid attempt")
	}
	if strings.TrimSpace(t.ID) == "" {
		t.ID = productID
	}

	now := time.Now().UTC()
	u := now
	t.UpdatedAt = &u
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}

	metaRef := r.transferDoc(productID)
	attemptRef := r.attemptDoc(productID, t.Attempt)

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		if err := tx.Set(attemptRef, transferToDoc(t), firestore.MergeAll); err != nil {
			return err
		}

		latestAttempt := 0
		metaSnap, merr := tx.Get(metaRef)
		if merr == nil && metaSnap != nil && metaSnap.Exists() {
			latestAttempt = asInt(metaSnap.Data()["latestAttempt"])
		} else if merr != nil && status.Code(merr) != codes.NotFound {
			return merr
		}

		if t.Attempt >= latestAttempt {
			meta := map[string]any{
				"productId":     productID,
				"latestAttempt": t.Attempt,
				"latestStatus":  string(t.Status),
				"latestErrorType": func() any {
					if t.ErrorType == nil {
						return nil
					}
					return string(*t.ErrorType)
				}(),
				"latestUpdatedAt": now,
			}
			if err := tx.Set(metaRef, meta, firestore.MergeAll); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	out := t
	return &out, nil
}

func (r *TransferRepositoryFS) Patch(ctx context.Context, productID string, attempt int, patch transferdom.TransferPatch, _ *common.SaveOptions) (*transferdom.Transfer, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, transferdom.ErrInvalidProductID
	}
	if attempt <= 0 {
		return nil, errors.New("transfer: invalid attempt")
	}

	now := time.Now().UTC()
	if patch.UpdatedAt == nil || patch.UpdatedAt.IsZero() {
		patch.UpdatedAt = &now
	}

	metaRef := r.transferDoc(productID)
	attemptRef := r.attemptDoc(productID, attempt)

	var updated *transferdom.Transfer

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(attemptRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return errTransferNotFound
			}
			return err
		}

		updates := make([]firestore.Update, 0, 8)

		if patch.Status != nil {
			updates = append(updates, firestore.Update{Path: "status", Value: string(*patch.Status)})
		}
		if patch.ErrorType != nil {
			updates = append(updates, firestore.Update{Path: "errorType", Value: string(*patch.ErrorType)})
		}
		if patch.ErrorMsg != nil {
			m := strings.TrimSpace(*patch.ErrorMsg)
			if m == "" {
				updates = append(updates, firestore.Update{Path: "errorMsg", Value: firestore.Delete})
			} else {
				updates = append(updates, firestore.Update{Path: "errorMsg", Value: m})
			}
		}
		if patch.TxSignature != nil {
			s := strings.TrimSpace(*patch.TxSignature)
			if s == "" {
				updates = append(updates, firestore.Update{Path: "txSignature", Value: firestore.Delete})
			} else {
				updates = append(updates, firestore.Update{Path: "txSignature", Value: s})
			}
		}
		if patch.UpdatedAt != nil && !patch.UpdatedAt.IsZero() {
			updates = append(updates, firestore.Update{Path: "updatedAt", Value: patch.UpdatedAt.UTC()})
		}

		if len(updates) > 0 {
			if err := tx.Update(attemptRef, updates); err != nil {
				return err
			}
		}

		latestAttempt := 0
		metaSnap, merr := tx.Get(metaRef)
		if merr == nil && metaSnap != nil && metaSnap.Exists() {
			latestAttempt = asInt(metaSnap.Data()["latestAttempt"])
		} else if merr != nil && status.Code(merr) != codes.NotFound {
			return merr
		}

		if attempt >= latestAttempt {
			metaUpdates := map[string]any{
				"productId":       productID,
				"latestAttempt":   attempt,
				"latestUpdatedAt": patch.UpdatedAt.UTC(),
			}
			if patch.Status != nil {
				metaUpdates["latestStatus"] = string(*patch.Status)
			}
			if patch.ErrorType != nil {
				metaUpdates["latestErrorType"] = string(*patch.ErrorType)
			}
			if err := tx.Set(metaRef, metaUpdates, firestore.MergeAll); err != nil {
				return err
			}
		}

		t, derr := docToTransfer(snap, productID)
		if derr != nil {
			return derr
		}
		t.ApplyPatch(patch)
		updated = &t

		return nil
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (r *TransferRepositoryFS) Delete(ctx context.Context, productID string, attempt int) error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" || attempt <= 0 {
		return errTransferNotFound
	}

	_, err := r.attemptDoc(productID, attempt).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return errTransferNotFound
		}
		return err
	}
	return nil
}

// Reset deletes all transfers (meta + attempts). (dev/test only)
func (r *TransferRepositoryFS) Reset(ctx context.Context) error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}

	ait := r.Client.CollectionGroup("attempts").Documents(ctx)
	defer ait.Stop()

	var attemptRefs []*firestore.DocumentRef
	for {
		doc, err := ait.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		attemptRefs = append(attemptRefs, doc.Ref)
	}

	mit := r.transfersCol().Documents(ctx)
	defer mit.Stop()

	var metaRefs []*firestore.DocumentRef
	for {
		doc, err := mit.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		metaRefs = append(metaRefs, doc.Ref)
	}

	const chunkSize = 400

	delChunk := func(refs []*firestore.DocumentRef) error {
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

	if err := delChunk(attemptRefs); err != nil {
		return err
	}
	if err := delChunk(metaRefs); err != nil {
		return err
	}

	log.Printf("[transfer_repo_fs] Reset OK deletedAttempts=%d deletedMeta=%d", len(attemptRefs), len(metaRefs))
	return nil
}

// ============================================================
// Mapping
// ============================================================

func attemptDocID(attempt int) string {
	if attempt < 0 {
		attempt = 0
	}
	return fmt.Sprintf("att_%06d", attempt)
}

func transferToDoc(t transferdom.Transfer) map[string]any {
	m := map[string]any{
		"id":        strings.TrimSpace(t.ID),
		"productId": strings.TrimSpace(t.ProductID),
		"orderId":   strings.TrimSpace(t.OrderID),
		"avatarId":  strings.TrimSpace(t.AvatarID),

		"attempt": t.Attempt,

		"toWalletAddress": strings.TrimSpace(t.ToWalletAddress),

		"status": string(t.Status),
	}

	if t.TxSignature != nil && strings.TrimSpace(*t.TxSignature) != "" {
		m["txSignature"] = strings.TrimSpace(*t.TxSignature)
	}
	if t.ErrorType != nil && strings.TrimSpace(string(*t.ErrorType)) != "" {
		m["errorType"] = strings.TrimSpace(string(*t.ErrorType))
	}
	if t.ErrorMsg != nil && strings.TrimSpace(*t.ErrorMsg) != "" {
		m["errorMsg"] = strings.TrimSpace(*t.ErrorMsg)
	}

	if !t.CreatedAt.IsZero() {
		m["createdAt"] = t.CreatedAt.UTC()
	}
	if t.UpdatedAt != nil && !t.UpdatedAt.IsZero() {
		m["updatedAt"] = t.UpdatedAt.UTC()
	}

	return m
}

func docToTransfer(doc *firestore.DocumentSnapshot, fallbackProductID string) (transferdom.Transfer, error) {
	data := doc.Data()
	if data == nil {
		return transferdom.Transfer{}, fmt.Errorf("empty transfer doc: %s", doc.Ref.Path)
	}

	getStr := func(key string) string {
		if v, ok := data[key].(string); ok {
			return strings.TrimSpace(v)
		}
		if v, ok := data[key]; ok && v != nil {
			return strings.TrimSpace(fmt.Sprint(v))
		}
		return ""
	}
	getTime := func(key string) time.Time {
		if v, ok := data[key].(time.Time); ok && !v.IsZero() {
			return v.UTC()
		}
		return time.Time{}
	}
	getTimePtr := func(key string) *time.Time {
		t := getTime(key)
		if t.IsZero() {
			return nil
		}
		tt := t.UTC()
		return &tt
	}

	productID := getStr("productId")
	if productID == "" {
		productID = strings.TrimSpace(fallbackProductID)
		if productID == "" {
			if doc.Ref.Parent != nil && doc.Ref.Parent.Parent != nil {
				productID = strings.TrimSpace(doc.Ref.Parent.Parent.ID)
			}
		}
	}

	attempt := asInt(data["attempt"])
	if attempt <= 0 {
		if strings.HasPrefix(doc.Ref.ID, "att_") {
			var n int
			_, _ = fmt.Sscanf(doc.Ref.ID, "att_%d", &n)
			if n > 0 {
				attempt = n
			}
		}
	}

	createdAt := getTime("createdAt")
	if createdAt.IsZero() && !doc.CreateTime.IsZero() {
		createdAt = doc.CreateTime.UTC()
	}
	updatedAt := getTimePtr("updatedAt")

	st := transferdom.Status(strings.TrimSpace(getStr("status")))
	if st == "" {
		st = transferdom.StatusPending
	}

	var et *transferdom.ErrorType
	if s := strings.TrimSpace(getStr("errorType")); s != "" {
		x := transferdom.ErrorType(s)
		et = &x
	}

	var em *string
	if s := strings.TrimSpace(getStr("errorMsg")); s != "" {
		x := s
		em = &x
	}

	var tx *string
	if s := strings.TrimSpace(getStr("txSignature")); s != "" {
		x := s
		tx = &x
	}

	t := transferdom.Transfer{
		ID:        strings.TrimSpace(getStr("id")),
		Attempt:   attempt,
		ProductID: productID,
		OrderID:   strings.TrimSpace(getStr("orderId")),
		AvatarID:  strings.TrimSpace(getStr("avatarId")),

		ToWalletAddress: strings.TrimSpace(getStr("toWalletAddress")),
		TxSignature:     tx,

		Status:    st,
		ErrorType: et,
		ErrorMsg:  em,

		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	if strings.TrimSpace(t.ID) == "" {
		t.ID = t.ProductID
	}

	return t, nil
}
