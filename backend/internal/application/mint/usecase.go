// backend/internal/application/mint/usecase.go
package mint

import (
	"context"
	"errors"
	"log"
	"reflect"
	"sort"
	"strings"
	"time"

	dto "narratives/internal/application/mint/dto"
	qdto "narratives/internal/application/query/dto"
	resolver "narratives/internal/application/resolver"
	appusecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	inspectiondom "narratives/internal/domain/inspection"
	invdom "narratives/internal/domain/inventory" // inventory 連携
	mintdom "narratives/internal/domain/mint"
	pbpdom "narratives/internal/domain/productBlueprint"
	tokendom "narratives/internal/domain/token"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// チェーンミント起動用ポート
// ============================================================

// TokenMintPort は、MintUsecase から見た「オンチェーンミントを起動するための」ポートです。
// TokenUsecase がこのインターフェースを実装する想定です。
type TokenMintPort interface {
	MintFromMintRequest(ctx context.Context, mintID string) (*tokendom.MintResult, error)
}

// ============================================================
// Inventory Upsert Port（modelId 単位）
// ============================================================

// InventoryUpserter は inventories の upsert を行うための最小インターフェースです。
// inventories の docId を modelId_tokenBlueprintId にする方針のため、modelID を必須にする。
type InventoryUpserter interface {
	UpsertFromMintByModel(
		ctx context.Context,
		tokenBlueprintID string,
		productBlueprintID string,
		modelID string,
		productIDs []string,
	) (invdom.Mint, error)
}

// ============================================================
// MintUsecase 本体
// ============================================================

type MintUsecase struct {
	// 互換のため残しているが、company -> pb -> production の探索にはもう使わない方針
	pbRepo    mintdom.MintProductBlueprintRepo
	prodRepo  mintdom.MintProductionRepo
	inspRepo  mintdom.MintInspectionRepo
	modelRepo mintdom.MintModelRepo

	// TokenBlueprint の minted 状態や一覧を扱うためのリポジトリ
	tbRepo tbdom.RepositoryPort

	// Brand 一覧取得用
	brandSvc *branddom.Service

	// mints テーブル用リポジトリ
	mintRepo mintdom.MintRepository

	// inspections → passed productId 一覧を取得するためのポート
	passedProductLister mintdom.PassedProductLister

	// チェーンミント実行用ポート（TokenUsecase を想定）
	tokenMinter TokenMintPort

	// inventories への反映（modelId 単位）
	inventoryUC InventoryUpserter

	// createdBy(memberId) → 氏名 を解決するため
	// 既存DIを壊さないため、Setterで後から差し込む
	nameResolver *resolver.NameResolver
}

// NewMintUsecase は MintUsecase のコンストラクタです。
// NameResolver / InventoryUC は任意依存（Setterで後から差し込む）とする。
func NewMintUsecase(
	pbRepo mintdom.MintProductBlueprintRepo,
	prodRepo mintdom.MintProductionRepo,
	inspRepo mintdom.MintInspectionRepo,
	modelRepo mintdom.MintModelRepo,
	tbRepo tbdom.RepositoryPort,
	brandSvc *branddom.Service,
	mintRepo mintdom.MintRepository,
	passedProductLister mintdom.PassedProductLister,
	tokenMinter TokenMintPort,
) *MintUsecase {
	return &MintUsecase{
		pbRepo:              pbRepo,
		prodRepo:            prodRepo,
		inspRepo:            inspRepo,
		modelRepo:           modelRepo,
		tbRepo:              tbRepo,
		brandSvc:            brandSvc,
		mintRepo:            mintRepo,
		passedProductLister: passedProductLister,
		tokenMinter:         tokenMinter,
		inventoryUC:         nil,
		nameResolver:        nil,
	}
}

// DI 側で nameResolver を後から注入できるようにする
func (u *MintUsecase) SetNameResolver(r *resolver.NameResolver) {
	if u == nil {
		return
	}
	u.nameResolver = r
}

// ★ DI 側で InventoryUsecase（または互換の Upserter）を後から注入できるようにする
// ※ *usecase.InventoryUsecase が UpsertFromMintByModel を実装している前提
func (u *MintUsecase) SetInventoryUsecase(uc *appusecase.InventoryUsecase) {
	if u == nil {
		return
	}
	// コンパイル時に interface 実装を保証したいので代入時点でチェック
	var _ InventoryUpserter = uc
	u.inventoryUC = uc
}

// 互換: interface 注入したいケース用
func (u *MintUsecase) SetInventoryUpserter(up InventoryUpserter) {
	if u == nil {
		return
	}
	u.inventoryUC = up
}

// internal helper: createdBy(memberId) -> display name
// nameResolver が無い/解決できない場合は memberId を返す
func (u *MintUsecase) resolveCreatedByName(ctx context.Context, memberID string) string {
	memberID = strings.TrimSpace(memberID)
	if memberID == "" {
		return ""
	}

	if u != nil && u.nameResolver != nil {
		if name := strings.TrimSpace(u.nameResolver.ResolveMemberName(ctx, memberID)); name != "" {
			return name
		}
	}
	return memberID
}

// production から ProductBlueprintID を（存在すれば）取り出す
// ※ prodRepo の具体型に依存しないため、GetByID/Get を reflect で試す
func (u *MintUsecase) resolveProductBlueprintIDFromProduction(ctx context.Context, productionID string) string {
	if u == nil || u.prodRepo == nil {
		return ""
	}

	call := func(methodName string) (any, error) {
		rv := reflect.ValueOf(u.prodRepo)
		m := rv.MethodByName(methodName)
		if !m.IsValid() {
			return nil, errors.New("method not found: " + methodName)
		}
		out := m.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(productionID)})
		if len(out) != 2 {
			return nil, errors.New("unexpected return values from " + methodName)
		}
		if !out[1].IsNil() {
			if err, ok := out[1].Interface().(error); ok {
				return nil, err
			}
			return nil, errors.New("non-error type returned as error")
		}
		return out[0].Interface(), nil
	}

	var prod any
	if p, err := call("GetByID"); err == nil {
		prod = p
	} else if p, err := call("Get"); err == nil {
		prod = p
	} else {
		return ""
	}

	v := reflect.ValueOf(prod)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}

	for _, name := range []string{"ProductBlueprintID", "ProductBlueprintId"} {
		f := v.FieldByName(name)
		if !f.IsValid() {
			continue
		}
		if f.Kind() == reflect.String {
			return strings.TrimSpace(f.String())
		}
		if f.Kind() == reflect.Ptr && !f.IsNil() && f.Elem().Kind() == reflect.String {
			return strings.TrimSpace(f.Elem().String())
		}
	}

	return ""
}

