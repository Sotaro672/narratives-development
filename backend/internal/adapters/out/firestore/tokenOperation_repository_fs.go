// backend/internal/adapters/out/firestore/tokenOperation_repository_fs.go
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

	tod "narratives/internal/domain/tokenOperation"
)

// ========================================
// Firestore TokenOperation Repository
// ========================================

// Ensure interface implementation (uses RepositoryPort)
var _ tod.RepositoryPort = (*TokenOperationRepositoryFS)(nil)

// TokenOperationRepositoryFS implements tokenOperation.RepositoryPort using Firestore.
type TokenOperationRepositoryFS struct {
	Client *firestore.Client
}

func NewTokenOperationRepositoryFS(client *firestore.Client) *TokenOperationRepositoryFS {
	return &TokenOperationRepositoryFS{Client: client}
}

func (r *TokenOperationRepositoryFS) colOps() *firestore.CollectionRef {
	return r.Client.Collection("token_operations")
}

func (r *TokenOperationRepositoryFS) colHolders() *firestore.CollectionRef {
	return r.Client.Collection("token_holders")
}

func (r *TokenOperationRepositoryFS) colHistory() *firestore.CollectionRef {
	return r.Client.Collection("token_update_history")
}

func (r *TokenOperationRepositoryFS) colContents() *firestore.CollectionRef {
	return r.Client.Collection("token_operation_contents")
}

func (r *TokenOperationRepositoryFS) colProducts() *firestore.CollectionRef {
	return r.Client.Collection("product_details")
}

// ========================================
// Basic TokenOperation methods (for compatibility)
// These are extra vs RepositoryPort but kept for callers expecting PG-style API.
// ========================================

// GetByID returns a minimal tod.TokenOperation (ID, TokenBlueprintID, AssigneeID).
func (r *TokenOperationRepositoryFS) GetByID(ctx context.Context, id string) (tod.TokenOperation, error) {
	if r.Client == nil {
		return tod.TokenOperation{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return tod.TokenOperation{}, tod.ErrNotFound
	}

	snap, err := r.colOps().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return tod.TokenOperation{}, tod.ErrNotFound
	}
	if err != nil {
		return tod.TokenOperation{}, err
	}

	op, err := docToTokenOperation(snap)
	if err != nil {
		return tod.TokenOperation{}, err
	}
	return op, nil
}

// Exists checks if a TokenOperation with given ID exists.
func (r *TokenOperationRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}

	_, err := r.colOps().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Create inserts a new token_operations document using tod.TokenOperation.
// name/status/updatedBy はここでデフォルトを設定（PG 実装と同等の意味）。
func (r *TokenOperationRepositoryFS) Create(ctx context.Context, v tod.TokenOperation) (tod.TokenOperation, error) {
	if r.Client == nil {
		return tod.TokenOperation{}, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()
	id := strings.TrimSpace(v.ID)

	var ref *firestore.DocumentRef
	if id != "" {
		ref = r.colOps().Doc(id)
	} else {
		ref = r.colOps().NewDoc()
		id = ref.ID
	}

	data := map[string]any{
		"tokenBlueprintId": strings.TrimSpace(v.TokenBlueprintID),
		"assigneeId":       strings.TrimSpace(v.AssigneeID),
		"name":             "",
		"status":           "operational",
		"updatedAt":        now,
		"updatedBy":        "",
	}

	if _, err := ref.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return tod.TokenOperation{}, tod.ErrConflict
		}
		return tod.TokenOperation{}, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return tod.TokenOperation{}, err
	}
	return docToTokenOperation(snap)
}

