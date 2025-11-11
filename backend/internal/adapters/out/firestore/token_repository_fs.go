// backend/internal/adapters/out/firestore/token_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	tokendom "narratives/internal/domain/token"
)

// ============================================================
// Firestore-based Token Repository
// (Firestore implementation corresponding to TokenRepositoryPG)
// ============================================================

type TokenRepositoryFS struct {
	Client *firestore.Client
}

func NewTokenRepositoryFS(client *firestore.Client) *TokenRepositoryFS {
	return &TokenRepositoryFS{Client: client}
}

func (r *TokenRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("tokens")
}

// ============================================================
// TokenRepo facade for usecase.TokenRepo
// ============================================================

// GetByID(ctx, id) (tokendom.Token, error)
// In this domain, "id" corresponds to mintAddress (doc ID).
func (r *TokenRepositoryFS) GetByID(ctx context.Context, id string) (tokendom.Token, error) {
	return r.GetByMintAddress(ctx, id)
}

// Exists(ctx, id) (bool, error)
// Check existence by mintAddress (doc ID).
func (r *TokenRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}

	_, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Create(ctx, v tokendom.Token) (tokendom.Token, error)
// We treat MintAddress as the primary key & document ID (not auto-generated).
func (r *TokenRepositoryFS) Create(ctx context.Context, v tokendom.Token) (tokendom.Token, error) {
	if r.Client == nil {
		return tokendom.Token{}, errors.New("firestore client is nil")
	}

	mintAddress := strings.TrimSpace(v.MintAddress)
	if mintAddress == "" {
		return tokendom.Token{}, errors.New("mint address required for Create")
	}

	docRef := r.col().Doc(mintAddress)

	// mintedAt is Firestore-only metadata; set on create if not present.
	now := time.Now().UTC()
	data := tokenToDocData(v)
	if _, ok := data["mintedAt"]; !ok {
		data["mintedAt"] = now
	}

	if _, err := docRef.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return tokendom.Token{}, tokendom.ErrConflict
		}
		return tokendom.Token{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return tokendom.Token{}, err
	}
	return docToToken(snap)
}

// Save(ctx, v tokendom.Token) (tokendom.Token, error)
// Upsert-like: if token with MintAddress exists -> Update, else -> Create.
func (r *TokenRepositoryFS) Save(ctx context.Context, v tokendom.Token) (tokendom.Token, error) {
	if r.Client == nil {
		return tokendom.Token{}, errors.New("firestore client is nil")
	}

	mintAddress := strings.TrimSpace(v.MintAddress)
	if mintAddress == "" {
		return tokendom.Token{}, errors.New("mint address required for Save")
	}

	exists, err := r.Exists(ctx, mintAddress)
	if err != nil {
		return tokendom.Token{}, err
	}

	if !exists {
		// New document
		return r.Create(ctx, v)
	}

	// Existing document -> partial update via UpdateTokenInput
	patch := tokendom.UpdateTokenInput{
		MintRequestID: trimPtrOrNil(v.MintRequestID),
		Owner:         trimPtrOrNil(v.Owner),
	}

	return r.Update(ctx, mintAddress, patch)
}

// ============================================================
// Lower-level / richer query methods
// (List, Count, Transfer, GetStats, etc.)
// ============================================================

// GetByMintAddress fetches a token by its mint address (doc ID).
func (r *TokenRepositoryFS) GetByMintAddress(ctx context.Context, mintAddress string) (tokendom.Token, error) {
	if r.Client == nil {
		return tokendom.Token{}, errors.New("firestore client is nil")
	}

	mintAddress = strings.TrimSpace(mintAddress)
	if mintAddress == "" {
		return tokendom.Token{}, tokendom.ErrNotFound
	}

	snap, err := r.col().Doc(mintAddress).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return tokendom.Token{}, tokendom.ErrNotFound
	}
	if err != nil {
		return tokendom.Token{}, err
	}

	return docToToken(snap)
}

