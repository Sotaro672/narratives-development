// backend/internal/adapters/out/firestore/productBlueprint_repository_fs.go
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

	usecase "narratives/internal/application/usecase"
	pbdom "narratives/internal/domain/productBlueprint"
)

// ProductBlueprintRepositoryFS implements ProductBlueprintRepo using Firestore.
type ProductBlueprintRepositoryFS struct {
	Client *firestore.Client
}

func NewProductBlueprintRepositoryFS(client *firestore.Client) *ProductBlueprintRepositoryFS {
	return &ProductBlueprintRepositoryFS{Client: client}
}

func (r *ProductBlueprintRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("product_blueprints")
}

// history コレクション: product_blueprints_history/{blueprintId}/versions/{version}
func (r *ProductBlueprintRepositoryFS) historyCol(blueprintID string) *firestore.CollectionRef {
	return r.Client.Collection("product_blueprints_history").
		Doc(blueprintID).
		Collection("versions")
}

// Compile-time check: ensure this satisfies usecase.ProductBlueprintRepo
// および usecase.ProductBlueprintPrintedRepo.
var (
	_ usecase.ProductBlueprintRepo        = (*ProductBlueprintRepositoryFS)(nil)
	_ usecase.ProductBlueprintPrintedRepo = (*ProductBlueprintRepositoryFS)(nil)
)

// ========================
// Core methods (ProductBlueprintRepo)
// ========================

// GetByID returns a single ProductBlueprint by ID.
func (r *ProductBlueprintRepositoryFS) GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}

	return docToProductBlueprint(snap)
}

// ★ 追加: productBlueprintId から productName だけを取得するヘルパ
// usecase.mintProductBlueprintRepo / productBlueprint.Repository の
// GetProductNameByID を満たすための薄いラッパ。
func (r *ProductBlueprintRepositoryFS) GetProductNameByID(
	ctx context.Context,
	id string,
) (string, error) {
	if r.Client == nil {
		return "", errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return "", pbdom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", pbdom.ErrNotFound
		}
		return "", err
	}

	data := snap.Data()
	if data != nil {
		// まずはフィールドから直接読む（軽量）
		if v, ok := data["productName"].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v), nil
		}
		if v, ok := data["product_name"].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v), nil
		}
	}

	// なければ既存ヘルパでドメイン型に変換してから取り出す
	pb, err := docToProductBlueprint(snap)
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(pb.ProductName)
	if name == "" {
		return "", pbdom.ErrNotFound
	}
	return name, nil
}

// ★ 追加: productBlueprintId から Patch 全体を組み立てて返すヘルパ
// usecase.mintProductBlueprintRepo の GetPatchByID を満たすための実装。
func (r *ProductBlueprintRepositoryFS) GetPatchByID(
	ctx context.Context,
	id string,
) (pbdom.Patch, error) {
	if r.Client == nil {
		return pbdom.Patch{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return pbdom.Patch{}, pbdom.ErrNotFound
	}

	// まずは既存の GetByID でライブの ProductBlueprint を取得
	pb, err := r.GetByID(ctx, id)
	if err != nil {
		return pbdom.Patch{}, err
	}

	// Patch 用にポインタ値を組み立て
	name := pb.ProductName
	brandID := pb.BrandID
	itemType := pb.ItemType
	fit := pb.Fit
	material := pb.Material
	weight := pb.Weight
	qa := make([]string, len(pb.QualityAssurance))
	copy(qa, pb.QualityAssurance)
	productIdTag := pb.ProductIdTag
	assigneeID := pb.AssigneeID

	patch := pbdom.Patch{
		ProductName:      &name,
		BrandID:          &brandID,
		ItemType:         &itemType,
		Fit:              &fit,
		Material:         &material,
		Weight:           &weight,
		QualityAssurance: &qa,
		ProductIdTag:     &productIdTag,
		AssigneeID:       &assigneeID,
	}

	return patch, nil
}

// Exists reports whether a ProductBlueprint with given ID exists.
func (r *ProductBlueprintRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
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

// ListIDsByCompany は、指定された companyId を持つ product_blueprints の ID 一覧を返します。
// MintRequest 用のチェーン（companyId → productBlueprintId → production → mintRequest）で利用します。
func (r *ProductBlueprintRepositoryFS) ListIDsByCompany(
	ctx context.Context,
	companyID string,
) ([]string, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		// 空 companyId の場合は空配列を返す（エラーにはしない）
		return []string{}, nil
	}

	iter := r.col().
		Where("companyId", "==", companyID).
		Documents(ctx)
	defer iter.Stop()

	var ids []string
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		ids = append(ids, snap.Ref.ID)
	}
	return ids, nil
}