// Save updates tokenBlueprintId / assigneeId for an existing doc.
// 他の列(name/status/updatedBy等)は触らず、updatedAtのみ更新。
func (r *TokenOperationRepositoryFS) Save(ctx context.Context, v tod.TokenOperation) (tod.TokenOperation, error) {
	if r.Client == nil {
		return tod.TokenOperation{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(v.ID)
	if id == "" {
		return tod.TokenOperation{}, tod.ErrNotFound
	}

	ref := r.colOps().Doc(id)

	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return tod.TokenOperation{}, tod.ErrNotFound
	} else if err != nil {
		return tod.TokenOperation{}, err
	}

	updates := []firestore.Update{
		{Path: "tokenBlueprintId", Value: strings.TrimSpace(v.TokenBlueprintID)},
		{Path: "assigneeId", Value: strings.TrimSpace(v.AssigneeID)},
		{Path: "updatedAt", Value: time.Now().UTC()},
	}

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return tod.TokenOperation{}, tod.ErrNotFound
		}
		return tod.TokenOperation{}, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return tod.TokenOperation{}, err
	}
	return docToTokenOperation(snap)
}

// Delete removes a TokenOperation by ID.
func (r *TokenOperationRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return tod.ErrNotFound
	}

	ref := r.colOps().Doc(id)
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return tod.ErrNotFound
	} else if err != nil {
		return err
	}

	if _, err := ref.Delete(ctx); err != nil {
		return err
	}
	return nil
}

// ========================================
// RepositoryPort-required methods
// (OperationalToken view, holders, history, contents, product detail)
// ========================================

// GetOperationalTokens returns enriched OperationalToken list.
func (r *TokenOperationRepositoryFS) GetOperationalTokens(ctx context.Context) ([]*tod.OperationalToken, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	it := r.colOps().
		OrderBy("updatedAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc).
		Documents(ctx)
	defer it.Stop()

	var out []*tod.OperationalToken
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		op, err := r.buildOperationalTokenFromDoc(ctx, doc)
		if err != nil {
			return nil, err
		}
		out = append(out, op)
	}
	return out, nil
}

// GetOperationalTokenByID returns a single enriched OperationalToken.
func (r *TokenOperationRepositoryFS) GetOperationalTokenByID(ctx context.Context, id string) (*tod.OperationalToken, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: operational token not found", tod.ErrNotFound)
	}

	snap, err := r.colOps().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, fmt.Errorf("%w: operational token not found", tod.ErrNotFound)
	}
	if err != nil {
		return nil, err
	}

	op, err := r.buildOperationalTokenFromDoc(ctx, snap)
	if err != nil {
		return nil, err
	}
	return op, nil
}

// CreateOperationalToken creates a new token_operation for the given blueprint/assignee.
func (r *TokenOperationRepositoryFS) CreateOperationalToken(ctx context.Context, in tod.CreateOperationalTokenData) (*tod.OperationalToken, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()
	ref := r.colOps().NewDoc()

	data := map[string]any{
		"tokenBlueprintId": strings.TrimSpace(in.TokenBlueprintID),
		"assigneeId":       strings.TrimSpace(in.AssigneeID),
		"name":             "",
		"status":           "operational",
		"updatedAt":        now,
		"updatedBy":        "",
	}

	if _, err := ref.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, tod.ErrConflict
		}
		return nil, err
	}

	return r.GetOperationalTokenByID(ctx, ref.ID)
}

// UpdateOperationalToken updates fields of an operational token document.
func (r *TokenOperationRepositoryFS) UpdateOperationalToken(ctx context.Context, id string, in tod.UpdateOperationalTokenData) (*tod.OperationalToken, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: operational token not found", tod.ErrNotFound)
	}

	ref := r.colOps().Doc(id)

	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return nil, fmt.Errorf("%w: operational token not found", tod.ErrNotFound)
	} else if err != nil {
		return nil, err
	}

	var updates []firestore.Update

	if in.AssigneeID != nil {
		updates = append(updates, firestore.Update{
			Path:  "assigneeId",
			Value: strings.TrimSpace(*in.AssigneeID),
		})
	}
	if in.Name != nil {
		updates = append(updates, firestore.Update{
			Path:  "name",
			Value: strings.TrimSpace(*in.Name),
		})
	}
	if in.Status != nil {
		updates = append(updates, firestore.Update{
			Path:  "status",
			Value: strings.TrimSpace(*in.Status),
		})
	}
	if v := strings.TrimSpace(in.UpdatedBy); v != "" {
		updates = append(updates, firestore.Update{
			Path:  "updatedBy",
			Value: v,
		})
	}
	// always bump updatedAt
	updates = append(updates, firestore.Update{
		Path:  "updatedAt",
		Value: time.Now().UTC(),
	})

	if len(updates) == 0 {
		return r.GetOperationalTokenByID(ctx, id)
	}

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("%w: operational token not found", tod.ErrNotFound)
		}
		if status.Code(err) == codes.AlreadyExists {
			return nil, tod.ErrConflict
		}
		return nil, err
	}

	return r.GetOperationalTokenByID(ctx, id)
}