// List applies Filter + Sort + Paging using Firestore query + in-memory filtering.
func (r *TokenRepositoryFS) List(
	ctx context.Context,
	filter tokendom.Filter,
	sort tokendom.Sort,
	page tokendom.Page,
) (tokendom.PageResult, error) {
	if r.Client == nil {
		return tokendom.PageResult{}, errors.New("firestore client is nil")
	}

	q := r.col().Query
	q = applyTokenOrderByFS(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []tokendom.Token
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return tokendom.PageResult{}, err
		}
		t, err := docToToken(doc)
		if err != nil {
			return tokendom.PageResult{}, err
		}
		if matchTokenFilterFS(doc, t, filter) {
			all = append(all, t)
		}
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)
	total := len(all)

	if total == 0 {
		return tokendom.PageResult{
			Items:      []tokendom.Token{},
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

	return tokendom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// Count returns count of tokens matching filter (client-side).
func (r *TokenRepositoryFS) Count(ctx context.Context, filter tokendom.Filter) (int, error) {
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
		t, err := docToToken(doc)
		if err != nil {
			return 0, err
		}
		if matchTokenFilterFS(doc, t, filter) {
			total++
		}
	}
	return total, nil
}

// GetByOwner returns tokens owned by the given owner.
func (r *TokenRepositoryFS) GetByOwner(ctx context.Context, owner string) ([]tokendom.Token, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	owner = strings.TrimSpace(owner)
	if owner == "" {
		return []tokendom.Token{}, nil
	}

	q := r.col().Where("owner", "==", owner)
	it := q.Documents(ctx)
	defer it.Stop()

	var out []tokendom.Token
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		t, err := docToToken(doc)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// GetByMintRequest returns tokens matching given mintRequestID.
func (r *TokenRepositoryFS) GetByMintRequest(ctx context.Context, mintRequestID string) ([]tokendom.Token, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	mintRequestID = strings.TrimSpace(mintRequestID)
	if mintRequestID == "" {
		return []tokendom.Token{}, nil
	}

	q := r.col().Where("mintRequestId", "==", mintRequestID)
	it := q.Documents(ctx)
	defer it.Stop()

	var out []tokendom.Token
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		t, err := docToToken(doc)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// Update applies UpdateTokenInput to the token document.
func (r *TokenRepositoryFS) Update(
	ctx context.Context,
	mintAddress string,
	in tokendom.UpdateTokenInput,
) (tokendom.Token, error) {
	if r.Client == nil {
		return tokendom.Token{}, errors.New("firestore client is nil")
	}

	mintAddress = strings.TrimSpace(mintAddress)
	if mintAddress == "" {
		return tokendom.Token{}, tokendom.ErrNotFound
	}

	ref := r.col().Doc(mintAddress)

	// Ensure exists
	snap, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return tokendom.Token{}, tokendom.ErrNotFound
	}
	if err != nil {
		return tokendom.Token{}, err
	}

	var updates []firestore.Update

	if in.MintRequestID != nil {
		updates = append(updates, firestore.Update{
			Path:  "mintRequestId",
			Value: strings.TrimSpace(*in.MintRequestID),
		})
	}
	if in.Owner != nil {
		owner := strings.TrimSpace(*in.Owner)
		updates = append(updates, firestore.Update{
			Path:  "owner",
			Value: owner,
		})
		// bump lastTransferredAt when owner changes
		updates = append(updates, firestore.Update{
			Path:  "lastTransferredAt",
			Value: time.Now().UTC(),
		})
	}

	if len(updates) == 0 {
		// no-op, return current
		return docToToken(snap)
	}

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return tokendom.Token{}, tokendom.ErrNotFound
		}
		return tokendom.Token{}, err
	}

	snap, err = ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return tokendom.Token{}, tokendom.ErrNotFound
		}
		return tokendom.Token{}, err
	}
	return docToToken(snap)
}

