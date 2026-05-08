// backend/internal/adapters/out/firestore/tokenBlueprint_repository_fs.go
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
	domcommon "narratives/internal/domain/common"
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

func (r *TokenBlueprintRepositoryFS) GetByID(
	ctx context.Context,
	id string,
) (*tbdom.TokenBlueprint, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

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

// GetPatchByID returns a lightweight Patch used by read-models.
func (r *TokenBlueprintRepositoryFS) GetPatchByID(
	ctx context.Context,
	id string,
) (tbdom.Patch, error) {
	if r.Client == nil {
		return tbdom.Patch{}, errors.New("firestore client is nil")
	}

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

	var raw struct {
		Name        string `firestore:"name"`
		Symbol      string `firestore:"symbol"`
		BrandID     string `firestore:"brandId"`
		BrandName   string `firestore:"brandName"`
		CompanyID   string `firestore:"companyId"`
		Description string `firestore:"description"`
		MetadataURI string `firestore:"metadataUri"`
		Minted      bool   `firestore:"minted"`
		IconURL     string `firestore:"iconUrl"`
	}

	if err := snap.DataTo(&raw); err != nil {
		return tbdom.Patch{}, err
	}

	return tbdom.Patch{
		ID:          id,
		TokenName:   raw.Name,
		Symbol:      raw.Symbol,
		BrandID:     raw.BrandID,
		BrandName:   raw.BrandName,
		CompanyID:   raw.CompanyID,
		Description: raw.Description,
		Minted:      raw.Minted,
		MetadataURI: raw.MetadataURI,
		IconURL:     raw.IconURL,
	}, nil
}

// GetNameByID returns only the Name field of a TokenBlueprint.
func (r *TokenBlueprintRepositoryFS) GetNameByID(
	ctx context.Context,
	id string,
) (string, error) {
	if r.Client == nil {
		return "", errors.New("firestore client is nil")
	}

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

	return raw.Name, nil
}

func (r *TokenBlueprintRepositoryFS) List(
	ctx context.Context,
	filter tbdom.Filter,
	page domcommon.Page,
) (domcommon.PageResult[tbdom.TokenBlueprint], error) {
	if r.Client == nil {
		return domcommon.PageResult[tbdom.TokenBlueprint]{}, errors.New("firestore client is nil")
	}

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
			return domcommon.PageResult[tbdom.TokenBlueprint]{}, err
		}

		tb, err := docToTokenBlueprint(doc)
		if err != nil {
			return domcommon.PageResult[tbdom.TokenBlueprint]{}, err
		}

		if matchTBFilter(tb, filter) {
			all = append(all, tb)
		}
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)
	total := len(all)

	if total == 0 {
		return domcommon.PageResult[tbdom.TokenBlueprint]{
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

	return domcommon.PageResult[tbdom.TokenBlueprint]{
		Items:      items,
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *TokenBlueprintRepositoryFS) ListByCompanyID(
	ctx context.Context,
	companyID string,
	page domcommon.Page,
) (domcommon.PageResult[tbdom.TokenBlueprint], error) {
	if r.Client == nil {
		return domcommon.PageResult[tbdom.TokenBlueprint]{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	if companyID == "" {
		return domcommon.PageResult[tbdom.TokenBlueprint]{
			Items:      []tbdom.TokenBlueprint{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	q := r.col().
		Where("companyId", "==", companyID).
		OrderBy("createdAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc).
		Offset(offset).
		Limit(perPage)

	it := q.Documents(ctx)
	defer it.Stop()

	items := make([]tbdom.TokenBlueprint, 0, perPage)

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return domcommon.PageResult[tbdom.TokenBlueprint]{}, err
		}

		tb, err := docToTokenBlueprint(doc)
		if err != nil {
			return domcommon.PageResult[tbdom.TokenBlueprint]{}, err
		}

		if tb.DeletedAt != nil {
			continue
		}

		items = append(items, tb)
	}

	return domcommon.PageResult[tbdom.TokenBlueprint]{
		Items:      items,
		TotalCount: 0,
		TotalPages: 0,
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *TokenBlueprintRepositoryFS) ListByBrandID(
	ctx context.Context,
	brandID string,
	page domcommon.Page,
) (domcommon.PageResult[tbdom.TokenBlueprint], error) {
	if r.Client == nil {
		return domcommon.PageResult[tbdom.TokenBlueprint]{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	if brandID == "" {
		return domcommon.PageResult[tbdom.TokenBlueprint]{
			Items:      []tbdom.TokenBlueprint{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	q := r.col().
		Where("brandId", "==", brandID).
		OrderBy("createdAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc).
		Offset(offset).
		Limit(perPage)

	it := q.Documents(ctx)
	defer it.Stop()

	items := make([]tbdom.TokenBlueprint, 0, perPage)

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return domcommon.PageResult[tbdom.TokenBlueprint]{}, err
		}

		tb, err := docToTokenBlueprint(doc)
		if err != nil {
			return domcommon.PageResult[tbdom.TokenBlueprint]{}, err
		}

		if tb.DeletedAt != nil {
			continue
		}

		items = append(items, tb)
	}

	return domcommon.PageResult[tbdom.TokenBlueprint]{
		Items:      items,
		TotalCount: 0,
		TotalPages: 0,
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *TokenBlueprintRepositoryFS) Create(
	ctx context.Context,
	in tbdom.CreateTokenBlueprintInput,
) (*tbdom.TokenBlueprint, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()

	createdAt := now
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}

	updatedAt := now
	if in.UpdatedAt != nil && !in.UpdatedAt.IsZero() {
		updatedAt = in.UpdatedAt.UTC()
	}

	contentFiles := sanitizeContentFiles(in.ContentFiles)
	minted := false

	docRef := r.col().NewDoc()

	data := map[string]any{
		"name":         in.Name,
		"symbol":       in.Symbol,
		"brandId":      in.BrandID,
		"companyId":    in.CompanyID,
		"description":  in.Description,
		"iconUrl":      strings.TrimSpace(in.IconURL),
		"contentFiles": toFSContentFiles(contentFiles),
		"assigneeId":   in.AssigneeID,
		"minted":       minted,
		"createdAt":    createdAt,
		"updatedAt":    updatedAt,
		"deletedAt":    nil,
		"deletedBy":    nil,
		"metadataUri":  strings.TrimSpace(in.MetadataURI),
	}

	if in.CreatedBy != "" {
		data["createdBy"] = in.CreatedBy
	}

	if in.UpdatedBy != "" {
		data["updatedBy"] = in.UpdatedBy
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

	if id == "" {
		return nil, tbdom.ErrNotFound
	}

	ref := r.col().Doc(id)

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
				Value: *p,
			})
		}
	}

	setStr("name", in.Name)
	setStr("symbol", in.Symbol)
	setStr("iconUrl", in.IconURL)

	if in.BrandID != nil {
		updates = append(updates, firestore.Update{
			Path:  "brandId",
			Value: *in.BrandID,
		})
	}

	if in.Description != nil {
		updates = append(updates, firestore.Update{
			Path:  "description",
			Value: *in.Description,
		})
	}

	if in.AssigneeID != nil {
		updates = append(updates, firestore.Update{
			Path:  "assigneeId",
			Value: *in.AssigneeID,
		})
	}

	if in.MetadataURI != nil {
		updates = append(updates, firestore.Update{
			Path:  "metadataUri",
			Value: *in.MetadataURI,
		})
	}

	if in.Minted != nil {
		updates = append(updates, firestore.Update{
			Path:  "minted",
			Value: *in.Minted,
		})
	}

	if in.ContentFiles != nil {
		files := sanitizeContentFiles(*in.ContentFiles)
		updates = append(updates, firestore.Update{
			Path:  "contentFiles",
			Value: toFSContentFiles(files),
		})
	}

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

	if in.UpdatedBy != nil {
		v := *in.UpdatedBy
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
		v := *in.DeletedBy
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

func (r *TokenBlueprintRepositoryFS) Delete(
	ctx context.Context,
	id string,
) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

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

func (r *TokenBlueprintRepositoryFS) IsSymbolUnique(
	ctx context.Context,
	symbol string,
	excludeID string,
) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

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
		if excludeID != "" && doc.Ref.ID == excludeID {
			continue
		}
		return false, nil
	}

	return true, nil
}

func (r *TokenBlueprintRepositoryFS) IsNameUnique(
	ctx context.Context,
	name string,
	excludeID string,
) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

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
		if excludeID != "" && doc.Ref.ID == excludeID {
			continue
		}
		return false, nil
	}

	return true, nil
}

// ========================================
// Helpers
// ========================================

func docToTokenBlueprint(
	doc *firestore.DocumentSnapshot,
) (tbdom.TokenBlueprint, error) {
	var raw struct {
		Name         string           `firestore:"name"`
		Symbol       string           `firestore:"symbol"`
		BrandID      string           `firestore:"brandId"`
		CompanyID    string           `firestore:"companyId"`
		Description  string           `firestore:"description"`
		IconURL      string           `firestore:"iconUrl"`
		ContentFiles []map[string]any `firestore:"contentFiles"`
		AssigneeID   string           `firestore:"assigneeId"`
		Minted       bool             `firestore:"minted"`
		CreatedAt    time.Time        `firestore:"createdAt"`
		CreatedBy    string           `firestore:"createdBy"`
		UpdatedAt    time.Time        `firestore:"updatedAt"`
		UpdatedBy    string           `firestore:"updatedBy"`
		DeletedAt    *time.Time       `firestore:"deletedAt"`
		DeletedBy    *string          `firestore:"deletedBy"`
		MetadataURI  string           `firestore:"metadataUri"`
	}

	if err := doc.DataTo(&raw); err != nil {
		return tbdom.TokenBlueprint{}, err
	}

	files, err := fromFSContentFiles(raw.ContentFiles)
	if err != nil {
		return tbdom.TokenBlueprint{}, err
	}

	tb := tbdom.TokenBlueprint{
		ID:           doc.Ref.ID,
		Name:         raw.Name,
		Symbol:       raw.Symbol,
		BrandID:      raw.BrandID,
		CompanyID:    raw.CompanyID,
		Description:  raw.Description,
		IconURL:      raw.IconURL,
		ContentFiles: files,
		AssigneeID:   raw.AssigneeID,
		Minted:       raw.Minted,
		CreatedAt:    raw.CreatedAt.UTC(),
		CreatedBy:    raw.CreatedBy,
		UpdatedAt:    raw.UpdatedAt.UTC(),
		UpdatedBy:    raw.UpdatedBy,
		MetadataURI:  raw.MetadataURI,
	}

	if raw.DeletedAt != nil && !raw.DeletedAt.IsZero() {
		t := raw.DeletedAt.UTC()
		tb.DeletedAt = &t
	}

	if raw.DeletedBy != nil {
		if v := *raw.DeletedBy; v != "" {
			tb.DeletedBy = &v
		}
	}

	return tb, nil
}

func matchTBFilter(tb tbdom.TokenBlueprint, f tbdom.Filter) bool {
	inList := func(v string, xs []string) bool {
		if len(xs) == 0 {
			return true
		}

		for _, x := range xs {
			if x == v {
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

	if len(f.Symbols) > 0 {
		match := false
		for _, x := range f.Symbols {
			if x == tb.Symbol {
				match = true
				break
			}
		}

		if !match {
			return false
		}
	}

	if f.NameLike != "" {
		if !strings.Contains(strings.ToLower(tb.Name), strings.ToLower(f.NameLike)) {
			return false
		}
	}

	if f.SymbolLike != "" {
		if !strings.Contains(strings.ToLower(tb.Symbol), strings.ToLower(f.SymbolLike)) {
			return false
		}
	}

	if f.Created.From != nil && tb.CreatedAt.Before(f.Created.From.UTC()) {
		return false
	}

	if f.Created.To != nil && !tb.CreatedAt.Before(f.Created.To.UTC()) {
		return false
	}

	if f.Updated.From != nil && tb.UpdatedAt.Before(f.Updated.From.UTC()) {
		return false
	}

	if f.Updated.To != nil && !tb.UpdatedAt.Before(f.Updated.To.UTC()) {
		return false
	}

	return true
}

func sanitizeContentFiles(xs []tbdom.ContentFile) []tbdom.ContentFile {
	out := make([]tbdom.ContentFile, 0, len(xs))
	seen := make(map[string]struct{}, len(xs))

	for _, f := range xs {
		if f.ID == "" {
			continue
		}

		if _, ok := seen[f.ID]; ok {
			continue
		}

		seen[f.ID] = struct{}{}
		out = append(out, f)
	}

	return out
}

func toFSContentFiles(xs []tbdom.ContentFile) []map[string]any {
	out := make([]map[string]any, 0, len(xs))

	for _, f := range xs {
		m := map[string]any{
			"id":          f.ID,
			"name":        f.Name,
			"type":        string(f.Type),
			"contentType": f.ContentType,
			"size":        f.Size,
			"objectPath":  f.ObjectPath,
			"url":         f.URL,
			"visibility":  string(f.Visibility),
			"createdAt":   f.CreatedAt,
			"createdBy":   f.CreatedBy,
			"updatedAt":   f.UpdatedAt,
			"updatedBy":   f.UpdatedBy,
		}

		out = append(out, m)
	}

	return out
}

func fromFSContentFiles(xs []map[string]any) ([]tbdom.ContentFile, error) {
	out := make([]tbdom.ContentFile, 0, len(xs))

	for _, m := range xs {
		var f tbdom.ContentFile

		if v, ok := m["id"].(string); ok {
			f.ID = v
		}

		if v, ok := m["name"].(string); ok {
			f.Name = v
		}

		if v, ok := m["type"].(string); ok {
			f.Type = tbdom.ContentFileType(v)
		}

		if v, ok := m["contentType"].(string); ok {
			f.ContentType = v
		}

		if v, ok := m["objectPath"].(string); ok {
			f.ObjectPath = v
		}

		if v, ok := m["url"].(string); ok {
			f.URL = v
		}

		if v, ok := m["visibility"].(string); ok {
			f.Visibility = tbdom.ContentVisibility(v)
		}

		switch v := m["size"].(type) {
		case int64:
			f.Size = v
		case int:
			f.Size = int64(v)
		case int32:
			f.Size = int64(v)
		case float64:
			f.Size = int64(v)
		case nil:
			f.Size = 0
		default:
			return nil, fmt.Errorf("contentFiles.size: unexpected type %T", m["size"])
		}

		if v, ok := m["createdBy"].(string); ok {
			f.CreatedBy = v
		}

		if v, ok := m["updatedBy"].(string); ok {
			f.UpdatedBy = v
		}

		if v, ok := m["createdAt"].(time.Time); ok {
			f.CreatedAt = v.UTC()
		}

		if v, ok := m["updatedAt"].(time.Time); ok {
			f.UpdatedAt = v.UTC()
		}

		if string(f.Visibility) == "" {
			f.Visibility = tbdom.VisibilityPrivate
		}

		if err := f.Validate(); err != nil {
			return nil, err
		}

		out = append(out, f)
	}

	return sanitizeContentFiles(out), nil
}