// ★ 追加: printed == true だけを、指定 ID 群から取得
// - ListIDsByCompany → ListPrinted で 1 セットの利用を想定
func (r *ProductBlueprintRepositoryFS) ListPrinted(
	ctx context.Context,
	ids []string,
) ([]pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	// ID 正規化 & 重複排除
	uniq := make(map[string]struct{}, len(ids))
	var cleaned []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := uniq[id]; ok {
			continue
		}
		uniq[id] = struct{}{}
		cleaned = append(cleaned, id)
	}
	if len(cleaned) == 0 {
		return []pbdom.ProductBlueprint{}, nil
	}

	out := make([]pbdom.ProductBlueprint, 0, len(cleaned))
	for _, id := range cleaned {
		snap, err := r.col().Doc(id).Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				continue
			}
			return nil, err
		}
		pb, err := docToProductBlueprint(snap)
		if err != nil {
			return nil, err
		}

		if pb.Printed {
			out = append(out, pb)
		}
	}
	return out, nil
}

// ★ 追加: printed: false → true へ更新する（usecase.ProductBlueprintPrintedRepo 用）
func (r *ProductBlueprintRepositoryFS) MarkPrinted(
	ctx context.Context,
	id string,
) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return pbdom.ProductBlueprint{}, pbdom.ErrInvalidID
	}

	docRef := r.col().Doc(id)

	// 現在状態を取得して printed を確認
	snap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}

	data := snap.Data()
	if data != nil {
		if v, ok := data["printed"]; ok {
			switch x := v.(type) {
			case bool:
				if x {
					// すでに printed の場合は Forbidden 扱い（再印刷禁止など）
					return pbdom.ProductBlueprint{}, pbdom.ErrForbidden
				}
			case string:
				// 旧データ互換: "printed" を true 相当とみなす
				if strings.TrimSpace(x) == "printed" {
					return pbdom.ProductBlueprint{}, pbdom.ErrForbidden
				}
			}
		}
	}

	// printed = true, updatedAt = now をセット
	now := time.Now().UTC()
	if _, err := docRef.Update(ctx, []firestore.Update{
		{Path: "printed", Value: true},
		{Path: "updatedAt", Value: now},
	}); err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}

	// 再取得してドメイン型へ変換
	snap, err = docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ProductBlueprint{}, pbdom.ErrNotFound
		}
		return pbdom.ProductBlueprint{}, err
	}

	return docToProductBlueprint(snap)
}

// List returns all ProductBlueprints (optionally filtered by companyId in context).
// （簡易版: フィルタ/ソート/ページングは usecase 側でラップして利用）
func (r *ProductBlueprintRepositoryFS) List(ctx context.Context) ([]pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	q := r.col().Query

	// usecase.CompanyIDFromContext で context から companyId を取得し、
	// 指定があればテナント単位で絞り込む
	if cid := strings.TrimSpace(usecase.CompanyIDFromContext(ctx)); cid != "" {
		q = q.Where("companyId", "==", cid)
	}

	snaps, err := q.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	out := make([]pbdom.ProductBlueprint, 0, len(snaps))
	for _, snap := range snaps {
		pb, err := docToProductBlueprint(snap)
		if err != nil {
			return nil, err
		}
		out = append(out, pb)
	}
	return out, nil
}

// ListDeleted returns only product_blueprints whose deletedAt is NOT null
// (i.e. logically deleted ones), optionally filtered by companyId in context.
func (r *ProductBlueprintRepositoryFS) ListDeleted(ctx context.Context) ([]pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	q := r.col().Query

	// テナントスコープ: companyId が context に入っていれば絞り込む
	if cid := strings.TrimSpace(usecase.CompanyIDFromContext(ctx)); cid != "" {
		q = q.Where("companyId", "==", cid)
	}

	// deletedAt が存在して 0 以外の Timestamp が入っているものだけを取得
	// （フィールド未定義のドキュメントはこの range 条件にマッチしない）
	q = q.Where("deletedAt", ">", time.Time{})

	snaps, err := q.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	out := make([]pbdom.ProductBlueprint, 0, len(snaps))
	for _, snap := range snaps {
		pb, err := docToProductBlueprint(snap)
		if err != nil {
			return nil, err
		}
		out = append(out, pb)
	}
	return out, nil
}

