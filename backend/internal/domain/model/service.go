package model

import (
	"context"
	"strings"
	"sync"
)

// Service - モデルドメインのビジネスロジック層
type Service struct {
	repo   RepositoryPort
	pbRepo ProductBlueprintPort // 任意依存（プロダクト名など取得用）
}

// NewService - コンストラクタ
func NewService(repo RepositoryPort, pbRepo ProductBlueprintPort) *Service {
	return &Service{repo: repo, pbRepo: pbRepo}
}

// ========================================
// 変換系（フロントの純粋関数相当）
// ========================================

// TransformModelDataToLegacyFormatFromVariations
// variationsから旧フォーマット（SizeVariation/ModelNumber）へ変換
func TransformModelDataToLegacyFormatFromVariations(
	variations []ModelVariation,
) (sizeVariations []SizeVariation, modelNumbers []ModelNumber) {
	sizeMap := make(map[string]SizeVariation)
	for _, v := range variations {
		if _, ok := sizeMap[v.Size]; !ok {
			sizeMap[v.Size] = SizeVariation{
				ID:           v.ID,
				Size:         v.Size,
				Measurements: v.Measurements,
			}
		}
		modelNumbers = append(modelNumbers, ModelNumber{
			Size:        v.Size,
			Color:       v.Color,
			ModelNumber: v.ModelNumber,
		})
	}
	sizeVariations = make([]SizeVariation, 0, len(sizeMap))
	for _, sv := range sizeMap {
		sizeVariations = append(sizeVariations, sv)
	}
	return
}

// HandleColorChange - カラー変更時に削除されたカラーのmodelNumbersを除去
func HandleColorChange(oldColors, newColors []string, current []ModelNumber) []ModelNumber {
	newSet := make(map[string]struct{}, len(newColors))
	for _, c := range newColors {
		newSet[c] = struct{}{}
	}
	out := make([]ModelNumber, 0, len(current))
	for _, mn := range current {
		if _, ok := newSet[mn.Color]; ok {
			out = append(out, mn)
		}
	}
	return out
}

// HandleSizeVariationChange - サイズの削除/リネームを反映
func HandleSizeVariationChange(oldSizes, newSizes []SizeVariation, current []ModelNumber) []ModelNumber {
	// 削除されたサイズ
	newByID := make(map[string]SizeVariation, len(newSizes))
	for _, s := range newSizes {
		newByID[s.ID] = s
	}
	removed := make(map[string]struct{})
	renamed := make(map[string]string) // oldName -> newName

	oldByID := make(map[string]SizeVariation, len(oldSizes))
	for _, s := range oldSizes {
		oldByID[s.ID] = s
		if ns, ok := newByID[s.ID]; ok {
			if ns.Size != s.Size {
				renamed[s.Size] = ns.Size
			}
		} else {
			removed[s.Size] = struct{}{}
		}
	}

	out := make([]ModelNumber, 0, len(current))
	for _, mn := range current {
		if _, isRemoved := removed[mn.Size]; isRemoved {
			continue
		}
		if nn, ok := renamed[mn.Size]; ok {
			mn.Size = nn
		}
		out = append(out, mn)
	}
	return out
}

// UpdateModelNumberMatrix - カラー×サイズの直積を再構成
func UpdateModelNumberMatrix(colors []string, sizes []SizeVariation, current []ModelNumber) []ModelNumber {
	// 既存のマトリクスを引き継ぎつつ足りない組み合わせを追加
	exists := make(map[string]ModelNumber)
	for _, mn := range current {
		key := mn.Size + "||" + mn.Color
		exists[key] = mn
	}
	var out []ModelNumber
	for _, s := range sizes {
		sz := strings.TrimSpace(s.Size)
		if sz == "" {
			continue
		}
		for _, c := range colors {
			cc := strings.TrimSpace(c)
			if cc == "" {
				continue
			}
			key := sz + "||" + cc
			if mn, ok := exists[key]; ok {
				out = append(out, mn)
			} else {
				out = append(out, ModelNumber{
					Size:        sz,
					Color:       cc,
					ModelNumber: "",
				})
			}
		}
	}
	return out
}