// Delete removes token by mintAddress (doc ID).
func (r *TokenRepositoryFS) Delete(ctx context.Context, mintAddress string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	mintAddress = strings.TrimSpace(mintAddress)
	if mintAddress == "" {
		return tokendom.ErrNotFound
	}

	ref := r.col().Doc(mintAddress)
	_, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return tokendom.ErrNotFound
	}
	if err != nil {
		return err
	}

	if _, err := ref.Delete(ctx); err != nil {
		return err
	}
	return nil
}

// Transfer is convenience for owner change + lastTransferredAt bump.
func (r *TokenRepositoryFS) Transfer(ctx context.Context, mintAddress, newOwner string) (tokendom.Token, error) {
	in := tokendom.UpdateTokenInput{
		Owner: &newOwner,
	}
	return r.Update(ctx, mintAddress, in)
}

// Burn is alias for Delete.
func (r *TokenRepositoryFS) Burn(ctx context.Context, mintAddress string) error {
	return r.Delete(ctx, mintAddress)
}

// GetStats computes stats by scanning tokens in Firestore.
// Note: client-side aggregation (not efficient for huge collections).
func (r *TokenRepositoryFS) GetStats(ctx context.Context) (tokendom.TokenStats, error) {
	if r.Client == nil {
		return tokendom.TokenStats{}, errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	var stats tokendom.TokenStats
	stats.ByOwner = map[string]int{}
	stats.ByMintRequest = map[string]int{}

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return tokendom.TokenStats{}, err
		}
		data := doc.Data()
		if data == nil {
			continue
		}

		stats.TotalTokens++

		mr := strings.TrimSpace(getString(data, "mintRequestId", "mint_request_id"))
		if mr != "" {
			stats.ByMintRequest[mr]++
		}

		owner := strings.TrimSpace(getString(data, "owner"))
		if owner != "" {
			stats.ByOwner[owner]++
		}
	}

	// UniqueOwners & UniqueMintRequests
	stats.UniqueOwners = len(stats.ByOwner)
	stats.UniqueMintRequests = len(stats.ByMintRequest)

	// TopOwners
	for owner, cnt := range stats.ByOwner {
		stats.TopOwners = append(stats.TopOwners, struct {
			Owner string
			Count int
		}{
			Owner: owner,
			Count: cnt,
		})
	}

	// TopMintRequests
	for mr, cnt := range stats.ByMintRequest {
		stats.TopMintRequests = append(stats.TopMintRequests, struct {
			MintRequestID string
			Count         int
		}{
			MintRequestID: mr,
			Count:         cnt,
		})
	}

	// NOTE: Sorting for TopOwners/TopMintRequests can be added if needed.

	return stats, nil
}

// WithTx uses Firestore RunTransaction to execute fn with transactional context.
// Note: the current repository methods do not accept a *firestore.Transaction;
// callers that need strict transactional semantics should ensure fn only uses
// operations that are compatible with this pattern.
func (r *TokenRepositoryFS) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	return r.Client.RunTransaction(ctx, func(txCtx context.Context, _ *firestore.Transaction) error {
		return fn(txCtx)
	})
}