// Create inserts a new ProductBlueprint (no upsert).
// If ID is empty, it is auto-generated.
// If CreatedAt/UpdatedAt are zero, they are set to now (UTC).
func (r *ProductBlueprintRepositoryFS) Create(
	ctx context.Context,
	pb pbdom.ProductBlueprint,
) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()

	id := strings.TrimSpace(pb.ID)
	var docRef *firestore.DocumentRef
	if id == "" {
		docRef = r.col().NewDoc()
		pb.ID = docRef.ID
	} else {
		docRef = r.col().Doc(id)
	}

	if pb.CreatedAt.IsZero() {
		pb.CreatedAt = now
	} else {
		pb.CreatedAt = pb.CreatedAt.UTC()
	}
	if pb.UpdatedAt.IsZero() {
		pb.UpdatedAt = now
	} else {
		pb.UpdatedAt = pb.UpdatedAt.UTC()
	}

	data, err := productBlueprintToDoc(pb, pb.CreatedAt, pb.UpdatedAt)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}
	data["id"] = pb.ID

	if _, err := docRef.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return pbdom.ProductBlueprint{}, pbdom.ErrConflict
		}
		return pbdom.ProductBlueprint{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}
	return docToProductBlueprint(snap)
}

// Save upserts a ProductBlueprint.
// If ID is empty, a new one is generated.
// If CreatedAt is zero, it is set to now (UTC).
// UpdatedAt is always set to now (UTC) when saving.
func (r *ProductBlueprintRepositoryFS) Save(
	ctx context.Context,
	pb pbdom.ProductBlueprint,
) (pbdom.ProductBlueprint, error) {
	if r.Client == nil {
		return pbdom.ProductBlueprint{}, errors.New("firestore client is nil")
	}

	now := time.Now().UTC()

	id := strings.TrimSpace(pb.ID)
	var docRef *firestore.DocumentRef
	if id == "" {
		docRef = r.col().NewDoc()
		pb.ID = docRef.ID
	} else {
		docRef = r.col().Doc(id)
	}

	if pb.CreatedAt.IsZero() {
		pb.CreatedAt = now
	} else {
		pb.CreatedAt = pb.CreatedAt.UTC()
	}
	pb.UpdatedAt = now

	data, err := productBlueprintToDoc(pb, pb.CreatedAt, pb.UpdatedAt)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}
	data["id"] = pb.ID

	// 完全上書き（MergeAll は使わない）
	if _, err := docRef.Set(ctx, data); err != nil {
		return pbdom.ProductBlueprint{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return pbdom.ProductBlueprint{}, err
	}
	return docToProductBlueprint(snap)
}

// Delete removes a ProductBlueprint by ID (物理削除用).
// 通常の画面操作からは usecase.ProductBlueprintUsecase.Delete → SoftDeleteWithModels を利用し、
// この Delete は 90日後のクリーンアップジョブなどから直接呼ぶ想定。
func (r *ProductBlueprintRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return pbdom.ErrNotFound
	}

	_, err := r.col().Doc(id).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ErrNotFound
		}
		return err
	}
	return nil
}

// SoftDeleteWithModels:
//   - 現在は product_blueprints/{id} に deletedAt を立てて論理削除するだけ。
//   - models コレクションのドキュメントには一切変更を加えない。
//
// ※ 現在はユースケース側で SoftDelete + ExpireAt 設定を行うため、
// このメソッドは将来的に廃止する方向のレガシー実装扱い。
func (r *ProductBlueprintRepositoryFS) SoftDeleteWithModels(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return pbdom.ErrInvalidID
	}

	now := time.Now().UTC()

	// まず product_blueprints/{id} が存在するかチェック
	pbRef := r.col().Doc(id)
	if _, err := pbRef.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ErrNotFound
		}
		return err
	}

	// productBlueprint を論理削除（models 側には何もしない）
	if _, err := pbRef.Update(ctx, []firestore.Update{
		{Path: "deletedAt", Value: now},
		// DeletedBy は context からのユーザーIDなどを後で組み込む想定
	}); err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ErrNotFound
		}
		return err
	}

	return nil
}