// inspection batch を productionId から 1件取得（互換）
func (u *MintUsecase) loadInspectionBatchByProductionID(ctx context.Context, productionID string) (*inspectiondom.InspectionBatch, error) {
	if u == nil || u.inspRepo == nil {
		return nil, errors.New("inspection repo is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return nil, errors.New("productionID is empty")
	}

	// 1) GetByProductionID があれば最優先
	if getter, ok := any(u.inspRepo).(interface {
		GetByProductionID(ctx context.Context, productionID string) (inspectiondom.InspectionBatch, error)
	}); ok {
		b, err := getter.GetByProductionID(ctx, pid)
		if err != nil {
			return nil, err
		}
		return &b, nil
	}

	// 2) ListByProductionID のみでもOK
	if lister, ok := any(u.inspRepo).(interface {
		ListByProductionID(ctx context.Context, productionIDs []string) ([]inspectiondom.InspectionBatch, error)
	}); ok {
		list, err := lister.ListByProductionID(ctx, []string{pid})
		if err != nil {
			return nil, err
		}
		if len(list) == 0 {
			return nil, inspectiondom.ErrNotFound
		}
		// productionId 一致を優先
		for i := range list {
			if strings.TrimSpace(list[i].ProductionID) == pid {
				b := list[i]
				return &b, nil
			}
		}
		b := list[0]
		return &b, nil
	}

	return nil, errors.New("inspection repo does not support GetByProductionID/ListByProductionID")
}

// ErrCompanyIDMissing は context から companyId が解決できない場合のエラーです。
var ErrCompanyIDMissing = errors.New("companyId not found in context")

// ============================================================
// ★ 修復: POST /mint/requests/{mintRequestId}/mint 用
// - actorId は context memberId ではなく mint.createdBy を優先
// - 二重mint防止
// - mints に onchain結果（署名/ミントアドレス）を保存（フィールドが存在する場合）
// - inventories は modelId ごとに UpsertFromMintByModel を呼ぶ
// ============================================================

// MintFromMintRequest runs onchain mint for an existing mint request (docId = mintRequestID).
func (u *MintUsecase) MintFromMintRequest(ctx context.Context, mintRequestID string) (*tokendom.MintResult, error) {
	start := time.Now()

	log.Printf(
		"[mint_usecase] MintFromMintRequest start mintRequestId=%q usecase_nil=%t tokenMinter_nil=%t mintRepo_nil=%t tbRepo_nil=%t inventoryUC_nil=%t",
		strings.TrimSpace(mintRequestID),
		u == nil,
		u == nil || u.tokenMinter == nil,
		u == nil || u.mintRepo == nil,
		u == nil || u.tbRepo == nil,
		u == nil || u.inventoryUC == nil,
	)

	if u == nil {
		log.Printf("[mint_usecase] MintFromMintRequest abort reason=usecase_nil elapsed=%s", time.Since(start))
		return nil, errors.New("mint usecase is nil")
	}
	mintRequestID = strings.TrimSpace(mintRequestID)
	if mintRequestID == "" {
		log.Printf("[mint_usecase] MintFromMintRequest abort reason=empty_mintRequestId elapsed=%s", time.Since(start))
		return nil, errors.New("mintRequestID is empty")
	}
	if u.tokenMinter == nil {
		log.Printf("[mint_usecase] MintFromMintRequest abort reason=token_minter_nil mintRequestId=%q elapsed=%s", mintRequestID, time.Since(start))
		return nil, errors.New("token minter is nil")
	}
	if u.mintRepo == nil {
		log.Printf("[mint_usecase] MintFromMintRequest abort reason=mintRepo_nil mintRequestId=%q elapsed=%s", mintRequestID, time.Since(start))
		return nil, errors.New("mint repo is nil")
	}

	// helper: mintEnt を安全に取得する（Policy A: docId = productionId）
	loadMint := func() *mintdom.Mint {
		if u.mintRepo == nil {
			return nil
		}
		if getter, ok := any(u.mintRepo).(interface {
			GetByID(ctx context.Context, id string) (mintdom.Mint, error)
		}); ok {
			m, err := getter.GetByID(ctx, mintRequestID)
			if err == nil {
				return &m
			}
			if errors.Is(err, mintdom.ErrNotFound) {
				return nil
			}
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				return nil
			}
			log.Printf("[mint_usecase] MintFromMintRequest mint(GetByID) error id=%q err=%v", mintRequestID, err)
			return nil
		}
		if getter, ok := any(u.mintRepo).(interface {
			Get(ctx context.Context, id string) (mintdom.Mint, error)
		}); ok {
			m, err := getter.Get(ctx, mintRequestID)
			if err == nil {
				return &m
			}
			if errors.Is(err, mintdom.ErrNotFound) {
				return nil
			}
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				return nil
			}
			log.Printf("[mint_usecase] MintFromMintRequest mint(Get) error id=%q err=%v", mintRequestID, err)
			return nil
		}
		log.Printf("[mint_usecase] MintFromMintRequest mintRepo has no GetByID/Get (type=%T)", u.mintRepo)
		return nil
	}

	// 1) Mint を事前取得（actorId/ tokenBlueprintId / products / 二重mint判定に必須）
	mintEnt := loadMint()
	if mintEnt == nil {
		log.Printf("[mint_usecase] MintFromMintRequest abort reason=mint_not_found mintRequestId=%q elapsed=%s", mintRequestID, time.Since(start))
		return nil, mintdom.ErrNotFound
	}

	passedProductIDs := normalizeMintProducts(any(mintEnt.Products))

	log.Printf(
		"[mint_usecase] MintFromMintRequest loaded mint id=%q tokenBlueprintId=%q minted=%t products=%d createdBy=%q",
		strings.TrimSpace(mintEnt.ID),
		strings.TrimSpace(mintEnt.TokenBlueprintID),
		mintEnt.Minted,
		len(passedProductIDs),
		strings.TrimSpace(mintEnt.CreatedBy),
	)

	// actorID は mint.createdBy を優先
	actorID := strings.TrimSpace(mintEnt.CreatedBy)
	if actorID == "" {
		actorID = strings.TrimSpace(appusecase.MemberIDFromContext(ctx))
	}
	if actorID == "" {
		log.Printf("[mint_usecase] MintFromMintRequest abort reason=actor_missing mintRequestId=%q elapsed=%s", mintRequestID, time.Since(start))
		return nil, errors.New("actorID is missing (mint.createdBy and context memberId are empty)")
	}
	log.Printf("[mint_usecase] MintFromMintRequest actorId=%q mintRequestId=%q", actorID, mintRequestID)

	// 0) 二重mint防止
	if mintEnt.Minted {
		existing := buildMintResultFromMint(*mintEnt)
		log.Printf(
			"[mint_usecase] MintFromMintRequest skip onchain reason=already_minted mintRequestId=%q signature=%q mintAddress=%q elapsed=%s",
			mintRequestID,
			func() string {
				if existing == nil {
					return ""
				}
				return strings.TrimSpace(existing.Signature)
			}(),
			func() string {
				if existing == nil {
					return ""
				}
				return strings.TrimSpace(existing.MintAddress)
			}(),
			time.Since(start),
		)
		return existing, nil
	}

	// 2) オンチェーンミント実行
	log.Printf("[mint_usecase] MintFromMintRequest onchain mint start mintRequestId=%q", mintRequestID)

	onchainStart := time.Now()
	result, err := u.tokenMinter.MintFromMintRequest(ctx, mintRequestID)
	onchainElapsed := time.Since(onchainStart)

	if err != nil {
		log.Printf(
			"[mint_usecase] MintFromMintRequest onchain mint error mintRequestId=%q err=%v elapsed=%s totalElapsed=%s",
			mintRequestID, err, onchainElapsed, time.Since(start),
		)
		return nil, err
	}

	log.Printf(
		"[mint_usecase] MintFromMintRequest onchain mint ok mintRequestId=%q elapsed=%s signature=%q mintAddress=%q slot=%d",
		mintRequestID,
		onchainElapsed,
		func() string {
			if result == nil {
				return ""
			}
			return strings.TrimSpace(result.Signature)
		}(),
		func() string {
			if result == nil {
				return ""
			}
			return strings.TrimSpace(result.MintAddress)
		}(),
		func() uint64 {
			if result == nil {
				return 0
			}
			return result.Slot
		}(),
	)

	// 3) TokenBlueprint minted=true（未mint の場合のみ）
	tbID := strings.TrimSpace(mintEnt.TokenBlueprintID)
	if tbID != "" {
		tbStart := time.Now()
		errTB := u.markTokenBlueprintMinted(ctx, tbID, actorID)
		tbElapsed := time.Since(tbStart)

		if errTB != nil {
			log.Printf(
				"[mint_usecase] MintFromMintRequest markTokenBlueprintMinted error tokenBlueprintId=%q actorId=%q err=%v elapsed=%s",
				tbID, actorID, errTB, tbElapsed,
			)
		} else {
			log.Printf(
				"[mint_usecase] MintFromMintRequest markTokenBlueprintMinted ok tokenBlueprintId=%q actorId=%q elapsed=%s",
				tbID, actorID, tbElapsed,
			)
		}
	} else {
		log.Printf("[mint_usecase] MintFromMintRequest skip markTokenBlueprintMinted reason=empty_tokenBlueprintId")
	}

	// 4) mints 側 minted/mintedAt + onchain結果を更新（失敗してもログだけ）
	if u.mintRepo != nil {
		if updater, ok := any(u.mintRepo).(interface {
			Update(ctx context.Context, m mintdom.Mint) (mintdom.Mint, error)
		}); ok {
			now := time.Now().UTC()
			m := *mintEnt

			// Policy A: docId を必ず mintRequestId に揃える
			m.ID = mintRequestID
			setIfExistsString(&m, "InspectionID", mintRequestID)

			m.Minted = true
			m.MintedAt = &now

			if result != nil {
				sig := strings.TrimSpace(result.Signature)
				addr := strings.TrimSpace(result.MintAddress)

				if sig != "" {
					setIfExistsString(&m, "OnChainTxSignature", sig)
					setIfExistsString(&m, "OnchainTxSignature", sig)
					setIfExistsString(&m, "TxSignature", sig)
					setIfExistsString(&m, "Signature", sig)
				}
				if addr != "" {
					setIfExistsString(&m, "MintAddress", addr)
					setIfExistsString(&m, "OnChainMintAddress", addr)
					setIfExistsString(&m, "OnchainMintAddress", addr)
				}
			}

			updStart := time.Now()
			updated, errUpd := updater.Update(ctx, m)
			updElapsed := time.Since(updStart)

			if errUpd != nil {
				log.Printf(
					"[mint_usecase] MintFromMintRequest mintRepo.Update error mintRequestId=%q err=%v elapsed=%s",
					mintRequestID, errUpd, updElapsed,
				)
			} else {
				log.Printf(
					"[mint_usecase] MintFromMintRequest mintRepo.Update ok mintRequestId=%q minted=%t mintedAt=%v elapsed=%s",
					mintRequestID, updated.Minted, updated.MintedAt, updElapsed,
				)
			}
		} else {
			log.Printf("[mint_usecase] MintFromMintRequest skip mintRepo.Update reason=no_update_method")
		}
	} else {
		log.Printf("[mint_usecase] MintFromMintRequest skip mintRepo.Update reason=mintRepo_nil")
	}

	// 5) inventories Upsert（modelId ごとに UpsertFromMintByModel）
	if u.inventoryUC == nil {
		log.Printf("[mint_usecase] MintFromMintRequest inventoryUC is nil -> skip inventory upsert mintRequestId=%q", mintRequestID)
	} else {
		pbID := strings.TrimSpace(u.resolveProductBlueprintIDFromProduction(ctx, mintRequestID))

		log.Printf(
			"[mint_usecase] MintFromMintRequest inventory upsert(by-model) start mintRequestId=%q tokenBlueprintId=%q productBlueprintId=%q passedProducts=%d",
			mintRequestID, tbID, pbID, len(passedProductIDs),
		)

		if tbID == "" || pbID == "" || len(passedProductIDs) == 0 {
			log.Printf(
				"[mint_usecase] MintFromMintRequest inventory upsert(by-model) skip reason=missing_fields mintRequestId=%q tbID=%q pbID=%q products=%d",
				mintRequestID, tbID, pbID, len(passedProductIDs),
			)
		} else {
			// inspection から modelId を引いて、modelId ごとに productId を束ねる
			batch, berr := u.loadInspectionBatchByProductionID(ctx, mintRequestID)
			if berr != nil || batch == nil {
				log.Printf(
					"[mint_usecase] MintFromMintRequest inventory upsert(by-model) skip reason=inspection_load_failed mintRequestId=%q err=%v",
					mintRequestID, berr,
				)
			} else {
				passedSet := make(map[string]struct{}, len(passedProductIDs))
				for _, p := range passedProductIDs {
					passedSet[p] = struct{}{}
				}

				byModel := map[string][]string{}
				for _, it := range batch.Inspections {
					pid := strings.TrimSpace(it.ProductID)
					if pid == "" {
						continue
					}
					if _, ok := passedSet[pid]; !ok {
						continue
					}
					mid := strings.TrimSpace(it.ModelID)
					if mid == "" {
						// modelId がないデータは upsert できない（docId 方針に合わない）
						continue
					}
					byModel[mid] = append(byModel[mid], pid)
				}

				modelIDs := make([]string, 0, len(byModel))
				for mid := range byModel {
					modelIDs = append(modelIDs, mid)
				}
				sort.Strings(modelIDs)

				if len(modelIDs) == 0 {
					log.Printf(
						"[mint_usecase] MintFromMintRequest inventory upsert(by-model) skip reason=no_model_groups mintRequestId=%q passed=%d inspections=%d",
						mintRequestID, len(passedProductIDs), len(batch.Inspections),
					)
				} else {
					for _, mid := range modelIDs {
						pids := normalizeIDs(byModel[mid])
						if len(pids) == 0 {
							continue
						}

						invStart := time.Now()
						invEnt, invErr := u.inventoryUC.UpsertFromMintByModel(ctx, tbID, pbID, mid, pids)
						invElapsed := time.Since(invStart)

						if invErr != nil {
							log.Printf(
								"[mint_usecase] MintFromMintRequest inventory upsert(by-model) error mintRequestId=%q tokenBlueprintId=%q productBlueprintId=%q modelId=%q products=%d err=%v elapsed=%s",
								mintRequestID, tbID, pbID, mid, len(pids), invErr, invElapsed,
							)
							// ここで return しても良いが、現状は mint 成功を優先してログのみ
							continue
						}

						log.Printf(
							"[mint_usecase] MintFromMintRequest inventory upsert(by-model) ok inventoryId=%q modelId=%q accumulation=%d products=%d elapsed=%s",
							strings.TrimSpace(invEnt.ID),
							mid,
							invEnt.Accumulation,
							func() int {
								if invEnt.Products == nil {
									return 0
								}
								return len(invEnt.Products)
							}(),
							invElapsed,
						)
					}
				}
			}
		}
	}

	log.Printf(
		"[mint_usecase] MintFromMintRequest done mintRequestId=%q elapsed=%s result_nil=%t",
		mintRequestID, time.Since(start), result == nil,
	)

	return result, nil
}

