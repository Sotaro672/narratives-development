package firestore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ========================================
// Firestore TokenBlueprint Repository
// ========================================

type TokenBlueprintRepositoryFS struct {
	Client *firestore.Client
}

func NewTokenBlueprintRepositoryFS(client *firestore.Client) *TokenBlueprintRepositoryFS {
	return &TokenBlueprintRepositoryFS{Client: client}
}

func (r *TokenBlueprintRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("token_blueprints")
}

// ========================================
// RepositoryPort impl
// ========================================

func (r *TokenBlueprintRepositoryFS) GetByID(ctx context.Context, id string) (*tbdom.TokenBlueprint, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, tbdom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, tbdom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	tb, err := docToTokenBlueprint(snap)
	if err != nil {
		return nil, err
	}
	return &tb, nil
}

// ✅ NEW: GetPatchByID returns a lightweight Patch used by read-models (e.g. Inventory detail).
// - 実装は Firestore から必要最小限を読み出し、tbdom.Patch に詰める。
// - Patch のフィールド名が将来変わっても壊れにくいよう、reflect で「存在するフィールドだけ」セットする。
// - ★重要: Patch 側が string / *string のどちらでも値が入るように「両方セット」する
// ✅ NEW: GetPatchByID returns a lightweight Patch used by read-models (e.g. Inventory detail).
func (r *TokenBlueprintRepositoryFS) GetPatchByID(ctx context.Context, id string) (tbdom.Patch, error) {
	if r.Client == nil {
		return tbdom.Patch{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return tbdom.Patch{}, tbdom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return tbdom.Patch{}, tbdom.ErrNotFound
	}
	if err != nil {
		return tbdom.Patch{}, err
	}

	// Patch 用に最低限だけ読む（TokenBlueprint 全体に依存しない）
	var raw struct {
		Name        string  `firestore:"name"`
		Symbol      string  `firestore:"symbol"`
		BrandID     string  `firestore:"brandId"`
		CompanyID   string  `firestore:"companyId"`
		Description string  `firestore:"description"`
		IconID      *string `firestore:"iconId"`
		MetadataURI string  `firestore:"metadataUri"`
		Minted      any     `firestore:"minted"` // bool / string / nil 想定
	}
	if err := snap.DataTo(&raw); err != nil {
		return tbdom.Patch{}, err
	}

	// minted 旧データ互換
	var mintedBool bool
	switch v := raw.Minted.(type) {
	case bool:
		mintedBool = v
	case string:
		mintedBool = strings.TrimSpace(v) == "minted"
	case nil:
		mintedBool = false
	default:
		mintedBool = false
	}

	trim := func(s string) string { return strings.TrimSpace(s) }

	// ★ 先に patch を作る（これが無いと undefined: patch になる）
	patch := tbdom.Patch{}

	// ★ Patch が string / *string どちらでも入るように両方セットする
	setBoth := func(names []string, v string) {
		v = trim(v)
		// string フィールドがあれば入る
		setPatchString(&patch, names, v)
		// *string フィールドがあれば入る（v=="" は nil にしたいので弾く）
		if v != "" {
			setPatchPtrString(&patch, names, v)
		}
	}

	// id
	setPatchString(&patch, []string{"ID", "TokenBlueprintID"}, trim(id))

	// name/symbol
	setBoth([]string{"Name", "TokenName", "TokenBlueprintName"}, raw.Name)
	setBoth([]string{"Symbol"}, raw.Symbol)

	// ★重要: list_create_query.go は patch.BrandID(*string) を参照するので必ず入れる
	setBoth([]string{"BrandID", "BrandId"}, raw.BrandID)
	setBoth([]string{"CompanyID", "CompanyId"}, raw.CompanyID)

	// description / metadataUri / minted
	setBoth([]string{"Description"}, raw.Description)
	setPatchBool(&patch, []string{"Minted"}, mintedBool)
	setBoth([]string{"MetadataURI", "MetadataUrl", "MetadataURL"}, raw.MetadataURI)

	// iconId は *string / string どちらでも入れられるようにする
	if raw.IconID != nil {
		v := trim(*raw.IconID)
		if v != "" {
			setPatchPtrString(&patch, []string{"IconID", "IconId"}, v)
			setPatchString(&patch, []string{"IconID", "IconId"}, v)
		}
	}

	return patch, nil
}

// GetNameByID returns only the Name field of a TokenBlueprint.
func (r *TokenBlueprintRepositoryFS) GetNameByID(ctx context.Context, id string) (string, error) {
	if r.Client == nil {
		return "", errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return "", tbdom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return "", tbdom.ErrNotFound
	}
	if err != nil {
		return "", err
	}

	var raw struct {
		Name string `firestore:"name"`
	}
	if err := snap.DataTo(&raw); err != nil {
		return "", err
	}

	return strings.TrimSpace(raw.Name), nil
}

func (r *TokenBlueprintRepositoryFS) List(
	ctx context.Context,
	filter tbdom.Filter,
	page tbdom.Page,
) (tbdom.PageResult, error) {
	if r.Client == nil {
		return tbdom.PageResult{}, errors.New("firestore client is nil")
	}

	// デフォルト: createdAt DESC, doc ID DESC
	q := r.col().
		OrderBy("createdAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []tbdom.TokenBlueprint
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return tbdom.PageResult{}, err
		}
		tb, err := docToTokenBlueprint(doc)
		if err != nil {
			return tbdom.PageResult{}, err
		}
		if matchTBFilter(tb, filter) {
			all = append(all, tb)
		}
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)
	total := len(all)

	if total == 0 {
		return tbdom.PageResult{
			Items:      []tbdom.TokenBlueprint{},
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

	return tbdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// ListByCompanyID: companyId で限定した一覧取得。
// ★修正: Firestore の Where で companyId を確実に絞る（in-memory 全走査をやめる）
func (r *TokenBlueprintRepositoryFS) ListByCompanyID(
	ctx context.Context,
	companyID string,
	page tbdom.Page,
) (tbdom.PageResult, error) {
	if r.Client == nil {
		return tbdom.PageResult{}, errors.New("firestore client is nil")
	}

	cid := strings.TrimSpace(companyID)
	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	if cid == "" {
		return tbdom.PageResult{
			Items:      []tbdom.TokenBlueprint{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	// companyId を Firestore クエリで確実に絞る（minted は絞らない）
	baseQ := r.col().
		Where("companyId", "==", cid).
		OrderBy("createdAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc)

	// totalCount を計算（deletedAt が入っているものは除外）
	total := 0
	{
		it := baseQ.Documents(ctx)
		defer it.Stop()

		for {
			doc, err := it.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				return tbdom.PageResult{}, err
			}
			tb, err := docToTokenBlueprint(doc)
			if err != nil {
				return tbdom.PageResult{}, err
			}
			if tb.DeletedAt != nil {
				continue
			}
			total++
		}
	}

	if total == 0 {
		return tbdom.PageResult{
			Items:      []tbdom.TokenBlueprint{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	// ページングして items を取得
	q := baseQ.Offset(offset).Limit(perPage)

	it := q.Documents(ctx)
	defer it.Stop()

	items := make([]tbdom.TokenBlueprint, 0, perPage)
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return tbdom.PageResult{}, err
		}

		tb, err := docToTokenBlueprint(doc)
		if err != nil {
			return tbdom.PageResult{}, err
		}
		if tb.DeletedAt != nil {
			continue
		}

		items = append(items, tb)
	}

	return tbdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *TokenBlueprintRepositoryFS) Count(ctx context.Context, filter tbdom.Filter) (int, error) {
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
		tb, err := docToTokenBlueprint(doc)
		if err != nil {
			return 0, err
		}
		if matchTBFilter(tb, filter) {
			total++
		}
	}
	return total, nil
}

func (r *TokenBlueprintRepositoryFS) Create(ctx context.Context, in tbdom.CreateTokenBlueprintInput) (*tbdom.TokenBlueprint, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()

	// CreatedAt は nil の場合のみ now を使う
	createdAt := now
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}

	files := sanitizeStrings(in.ContentFiles)

	var iconID *string
	if in.IconID != nil {
		if v := strings.TrimSpace(*in.IconID); v != "" {
			iconID = &v
		}
	}

	// minted は create 時は必ず false
	minted := false

	docRef := r.col().NewDoc()

	data := map[string]any{
		"name":         strings.TrimSpace(in.Name),
		"symbol":       strings.TrimSpace(in.Symbol),
		"brandId":      strings.TrimSpace(in.BrandID),
		"companyId":    strings.TrimSpace(in.CompanyID), // ★ companyId
		"description":  strings.TrimSpace(in.Description),
		"contentFiles": files,
		"assigneeId":   strings.TrimSpace(in.AssigneeID),
		"minted":       minted, // ★ minted を bool で Firestore に保存
		"createdAt":    createdAt,
		"deletedAt":    nil,
		"deletedBy":    nil,
		// "metadataUri" は create 時点では未設定（Publish 時に付与）
	}

	if iconID != nil {
		data["iconId"] = *iconID
	}

	if s := strings.TrimSpace(in.CreatedBy); s != "" {
		data["createdBy"] = s
	}
	if s := strings.TrimSpace(in.UpdatedBy); s != "" {
		data["updatedBy"] = s
	}

	// ★ UpdatedAt は入力がある場合のみ保存する
	if in.UpdatedAt != nil && !in.UpdatedAt.IsZero() {
		data["updatedAt"] = in.UpdatedAt.UTC()
	}

	if _, err := docRef.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, tbdom.ErrConflict
		}
		return nil, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return nil, err
	}
	tb, err := docToTokenBlueprint(snap)
	if err != nil {
		return nil, err
	}
	return &tb, nil
}

func (r *TokenBlueprintRepositoryFS) Update(
	ctx context.Context,
	id string,
	in tbdom.UpdateTokenBlueprintInput,
) (*tbdom.TokenBlueprint, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, tbdom.ErrNotFound
	}

	ref := r.col().Doc(id)

	// Ensure exists
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return nil, tbdom.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	var updates []firestore.Update

	setStr := func(field string, p *string) {
		if p != nil {
			updates = append(updates, firestore.Update{
				Path:  field,
				Value: strings.TrimSpace(*p),
			})
		}
	}

	setStr("name", in.Name)
	setStr("symbol", in.Symbol)
	setStr("brandId", in.BrandID)
	setStr("description", in.Description)
	setStr("assigneeId", in.AssigneeID)
	// companyId は現状更新しない想定（必要ならここに追加）

	// ★ metadataUri の更新
	if in.MetadataURI != nil {
		updates = append(updates, firestore.Update{
			Path:  "metadataUri",
			Value: strings.TrimSpace(*in.MetadataURI),
		})
	}

	// minted フィールド更新（bool）
	if in.Minted != nil {
		updates = append(updates, firestore.Update{
			Path:  "minted",
			Value: *in.Minted,
		})
	}

	// iconId (empty => null)
	if in.IconID != nil {
		v := strings.TrimSpace(*in.IconID)
		if v == "" {
			updates = append(updates, firestore.Update{
				Path:  "iconId",
				Value: nil,
			})
		} else {
			updates = append(updates, firestore.Update{
				Path:  "iconId",
				Value: v,
			})
		}
	}

	// contentFiles
	if in.ContentFiles != nil {
		files := sanitizeStrings(*in.ContentFiles)
		updates = append(updates, firestore.Update{
			Path:  "contentFiles",
			Value: files,
		})
	}

	// updatedAt
	if in.UpdatedAt != nil {
		if in.UpdatedAt.IsZero() {
			updates = append(updates, firestore.Update{
				Path:  "updatedAt",
				Value: nil,
			})
		} else {
			updates = append(updates, firestore.Update{
				Path:  "updatedAt",
				Value: in.UpdatedAt.UTC(),
			})
		}
	} else {
		updates = append(updates, firestore.Update{
			Path:  "updatedAt",
			Value: time.Now().UTC(),
		})
	}

	// updatedBy
	if in.UpdatedBy != nil {
		v := strings.TrimSpace(*in.UpdatedBy)
		if v == "" {
			updates = append(updates, firestore.Update{
				Path:  "updatedBy",
				Value: nil,
			})
		} else {
			updates = append(updates, firestore.Update{
				Path:  "updatedBy",
				Value: v,
			})
		}
	}

	// deletedAt / deletedBy
	if in.DeletedAt != nil {
		if in.DeletedAt.IsZero() {
			updates = append(updates, firestore.Update{
				Path:  "deletedAt",
				Value: nil,
			})
		} else {
			updates = append(updates, firestore.Update{
				Path:  "deletedAt",
				Value: in.DeletedAt.UTC(),
			})
		}
	}
	if in.DeletedBy != nil {
		v := strings.TrimSpace(*in.DeletedBy)
		if v == "" {
			updates = append(updates, firestore.Update{
				Path:  "deletedBy",
				Value: nil,
			})
		} else {
			updates = append(updates, firestore.Update{
				Path:  "deletedBy",
				Value: v,
			})
		}
	}

	if len(updates) == 0 {
		// no-op
		snap, err := ref.Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return nil, tbdom.ErrNotFound
			}
			return nil, err
		}
		tb, err := docToTokenBlueprint(snap)
		if err != nil {
			return nil, err
		}
		return &tb, nil
	}

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, tbdom.ErrNotFound
		}
		return nil, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, tbdom.ErrNotFound
		}
		return nil, err
	}
	tb, err := docToTokenBlueprint(snap)
	if err != nil {
		return nil, err
	}
	return &tb, nil
}