// UpdateSingleModelNumber - メモリ上のモデルナンバー1件更新
func UpdateSingleModelNumber(modelNumbers []ModelNumber, size, color, newModelNumber string) []ModelNumber {
	out := make([]ModelNumber, len(modelNumbers))
	copy(out, modelNumbers)
	for i := range out {
		if out[i].Size == size && out[i].Color == color {
			out[i].ModelNumber = newModelNumber
			break
		}
	}
	return out
}

// ConvertToSizeVariationsFromModelVariations - サイズ配列とモデルバリエーションからSizeVariationへ
func ConvertToSizeVariationsFromModelVariations(sizes []string, modelVariations []ModelVariation) []SizeVariation {
	out := make([]SizeVariation, 0, len(sizes))
	for i, sz := range sizes {
		var meas Measurements
		for _, mv := range modelVariations {
			if mv.Size == sz {
				meas = mv.Measurements
				break
			}
		}
		out = append(out, SizeVariation{
			ID:           "size-" + itoa(i),
			Size:         sz,
			Measurements: meas,
		})
	}
	return out
}

// ========================================
// Handler/UseCase（RepositoryPortを利用）
// ========================================

func (s *Service) CreateModelVariation(ctx context.Context, productID string, v NewModelVariation) (*ModelVariation, error) {
	if strings.TrimSpace(productID) == "" {
		return nil, ErrProductIDRequired
	}
	return s.repo.CreateModelVariation(ctx, productID, v)
}

func (s *Service) UpdateModelVariation(ctx context.Context, variationID string, updates ModelVariationUpdate) (*ModelVariation, error) {
	if strings.TrimSpace(variationID) == "" {
		return nil, ErrVariationIDRequired
	}
	return s.repo.UpdateModelVariation(ctx, variationID, updates)
}

func (s *Service) DeleteModelVariation(ctx context.Context, variationID string) (*ModelVariation, error) {
	if strings.TrimSpace(variationID) == "" {
		return nil, ErrVariationIDRequired
	}
	return s.repo.DeleteModelVariation(ctx, variationID)
}

func (s *Service) UpdateModelData(ctx context.Context, productID string, updates ModelDataUpdate) (*ModelData, error) {
	if strings.TrimSpace(productID) == "" {
		return nil, ErrProductIDRequired
	}
	return s.repo.UpdateModelData(ctx, productID, updates)
}

// BulkUpdateModelVariations - variationsを一括置換（リポジトリがサポートする前提）
func (s *Service) BulkUpdateModelVariations(ctx context.Context, productID string, variations []ModelVariation) (*ModelData, error) {
	if strings.TrimSpace(productID) == "" {
		return nil, ErrProductIDRequired
	}
	updates := ModelDataUpdate{"variations": variations}
	return s.repo.UpdateModelData(ctx, productID, updates)
}

// UpdateSingleModelNumberOnServer - サーバー上の特定サイズ/カラーのモデルナンバーを更新
func (s *Service) UpdateSingleModelNumberOnServer(ctx context.Context, productID, size, color, newModelNumber string) error {
	vars, err := s.repo.GetModelVariations(ctx, productID)
	if err != nil {
		return err
	}
	for _, v := range vars {
		if v.Size == size && v.Color == color {
			_, err = s.repo.UpdateModelVariation(ctx, v.ID, ModelVariationUpdate{
				ModelNumber: &newModelNumber,
			})
			return err
		}
	}
	return ErrTargetVariationNotFound
}

// BulkUpdateModelNumbersOnServer - 複数のモデルナンバーを更新
func (s *Service) BulkUpdateModelNumbersOnServer(ctx context.Context, productID string, modelNumbers []ModelNumber) error {
	if len(modelNumbers) == 0 {
		return nil
	}
	vars, err := s.repo.GetModelVariations(ctx, productID)
	if err != nil {
		return err
	}
	// map作成
	target := make(map[string]string, len(modelNumbers)) // key: size||color -> modelNumber
	for _, mn := range modelNumbers {
		target[mn.Size+"||"+mn.Color] = mn.ModelNumber
	}
	// 対象を更新
	for _, v := range vars {
		if newNum, ok := target[v.Size+"||"+v.Color]; ok && newNum != v.ModelNumber {
			if _, err := s.repo.UpdateModelVariation(ctx, v.ID, ModelVariationUpdate{ModelNumber: &newNum}); err != nil {
				return err
			}
		}
	}
	return nil
}