// ============================================================
// Additional API: mints を inspectionIds(docId) で取得
// ============================================================

// ListMintsByInspectionIDs は、inspectionIds（= productionIds = docId）に紐づく mints を
// inspectionId をキーにした map で返します。
func (u *MintUsecase) ListMintsByInspectionIDs(
	ctx context.Context,
	inspectionIDs []string,
) (map[string]mintdom.Mint, error) {

	if u == nil {
		return nil, errors.New("mint usecase is nil")
	}
	if u.mintRepo == nil {
		return nil, errors.New("mint repo is nil")
	}

	seen := make(map[string]struct{}, len(inspectionIDs))
	ids := make([]string, 0, len(inspectionIDs))

	for _, id := range inspectionIDs {
		s := strings.TrimSpace(id)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		ids = append(ids, s)
	}

	if len(ids) == 0 {
		return map[string]mintdom.Mint{}, nil
	}

	sort.Strings(ids)

	// 最優先: mintRepo が docId 同一前提の ListByProductionID を持つ
	if lister, ok := u.mintRepo.(interface {
		ListByProductionID(ctx context.Context, productionIDs []string) (map[string]mintdom.Mint, error)
	}); ok {
		return lister.ListByProductionID(ctx, ids)
	}

	// 次点: GetByID / Get で docId を個別取得
	if getter, ok := u.mintRepo.(interface {
		GetByID(ctx context.Context, id string) (mintdom.Mint, error)
	}); ok {
		out := make(map[string]mintdom.Mint, len(ids))
		for _, id := range ids {
			m, err := getter.GetByID(ctx, id)
			if err != nil {
				if errors.Is(err, mintdom.ErrNotFound) || strings.Contains(strings.ToLower(err.Error()), "not found") {
					continue
				}
				return nil, err
			}
			out[id] = m
		}
		return out, nil
	}

	if getter, ok := u.mintRepo.(interface {
		Get(ctx context.Context, id string) (mintdom.Mint, error)
	}); ok {
		out := make(map[string]mintdom.Mint, len(ids))
		for _, id := range ids {
			m, err := getter.Get(ctx, id)
			if err != nil {
				if errors.Is(err, mintdom.ErrNotFound) || strings.Contains(strings.ToLower(err.Error()), "not found") {
					continue
				}
				return nil, err
			}
			out[id] = m
		}
		return out, nil
	}

	return nil, errors.New("mint repo does not support ListByProductionID/GetByID/Get")
}