// RestoreWithModels:
//   - 論理削除された product_blueprints/{id} の deletedAt / deletedBy をクリアして復旧。
//   - models コレクションのドキュメントには一切変更を加えない。
func (r *ProductBlueprintRepositoryFS) RestoreWithModels(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return pbdom.ErrInvalidID
	}

	pbRef := r.col().Doc(id)
	if _, err := pbRef.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ErrNotFound
		}
		return err
	}

	// productBlueprint の論理削除を解除（models 側には何もしない）
	if _, err := pbRef.Update(ctx, []firestore.Update{
		{Path: "deletedAt", Value: firestore.Delete},
		{Path: "deletedBy", Value: firestore.Delete},
	}); err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.ErrNotFound
		}
		return err
	}
	return nil
}

// ========================
// History (snapshot, versioned)
// ========================

// SaveHistorySnapshot は product_blueprints_history/{blueprintId}/versions/{version}
// に、その時点の ProductBlueprint のスナップショットを保存する。
func (r *ProductBlueprintRepositoryFS) SaveHistorySnapshot(
	ctx context.Context,
	blueprintID string,
	h pbdom.HistoryRecord,
) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	blueprintID = strings.TrimSpace(blueprintID)
	if blueprintID == "" {
		return pbdom.ErrInvalidID
	}

	// Blueprint.ID が空なら補正、異なる場合も blueprintID を優先
	if strings.TrimSpace(h.Blueprint.ID) == "" || h.Blueprint.ID != blueprintID {
		h.Blueprint.ID = blueprintID
	}

	// UpdatedAt/UpdatedBy（履歴メタ）は Blueprint 側になければ HistoryRecord 側から補完
	if h.UpdatedAt.IsZero() {
		h.UpdatedAt = h.Blueprint.UpdatedAt
	}
	if h.UpdatedAt.IsZero() {
		h.UpdatedAt = time.Now().UTC()
	}

	// ドキュメント ID は version 番号文字列
	docID := fmt.Sprintf("%d", h.Version)
	docRef := r.historyCol(blueprintID).Doc(docID)

	data, err := productBlueprintToDoc(h.Blueprint, h.Blueprint.CreatedAt, h.Blueprint.UpdatedAt)
	if err != nil {
		return err
	}
	data["id"] = blueprintID
	data["version"] = h.Version
	data["historyUpdatedAt"] = h.UpdatedAt.UTC()
	if h.UpdatedBy != nil {
		if s := strings.TrimSpace(*h.UpdatedBy); s != "" {
			data["historyUpdatedBy"] = s
		}
	}

	if _, err := docRef.Set(ctx, data); err != nil {
		return err
	}
	return nil
}

// ListHistory は blueprintID に紐づく全バージョンの履歴を取得する。
// 基本的には version の降順（新しい順）で返す。
func (r *ProductBlueprintRepositoryFS) ListHistory(
	ctx context.Context,
	blueprintID string,
) ([]pbdom.HistoryRecord, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	blueprintID = strings.TrimSpace(blueprintID)
	if blueprintID == "" {
		return nil, pbdom.ErrInvalidID
	}

	q := r.historyCol(blueprintID).OrderBy("version", firestore.Desc)
	snaps, err := q.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	out := make([]pbdom.HistoryRecord, 0, len(snaps))
	for _, snap := range snaps {
		data := snap.Data()
		if data == nil {
			continue
		}

		pb, err := docToProductBlueprint(snap)
		if err != nil {
			return nil, err
		}

		// version の取り出し
		var version int64
		if v, ok := data["version"]; ok {
			switch x := v.(type) {
			case int64:
				version = x
			case int:
				version = int64(x)
			case float64:
				version = int64(x)
			}
		}

		// UpdatedAt/UpdatedBy（履歴メタ）
		var histUpdatedAt time.Time
		if v, ok := data["historyUpdatedAt"].(time.Time); ok && !v.IsZero() {
			histUpdatedAt = v.UTC()
		} else {
			histUpdatedAt = pb.UpdatedAt
		}

		var histUpdatedBy *string
		if v, ok := data["historyUpdatedBy"].(string); ok && strings.TrimSpace(v) != "" {
			s := strings.TrimSpace(v)
			histUpdatedBy = &s
		} else {
			histUpdatedBy = pb.UpdatedBy
		}

		out = append(out, pbdom.HistoryRecord{
			Blueprint: pb,
			Version:   version,
			UpdatedAt: histUpdatedAt,
			UpdatedBy: histUpdatedBy,
		})
	}
	return out, nil
}