// ========================================
// Holders / History / Contents / ProductDetail
// ========================================

func (r *TokenOperationRepositoryFS) GetHoldersByTokenID(
	ctx context.Context,
	params tod.HolderSearchParams,
) (holders []*tod.Holder, total int, err error) {
	if r.Client == nil {
		return nil, 0, errors.New("firestore client is nil")
	}

	tokenID := strings.TrimSpace(params.TokenID)
	if tokenID == "" {
		return []*tod.Holder{}, 0, nil
	}

	q := r.colHolders().Where("tokenId", "==", tokenID)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []*tod.Holder
	search := strings.ToLower(strings.TrimSpace(params.Query))

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, 0, err
		}
		h, err := docToHolder(doc)
		if err != nil {
			return nil, 0, err
		}
		if search != "" && !strings.Contains(strings.ToLower(h.WalletAddress), search) {
			continue
		}
		all = append(all, h)
	}

	total = len(all)

	offset := params.Offset
	if offset < 0 {
		offset = 0
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}

	return all[offset:end], total, nil
}

func (r *TokenOperationRepositoryFS) GetTokenUpdateHistory(
	ctx context.Context,
	params tod.TokenUpdateHistorySearchParams,
) ([]*tod.TokenUpdateHistory, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	tokenID := strings.TrimSpace(params.TokenID)
	if tokenID == "" {
		return []*tod.TokenUpdateHistory{}, nil
	}

	q := r.colHistory().
		Where("tokenId", "==", tokenID).
		OrderBy("createdAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc)

	offset := params.Offset
	if offset < 0 {
		offset = 0
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 100
	}

	if offset > 0 {
		q = q.Offset(offset)
	}
	q = q.Limit(limit)

	it := q.Documents(ctx)
	defer it.Stop()

	var out []*tod.TokenUpdateHistory
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		h, err := docToUpdateHistory(doc)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, nil
}

func (r *TokenOperationRepositoryFS) GetTokenContents(ctx context.Context, tokenID string) ([]*tod.TokenContent, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return []*tod.TokenContent{}, nil
	}

	q := r.colContents().
		Where("tokenId", "==", tokenID).
		OrderBy("createdAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc)

	it := q.Documents(ctx)
	defer it.Stop()

	var out []*tod.TokenContent
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		tc, err := docToTokenContent(doc)
		if err != nil {
			return nil, err
		}
		out = append(out, tc)
	}
	return out, nil
}

func (r *TokenOperationRepositoryFS) AddTokenContent(
	ctx context.Context,
	tokenID string,
	typ, url, description, publishedBy string,
) (*tod.TokenContent, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return nil, fmt.Errorf("%w: token id required", tod.ErrNotFound)
	}

	now := time.Now().UTC()

	ref := r.colContents().NewDoc()
	data := map[string]any{
		"tokenId":     tokenID,
		"type":        strings.TrimSpace(typ),
		"url":         strings.TrimSpace(url),
		"description": strings.TrimSpace(description),
		"publishedBy": strings.TrimSpace(publishedBy),
		"createdAt":   now,
	}

	if _, err := ref.Create(ctx, data); err != nil {
		return nil, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}
	return docToTokenContent(snap)
}