func (r *TokenBlueprintRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return tbdom.ErrNotFound
	}

	ref := r.col().Doc(id)
	if _, err := ref.Get(ctx); status.Code(err) == codes.NotFound {
		return tbdom.ErrNotFound
	} else if err != nil {
		return err
	}

	if _, err := ref.Delete(ctx); err != nil {
		return err
	}
	return nil
}

// IsSymbolUnique checks if symbol is unique, excluding given ID.
func (r *TokenBlueprintRepositoryFS) IsSymbolUnique(ctx context.Context, symbol string, excludeID string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return false, nil
	}

	q := r.col().Where("symbol", "==", symbol)
	it := q.Documents(ctx)
	defer it.Stop()

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return false, err
		}
		if strings.TrimSpace(excludeID) != "" && doc.Ref.ID == strings.TrimSpace(excludeID) {
			continue
		}
		return false, nil
	}
	return true, nil
}

// IsNameUnique checks if name is unique, excluding given ID.
func (r *TokenBlueprintRepositoryFS) IsNameUnique(ctx context.Context, name string, excludeID string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return false, nil
	}

	q := r.col().Where("name", "==", name)
	it := q.Documents(ctx)
	defer it.Stop()

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return false, err
		}
		if strings.TrimSpace(excludeID) != "" && doc.Ref.ID == strings.TrimSpace(excludeID) {
			continue
		}
		return false, nil
	}
	return true, nil
}