// GetHistoryByVersion は特定バージョンの履歴を 1 件取得する。
func (r *ProductBlueprintRepositoryFS) GetHistoryByVersion(
	ctx context.Context,
	blueprintID string,
	version int64,
) (pbdom.HistoryRecord, error) {
	if r.Client == nil {
		return pbdom.HistoryRecord{}, errors.New("firestore client is nil")
	}

	blueprintID = strings.TrimSpace(blueprintID)
	if blueprintID == "" {
		return pbdom.HistoryRecord{}, pbdom.ErrInvalidID
	}

	docID := fmt.Sprintf("%d", version)
	snap, err := r.historyCol(blueprintID).Doc(docID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return pbdom.HistoryRecord{}, pbdom.ErrNotFound
		}
		return pbdom.HistoryRecord{}, err
	}

	data := snap.Data()
	if data == nil {
		return pbdom.HistoryRecord{}, fmt.Errorf("empty history document: %s", snap.Ref.Path)
	}

	pb, err := docToProductBlueprint(snap)
	if err != nil {
		return pbdom.HistoryRecord{}, err
	}

	// version の取り出し（念のため doc からも読んでおく）
	var ver int64
	if v, ok := data["version"]; ok {
		switch x := v.(type) {
		case int64:
			ver = x
		case int:
			ver = int64(x)
		case float64:
			ver = int64(x)
		}
	}
	if ver == 0 {
		ver = version
	}

	var histUpdatedAt time.Time
	if v, ok := data["historyUpdatedAt"].(time.Time); ok && !v.IsZero() {
		histUpdatedAt = v.UTC()
	} else {
		histUpdatedAt = pb.UpdatedAt
	}

	var histUpdatedBy *string
	if v, ok := data["historyUpdatedBy"].(string); ok && strings.TrimSpace(v) != "" {
		s := strings.TrimSpace(v)
		histUpdatedBy = &s
	} else {
		histUpdatedBy = pb.UpdatedBy
	}

	return pbdom.HistoryRecord{
		Blueprint: pb,
		Version:   ver,
		UpdatedAt: histUpdatedAt,
		UpdatedBy: histUpdatedBy,
	}, nil
}

// ========================
// Helpers
// ========================

