// backend/internal/adapters/out/firestore/tokenBlueprint_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
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

func (r *TokenBlueprintRepositoryFS) GetByID(ctx context.Context, id string) (*tbdom.TokenBlueprint, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if id == "" {
		return nil, tbdom.ErrInvalidID
	}

	snap, err := r.col().Doc(id).Get(ctx)
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

func (r *TokenBlueprintRepositoryFS) ListByCompanyID(ctx context.Context, companyID string, page domcommon.Page) (domcommon.PageResult[tbdom.TokenBlueprint], error) {
	if r == nil || r.Client == nil {
		return domcommon.PageResult[tbdom.TokenBlueprint]{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)
	if companyID == "" {
		return domcommon.PageResult[tbdom.TokenBlueprint]{
			Items: []tbdom.TokenBlueprint{}, TotalCount: 0, TotalPages: 0, Page: pageNum, PerPage: perPage,
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
		Items: items, TotalCount: 0, TotalPages: 0, Page: pageNum, PerPage: perPage,
	}, nil
}

func (r *TokenBlueprintRepositoryFS) ListByBrandID(ctx context.Context, brandID string, page domcommon.Page) (domcommon.PageResult[tbdom.TokenBlueprint], error) {
	if r == nil || r.Client == nil {
		return domcommon.PageResult[tbdom.TokenBlueprint]{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)
	if brandID == "" {
		return domcommon.PageResult[tbdom.TokenBlueprint]{
			Items: []tbdom.TokenBlueprint{}, TotalCount: 0, TotalPages: 0, Page: pageNum, PerPage: perPage,
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
		Items: items, TotalCount: 0, TotalPages: 0, Page: pageNum, PerPage: perPage,
	}, nil
}

func (r *TokenBlueprintRepositoryFS) Create(ctx context.Context, in tbdom.CreateTokenBlueprintInput) (*tbdom.TokenBlueprint, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if in.CreatedBy == "" {
		return nil, tbdom.ErrInvalidCreatedBy
	}
	if in.UpdatedBy == "" {
		return nil, tbdom.ErrInvalidUpdatedBy
	}
	if err := tbdom.ValidateContentFiles(in.ContentFiles); err != nil {
		return nil, err
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

	docRef := r.col().NewDoc()
	candidate := tbdom.TokenBlueprint{
		ID:               docRef.ID,
		Name:             in.Name,
		Symbol:           in.Symbol,
		BrandID:          in.BrandID,
		CompanyID:        in.CompanyID,
		Description:      in.Description,
		IconURL:          in.IconURL,
		IconObjectPath:   in.IconObjectPath,
		IconFileName:     in.IconFileName,
		IconContentType:  in.IconContentType,
		IconSize:         in.IconSize,
		ContentFiles:     in.ContentFiles,
		AssigneeID:       in.AssigneeID,
		Minted:           false,
		ModerationStatus: tbdom.ModerationStatusActive,
		CreatedAt:        createdAt,
		CreatedBy:        in.CreatedBy,
		UpdatedAt:        updatedAt,
		UpdatedBy:        in.UpdatedBy,
		MetadataURI:      in.MetadataURI,
	}
	if err := validatePersistedTokenBlueprint(candidate); err != nil {
		return nil, err
	}

	data := map[string]any{
		"name":             in.Name,
		"symbol":           in.Symbol,
		"brandId":          in.BrandID,
		"companyId":        in.CompanyID,
		"description":      in.Description,
		"iconUrl":          in.IconURL,
		"iconObjectPath":   in.IconObjectPath,
		"iconFileName":     in.IconFileName,
		"iconContentType":  in.IconContentType,
		"iconSize":         in.IconSize,
		"contentFiles":     toFSContentFiles(in.ContentFiles),
		"assigneeId":       in.AssigneeID,
		"minted":           false,
		"moderationStatus": string(tbdom.ModerationStatusActive),
		"createdAt":        createdAt,
		"createdBy":        in.CreatedBy,
		"updatedAt":        updatedAt,
		"updatedBy":        in.UpdatedBy,
		"metadataUri":      in.MetadataURI,
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

func (r *TokenBlueprintRepositoryFS) Update(ctx context.Context, id string, in tbdom.UpdateTokenBlueprintInput) (*tbdom.TokenBlueprint, error) {
	if r == nil || r.Client == nil {
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
	snap, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, tbdom.ErrNotFound
		}
		return nil, err
	}

	current, err := docToTokenBlueprint(snap)
	if err != nil {
		return nil, err
	}

	candidate := current
	if in.Name != nil {
		candidate.Name = *in.Name
	}
	if in.Symbol != nil {
		candidate.Symbol = *in.Symbol
	}
	if in.BrandID != nil {
		candidate.BrandID = *in.BrandID
	}
	if in.Description != nil {
		candidate.Description = *in.Description
	}
	if in.IconURL != nil {
		candidate.IconURL = *in.IconURL
	}
	if in.IconObjectPath != nil {
		candidate.IconObjectPath = *in.IconObjectPath
	}
	if in.IconFileName != nil {
		candidate.IconFileName = *in.IconFileName
	}
	if in.IconContentType != nil {
		candidate.IconContentType = *in.IconContentType
	}
	if in.IconSize != nil {
		candidate.IconSize = *in.IconSize
	}
	if in.ContentFiles != nil {
		candidate.ContentFiles = *in.ContentFiles
	}
	if in.AssigneeID != nil {
		candidate.AssigneeID = *in.AssigneeID
	}
	if in.Minted != nil {
		candidate.Minted = *in.Minted
	}
	if in.MetadataURI != nil {
		candidate.MetadataURI = *in.MetadataURI
	}

	candidate.UpdatedAt = *in.UpdatedAt
	candidate.UpdatedBy = *in.UpdatedBy
	if err := validatePersistedTokenBlueprint(candidate); err != nil {
		return nil, err
	}

	updates := make([]firestore.Update, 0, 16)
	setString := func(field string, value *string) {
		if value != nil {
			updates = append(updates, firestore.Update{Path: field, Value: *value})
		}
	}
	setInt64 := func(field string, value *int64) {
		if value != nil {
			updates = append(updates, firestore.Update{Path: field, Value: *value})
		}
	}

	setString("name", in.Name)
	setString("symbol", in.Symbol)
	setString("brandId", in.BrandID)
	setString("description", in.Description)
	setString("iconUrl", in.IconURL)
	setString("iconObjectPath", in.IconObjectPath)
	setString("iconFileName", in.IconFileName)
	setString("iconContentType", in.IconContentType)
	setInt64("iconSize", in.IconSize)
	setString("assigneeId", in.AssigneeID)
	setString("metadataUri", in.MetadataURI)

	if in.Minted != nil {
		updates = append(updates, firestore.Update{Path: "minted", Value: *in.Minted})
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

	updates = append(
		updates,
		firestore.Update{Path: "updatedAt", Value: in.UpdatedAt.UTC()},
		firestore.Update{Path: "updatedBy", Value: *in.UpdatedBy},
	)

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, tbdom.ErrNotFound
		}
		return nil, err
	}

	snap, err = ref.Get(ctx)
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

// UpdateModerationStatus updates only AMOL-side moderation state.
// TokenBlueprint document, Firebase Storage assets, metadataUri and on-chain state
// are intentionally preserved.
func (r *TokenBlueprintRepositoryFS) UpdateModerationStatus(
	ctx context.Context,
	id string,
	moderationStatus tbdom.ModerationStatus,
	updatedBy string,
	updatedAt time.Time,
) (*tbdom.TokenBlueprint, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if id == "" {
		return nil, tbdom.ErrInvalidID
	}
	if !tbdom.IsValidModerationStatus(moderationStatus) {
		return nil, tbdom.ErrInvalidModerationStatus
	}
	if updatedBy == "" {
		return nil, tbdom.ErrInvalidUpdatedBy
	}
	if updatedAt.IsZero() {
		return nil, tbdom.ErrInvalidUpdatedAt
	}

	current, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := current.SetModerationStatus(
		moderationStatus,
		updatedBy,
		updatedAt,
	); err != nil {
		return nil, err
	}

	if err := validatePersistedTokenBlueprint(*current); err != nil {
		return nil, err
	}

	ref := r.col().Doc(id)
	if _, err := ref.Update(ctx, []firestore.Update{
		{
			Path:  "moderationStatus",
			Value: string(current.EffectiveModerationStatus()),
		},
		{
			Path:  "updatedAt",
			Value: current.UpdatedAt,
		},
		{
			Path:  "updatedBy",
			Value: current.UpdatedBy,
		},
	}); err != nil {
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
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}
	if id == "" {
		return tbdom.ErrInvalidID
	}

	ref := r.col().Doc(id)
	if _, err := ref.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return tbdom.ErrNotFound
		}
		return err
	}

	_, err := ref.Delete(ctx)
	return err
}

func (r *TokenBlueprintRepositoryFS) IsSymbolUnique(ctx context.Context, symbol string, excludeID string) (bool, error) {
	if r == nil || r.Client == nil {
		return false, errors.New("firestore client is nil")
	}
	if symbol == "" {
		return false, tbdom.ErrInvalidSymbol
	}

	it := r.col().Where("symbol", "==", symbol).Documents(ctx)
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

func (r *TokenBlueprintRepositoryFS) IsNameUnique(ctx context.Context, name string, excludeID string) (bool, error) {
	if r == nil || r.Client == nil {
		return false, errors.New("firestore client is nil")
	}
	if name == "" {
		return false, tbdom.ErrInvalidName
	}

	it := r.col().Where("name", "==", name).Documents(ctx)
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
// Firestore DTO / mapping
// ========================================

type tokenBlueprintRepositoryDoc struct {
	Name             string                                   `firestore:"name"`
	Symbol           string                                   `firestore:"symbol"`
	BrandID          string                                   `firestore:"brandId"`
	CompanyID        string                                   `firestore:"companyId"`
	Description      string                                   `firestore:"description"`
	IconURL          string                                   `firestore:"iconUrl"`
	IconObjectPath   string                                   `firestore:"iconObjectPath"`
	IconFileName     string                                   `firestore:"iconFileName"`
	IconContentType  string                                   `firestore:"iconContentType"`
	IconSize         *int64                                   `firestore:"iconSize"`
	ContentFiles     []tokenBlueprintRepositoryContentFileDoc `firestore:"contentFiles"`
	AssigneeID       string                                   `firestore:"assigneeId"`
	Minted           *bool                                    `firestore:"minted"`
	ModerationStatus string                                   `firestore:"moderationStatus"`
	CreatedAt        time.Time                                `firestore:"createdAt"`
	CreatedBy        string                                   `firestore:"createdBy"`
	UpdatedAt        time.Time                                `firestore:"updatedAt"`
	UpdatedBy        string                                   `firestore:"updatedBy"`
	MetadataURI      string                                   `firestore:"metadataUri"`
}

type tokenBlueprintRepositoryContentFileDoc struct {
	ID          string    `firestore:"id"`
	Name        string    `firestore:"name"`
	Type        string    `firestore:"type"`
	ContentType string    `firestore:"contentType"`
	URL         string    `firestore:"url"`
	ObjectPath  string    `firestore:"objectPath"`
	IsPublic    *bool     `firestore:"isPublic"`
	Size        *int64    `firestore:"size"`
	CreatedAt   time.Time `firestore:"createdAt"`
	CreatedBy   string    `firestore:"createdBy"`
	UpdatedAt   time.Time `firestore:"updatedAt"`
	UpdatedBy   string    `firestore:"updatedBy"`
}

func docToTokenBlueprint(doc *firestore.DocumentSnapshot) (tbdom.TokenBlueprint, error) {
	if doc == nil || doc.Ref == nil || doc.Ref.ID == "" {
		return tbdom.TokenBlueprint{}, tbdom.ErrInvalidID
	}

	var raw tokenBlueprintRepositoryDoc
	if err := doc.DataTo(&raw); err != nil {
		return tbdom.TokenBlueprint{}, fmt.Errorf(
			"decode token_blueprints document %q: %w",
			doc.Ref.ID,
			err,
		)
	}
	if raw.IconSize == nil {
		return tbdom.TokenBlueprint{}, fmt.Errorf(
			"invalid token_blueprints document %q: iconSize is missing",
			doc.Ref.ID,
		)
	}
	if raw.Minted == nil {
		return tbdom.TokenBlueprint{}, fmt.Errorf(
			"invalid token_blueprints document %q: minted is missing",
			doc.Ref.ID,
		)
	}

	files, err := fromFSContentFiles(raw.ContentFiles)
	if err != nil {
		return tbdom.TokenBlueprint{}, fmt.Errorf(
			"invalid token_blueprints document %q: %w",
			doc.Ref.ID,
			err,
		)
	}

	tb := tbdom.TokenBlueprint{
		ID:              doc.Ref.ID,
		Name:            raw.Name,
		Symbol:          raw.Symbol,
		BrandID:         raw.BrandID,
		CompanyID:       raw.CompanyID,
		Description:     raw.Description,
		IconURL:         raw.IconURL,
		IconObjectPath:  raw.IconObjectPath,
		IconFileName:    raw.IconFileName,
		IconContentType: raw.IconContentType,
		IconSize:        *raw.IconSize,
		ContentFiles:    files,
		AssigneeID:      raw.AssigneeID,
		Minted:          *raw.Minted,
		ModerationStatus: tbdom.NormalizeModerationStatus(
			tbdom.ModerationStatus(raw.ModerationStatus),
		),
		CreatedAt:   raw.CreatedAt,
		CreatedBy:   raw.CreatedBy,
		UpdatedAt:   raw.UpdatedAt,
		UpdatedBy:   raw.UpdatedBy,
		MetadataURI: raw.MetadataURI,
	}

	if err := validatePersistedTokenBlueprint(tb); err != nil {
		return tbdom.TokenBlueprint{}, fmt.Errorf(
			"invalid token_blueprints document %q: %w",
			doc.Ref.ID,
			err,
		)
	}
	return tb, nil
}

func toFSContentFiles(xs []tbdom.ContentFile) []map[string]any {
	out := make([]map[string]any, 0, len(xs))
	for _, f := range xs {
		out = append(out, map[string]any{
			"id":          f.ID,
			"name":        f.Name,
			"type":        string(f.Type),
			"contentType": f.ContentType,
			"url":         f.URL,
			"objectPath":  f.ObjectPath,
			"isPublic":    f.IsPublic,
			"size":        f.Size,
			"createdAt":   f.CreatedAt,
			"createdBy":   f.CreatedBy,
			"updatedAt":   f.UpdatedAt,
			"updatedBy":   f.UpdatedBy,
		})
	}
	return out
}

func fromFSContentFiles(xs []tokenBlueprintRepositoryContentFileDoc) ([]tbdom.ContentFile, error) {
	out := make([]tbdom.ContentFile, 0, len(xs))
	for i, raw := range xs {
		if raw.IsPublic == nil {
			return nil, fmt.Errorf(
				"%w: contentFiles[%d].isPublic is missing",
				tbdom.ErrInvalidContentFile,
				i,
			)
		}
		if raw.Size == nil {
			return nil, fmt.Errorf(
				"%w: contentFiles[%d].size is missing",
				tbdom.ErrInvalidContentFile,
				i,
			)
		}

		out = append(out, tbdom.ContentFile{
			ID:          raw.ID,
			Name:        raw.Name,
			Type:        tbdom.ContentFileType(raw.Type),
			ContentType: raw.ContentType,
			URL:         raw.URL,
			ObjectPath:  raw.ObjectPath,
			IsPublic:    *raw.IsPublic,
			Size:        *raw.Size,
			CreatedAt:   raw.CreatedAt,
			CreatedBy:   raw.CreatedBy,
			UpdatedAt:   raw.UpdatedAt,
			UpdatedBy:   raw.UpdatedBy,
		})
	}

	if err := tbdom.ValidateContentFiles(out); err != nil {
		return nil, err
	}
	return out, nil
}

func validatePersistedTokenBlueprint(tb tbdom.TokenBlueprint) error {
	if _, err := tbdom.New(
		tb.ID,
		tb.Name,
		tb.Symbol,
		tb.BrandID,
		tb.CompanyID,
		tb.Description,
		tb.ContentFiles,
		tb.AssigneeID,
		tb.CreatedAt,
		tb.CreatedBy,
		tb.UpdatedAt,
	); err != nil {
		return err
	}
	if tb.UpdatedBy == "" {
		return tbdom.ErrInvalidUpdatedBy
	}
	if !tbdom.IsValidModerationStatus(tb.ModerationStatus) {
		return tbdom.ErrInvalidModerationStatus
	}
	if tb.IconSize < 0 {
		return tbdom.ErrInvalidIconSize
	}

	hasAnyIconField :=
		tb.IconURL != "" ||
			tb.IconObjectPath != "" ||
			tb.IconFileName != "" ||
			tb.IconContentType != "" ||
			tb.IconSize != 0

	if hasAnyIconField {
		if tb.IconURL == "" {
			return tbdom.ErrInvalidIconURL
		}
		if tb.IconObjectPath == "" {
			return tbdom.ErrInvalidIconObjectPath
		}
		if tb.IconFileName == "" {
			return tbdom.ErrInvalidIconFileName
		}
	}

	return nil
}

// ========================================
// compile-time interface check
// ========================================

var _ tbdom.RepositoryPort = (*TokenBlueprintRepositoryFS)(nil)