// UploadIcon / UploadContentFile remain storage responsibilities.
func (r *TokenBlueprintRepositoryFS) UploadIcon(ctx context.Context, fileName, contentType string, _ io.Reader) (string, error) {
	return "", fmt.Errorf("UploadIcon: not implemented in Firestore repository")
}

func (r *TokenBlueprintRepositoryFS) UploadContentFile(ctx context.Context, fileName, contentType string, _ io.Reader) (string, error) {
	return "", fmt.Errorf("UploadContentFile: not implemented in Firestore repository")
}

// WithTx: Firestore transactions wrapper (simple passthrough).
func (r *TokenBlueprintRepositoryFS) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	// For more complex cases, adapt fn to use RunTransaction.
	return fn(ctx)
}

// Reset: delete all docs (for tests/dev) using transactions instead of deprecated Batch.
func (r *TokenBlueprintRepositoryFS) Reset(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	var snaps []*firestore.DocumentSnapshot
	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		snaps = append(snaps, doc)
	}

	const chunkSize = 400
	for i := 0; i < len(snaps); i += chunkSize {
		end := i + chunkSize
		if end > len(snaps) {
			end = len(snaps)
		}

		if err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			for _, s := range snaps[i:end] {
				if err := tx.Delete(s.Ref); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

// ========================================
// Helpers
// ========================================

func docToTokenBlueprint(doc *firestore.DocumentSnapshot) (tbdom.TokenBlueprint, error) {
	var raw struct {
		Name         string     `firestore:"name"`
		Symbol       string     `firestore:"symbol"`
		BrandID      string     `firestore:"brandId"`
		CompanyID    string     `firestore:"companyId"` // ★ companyId
		Description  string     `firestore:"description"`
		IconID       *string    `firestore:"iconId"`
		ContentFiles []string   `firestore:"contentFiles"`
		AssigneeID   string     `firestore:"assigneeId"`
		Minted       any        `firestore:"minted"` // ★ minted: bool or 旧 string
		CreatedAt    time.Time  `firestore:"createdAt"`
		CreatedBy    string     `firestore:"createdBy"`
		UpdatedAt    time.Time  `firestore:"updatedAt"`
		UpdatedBy    string     `firestore:"updatedBy"`
		DeletedAt    *time.Time `firestore:"deletedAt"`
		DeletedBy    *string    `firestore:"deletedBy"`
		// ★ Arweave metadata URI
		MetadataURI string `firestore:"metadataUri"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return tbdom.TokenBlueprint{}, err
	}

	// minted 互換
	var mintedBool bool
	switch v := raw.Minted.(type) {
	case bool:
		mintedBool = v
	case string:
		switch strings.TrimSpace(v) {
		case "minted":
			mintedBool = true
		default:
			mintedBool = false
		}
	case nil:
		mintedBool = false
	default:
		mintedBool = false
	}

	tb := tbdom.TokenBlueprint{
		ID:           strings.TrimSpace(doc.Ref.ID),
		Name:         strings.TrimSpace(raw.Name),
		Symbol:       strings.TrimSpace(raw.Symbol),
		BrandID:      strings.TrimSpace(raw.BrandID),
		CompanyID:    strings.TrimSpace(raw.CompanyID),
		Description:  strings.TrimSpace(raw.Description),
		ContentFiles: sanitizeStrings(raw.ContentFiles),
		AssigneeID:   strings.TrimSpace(raw.AssigneeID),
		Minted:       mintedBool,
		CreatedAt:    raw.CreatedAt.UTC(),
		CreatedBy:    strings.TrimSpace(raw.CreatedBy),
		UpdatedAt:    raw.UpdatedAt.UTC(),
		UpdatedBy:    strings.TrimSpace(raw.UpdatedBy),
		MetadataURI:  strings.TrimSpace(raw.MetadataURI),
	}

	if raw.IconID != nil {
		if v := strings.TrimSpace(*raw.IconID); v != "" {
			tb.IconID = &v
		}
	}
	if raw.DeletedAt != nil && !raw.DeletedAt.IsZero() {
		t := raw.DeletedAt.UTC()
		tb.DeletedAt = &t
	}
	if raw.DeletedBy != nil {
		if v := strings.TrimSpace(*raw.DeletedBy); v != "" {
			tb.DeletedBy = &v
		}
	}

	return tb, nil
}

// matchTBFilter applies tbdom.Filter in-memory.
func matchTBFilter(tb tbdom.TokenBlueprint, f tbdom.Filter) bool {
	trim := func(s string) string { return strings.TrimSpace(s) }

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

	if len(f.IDs) > 0 && !inList(tb.ID, f.IDs) {
		return false
	}
	if len(f.BrandIDs) > 0 && !inList(tb.BrandID, f.BrandIDs) {
		return false
	}
	if len(f.CompanyIDs) > 0 && !inList(tb.CompanyID, f.CompanyIDs) {
		return false
	}
	if len(f.AssigneeIDs) > 0 && !inList(tb.AssigneeID, f.AssigneeIDs) {
		return false
	}
	if len(f.Symbols) > 0 && !inList(tb.Symbol, f.Symbols) {
		return false
	}

	if v := trim(f.NameLike); v != "" {
		if !strings.Contains(strings.ToLower(tb.Name), strings.ToLower(v)) {
			return false
		}
	}
	if v := trim(f.SymbolLike); v != "" {
		if !strings.Contains(strings.ToLower(tb.Symbol), strings.ToLower(v)) {
			return false
		}
	}

	if f.HasIcon != nil {
		has := tb.IconID != nil && trim(*tb.IconID) != ""
		if *f.HasIcon && !has {
			return false
		}
		if !*f.HasIcon && has {
			return false
		}
	}

	if f.CreatedFrom != nil && tb.CreatedAt.Before(f.CreatedFrom.UTC()) {
		return false
	}
	if f.CreatedTo != nil && !tb.CreatedAt.Before(f.CreatedTo.UTC()) {
		return false
	}
	if f.UpdatedFrom != nil && tb.UpdatedAt.Before(f.UpdatedFrom.UTC()) {
		return false
	}
	if f.UpdatedTo != nil && !tb.UpdatedAt.Before(f.UpdatedTo.UTC()) {
		return false
	}

	return true
}

func sanitizeStrings(xs []string) []string {
	out := make([]string, 0, len(xs))
	seen := make(map[string]struct{}, len(xs))
	for _, v := range xs {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// ========================================
// Patch setters (reflect-based; tolerate Patch schema changes)
// ========================================

func setPatchString(p *tbdom.Patch, names []string, v string) {
	if p == nil {
		return
	}
	rv := reflect.ValueOf(p).Elem()
	for _, name := range names {
		f := rv.FieldByName(name)
		if !f.IsValid() || !f.CanSet() {
			continue
		}
		if f.Kind() == reflect.String {
			f.SetString(v)
			return
		}
	}
}

func setPatchBool(p *tbdom.Patch, names []string, v bool) {
	if p == nil {
		return
	}
	rv := reflect.ValueOf(p).Elem()
	for _, name := range names {
		f := rv.FieldByName(name)
		if !f.IsValid() || !f.CanSet() {
			continue
		}
		if f.Kind() == reflect.Bool {
			f.SetBool(v)
			return
		}
	}
}

func setPatchPtrString(p *tbdom.Patch, names []string, v string) {
	if p == nil {
		return
	}
	rv := reflect.ValueOf(p).Elem()
	for _, name := range names {
		f := rv.FieldByName(name)
		if !f.IsValid() || !f.CanSet() {
			continue
		}
		if f.Kind() == reflect.Pointer && f.Type().Elem().Kind() == reflect.String {
			s := v
			f.Set(reflect.ValueOf(&s))
			return
		}
	}
}