func (r *TokenOperationRepositoryFS) DeleteTokenContent(ctx context.Context, contentID string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	contentID = strings.TrimSpace(contentID)
	if contentID == "" {
		return fmt.Errorf("%w: token content not found", tod.ErrNotFound)
	}

	ref := r.colContents().Doc(contentID)
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return fmt.Errorf("%w: token content not found", tod.ErrNotFound)
	} else if err != nil {
		return err
	}

	if _, err := ref.Delete(ctx); err != nil {
		return err
	}
	return nil
}

func (r *TokenOperationRepositoryFS) GetProductDetailByID(ctx context.Context, productID string) (*tod.ProductDetail, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, fmt.Errorf("%w: product detail not found", tod.ErrNotFound)
	}

	snap, err := r.colProducts().Doc(productID).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, fmt.Errorf("%w: product detail not found", tod.ErrNotFound)
	}
	if err != nil {
		return nil, err
	}

	pd, err := docToProductDetail(snap)
	if err != nil {
		return nil, err
	}
	return &pd, nil
}

// ResetTokenOperations is mainly for tests/dev: clears token_operations related collections.
func (r *TokenOperationRepositoryFS) ResetTokenOperations(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	clearCol := func(col *firestore.CollectionRef) error {
		it := col.Documents(ctx)
		batch := r.Client.Batch()
		count := 0
		for {
			doc, err := it.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				return err
			}
			batch.Delete(doc.Ref)
			count++
			if count%400 == 0 {
				if _, err := batch.Commit(ctx); err != nil {
					return err
				}
				batch = r.Client.Batch()
			}
		}
		if count > 0 {
			if _, err := batch.Commit(ctx); err != nil {
				return err
			}
		}
		return nil
	}

	if err := clearCol(r.colContents()); err != nil {
		return err
	}
	if err := clearCol(r.colHistory()); err != nil {
		return err
	}
	if err := clearCol(r.colHolders()); err != nil {
		return err
	}
	if err := clearCol(r.colOps()); err != nil {
		return err
	}
	return nil
}

// WithTx: Firestoreトランザクションラッパ（簡易版）
// 必要であれば RunTransaction に差し替え可能。
func (r *TokenOperationRepositoryFS) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	// シンプルにそのまま実行（Firestoreの複雑なTx要件が無い前提）
	return fn(ctx)
}

// ========================================
// Helpers: Firestore -> Domain mapping
// ========================================

func docToTokenOperation(doc *firestore.DocumentSnapshot) (tod.TokenOperation, error) {
	data := doc.Data()
	if data == nil {
		return tod.TokenOperation{}, tod.ErrNotFound
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}

	return tod.TokenOperation{
		ID:               strings.TrimSpace(doc.Ref.ID),
		TokenBlueprintID: getStr("tokenBlueprintId", "token_blueprint_id"),
		AssigneeID:       getStr("assigneeId", "assignee_id"),
	}, nil
}

func (r *TokenOperationRepositoryFS) buildOperationalTokenFromDoc(
	ctx context.Context,
	doc *firestore.DocumentSnapshot,
) (*tod.OperationalToken, error) {
	data := doc.Data()
	if data == nil {
		return nil, tod.ErrNotFound
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	getTime := func(keys ...string) time.Time {
		for _, k := range keys {
			if v, ok := data[k].(time.Time); ok {
				return v.UTC()
			}
		}
		return time.Time{}
	}

	id := strings.TrimSpace(doc.Ref.ID)
	tbid := getStr("tokenBlueprintId", "token_blueprint_id")
	aid := getStr("assigneeId", "assignee_id")

	name := getStr("name")
	status := getStr("status")
	updatedBy := getStr("updatedBy", "updated_by")
	updatedAt := getTime("updatedAt", "updated_at")
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}

	// Fetch TokenBlueprint
	var tokenName, symbol, brandID string
	if tbid != "" {
		if tbSnap, err := r.Client.Collection("token_blueprints").Doc(tbid).Get(ctx); err == nil {
			if tbData := tbSnap.Data(); tbData != nil {
				if v, ok := tbData["name"].(string); ok {
					tokenName = strings.TrimSpace(v)
				}
				if v, ok := tbData["symbol"].(string); ok {
					symbol = strings.TrimSpace(v)
				}
				if v, ok := tbData["brandId"].(string); ok {
					brandID = strings.TrimSpace(v)
				} else if v, ok := tbData["brand_id"].(string); ok {
					brandID = strings.TrimSpace(v)
				}
			}
		}
	}

	// Fetch Brand
	var brandName string
	if brandID != "" {
		if bSnap, err := r.Client.Collection("brands").Doc(brandID).Get(ctx); err == nil {
			if bd, ok := bSnap.Data()["name"].(string); ok {
				brandName = strings.TrimSpace(bd)
			}
		}
	}

	// Fetch Member (assignee)
	var assigneeName string
	if aid != "" {
		if mSnap, err := r.Client.Collection("members").Doc(aid).Get(ctx); err == nil {
			if md, ok := mSnap.Data()["name"].(string); ok {
				assigneeName = strings.TrimSpace(md)
			}
		}
	}

	return &tod.OperationalToken{
		ID:               id,
		TokenBlueprintID: tbid,
		AssigneeID:       aid,
		TokenName:        tokenName,
		Symbol:           symbol,
		BrandID:          brandID,
		AssigneeName:     assigneeName,
		BrandName:        brandName,
		Name:             name,
		Status:           status,
		UpdatedAt:        updatedAt,
		UpdatedBy:        updatedBy,
	}, nil
}

