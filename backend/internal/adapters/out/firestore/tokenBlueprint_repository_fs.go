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
		return nil, tbdom.ErrInvalidID
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
		return tbdom.Patch{}, tbdom.ErrInvalidID
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
		return "", tbdom.ErrInvalidID
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

	if in.CreatedBy == "" {
		return nil, tbdom.ErrInvalidCreatedBy
	}
	if in.UpdatedBy == "" {
		return nil, tbdom.ErrInvalidUpdatedBy
	}

	now := time.Now().UTC()

	createdAt := now
	if in.CreatedAt != nil {
		if in.CreatedAt.IsZero() {
			return nil, tbdom.ErrInvalidCreatedAt
		}
		createdAt = in.CreatedAt.UTC()
	}

	updatedAt := now
	if in.UpdatedAt != nil {
		if in.UpdatedAt.IsZero() {
			return nil, tbdom.ErrInvalidUpdatedAt
		}
		updatedAt = in.UpdatedAt.UTC()
	}

	if err := validateContentFilesForFS(in.ContentFiles); err != nil {
		return nil, err
	}

	docRef := r.col().NewDoc()

	data := map[string]any{
		"name":         in.Name,
		"symbol":       in.Symbol,
		"brandId":      in.BrandID,
		"companyId":    in.CompanyID,
		"description":  in.Description,
		"iconUrl":      in.IconURL,
		"contentFiles": toFSContentFiles(in.ContentFiles),
		"assigneeId":   in.AssigneeID,
		"minted":       false,
		"createdAt":    createdAt,
		"createdBy":    in.CreatedBy,
		"updatedAt":    updatedAt,
		"updatedBy":    in.UpdatedBy,
		"deletedAt":    nil,
		"deletedBy":    nil,
		"metadataUri":  in.MetadataURI,
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
		return nil, tbdom.ErrInvalidID
	}

	if in.UpdatedAt == nil || in.UpdatedAt.IsZero() {
		return nil, tbdom.ErrInvalidUpdatedAt
	}

	if in.UpdatedBy == nil || *in.UpdatedBy == "" {
		return nil, tbdom.ErrInvalidUpdatedBy
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
		if err := validateContentFilesForFS(*in.ContentFiles); err != nil {
			return nil, err
		}

		updates = append(updates, firestore.Update{
			Path:  "contentFiles",
			Value: toFSContentFiles(*in.ContentFiles),
		})
	}

	updates = append(updates, firestore.Update{
		Path:  "updatedAt",
		Value: in.UpdatedAt.UTC(),
	})
	updates = append(updates, firestore.Update{
		Path:  "updatedBy",
		Value: *in.UpdatedBy,
	})

	if in.DeletedAt != nil {
		if in.DeletedAt.IsZero() {
			return nil, tbdom.ErrInvalid
		}

		updates = append(updates, firestore.Update{
			Path:  "deletedAt",
			Value: in.DeletedAt.UTC(),
		})
	}

	if in.DeletedBy != nil {
		if *in.DeletedBy == "" {
			return nil, tbdom.ErrInvalidDeletedBy
		}

		updates = append(updates, firestore.Update{
			Path:  "deletedBy",
			Value: *in.DeletedBy,
		})
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
		return tbdom.ErrInvalidID
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
		return false, tbdom.ErrInvalidSymbol
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
		return false, tbdom.ErrInvalidName
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

	if raw.CreatedAt.IsZero() {
		return tbdom.TokenBlueprint{}, tbdom.ErrInvalidCreatedAt
	}
	if raw.CreatedBy == "" {
		return tbdom.TokenBlueprint{}, tbdom.ErrInvalidCreatedBy
	}
	if raw.UpdatedAt.IsZero() {
		return tbdom.TokenBlueprint{}, tbdom.ErrInvalidUpdatedAt
	}
	if raw.UpdatedBy == "" {
		return tbdom.TokenBlueprint{}, tbdom.ErrInvalidUpdatedBy
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

	if raw.DeletedAt != nil {
		if raw.DeletedAt.IsZero() {
			return tbdom.TokenBlueprint{}, tbdom.ErrInvalid
		}

		t := raw.DeletedAt.UTC()
		tb.DeletedAt = &t
	}

	if raw.DeletedBy != nil {
		if *raw.DeletedBy == "" {
			return tbdom.TokenBlueprint{}, tbdom.ErrInvalidDeletedBy
		}

		v := *raw.DeletedBy
		tb.DeletedBy = &v
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

func validateContentFilesForFS(xs []tbdom.ContentFile) error {
	seen := make(map[string]struct{}, len(xs))

	for i, f := range xs {
		if err := f.Validate(); err != nil {
			return err
		}

		if _, ok := seen[f.ID]; ok {
			return tbdom.WrapConflict(
				nil,
				fmt.Sprintf("contentFiles[%d].id duplicated: %s", i, f.ID),
			)
		}

		seen[f.ID] = struct{}{}
	}

	return nil
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
	seen := make(map[string]struct{}, len(xs))

	for i, m := range xs {
		var f tbdom.ContentFile

		v, ok := m["id"].(string)
		if !ok {
			return nil, fmt.Errorf("%w: contentFiles[%d].id", tbdom.ErrInvalidContentFile, i)
		}
		f.ID = v

		v, ok = m["name"].(string)
		if !ok {
			return nil, fmt.Errorf("%w: contentFiles[%d].name", tbdom.ErrInvalidContentFile, i)
		}
		f.Name = v

		tv, ok := m["type"].(string)
		if !ok {
			return nil, fmt.Errorf("%w: contentFiles[%d].type", tbdom.ErrInvalidContentType, i)
		}
		f.Type = tbdom.ContentFileType(tv)

		if v, ok := m["contentType"].(string); ok {
			f.ContentType = v
		}

		v, ok = m["objectPath"].(string)
		if !ok {
			return nil, fmt.Errorf("%w: contentFiles[%d].objectPath", tbdom.ErrInvalidContentFile, i)
		}
		f.ObjectPath = v

		if v, ok := m["url"].(string); ok {
			f.URL = v
		}

		vv, ok := m["visibility"].(string)
		if !ok {
			return nil, fmt.Errorf("%w: contentFiles[%d].visibility", tbdom.ErrInvalidContentVisibility, i)
		}
		f.Visibility = tbdom.ContentVisibility(vv)

		size, ok := m["size"].(int64)
		if !ok {
			return nil, fmt.Errorf("contentFiles[%d].size must be int64: got %T", i, m["size"])
		}
		f.Size = size

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

		if err := f.Validate(); err != nil {
			return nil, err
		}

		if _, ok := seen[f.ID]; ok {
			return nil, tbdom.WrapConflict(
				nil,
				fmt.Sprintf("contentFiles[%d].id duplicated: %s", i, f.ID),
			)
		}

		seen[f.ID] = struct{}{}
		out = append(out, f)
	}

	return out, nil
}