// ============================================================
// Additional API: mints(list) を inspectionIds で取得し、名前解決して DTO を組み立てる
// ============================================================

func (u *MintUsecase) ListMintListRowsByInspectionIDs(
	ctx context.Context,
	inspectionIDs []string,
) (map[string]dto.MintListRowDTO, error) {

	if u == nil {
		return nil, errors.New("mint usecase is nil")
	}
	if u.mintRepo == nil {
		return nil, errors.New("mint repo is nil")
	}

	mintsByInspectionID, err := u.ListMintsByInspectionIDs(ctx, inspectionIDs)
	if err != nil {
		return nil, err
	}
	if len(mintsByInspectionID) == 0 {
		return map[string]dto.MintListRowDTO{}, nil
	}

	tbNameByID := map[string]string{}
	if u.tbRepo != nil {
		tbIDSet := map[string]struct{}{}
		for _, m := range mintsByInspectionID {
			tbID := strings.TrimSpace(m.TokenBlueprintID)
			if tbID == "" {
				continue
			}
			tbIDSet[tbID] = struct{}{}
		}

		tbIDs := make([]string, 0, len(tbIDSet))
		for id := range tbIDSet {
			tbIDs = append(tbIDs, id)
		}
		sort.Strings(tbIDs)

		for _, tbID := range tbIDs {
			tb, err := u.tbRepo.GetByID(ctx, tbID)
			if err != nil {
				continue
			}
			tbNameByID[tbID] = strings.TrimSpace(tb.Name)
		}
	}

	out := make(map[string]dto.MintListRowDTO, len(mintsByInspectionID))
	keys := make([]string, 0, len(mintsByInspectionID))
	for k := range mintsByInspectionID {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	log.Printf("[mint_usecase] ListMintListRowsByInspectionIDs start ids=%d mints=%d nameResolver=%t",
		len(inspectionIDs), len(keys), u.nameResolver != nil,
	)

	for _, inspectionID := range keys {
		m := mintsByInspectionID[inspectionID]

		iid := strings.TrimSpace(inspectionID)
		mintID := strings.TrimSpace(m.ID)
		tbID := strings.TrimSpace(m.TokenBlueprintID)

		tokenName := ""
		if tbID != "" {
			if n, ok := tbNameByID[tbID]; ok {
				tokenName = n
			}
		}

		createdByName := u.resolveCreatedByName(ctx, m.CreatedBy)

		var mintedAt *string
		if m.MintedAt != nil && !m.MintedAt.IsZero() {
			s := m.MintedAt.UTC().Format(time.RFC3339)
			mintedAt = &s
		}

		out[iid] = dto.MintListRowDTO{
			InspectionID:   iid,
			MintID:         mintID,
			TokenBlueprint: tbID,

			TokenName:     tokenName,
			CreatedByName: createdByName,
			MintedAt:      mintedAt,
		}
	}

	log.Printf("[mint_usecase] ListMintListRowsByInspectionIDs done out=%d sampleKey=%q",
		len(out),
		func() string {
			if len(keys) == 0 {
				return ""
			}
			return keys[0]
		}(),
	)

	return out, nil
}

func (u *MintUsecase) ListMintListRowsByProductionIDs(
	ctx context.Context,
	productionIDs []string,
) (map[string]dto.MintListRowDTO, error) {
	return u.ListMintListRowsByInspectionIDs(ctx, productionIDs)
}

// ============================================================
// Additional API: ProductBlueprint Patch 解決
// ============================================================

func (u *MintUsecase) GetProductBlueprintPatchByID(
	ctx context.Context,
	productBlueprintID string,
) (pbpdom.Patch, error) {

	if u == nil {
		return pbpdom.Patch{}, errors.New("mint usecase is nil")
	}
	if u.pbRepo == nil {
		return pbpdom.Patch{}, errors.New("productBlueprint repo is nil")
	}

	id := strings.TrimSpace(productBlueprintID)
	if id == "" {
		return pbpdom.Patch{}, errors.New("productBlueprintID is empty")
	}

	patch, err := u.pbRepo.GetPatchByID(ctx, id)
	if err != nil {
		return pbpdom.Patch{}, err
	}

	return patch, nil
}

// ============================================================
// model variations -> modelMeta（任意）
// ============================================================

type modelMetaLister interface {
	ListModelMetaByIDs(ctx context.Context, modelIDs []string) (map[string]qdto.MintModelMetaEntry, error)
}

type modelMetaGetter interface {
	GetModelMetaByID(ctx context.Context, modelID string) (*qdto.MintModelMetaEntry, error)
}

func (u *MintUsecase) resolveModelMetaByModelIDs(
	ctx context.Context,
	modelIDs []string,
) (map[string]qdto.MintModelMetaEntry, error) {

	if u == nil {
		return map[string]qdto.MintModelMetaEntry{}, nil
	}
	if u.modelRepo == nil {
		return map[string]qdto.MintModelMetaEntry{}, nil
	}

	seen := map[string]struct{}{}
	ids := make([]string, 0, len(modelIDs))
	for _, id := range modelIDs {
		s := strings.TrimSpace(id)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		ids = append(ids, s)
	}
	if len(ids) == 0 {
		return map[string]qdto.MintModelMetaEntry{}, nil
	}
	sort.Strings(ids)

	if l, ok := any(u.modelRepo).(modelMetaLister); ok {
		m, err := l.ListModelMetaByIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
		if m == nil {
			return map[string]qdto.MintModelMetaEntry{}, nil
		}
		for k, v := range m {
			if strings.TrimSpace(v.ModelID) == "" {
				v.ModelID = strings.TrimSpace(k)
				m[k] = v
			}
		}
		return m, nil
	}

	if g, ok := any(u.modelRepo).(modelMetaGetter); ok {
		out := make(map[string]qdto.MintModelMetaEntry, len(ids))
		for _, id := range ids {
			ent, err := g.GetModelMetaByID(ctx, id)
			if err != nil {
				continue
			}
			if ent == nil {
				continue
			}
			v := *ent
			if strings.TrimSpace(v.ModelID) == "" {
				v.ModelID = id
			}
			out[id] = v
		}
		return out, nil
	}

	return map[string]qdto.MintModelMetaEntry{}, nil
}

func (u *MintUsecase) ResolveModelMetaFromInspectionBatch(
	ctx context.Context,
	batch inspectiondom.InspectionBatch,
) (map[string]qdto.MintModelMetaEntry, error) {

	modelIDs := make([]string, 0, len(batch.Inspections))
	for _, it := range batch.Inspections {
		modelIDs = append(modelIDs, strings.TrimSpace(it.ModelID))
	}

	return u.resolveModelMetaByModelIDs(ctx, modelIDs)
}

// ============================================================
// ★ 修復: /mint/inspections/{id}/request を「従来どおりミントまで実行」に戻す
// - Inspection へ mintId を記録 + mints 作成
// - そのまま MintFromMintRequest を起動（オンチェーン + 更新）
// ============================================================

func (u *MintUsecase) UpdateRequestInfo(
	ctx context.Context,
	productionID string,
	tokenBlueprintID string,
	scheduledBurnDate *string,
) (inspectiondom.InspectionBatch, error) {

	var empty inspectiondom.InspectionBatch

	if u == nil {
		return empty, errors.New("mint usecase is nil")
	}
	if u.inspRepo == nil {
		return empty, errors.New("inspection repo is nil")
	}
	if u.mintRepo == nil {
		return empty, errors.New("mint repo is nil")
	}
	if u.passedProductLister == nil {
		return empty, errors.New("passedProductLister is nil")
	}
	if u.tbRepo == nil {
		return empty, errors.New("tokenBlueprint repo is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return empty, errors.New("productionID is empty")
	}

	tbID := strings.TrimSpace(tokenBlueprintID)
	if tbID == "" {
		return empty, errors.New("tokenBlueprintID is empty")
	}

	memberID := strings.TrimSpace(appusecase.MemberIDFromContext(ctx))
	if memberID == "" {
		return empty, errors.New("memberID not found in context")
	}

	now := time.Now().UTC()

	tb, err := u.tbRepo.GetByID(ctx, tbID)
	if err != nil {
		return empty, err
	}
	brandID := strings.TrimSpace(tb.BrandID)
	if brandID == "" {
		return empty, errors.New("brandID is empty on tokenBlueprint")
	}

	passedProductIDs, err := u.passedProductLister.ListPassedProductIDsByProductionID(ctx, pid)
	if err != nil {
		return empty, err
	}
	if len(passedProductIDs) == 0 {
		return empty, errors.New("no passed products for this production")
	}

	// mint entity 作成
	mintEntity, err := mintdom.NewMint(
		pid,
		brandID,
		tbID,
		passedProductIDs,
		memberID,
		now,
	)
	if err != nil {
		return empty, err
	}

	// ★ Policy A: docId = productionId（必ず揃える）
	mintEntity.ID = pid
	setIfExistsString(&mintEntity, "InspectionID", pid)

	// minted は request 作成時は必ず false（念のため）
	mintEntity.Minted = false
	mintEntity.MintedAt = nil

	if scheduledBurnDate != nil {
		if s := strings.TrimSpace(*scheduledBurnDate); s != "" {
			t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
			if err != nil {
				return empty, errors.New("invalid scheduledBurnDate format (expected YYYY-MM-DD)")
			}
			utc := t.UTC()
			mintEntity.ScheduledBurnDate = &utc
		}
	}

	// Create は Policy A の docId 固定（productionId）で保存される
	savedMint, err := u.mintRepo.Create(ctx, mintEntity)
	if err != nil {
		return empty, err
	}

	mid := strings.TrimSpace(savedMint.ID)
	if mid == "" {
		return empty, errors.New("saved mintID is empty")
	}

	// Inspection に mintId を紐付け（= productionId と同値になる想定）
	batch, err := u.inspRepo.UpdateMintID(ctx, pid, &mid)
	if err != nil {
		return empty, err
	}

	log.Printf("[mint_usecase] UpdateRequestInfo done (request created) productionId=%q mintId=%q tokenBlueprintId=%q passedProducts=%d",
		pid, mid, tbID, len(passedProductIDs),
	)

	// ✅ mint request 作成直後に MintFromMintRequest を自動実行（維持）
	if u.tokenMinter == nil {
		return empty, errors.New("token minter is nil")
	}

	if _, err := u.MintFromMintRequest(ctx, pid); err != nil {
		log.Printf("[mint_usecase] UpdateRequestInfo auto MintFromMintRequest failed productionId=%q err=%v", pid, err)
		return empty, err
	}

	return batch, nil
}

func (u *MintUsecase) markTokenBlueprintMinted(ctx context.Context, tokenBlueprintID string, actorID string) error {
	if u == nil {
		return errors.New("mint usecase is nil")
	}
	if u.tbRepo == nil {
		return errors.New("tokenBlueprint repo is nil")
	}

	id := strings.TrimSpace(tokenBlueprintID)
	if id == "" {
		return errors.New("tokenBlueprintID is empty")
	}

	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return errors.New("actorID is empty")
	}

	tb, err := u.tbRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if tb.Minted {
		return nil
	}

	now := time.Now().UTC()
	minted := true
	updatedBy := actorID

	_, err = u.tbRepo.Update(ctx, id, tbdom.UpdateTokenBlueprintInput{
		Minted:    &minted,
		UpdatedAt: &now,
		UpdatedBy: &updatedBy,
	})
	return err
}

// ============================================================
// Additional API: Brand 一覧（current company）
// ============================================================

func (u *MintUsecase) ListBrandsForCurrentCompany(
	ctx context.Context,
	page branddom.Page,
) (branddom.PageResult[branddom.Brand], error) {

	var empty branddom.PageResult[branddom.Brand]

	if u == nil {
		return empty, errors.New("mint usecase is nil")
	}
	if u.brandSvc == nil {
		return empty, errors.New("brand service is nil")
	}

	companyID := strings.TrimSpace(appusecase.CompanyIDFromContext(ctx))
	if companyID == "" {
		return empty, ErrCompanyIDMissing
	}

	return u.brandSvc.ListByCompanyID(ctx, companyID, page)
}

// ============================================================
// Additional API: TokenBlueprint 一覧（brandId フィルタ）
// ============================================================

func (u *MintUsecase) ListTokenBlueprintsByBrand(
	ctx context.Context,
	brandID string,
	page tbdom.Page,
) (tbdom.PageResult, error) {

	var empty tbdom.PageResult

	if u == nil {
		return empty, errors.New("mint usecase is nil")
	}
	if u.tbRepo == nil {
		return empty, errors.New("tokenBlueprint repo is nil")
	}

	brandID = strings.TrimSpace(brandID)
	if brandID == "" {
		return empty, errors.New("brandID is empty")
	}

	return tbdom.ListByBrandID(ctx, u.tbRepo, brandID, page)
}

// ListInspectionBatchesByProductionIDs fetches inspection batches by production docIds.
func (u *MintUsecase) ListInspectionBatchesByProductionIDs(
	ctx context.Context,
	productionIDs []string,
) ([]inspectiondom.InspectionBatch, error) {

	if u == nil {
		return nil, errors.New("mint usecase is nil")
	}
	if u.inspRepo == nil {
		return nil, errors.New("inspection repo is nil")
	}

	seen := make(map[string]struct{}, len(productionIDs))
	ids := make([]string, 0, len(productionIDs))

	for _, id := range productionIDs {
		s := strings.TrimSpace(id)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		ids = append(ids, s)
	}

	if len(ids) == 0 {
		return []inspectiondom.InspectionBatch{}, nil
	}

	sort.Strings(ids)

	return u.inspRepo.ListByProductionID(ctx, ids)
}

// ============================================================
// Detail API for GET /mint/inspections/{productionId}
// ============================================================

func (u *MintUsecase) GetMintRequestDetail(
	ctx context.Context,
	productionID string,
) (inspectiondom.InspectionBatch, error) {

	var empty inspectiondom.InspectionBatch

	if u == nil {
		return empty, errors.New("mint usecase is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return empty, errors.New("productionID is empty")
	}

	batches, err := u.ListInspectionBatchesByProductionIDs(ctx, []string{pid})
	if err != nil {
		return empty, err
	}
	if len(batches) == 0 {
		return empty, inspectiondom.ErrNotFound
	}

	for _, b := range batches {
		if strings.TrimSpace(b.ProductionID) == pid {
			return b, nil
		}
	}

	return batches[0], nil
}

// ============================================================
// Local helpers
// ============================================================

// normalizeMintProducts は、Mint.Products の型が []string / map[string]string のどちらでも
// 取り出せるようにしておくためのヘルパです（将来の型変更に強くする）。
func normalizeMintProducts(raw any) []string {
	if raw == nil {
		return []string{}
	}

	rv := reflect.ValueOf(raw)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return []string{}
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		out := make([]string, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			it := rv.Index(i)
			if it.Kind() == reflect.String {
				s := strings.TrimSpace(it.String())
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return normalizeIDs(out)

	case reflect.Map:
		out := make([]string, 0, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			k := iter.Key()
			if k.Kind() == reflect.String {
				s := strings.TrimSpace(k.String())
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return normalizeIDs(out)

	default:
		return []string{}
	}
}

func normalizeIDs(raw []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// setIfExistsString は struct の string field が存在する場合だけ値をセットする（互換用）
func setIfExistsString(target any, fieldName string, value string) {
	rv := reflect.ValueOf(target)
	if !rv.IsValid() {
		return
	}
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}
	f := rv.FieldByName(fieldName)
	if !f.IsValid() || !f.CanSet() || f.Kind() != reflect.String {
		return
	}
	f.SetString(strings.TrimSpace(value))
}

// getIfExistsString は struct に fieldName の string フィールドがあれば取得する（互換用）
func getIfExistsString(target any, fieldName string) string {
	rv := reflect.ValueOf(target)
	if !rv.IsValid() {
		return ""
	}
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return ""
	}
	f := rv.FieldByName(fieldName)
	if !f.IsValid() {
		return ""
	}
	if f.Kind() == reflect.String {
		return strings.TrimSpace(f.String())
	}
	if f.Kind() == reflect.Ptr && !f.IsNil() && f.Elem().Kind() == reflect.String {
		return strings.TrimSpace(f.Elem().String())
	}
	return ""
}

// buildMintResultFromMint は minted 済みの Mint から、保存済みの署名/アドレスがあれば返す（無ければ空）
func buildMintResultFromMint(m mintdom.Mint) *tokendom.MintResult {
	sig := ""
	for _, name := range []string{
		"OnChainTxSignature",
		"OnchainTxSignature",
		"TxSignature",
		"Signature",
	} {
		if v := getIfExistsString(m, name); v != "" {
			sig = v
			break
		}
	}

	addr := ""
	for _, name := range []string{
		"MintAddress",
		"OnChainMintAddress",
		"OnchainMintAddress",
	} {
		if v := getIfExistsString(m, name); v != "" {
			addr = v
			break
		}
	}

	return &tokendom.MintResult{
		Signature:   sig,
		MintAddress: addr,
		Slot:        0,
	}
}