// AddSizeVariationOnServer - 新サイズを全カラーに追加
func (s *Service) AddSizeVariationOnServer(ctx context.Context, productID string, size SizeVariation, colors []string) error {
	for _, color := range colors {
		_, err := s.repo.CreateModelVariation(ctx, productID, NewModelVariation{
			Size:         size.Size,
			Color:        color,
			ModelNumber:  "",
			Measurements: size.Measurements,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateSizeVariationOnServer - 既存サイズ名/寸法更新
func (s *Service) UpdateSizeVariationOnServer(ctx context.Context, productID, oldSize string, newSize SizeVariation) error {
	vars, err := s.repo.GetModelVariations(ctx, productID)
	if err != nil {
		return err
	}
	for _, v := range vars {
		if v.Size == oldSize {
			ns := newSize.Size
			upd := ModelVariationUpdate{
				Measurements: newSize.Measurements,
			}
			if strings.TrimSpace(ns) != "" && ns != v.Size {
				upd.Size = &ns
			}
			if _, err := s.repo.UpdateModelVariation(ctx, v.ID, upd); err != nil {
				return err
			}
		}
	}
	return nil
}

// DeleteSizeVariationOnServer - 指定サイズの全Variation削除
func (s *Service) DeleteSizeVariationOnServer(ctx context.Context, productID, size string) (int, error) {
	vars, err := s.repo.GetModelVariations(ctx, productID)
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, v := range vars {
		if v.Size == size {
			if _, err := s.repo.DeleteModelVariation(ctx, v.ID); err != nil {
				return deleted, err
			}
			deleted++
		}
	}
	if deleted == 0 {
		return 0, ErrNoVariationsFoundForSize
	}
	return deleted, nil
}

// AddColorVariationOnServer - 新カラーを全サイズに追加
func (s *Service) AddColorVariationOnServer(ctx context.Context, productID, color string, sizes []SizeVariation) error {
	for _, sv := range sizes {
		_, err := s.repo.CreateModelVariation(ctx, productID, NewModelVariation{
			Size:         sv.Size,
			Color:        color,
			ModelNumber:  "",
			Measurements: sv.Measurements,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteColorVariationOnServer - 指定カラーの全Variation削除
func (s *Service) DeleteColorVariationOnServer(ctx context.Context, productID, color string) (int, error) {
	vars, err := s.repo.GetModelVariations(ctx, productID)
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, v := range vars {
		if v.Color == color {
			if _, err := s.repo.DeleteModelVariation(ctx, v.ID); err != nil {
				return deleted, err
			}
			deleted++
		}
	}
	if deleted == 0 {
		return 0, ErrNoVariationsFoundForColor
	}
	return deleted, nil
}

// ========================================
// Resolver系（モデル番号 => 製品情報）
// ========================================

// ProductBlueprintPort - プロダクト名取得用のポート
type ProductBlueprintPort interface {
	GetByID(ctx context.Context, id string) (*ProductBlueprint, error)
}

type ProductBlueprint struct {
	ID          string
	ProductName string
}

// ProductNameResult - TS側の戻り値に相当
type ProductNameResult struct {
	ProductName        string
	ProductBlueprintID string
}

// ModelWithProductInfo - TS側の戻り値に相当
type ModelWithProductInfo struct {
	ModelNumber        string       `json:"modelNumber"`
	ProductID          string       `json:"productId"`
	ProductBlueprintID string       `json:"productBlueprintId"`
	ProductName        string       `json:"productName"`
	Size               string       `json:"size"`
	Color              string       `json:"color"`
	Measurements       Measurements `json:"measurements,omitempty"`
}

// GetProductBlueprintIDByModelNumber - 候補productID群から探索
func (s *Service) GetProductBlueprintIDByModelNumber(ctx context.Context, modelNumber string, candidateProductIDs []string) (string, error) {
	for _, pid := range candidateProductIDs {
		vars, err := s.repo.GetModelVariations(ctx, pid)
		if err != nil {
			return "", err
		}
		for _, v := range vars {
			if v.ModelNumber == modelNumber {
				return pid, nil
			}
		}
	}
	return "", ErrProductBlueprintIDNotFound
}

// GetProductNameByModelNumber - プロダクト名を取得
func (s *Service) GetProductNameByModelNumber(ctx context.Context, modelNumber string, candidateProductIDs []string) (*ProductNameResult, error) {
	pbID, err := s.GetProductBlueprintIDByModelNumber(ctx, modelNumber, candidateProductIDs)
	if err != nil {
		return nil, err
	}
	if s.pbRepo == nil {
		return &ProductNameResult{ProductName: "", ProductBlueprintID: pbID}, nil
	}
	pb, err := s.pbRepo.GetByID(ctx, pbID)
	if err != nil {
		return nil, err
	}
	if pb == nil {
		return nil, ErrProductBlueprintNotFound
	}
	return &ProductNameResult{
		ProductName:        pb.ProductName,
		ProductBlueprintID: pbID,
	}, nil
}

// GetModelWithProductInfo - モデル番号から詳細
func (s *Service) GetModelWithProductInfo(ctx context.Context, modelNumber string, candidateProductIDs []string) (*ModelWithProductInfo, error) {
	pbID, err := s.GetProductBlueprintIDByModelNumber(ctx, modelNumber, candidateProductIDs)
	if err != nil {
		return nil, err
	}
	vars, err := s.repo.GetModelVariations(ctx, pbID)
	if err != nil {
		return nil, err
	}
	var matched *ModelVariation
	for i := range vars {
		if vars[i].ModelNumber == modelNumber {
			matched = &vars[i]
			break
		}
	}
	if matched == nil {
		return nil, ErrVariationNotFound
	}

	name := ""
	if s.pbRepo != nil {
		if pb, err := s.pbRepo.GetByID(ctx, pbID); err == nil && pb != nil {
			name = pb.ProductName
		}
	}

	return &ModelWithProductInfo{
		ModelNumber:        matched.ModelNumber,
		ProductBlueprintID: pbID,
		ProductName:        name,
		Size:               matched.Size,
		Color:              matched.Color,
		Measurements:       matched.Measurements,
	}, nil
}

// GetProductNamesByModelNumbers - 複数モデル番号の名称を一括取得
func (s *Service) GetProductNamesByModelNumbers(ctx context.Context, modelNumbers []string, candidateProductIDs []string) (map[string]ProductNameResult, error) {
	type result struct {
		mn  string
		pr  *ProductNameResult
		err error
	}
	out := make(map[string]ProductNameResult, len(modelNumbers))
	wg := sync.WaitGroup{}
	ch := make(chan result, len(modelNumbers))

	for _, mn := range modelNumbers {
		wg.Add(1)
		go func(modelNumber string) {
			defer wg.Done()
			pr, err := s.GetProductNameByModelNumber(ctx, modelNumber, candidateProductIDs)
			ch <- result{mn: modelNumber, pr: pr, err: err}
		}(mn)
	}
	wg.Wait()
	close(ch)

	for r := range ch {
		if r.err == nil && r.pr != nil {
			out[r.mn] = *r.pr
		}
	}
	return out, nil
}

// GetModelsWithProductInfoBatch - 複数モデル番号の詳細を一括取得
func (s *Service) GetModelsWithProductInfoBatch(ctx context.Context, modelNumbers []string, candidateProductIDs []string) (map[string]ModelWithProductInfo, error) {
	type result struct {
		mn  string
		mi  *ModelWithProductInfo
		err error
	}
	out := make(map[string]ModelWithProductInfo, len(modelNumbers))
	wg := sync.WaitGroup{}
	ch := make(chan result, len(modelNumbers))

	for _, mn := range modelNumbers {
		wg.Add(1)
		go func(modelNumber string) {
			defer wg.Done()
			mi, err := s.GetModelWithProductInfo(ctx, modelNumber, candidateProductIDs)
			ch <- result{mn: modelNumber, mi: mi, err: err}
		}(mn)
	}
	wg.Wait()
	close(ch)

	for r := range ch {
		if r.err == nil && r.mi != nil {
			out[r.mn] = *r.mi
		}
	}
	return out, nil
}

// ========================================
// ユーティリティ
// ========================================

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := v < 0
	if neg {
		v = -v
	}
	for v > 0 {
		i--
		buf[i] = byte('0' + (v % 10))
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// 参考にする定義（ドメイン内での使用）
// ModelDataはRepositoryPort側で定義済み。
// 参考: type ModelData struct { ProductID string; UpdatedAt time.Time; ... }