// Reset deletes all tokens (for tests/dev) using transactions instead of the
// deprecated WriteBatch API.
func (r *TokenRepositoryFS) Reset(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	for {
		// Fetch a chunk of documents to delete.
		it := r.col().Limit(400).Documents(ctx)
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
			// No more documents to delete.
			break
		}

		// Delete this chunk within a transaction.
		err := r.Client.RunTransaction(ctx, func(txCtx context.Context, tx *firestore.Transaction) error {
			for _, ref := range refs {
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
// Helpers
// ============================================================

func docToToken(doc *firestore.DocumentSnapshot) (tokendom.Token, error) {
	data := doc.Data()
	if data == nil {
		return tokendom.Token{}, errors.New("empty token document: " + doc.Ref.ID)
	}

	return tokendom.Token{
		MintAddress:   strings.TrimSpace(doc.Ref.ID),
		MintRequestID: strings.TrimSpace(getString(data, "mintRequestId", "mint_request_id")),
		Owner:         strings.TrimSpace(getString(data, "owner")),
	}, nil
}

func tokenToDocData(v tokendom.Token) map[string]any {
	m := map[string]any{
		"mintRequestId": strings.TrimSpace(v.MintRequestID),
		"owner":         strings.TrimSpace(v.Owner),
	}
	// MintAddress is encoded as document ID; we do not duplicate unless desired.
	return m
}

// matchTokenFilterFS applies tokendom.Filter using document data + domain fields.
func matchTokenFilterFS(doc *firestore.DocumentSnapshot, t tokendom.Token, f tokendom.Filter) bool {
	trim := func(s string) string { return strings.TrimSpace(s) }
	data := doc.Data()

	inList := func(v string, xs []string) bool {
		if len(xs) == 0 {
			return true
		}
		v = trim(v)
		for _, x := range xs {
			if trim(x) == v {
				return true
			}
		}
		return false
	}
	getTimeField := func(key string) time.Time {
		if data == nil {
			return time.Time{}
		}
		if raw, ok := data[key]; ok {
			if tt, ok2 := raw.(time.Time); ok2 {
				return tt.UTC()
			}
		}
		return time.Time{}
	}

	// MintAddresses, MintRequestIDs, Owners (IN filters)
	if len(f.MintAddresses) > 0 && !inList(t.MintAddress, f.MintAddresses) {
		return false
	}
	if len(f.MintRequestIDs) > 0 && !inList(t.MintRequestID, f.MintRequestIDs) {
		return false
	}
	if len(f.Owners) > 0 && !inList(t.Owner, f.Owners) {
		return false
	}

	// MintAddressLike
	if v := trim(f.MintAddressLike); v != "" {
		if !strings.Contains(strings.ToLower(t.MintAddress), strings.ToLower(v)) {
			return false
		}
	}

	// Time range filters use Firestore-only fields mintedAt / lastTransferredAt (if present)
	mintedAt := getTimeField("mintedAt")
	lastTransferredAt := getTimeField("lastTransferredAt")

	if f.MintedFrom != nil && (mintedAt.IsZero() || mintedAt.Before(f.MintedFrom.UTC())) {
		return false
	}
	if f.MintedTo != nil && (mintedAt.IsZero() || !mintedAt.Before(f.MintedTo.UTC())) {
		return false
	}
	if f.LastTransferredFrom != nil && (lastTransferredAt.IsZero() || lastTransferredAt.Before(f.LastTransferredFrom.UTC())) {
		return false
	}
	if f.LastTransferredTo != nil && (lastTransferredAt.IsZero() || !lastTransferredAt.Before(f.LastTransferredTo.UTC())) {
		return false
	}

	return true
}

// applyTokenOrderByFS maps tokendom.Sort to Firestore orderBy.
func applyTokenOrderByFS(q firestore.Query, s tokendom.Sort) firestore.Query {
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	var field string

	switch col {
	case "mintaddress", "mint_address":
		field = firestore.DocumentID
	case "mintrequestid", "mint_request_id":
		field = "mintRequestId"
	case "owner":
		field = "owner"
	case "mintedat", "minted_at":
		field = "mintedAt"
	case "lasttransferredat", "last_transferred_at":
		field = "lastTransferredAt"
	default:
		// Default: mintedAt DESC, then doc ID DESC
		return q.OrderBy("mintedAt", firestore.Desc).
			OrderBy(firestore.DocumentID, firestore.Desc)
	}

	dir := firestore.Desc
	if strings.EqualFold(string(s.Order), "asc") {
		dir = firestore.Asc
	}

	if field == firestore.DocumentID {
		return q.OrderBy(field, dir)
	}
	return q.OrderBy(field, dir).
		OrderBy(firestore.DocumentID, dir)
}

// getString tries keys in order and returns first string value.
func getString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok2 := v.(string); ok2 {
				return s
			}
		}
	}
	return ""
}

// trimPtrOrNil returns pointer to trimmed string or nil if blank.
func trimPtrOrNil(s string) *string {
	t := strings.TrimSpace(s)
	if t == "" {
		return nil
	}
	return &t
}