func docToProductBlueprint(doc *firestore.DocumentSnapshot) (pbdom.ProductBlueprint, error) {
	data := doc.Data()
	if data == nil {
		return pbdom.ProductBlueprint{}, fmt.Errorf("empty product_blueprints document: %s", doc.Ref.ID)
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
	getTimeVal := func(keys ...string) time.Time {
		for _, k := range keys {
			if v, ok := data[k].(time.Time); ok && !v.IsZero() {
				return v.UTC()
			}
		}
		return time.Time{}
	}

	getStringSlice := func(keys ...string) []string {
		for _, key := range keys {
			raw, ok := data[key]
			if !ok || raw == nil {
				continue
			}
			switch vv := raw.(type) {
			case []interface{}:
				out := make([]string, 0, len(vv))
				for _, x := range vv {
					if s, ok := x.(string); ok {
						s = strings.TrimSpace(s)
						if s != "" {
							out = append(out, s)
						}
					}
				}
				return dedupTrimStrings(out)
			case []string:
				return dedupTrimStrings(vv)
			}
		}
		return nil
	}

	qas := getStringSlice("qualityAssurance", "quality_assurance")
	tagTypeStr := getStr("productIdTagType", "product_id_tag_type")
	itemTypeStr := getStr("itemType", "item_type")

	// printed は bool（新）/ string（旧データ）両対応で読み取る
	var printedBool bool
	if v, ok := data["printed"]; ok {
		switch x := v.(type) {
		case bool:
			printedBool = x
		case string:
			s := strings.TrimSpace(x)
			// 旧データ: "printed" を true、それ以外/空は false とみなす
			printedBool = (s == "printed")
		}
	}

	var deletedAtPtr *time.Time
	if t := getTimeVal("deletedAt", "deleted_at"); !t.IsZero() {
		deletedAtPtr = &t
	}

	// ExpireAt（TTL 用フィールド）も Firestore から読み込む
	var expireAtPtr *time.Time
	if t := getTimeVal("expireAt", "expire_at"); !t.IsZero() {
		expireAtPtr = &t
	}

	// ID はフィールド "id" / "blueprintId" があればそれを優先し、なければ doc.Ref.ID
	id := getStr("id", "blueprintId", "blueprint_id")
	if id == "" {
		id = doc.Ref.ID
	}

	pb := pbdom.ProductBlueprint{
		ID:          id,
		ProductName: getStr("productName", "product_name"),
		BrandID:     getStr("brandId", "brand_id"),
		ItemType:    pbdom.ItemType(itemTypeStr),
		Fit:         getStr("fit"),
		Material:    getStr("material"),
		Weight:      getFloat64(data["weight"]),

		QualityAssurance: dedupTrimStrings(qas),
		ProductIdTag: pbdom.ProductIDTag{
			Type: pbdom.ProductIDTagType(tagTypeStr),
		},
		CompanyID:  getStr("companyId", "company_id"),
		AssigneeID: getStr("assigneeId", "assignee_id"),

		// New printed フィールド（bool）
		Printed: printedBool,

		CreatedBy: getStrPtr("createdBy", "created_by"),
		CreatedAt: getTimeVal("createdAt", "created_at"),
		UpdatedBy: getStrPtr("updatedBy", "updated_by"),
		UpdatedAt: getTimeVal("updatedAt", "updated_at"),
		DeletedBy: getStrPtr("deletedBy", "deleted_by"),
		DeletedAt: deletedAtPtr,
		ExpireAt:  expireAtPtr,
	}

	return pb, nil
}

func productBlueprintToDoc(v pbdom.ProductBlueprint, createdAt, updatedAt time.Time) (map[string]any, error) {
	m := map[string]any{
		"productName": strings.TrimSpace(v.ProductName),
		"brandId":     strings.TrimSpace(v.BrandID),
		"itemType":    strings.TrimSpace(string(v.ItemType)),
		"fit":         strings.TrimSpace(v.Fit),
		"material":    strings.TrimSpace(v.Material),
		"weight":      v.Weight,
		"assigneeId":  strings.TrimSpace(v.AssigneeID),
		"companyId":   strings.TrimSpace(v.CompanyID),
		"createdAt":   createdAt.UTC(),
		"updatedAt":   updatedAt.UTC(),
		"printed":     v.Printed,
	}

	if len(v.QualityAssurance) > 0 {
		m["qualityAssurance"] = dedupTrimStrings(v.QualityAssurance)
	}

	if v.ProductIdTag.Type != "" {
		m["productIdTagType"] = strings.TrimSpace(string(v.ProductIdTag.Type))
	}

	if v.CreatedBy != nil {
		if s := strings.TrimSpace(*v.CreatedBy); s != "" {
			m["createdBy"] = s
		}
	}
	if v.UpdatedBy != nil {
		if s := strings.TrimSpace(*v.UpdatedBy); s != "" {
			m["updatedBy"] = s
		}
	}
	if v.DeletedAt != nil && !v.DeletedAt.IsZero() {
		m["deletedAt"] = v.DeletedAt.UTC()
	}
	if v.DeletedBy != nil {
		if s := strings.TrimSpace(*v.DeletedBy); s != "" {
			m["deletedBy"] = s
		}
	}
	// ExpireAt も Firestore に書き出す（TTL 対象フィールド）
	if v.ExpireAt != nil && !v.ExpireAt.IsZero() {
		m["expireAt"] = v.ExpireAt.UTC()
	}

	return m, nil
}

func getFloat64(v any) float64 {
	switch x := v.(type) {
	case int:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	case float32:
		return float64(x)
	case float64:
		return x
	default:
		return 0
	}
}

func dedupTrimStrings(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}