func docToHolder(doc *firestore.DocumentSnapshot) (*tod.Holder, error) {
	data := doc.Data()
	if data == nil {
		return nil, tod.ErrNotFound
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	getTime := func(keys ...string) time.Time {
		for _, k := range keys {
			if v, ok := data[k].(time.Time); ok {
				return v.UTC()
			}
		}
		return time.Time{}
	}

	return &tod.Holder{
		ID:            strings.TrimSpace(doc.Ref.ID),
		TokenID:       getStr("tokenId", "token_id"),
		WalletAddress: getStr("walletAddress", "wallet_address"),
		Balance:       getStr("balance"),
		UpdatedAt:     getTime("updatedAt", "updated_at"),
	}, nil
}

func docToUpdateHistory(doc *firestore.DocumentSnapshot) (*tod.TokenUpdateHistory, error) {
	data := doc.Data()
	if data == nil {
		return nil, tod.ErrNotFound
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	getTime := func(keys ...string) time.Time {
		for _, k := range keys {
			if v, ok := data[k].(time.Time); ok {
				return v.UTC()
			}
		}
		return time.Time{}
	}

	return &tod.TokenUpdateHistory{
		ID:         strings.TrimSpace(doc.Ref.ID),
		TokenID:    getStr("tokenId", "token_id"),
		Event:      getStr("event"),
		AssigneeID: getStr("assigneeId", "assignee_id"),
		Note:       getStr("note"),
		CreatedAt:  getTime("createdAt", "created_at"),
	}, nil
}

func docToTokenContent(doc *firestore.DocumentSnapshot) (*tod.TokenContent, error) {
	data := doc.Data()
	if data == nil {
		return nil, tod.ErrNotFound
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	getTime := func(keys ...string) time.Time {
		for _, k := range keys {
			if v, ok := data[k].(time.Time); ok {
				return v.UTC()
			}
		}
		return time.Time{}
	}

	return &tod.TokenContent{
		ID:          strings.TrimSpace(doc.Ref.ID),
		TokenID:     getStr("tokenId", "token_id"),
		Type:        getStr("type"),
		URL:         getStr("url"),
		Description: getStr("description"),
		PublishedBy: getStr("publishedBy", "published_by"),
		CreatedAt:   getTime("createdAt", "created_at"),
	}, nil
}

func docToProductDetail(doc *firestore.DocumentSnapshot) (tod.ProductDetail, error) {
	data := doc.Data()
	if data == nil {
		return tod.ProductDetail{}, tod.ErrNotFound
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}

	return tod.ProductDetail{
		ID:          strings.TrimSpace(doc.Ref.ID),
		Name:        getStr("name"),
		Description: getStr("description"),
	}, nil
}
