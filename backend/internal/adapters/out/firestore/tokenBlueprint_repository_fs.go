// backend/internal/adapters/out/firestore/tokenBlueprint_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"math"
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

	if err := tbdom.ValidateContentFiles(in.ContentFiles); err != nil {
		return nil, err
	}

	docRef := r.col().NewDoc()

	data := map[string]any{
		"name":            in.Name,
		"symbol":          in.Symbol,
		"brandId":         in.BrandID,
		"companyId":       in.CompanyID,
		"description":     in.Description,
		"iconUrl":         in.IconURL,
		"iconObjectPath":  in.IconObjectPath,
		"iconFileName":    in.IconFileName,
		"iconContentType": in.IconContentType,
		"iconSize":        in.IconSize,
		"contentFiles":    toFSContentFiles(in.ContentFiles),
		"assigneeId":      in.AssigneeID,
		"minted":          false,
		"createdAt":       createdAt,
		"createdBy":       in.CreatedBy,
		"updatedAt":       updatedAt,
		"updatedBy":       in.UpdatedBy,
		"metadataUri":     in.MetadataURI,
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

	setInt64 := func(field string, p *int64) {
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
	setStr("iconObjectPath", in.IconObjectPath)
	setStr("iconFileName", in.IconFileName)
	setStr("iconContentType", in.IconContentType)
	setInt64("iconSize", in.IconSize)

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
		if err := tbdom.ValidateContentFiles(*in.ContentFiles); err != nil {
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
		Name            string           `firestore:"name"`
		Symbol          string           `firestore:"symbol"`
		BrandID         string           `firestore:"brandId"`
		CompanyID       string           `firestore:"companyId"`
		Description     string           `firestore:"description"`
		IconURL         string           `firestore:"iconUrl"`
		IconObjectPath  string           `firestore:"iconObjectPath"`
		IconFileName    string           `firestore:"iconFileName"`
		IconContentType string           `firestore:"iconContentType"`
		IconSize        int64            `firestore:"iconSize"`
		ContentFiles    []map[string]any `firestore:"contentFiles"`
		AssigneeID      string           `firestore:"assigneeId"`
		Minted          bool             `firestore:"minted"`
		CreatedAt       time.Time        `firestore:"createdAt"`
		CreatedBy       string           `firestore:"createdBy"`
		UpdatedAt       time.Time        `firestore:"updatedAt"`
		UpdatedBy       string           `firestore:"updatedBy"`
		MetadataURI     string           `firestore:"metadataUri"`
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
		ID:          doc.Ref.ID,
		Name:        raw.Name,
		Symbol:      raw.Symbol,
		BrandID:     raw.BrandID,
		CompanyID:   raw.CompanyID,
		Description: raw.Description,

		IconURL:         raw.IconURL,
		IconObjectPath:  raw.IconObjectPath,
		IconFileName:    raw.IconFileName,
		IconContentType: raw.IconContentType,
		IconSize:        raw.IconSize,

		ContentFiles: files,
		AssigneeID:   raw.AssigneeID,
		Minted:       raw.Minted,
		CreatedAt:    raw.CreatedAt.UTC(),
		CreatedBy:    raw.CreatedBy,
		UpdatedAt:    raw.UpdatedAt.UTC(),
		UpdatedBy:    raw.UpdatedBy,
		MetadataURI:  raw.MetadataURI,
	}

	return tb, nil
}

func toFSContentFiles(xs []tbdom.ContentFile) []map[string]any {
	out := make([]map[string]any, 0, len(xs))

	for _, f := range xs {
		m := map[string]any{
			"id":          f.ID,
			"name":        f.Name,
			"type":        string(f.Type),
			"contentType": f.ContentType,
			"url":         f.URL,
			"objectPath":  f.ObjectPath,
			"visibility":  string(f.Visibility),
			"size":        f.Size,
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

	for i, m := range xs {
		var f tbdom.ContentFile

		v, ok := m["id"].(string)
		if !ok || v == "" {
			return nil, fmt.Errorf("%w: contentFiles[%d].id", tbdom.ErrInvalidContentFile, i)
		}
		f.ID = v

		v, ok = m["name"].(string)
		if !ok || v == "" {
			return nil, fmt.Errorf("%w: contentFiles[%d].name", tbdom.ErrInvalidContentFile, i)
		}
		f.Name = v

		tv, ok := m["type"].(string)
		if !ok || tv == "" {
			return nil, fmt.Errorf("%w: contentFiles[%d].type", tbdom.ErrInvalidContentType, i)
		}
		f.Type = tbdom.ContentFileType(tv)

		if v, ok := m["contentType"].(string); ok {
			f.ContentType = v
		}

		v, ok = m["url"].(string)
		if !ok || v == "" {
			return nil, fmt.Errorf("%w: contentFiles[%d].url", tbdom.ErrInvalidContentFile, i)
		}
		f.URL = v

		v, ok = m["objectPath"].(string)
		if !ok || v == "" {
			return nil, fmt.Errorf("%w: contentFiles[%d].objectPath", tbdom.ErrInvalidContentFile, i)
		}
		f.ObjectPath = v

		vv, ok := m["visibility"].(string)
		if !ok || vv == "" {
			return nil, fmt.Errorf("%w: contentFiles[%d].visibility", tbdom.ErrInvalidContentVisibility, i)
		}
		f.Visibility = tbdom.ContentVisibility(vv)

		size, err := int64FromAny(m["size"])
		if err != nil {
			return nil, fmt.Errorf("%w: contentFiles[%d].size: %v", tbdom.ErrInvalidContentFile, i, err)
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

		out = append(out, f)
	}

	if err := tbdom.ValidateContentFiles(out); err != nil {
		return nil, err
	}

	return out, nil
}

func int64FromAny(v any) (int64, error) {
	switch x := v.(type) {
	case nil:
		return 0, nil
	case int:
		if x < 0 {
			return 0, fmt.Errorf("negative size")
		}
		return int64(x), nil
	case int8:
		if x < 0 {
			return 0, fmt.Errorf("negative size")
		}
		return int64(x), nil
	case int16:
		if x < 0 {
			return 0, fmt.Errorf("negative size")
		}
		return int64(x), nil
	case int32:
		if x < 0 {
			return 0, fmt.Errorf("negative size")
		}
		return int64(x), nil
	case int64:
		if x < 0 {
			return 0, fmt.Errorf("negative size")
		}
		return x, nil
	case uint:
		if uint64(x) > math.MaxInt64 {
			return 0, fmt.Errorf("size overflows int64")
		}
		return int64(x), nil
	case uint8:
		return int64(x), nil
	case uint16:
		return int64(x), nil
	case uint32:
		return int64(x), nil
	case uint64:
		if x > math.MaxInt64 {
			return 0, fmt.Errorf("size overflows int64")
		}
		return int64(x), nil
	case float32:
		if x < 0 {
			return 0, fmt.Errorf("negative size")
		}
		if math.Trunc(float64(x)) != float64(x) {
			return 0, fmt.Errorf("size must be integer")
		}
		if float64(x) > math.MaxInt64 {
			return 0, fmt.Errorf("size overflows int64")
		}
		return int64(x), nil
	case float64:
		if x < 0 {
			return 0, fmt.Errorf("negative size")
		}
		if math.Trunc(x) != x {
			return 0, fmt.Errorf("size must be integer")
		}
		if x > math.MaxInt64 {
			return 0, fmt.Errorf("size overflows int64")
		}
		return int64(x), nil
	default:
		return 0, fmt.Errorf("unsupported size type %T", v)
	}
}
