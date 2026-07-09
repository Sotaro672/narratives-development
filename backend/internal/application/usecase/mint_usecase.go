// backend/internal/application/usecase/mint_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	invdom "narratives/internal/domain/inventory"
	mintdom "narratives/internal/domain/mint"
	tokendom "narratives/internal/domain/token"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

var ErrCompanyIDMissing = errors.New("companyId not found in context")

// ============================================================
// Mint request ports
// ============================================================

// MintRequestForUsecase は、MintUsecase が mint 実行フローを進めるために
// 必要となる MintRequest 情報だけを集約した DTO です。
type MintRequestForUsecase struct {
	ID string

	// TokenBlueprintID は、metadata URI の確保や tokenBlueprint minted 化に使います。
	TokenBlueprintID string

	// ActorID は、metadata URI 確保や tokenBlueprint minted 化の実行者として使います。
	ActorID string

	// 受取先アドレス（ブランドウォレット等）
	// NOTE:
	// - これは「NFT/トークンを受け取るアドレス」であり、FeePayer（ガス支払い）ではありません。
	// - FeePayer はインフラ側（mint/transfer 実装）で master wallet に統一しています。
	ToAddress string

	// productId ごとに 1 ミントしたい場合の productId 一覧。
	ProductIDs []string

	BlueprintName   string
	BlueprintSymbol string

	MetadataURI string
}

// MintRequestPort は、MintUsecase から見た「ミント対象 MintRequest」の
// 取得および更新を行うためのポートです。
//
// NOTE:
// - 旧フローでは MarkProductsAsMinted が親 Mint も minted=true にしていました。
// - 1件ずつmint化後は、productId単位の token 保存と親 Mint の完了判定を分ける必要があります。
// - 可能なら MintProductMintRecorder を実装し、MarkProductsAsMinted は互換用に留めてください。
type MintRequestPort interface {
	// LoadForMinting:
	// - ミント実行に必要な情報をロードします。
	// - TokenBlueprintID / ActorID / ToAddress / ProductIDs / BlueprintName /
	//   BlueprintSymbol / MetadataURI を返す想定です。
	LoadForMinting(ctx context.Context, id string) (*MintRequestForUsecase, error)

	// MarkProductsAsMinted:
	// - 旧互換用。
	// - productId ごとの mint 結果一覧を保存します。
	// - 新フローでは、親 Mint を全件完了前に minted=true にしないよう実装側の修正が必要です。
	MarkProductsAsMinted(ctx context.Context, id string, minted []MintedTokenForUsecase) error
}

// MintProductMintRecorder は、1 product の mint 成功結果を保存するための新ポートです。
//
// 推奨:
// - Firestore 実装側では productId と mintAddress の 1:1 token record を保存します。
// - 親 Mint の minted=true 更新はここでは行わず、全 task 完了時に MintUsecase 側で行います。
type MintProductMintRecorder interface {
	RecordProductAsMinted(
		ctx context.Context,
		mintID string,
		minted MintedTokenForUsecase,
	) error
}

// ============================================================
// Token mint dependency
// ============================================================

type TokenMintPort interface {
	MintProducts(ctx context.Context, input MintProductsInput) ([]MintedTokenForUsecase, error)
}

// ============================================================
// Mint task dependency
// ============================================================

// MintTaskEnqueuer は、Cloud Tasks 等に「次の1件mint処理」を投入するためのポートです。
type MintTaskEnqueuer interface {
	EnqueueMintTask(ctx context.Context, mintID string) error
}

// ============================================================
// TokenBlueprint dependencies
// ============================================================

type TokenBlueprintMetadataEnsurer interface {
	EnsureMetadataURI(ctx context.Context, tb *tbdom.TokenBlueprint, actorID string) (*tbdom.TokenBlueprint, error)
}

type TokenBlueprintMintMarker interface {
	MarkTokenBlueprintMinted(
		ctx context.Context,
		tokenBlueprintID string,
		actorID string,
	) (*tbdom.TokenBlueprint, error)
}

// ============================================================
// Inventory dependency
// ============================================================

type InventoryUpserter interface {
	UpsertFromMint(
		ctx context.Context,
		tokenBlueprintID string,
		productBlueprintID string,
		productIDs []string,
	) ([]invdom.Mint, error)
}

// ============================================================
// MintResultMapper
// ============================================================

type MintResultMapper struct{}

func NewMintResultMapper() *MintResultMapper {
	return &MintResultMapper{}
}

func (m *MintResultMapper) FromMint(ent mintdom.Mint) *tokendom.MintResult {
	return &tokendom.MintResult{
		Signature:   ent.OnChainTxSignature,
		MintAddress: "",
		Slot:        0,
	}
}

func (m *MintResultMapper) ApplyOnchainResult(ent *mintdom.Mint, result *tokendom.MintResult) error {
	if ent == nil {
		return errors.New("mint entity is nil")
	}
	if result == nil {
		return nil
	}

	if result.Signature != "" {
		ent.OnChainTxSignature = result.Signature
	}

	return nil
}

// ============================================================
// MintUsecase
// ============================================================

type MintUsecase struct {
	prodRepo mintdom.MintProductionRepo

	tbRepo tbdom.RepositoryPort

	mintRepo     mintdom.MintRepository
	mintTaskRepo mintdom.MintProductTaskRepository

	mintRequestPort       MintRequestPort
	mintProductMintRecord MintProductMintRecorder

	mintTaskEnqueuer MintTaskEnqueuer

	mintResultMapper *MintResultMapper

	passedProductLister mintdom.PassedProductLister

	tokenMinter TokenMintPort

	inventoryUC InventoryUpserter

	tbMetadataEnsurer TokenBlueprintMetadataEnsurer
	tbMintMarker      TokenBlueprintMintMarker
}

func NewMintUsecase(
	prodRepo mintdom.MintProductionRepo,
	tbRepo tbdom.RepositoryPort,
	mintRepo mintdom.MintRepository,
	passedProductLister mintdom.PassedProductLister,
	tokenMinter TokenMintPort,
) *MintUsecase {
	var mintRequestPort MintRequestPort
	if p, ok := any(mintRepo).(MintRequestPort); ok {
		mintRequestPort = p
	}

	var mintProductMintRecord MintProductMintRecorder
	if p, ok := any(mintRepo).(MintProductMintRecorder); ok {
		mintProductMintRecord = p
	}

	return &MintUsecase{
		prodRepo:              prodRepo,
		tbRepo:                tbRepo,
		mintRepo:              mintRepo,
		mintTaskRepo:          nil,
		mintRequestPort:       mintRequestPort,
		mintProductMintRecord: mintProductMintRecord,
		mintTaskEnqueuer:      nil,
		mintResultMapper:      NewMintResultMapper(),
		passedProductLister:   passedProductLister,
		tokenMinter:           tokenMinter,
		inventoryUC:           nil,
		tbMetadataEnsurer:     nil,
		tbMintMarker:          nil,
	}
}

func (u *MintUsecase) SetInventoryUsecase(uc *InventoryUsecase) {
	if u == nil {
		return
	}

	var _ InventoryUpserter = uc
	u.inventoryUC = uc
}

func (u *MintUsecase) SetMintTaskRepository(repo mintdom.MintProductTaskRepository) {
	if u == nil {
		return
	}
	u.mintTaskRepo = repo
}

func (u *MintUsecase) SetMintTaskEnqueuer(enqueuer MintTaskEnqueuer) {
	if u == nil {
		return
	}
	u.mintTaskEnqueuer = enqueuer
}

func (u *MintUsecase) SetMintProductMintRecorder(recorder MintProductMintRecorder) {
	if u == nil {
		return
	}
	u.mintProductMintRecord = recorder
}

func (u *MintUsecase) SetTokenBlueprintMetadataEnsurer(e TokenBlueprintMetadataEnsurer) {
	if u == nil {
		return
	}
	u.tbMetadataEnsurer = e
}

func (u *MintUsecase) SetTokenBlueprintMintMarker(marker TokenBlueprintMintMarker) {
	if u == nil {
		return
	}
	u.tbMintMarker = marker
}

// UpdateRequestInfo は mint request を起票し、productId 単位の mint task を作成します。
//
// 旧実装:
// - mint request 作成後、同じHTTPリクエスト内で MintFromMintRequest を同期実行していました。
// - 18件一括mint時に RPC 429 / 504 timeout の原因になります。
//
// 新実装:
// - mint request 作成
// - productId 単位の MintProductTask を作成
// - 最初の worker task を enqueue
// - HTTP には即時返却
func (u *MintUsecase) UpdateRequestInfo(
	ctx context.Context,
	productionID string,
	tokenBlueprintID string,
	scheduledBurnDate *string,
) (*tokendom.MintResult, error) {
	if u == nil {
		return nil, errors.New("mint usecase is nil")
	}
	if u.mintRepo == nil {
		return nil, errors.New("mint repo is nil")
	}
	if u.mintTaskRepo == nil {
		return nil, errors.New("mint task repo is nil")
	}
	if u.passedProductLister == nil {
		return nil, errors.New("passedProductLister is nil")
	}
	if u.tbRepo == nil {
		return nil, errors.New("tokenBlueprint repo is nil")
	}

	pid := productionID
	if pid == "" {
		return nil, errors.New("productionID is empty")
	}

	tbID := tokenBlueprintID
	if tbID == "" {
		return nil, errors.New("tokenBlueprintID is empty")
	}

	memberID := MemberIDFromContext(ctx)
	if memberID == "" {
		return nil, errors.New("memberID not found in context")
	}

	now := time.Now().UTC()

	tb, err := u.tbRepo.GetByID(ctx, tbID)
	if err != nil {
		return nil, err
	}
	if tb == nil {
		return nil, errors.New("tokenBlueprint not found")
	}

	brandID := tb.BrandID
	if brandID == "" {
		return nil, errors.New("brandID is empty on tokenBlueprint")
	}

	passedProductIDs, err := u.passedProductLister.ListPassedProductIDsByProductionID(ctx, pid)
	if err != nil {
		return nil, err
	}
	if len(passedProductIDs) == 0 {
		return nil, errors.New("no passed products for this production")
	}

	mintEntity, err := mintdom.NewMint(
		pid,
		brandID,
		tbID,
		passedProductIDs,
		memberID,
		now,
	)
	if err != nil {
		return nil, err
	}

	mintEntity.ID = pid
	mintEntity.Minted = false
	mintEntity.MintedAt = nil

	if scheduledBurnDate != nil {
		if s := *scheduledBurnDate; s != "" {
			t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
			if err != nil {
				return nil, errors.New("invalid scheduledBurnDate format (expected YYYY-MM-DD)")
			}
			utc := t.UTC()
			mintEntity.ScheduledBurnDate = &utc
		}
	}

	if err := mintEntity.MarkQueued(); err != nil {
		return nil, err
	}

	if _, err := u.mintRepo.Create(ctx, mintEntity); err != nil {
		return nil, err
	}

	if _, err := u.mintTaskRepo.CreateTasks(ctx, pid, passedProductIDs); err != nil {
		return nil, fmt.Errorf("create mint product tasks: %w", err)
	}

	if u.mintTaskEnqueuer != nil {
		if err := u.mintTaskEnqueuer.EnqueueMintTask(ctx, pid); err != nil {
			return nil, fmt.Errorf("enqueue mint task: %w", err)
		}
	}

	// 既存 handler 互換のため MintResult を返します。
	// 本来は handler 側を 202 Accepted + queued DTO に変更するのが理想です。
	return &tokendom.MintResult{
		Signature:   "QUEUED",
		MintAddress: "",
		Slot:        0,
	}, nil
}

func (u *MintUsecase) resolveProductBlueprintIDFromProduction(ctx context.Context, productionID string) string {
	if u == nil || u.prodRepo == nil {
		return ""
	}
	if productionID == "" {
		return ""
	}

	productBlueprintID, err := u.prodRepo.GetProductBlueprintIDByProductionID(ctx, productionID)
	if err != nil {
		return ""
	}

	return productBlueprintID
}

func lastMintResult(minted []MintedTokenForUsecase) *tokendom.MintResult {
	for i := len(minted) - 1; i >= 0; i-- {
		if minted[i].Result != nil {
			return minted[i].Result
		}
	}
	return nil
}

func (u *MintUsecase) ensureMetadataURI(
	ctx context.Context,
	tokenBlueprintID string,
	actorID string,
	currentMetadataURI string,
) (string, error) {
	metadataURI := currentMetadataURI

	tbID := tokenBlueprintID
	if tbID == "" {
		return metadataURI, nil
	}

	if u.tbMetadataEnsurer == nil {
		return metadataURI, nil
	}

	if u.tbRepo == nil {
		return "", fmt.Errorf("tokenBlueprint repo is nil")
	}

	tb, err := u.tbRepo.GetByID(ctx, tbID)
	if err != nil {
		return "", fmt.Errorf("get tokenBlueprint for metadata ensure: %w", err)
	}
	if tb == nil {
		return "", fmt.Errorf("tokenBlueprint not found (id=%s)", tbID)
	}

	updated, err := u.tbMetadataEnsurer.EnsureMetadataURI(ctx, tb, actorID)
	if err != nil {
		return "", fmt.Errorf("ensure metadata uri: %w", err)
	}
	if updated == nil {
		updated = tb
	}

	return updated.MetadataURI, nil
}

// MintFromMintRequest は旧互換用です。
//
// 新フローでは、HTTPリクエストからこのメソッドを直接呼ばず、
// ExecuteNextMintTask を worker / Cloud Tasks から呼び出してください。
func (u *MintUsecase) MintFromMintRequest(ctx context.Context, mintRequestID string) (*tokendom.MintResult, error) {
	return u.ExecuteNextMintTask(ctx, mintRequestID)
}

// ExecuteNextMintTask は mintID に紐づく次の実行可能 task を1件だけ処理します。
//
// フロー:
//  1. 親 Mint を取得
//  2. 次の PENDING / FAILED_RETRYABLE task を1件取得
//  3. task を MINTING に更新
//  4. productId 1件だけ on-chain mint
//  5. token record / task / inventory を更新
//  6. 未完了 task が残っていれば次の worker task を enqueue
//  7. 全件完了なら親 Mint を MINTED にする
func (u *MintUsecase) ExecuteNextMintTask(ctx context.Context, mintRequestID string) (*tokendom.MintResult, error) {
	log.Printf("[mint-task-usecase] start mintID=%s", mintRequestID)

	if u == nil {
		log.Printf("[mint-task-usecase] failed mintID=%s error=mint usecase is nil", mintRequestID)
		return nil, errors.New("mint usecase is nil")
	}
	if mintRequestID == "" {
		log.Printf("[mint-task-usecase] failed mintID=%s error=mintRequestID is empty", mintRequestID)
		return nil, errors.New("mintRequestID is empty")
	}
	if u.mintRepo == nil {
		log.Printf("[mint-task-usecase] failed mintID=%s error=mint repo is nil", mintRequestID)
		return nil, errors.New("mint repo is nil")
	}
	if u.mintTaskRepo == nil {
		log.Printf("[mint-task-usecase] failed mintID=%s error=mint task repo is nil", mintRequestID)
		return nil, errors.New("mint task repo is nil")
	}
	if u.mintResultMapper == nil {
		log.Printf("[mint-task-usecase] failed mintID=%s error=mint result mapper is nil", mintRequestID)
		return nil, errors.New("mint result mapper is nil")
	}

	log.Printf("[mint-task-usecase] GetByID start mintID=%s", mintRequestID)

	mintEntValue, err := u.mintRepo.GetByID(ctx, mintRequestID)
	if err != nil {
		log.Printf("[mint-task-usecase] GetByID failed mintID=%s error=%v", mintRequestID, err)
		return nil, err
	}
	mintEnt := &mintEntValue

	log.Printf(
		"[mint-task-usecase] GetByID ok mintID=%s status=%s minted=%t productCount=%d tokenBlueprintID=%s brandID=%s",
		mintRequestID,
		mintEnt.Status,
		mintEnt.Minted,
		len(mintEnt.Products),
		mintEnt.TokenBlueprintID,
		mintEnt.BrandID,
	)

	if mintEnt.Minted {
		log.Printf("[mint-task-usecase] parent already minted mintID=%s", mintRequestID)
		return u.mintResultMapper.FromMint(*mintEnt), nil
	}

	tbID := mintEnt.TokenBlueprintID
	if tbID == "" {
		log.Printf("[mint-task-usecase] failed mintID=%s error=tokenBlueprintID is empty on mint", mintRequestID)
		return nil, errors.New("tokenBlueprintID is empty on mint")
	}

	log.Printf("[mint-task-usecase] resolve productBlueprintID start mintID=%s", mintRequestID)

	pbID := u.resolveProductBlueprintIDFromProduction(ctx, mintRequestID)
	if pbID == "" {
		log.Printf("[mint-task-usecase] resolve productBlueprintID failed mintID=%s", mintRequestID)
		return nil, errors.New("productBlueprintID is empty (cannot upsert inventory)")
	}

	log.Printf("[mint-task-usecase] resolve productBlueprintID ok mintID=%s productBlueprintID=%s", mintRequestID, pbID)

	if len(mintEnt.Products) == 0 {
		log.Printf("[mint-task-usecase] failed mintID=%s error=no products for this mint request", mintRequestID)
		return nil, errors.New("no products for this mint request")
	}

	if u.tokenMinter == nil {
		log.Printf("[mint-task-usecase] failed mintID=%s error=token minter is nil", mintRequestID)
		return nil, errors.New("token minter is nil")
	}
	if u.mintRequestPort == nil {
		log.Printf("[mint-task-usecase] failed mintID=%s error=mint request port is nil", mintRequestID)
		return nil, errors.New("mint request port is nil")
	}

	log.Printf("[mint-task-usecase] mark parent minting start mintID=%s", mintRequestID)

	if err := mintEnt.MarkMinting(); err == nil {
		if _, updateErr := u.mintRepo.Update(ctx, *mintEnt); updateErr != nil {
			log.Printf("[mint-task-usecase] mark parent minting failed mintID=%s error=%v", mintRequestID, updateErr)
			return nil, fmt.Errorf("mark parent minting: %w", updateErr)
		}
		log.Printf("[mint-task-usecase] mark parent minting ok mintID=%s", mintRequestID)
	} else {
		log.Printf("[mint-task-usecase] mark parent minting skipped mintID=%s error=%v", mintRequestID, err)
	}

	log.Printf("[mint-task-usecase] LoadForMinting start mintID=%s", mintRequestID)

	req, err := u.mintRequestPort.LoadForMinting(ctx, mintRequestID)
	if err != nil {
		log.Printf("[mint-task-usecase] LoadForMinting failed mintID=%s error=%v", mintRequestID, err)
		return nil, fmt.Errorf("load mint request for minting: %w", err)
	}
	if req == nil {
		log.Printf("[mint-task-usecase] LoadForMinting failed mintID=%s error=request is nil", mintRequestID)
		return nil, fmt.Errorf("mint request %s is nil", mintRequestID)
	}

	log.Printf(
		"[mint-task-usecase] LoadForMinting ok mintID=%s reqID=%s productCount=%d tokenBlueprintID=%s actorID=%s hasToAddress=%t hasMetadataURI=%t name=%q symbol=%q",
		mintRequestID,
		req.ID,
		len(req.ProductIDs),
		req.TokenBlueprintID,
		req.ActorID,
		req.ToAddress != "",
		req.MetadataURI != "",
		req.BlueprintName,
		req.BlueprintSymbol,
	)

	reqID := req.ID
	if reqID == "" {
		reqID = mintRequestID
	}

	reqTBID := req.TokenBlueprintID
	if reqTBID == "" {
		reqTBID = tbID
	}

	actorID := req.ActorID
	if actorID == "" {
		actorID = mintEnt.CreatedBy
	}
	if actorID == "" {
		actorID = MemberIDFromContext(ctx)
	}

	log.Printf(
		"[mint-task-usecase] ensureMetadataURI start mintID=%s tokenBlueprintID=%s actorID=%s hasCurrentMetadataURI=%t hasEnsurer=%t",
		mintRequestID,
		reqTBID,
		actorID,
		req.MetadataURI != "",
		u.tbMetadataEnsurer != nil,
	)

	metadataURI, err := u.ensureMetadataURI(
		ctx,
		reqTBID,
		actorID,
		req.MetadataURI,
	)
	if err != nil {
		log.Printf("[mint-task-usecase] ensureMetadataURI failed mintID=%s error=%v", mintRequestID, err)
		return nil, err
	}
	if metadataURI == "" {
		log.Printf("[mint-task-usecase] ensureMetadataURI failed mintID=%s error=empty MetadataURI", mintRequestID)
		return nil, fmt.Errorf("mint request %s has empty MetadataURI", reqID)
	}

	log.Printf("[mint-task-usecase] ensureMetadataURI ok mintID=%s hasMetadataURI=%t", mintRequestID, metadataURI != "")

	toAddress := req.ToAddress
	if toAddress == "" {
		log.Printf("[mint-task-usecase] failed mintID=%s error=empty ToAddress", mintRequestID)
		return nil, fmt.Errorf("mint request %s has empty ToAddress", reqID)
	}

	name := req.BlueprintName
	symbol := req.BlueprintSymbol
	if name == "" || symbol == "" {
		log.Printf("[mint-task-usecase] failed mintID=%s error=empty name or symbol name=%q symbol=%q", mintRequestID, name, symbol)
		return nil, fmt.Errorf("mint request %s has empty name or symbol", reqID)
	}

	log.Printf("[mint-task-usecase] GetNextExecutableTask start mintID=%s", mintRequestID)

	task, err := u.mintTaskRepo.GetNextExecutableTask(ctx, mintRequestID)
	if err != nil {
		if errors.Is(err, mintdom.ErrMintProductTaskNotFound) {
			log.Printf("[mint-task-usecase] GetNextExecutableTask not found mintID=%s; finalizing if all completed", mintRequestID)
			return u.finalizeMintIfAllTasksCompleted(ctx, mintEnt, reqTBID, actorID)
		}
		log.Printf("[mint-task-usecase] GetNextExecutableTask failed mintID=%s error=%v", mintRequestID, err)
		return nil, fmt.Errorf("get next executable mint task: %w", err)
	}

	log.Printf(
		"[mint-task-usecase] GetNextExecutableTask ok mintID=%s productID=%s status=%s attemptCount=%d",
		mintRequestID,
		task.ProductID,
		task.Status,
		task.AttemptCount,
	)

	log.Printf("[mint-task-usecase] MarkMinting task start mintID=%s productID=%s", mintRequestID, task.ProductID)

	task, err = u.mintTaskRepo.MarkMinting(ctx, mintRequestID, task.ProductID)
	if err != nil {
		log.Printf("[mint-task-usecase] MarkMinting task failed mintID=%s productID=%s error=%v", mintRequestID, task.ProductID, err)
		return nil, fmt.Errorf("mark mint product task minting: %w", err)
	}

	log.Printf(
		"[mint-task-usecase] MarkMinting task ok mintID=%s productID=%s status=%s attemptCount=%d",
		mintRequestID,
		task.ProductID,
		task.Status,
		task.AttemptCount,
	)

	log.Printf("[mint-task-usecase] MintProducts start mintID=%s productID=%s", mintRequestID, task.ProductID)

	minted, err := u.tokenMinter.MintProducts(ctx, MintProductsInput{
		ToAddress:       toAddress,
		ProductIDs:      []string{task.ProductID},
		BlueprintName:   name,
		BlueprintSymbol: symbol,
		MetadataURI:     metadataURI,
	})
	if err != nil {
		log.Printf("[mint-task-usecase] MintProducts failed mintID=%s productID=%s error=%v", mintRequestID, task.ProductID, err)

		if failErr := u.markTaskFailed(ctx, mintRequestID, task.ProductID, err); failErr != nil {
			log.Printf("[mint-task-usecase] markTaskFailed also failed mintID=%s productID=%s error=%v", mintRequestID, task.ProductID, failErr)
			return nil, fmt.Errorf("mint product failed: %w; also failed to update task: %v", err, failErr)
		}

		if parentErr := u.markParentFailedRetryable(ctx, mintEnt); parentErr != nil {
			log.Printf("[mint-task-usecase] markParentFailedRetryable also failed mintID=%s productID=%s error=%v", mintRequestID, task.ProductID, parentErr)
			return nil, fmt.Errorf("mint product failed: %w; also failed to update parent: %v", err, parentErr)
		}

		return nil, err
	}

	log.Printf("[mint-task-usecase] MintProducts ok mintID=%s productID=%s resultCount=%d", mintRequestID, task.ProductID, len(minted))

	if len(minted) != 1 {
		log.Printf("[mint-task-usecase] unexpected minted result count mintID=%s productID=%s count=%d", mintRequestID, task.ProductID, len(minted))
		return nil, fmt.Errorf("expected exactly one minted result, got %d (mintRequestId=%s productId=%s)", len(minted), mintRequestID, task.ProductID)
	}

	mintedOne := minted[0]
	if mintedOne.ProductID == "" {
		mintedOne.ProductID = task.ProductID
	}
	if mintedOne.ProductID != task.ProductID {
		log.Printf("[mint-task-usecase] minted productID mismatch mintID=%s taskProductID=%s mintedProductID=%s", mintRequestID, task.ProductID, mintedOne.ProductID)
		return nil, fmt.Errorf(
			"minted productID mismatch: task=%s minted=%s",
			task.ProductID,
			mintedOne.ProductID,
		)
	}
	if mintedOne.Result == nil {
		log.Printf("[mint-task-usecase] minted result nil mintID=%s productID=%s", mintRequestID, task.ProductID)
		return nil, fmt.Errorf("onchain mint succeeded but result is nil (mintRequestId=%s productId=%s)", mintRequestID, task.ProductID)
	}

	log.Printf(
		"[mint-task-usecase] minted result ok mintID=%s productID=%s mintAddress=%s signature=%s slot=%d",
		mintRequestID,
		task.ProductID,
		mintedOne.Result.MintAddress,
		mintedOne.Result.Signature,
		mintedOne.Result.Slot,
	)

	log.Printf("[mint-task-usecase] recordMintedProduct start mintID=%s productID=%s", reqID, mintedOne.ProductID)

	if err := u.recordMintedProduct(ctx, reqID, mintedOne); err != nil {
		log.Printf("[mint-task-usecase] recordMintedProduct failed mintID=%s productID=%s error=%v", reqID, mintedOne.ProductID, err)
		return mintedOne.Result, fmt.Errorf("record minted product: %w", err)
	}

	log.Printf("[mint-task-usecase] recordMintedProduct ok mintID=%s productID=%s", reqID, mintedOne.ProductID)

	log.Printf("[mint-task-usecase] MarkMinted task start mintID=%s productID=%s", mintRequestID, task.ProductID)

	if _, err := u.mintTaskRepo.MarkMinted(
		ctx,
		mintRequestID,
		task.ProductID,
		mintedOne.Result.MintAddress,
		mintedOne.Result.Signature,
	); err != nil {
		log.Printf("[mint-task-usecase] MarkMinted task failed mintID=%s productID=%s error=%v", mintRequestID, task.ProductID, err)
		return mintedOne.Result, fmt.Errorf("mark mint product task minted: %w", err)
	}

	log.Printf("[mint-task-usecase] MarkMinted task ok mintID=%s productID=%s", mintRequestID, task.ProductID)

	if u.inventoryUC == nil {
		log.Printf("[mint-task-usecase] failed mintID=%s productID=%s error=inventory usecase is nil", mintRequestID, task.ProductID)
		return mintedOne.Result, errors.New("inventory usecase is nil (cannot upsert inventory)")
	}

	log.Printf("[mint-task-usecase] UpsertFromMint start mintID=%s productID=%s productBlueprintID=%s tokenBlueprintID=%s", mintRequestID, task.ProductID, pbID, reqTBID)

	if _, invErr := u.inventoryUC.UpsertFromMint(
		ctx,
		reqTBID,
		pbID,
		[]string{task.ProductID},
	); invErr != nil {
		log.Printf("[mint-task-usecase] UpsertFromMint failed mintID=%s productID=%s error=%v", mintRequestID, task.ProductID, invErr)
		return mintedOne.Result, invErr
	}

	log.Printf("[mint-task-usecase] UpsertFromMint ok mintID=%s productID=%s", mintRequestID, task.ProductID)

	log.Printf("[mint-task-usecase] updateParentAndMaybeEnqueueNext start mintID=%s productID=%s", mintRequestID, task.ProductID)

	if err := u.updateParentAndMaybeEnqueueNext(
		ctx,
		mintEnt,
		reqTBID,
		actorID,
		mintedOne.Result.Signature,
	); err != nil {
		log.Printf("[mint-task-usecase] updateParentAndMaybeEnqueueNext failed mintID=%s productID=%s error=%v", mintRequestID, task.ProductID, err)
		return mintedOne.Result, err
	}

	log.Printf("[mint-task-usecase] updateParentAndMaybeEnqueueNext ok mintID=%s productID=%s", mintRequestID, task.ProductID)
	log.Printf("[mint-task-usecase] completed mintID=%s productID=%s", mintRequestID, task.ProductID)

	return mintedOne.Result, nil
}

func (u *MintUsecase) recordMintedProduct(
	ctx context.Context,
	mintID string,
	minted MintedTokenForUsecase,
) error {
	if u == nil {
		return errors.New("mint usecase is nil")
	}

	if u.mintProductMintRecord != nil {
		return u.mintProductMintRecord.RecordProductAsMinted(ctx, mintID, minted)
	}

	if u.mintRequestPort != nil {
		return u.mintRequestPort.MarkProductsAsMinted(ctx, mintID, []MintedTokenForUsecase{minted})
	}

	return errors.New("mint product recorder is nil")
}

func (u *MintUsecase) markTaskFailed(
	ctx context.Context,
	mintID string,
	productID string,
	err error,
) error {
	if u == nil || u.mintTaskRepo == nil {
		return errors.New("mint task repo is nil")
	}

	message := ""
	if err != nil {
		message = err.Error()
	}

	if isRetryableMintError(err) {
		_, updateErr := u.mintTaskRepo.MarkFailedRetryable(ctx, mintID, productID, message)
		return updateErr
	}

	_, updateErr := u.mintTaskRepo.MarkFailedFatal(ctx, mintID, productID, message)
	return updateErr
}

func (u *MintUsecase) markParentFailedRetryable(
	ctx context.Context,
	mintEnt *mintdom.Mint,
) error {
	if u == nil || u.mintRepo == nil {
		return errors.New("mint repo is nil")
	}
	if mintEnt == nil {
		return errors.New("mint entity is nil")
	}

	if err := mintEnt.MarkFailedRetryable(); err != nil {
		return err
	}

	_, err := u.mintRepo.Update(ctx, *mintEnt)
	return err
}

func (u *MintUsecase) updateParentAndMaybeEnqueueNext(
	ctx context.Context,
	mintEnt *mintdom.Mint,
	reqTBID string,
	actorID string,
	latestSignature string,
) error {
	if u == nil {
		return errors.New("mint usecase is nil")
	}
	if mintEnt == nil {
		return errors.New("mint entity is nil")
	}
	if u.mintTaskRepo == nil {
		return errors.New("mint task repo is nil")
	}
	if u.mintRepo == nil {
		return errors.New("mint repo is nil")
	}

	tasks, err := u.mintTaskRepo.ListByMintID(ctx, mintEnt.ID)
	if err != nil {
		return fmt.Errorf("list mint product tasks: %w", err)
	}

	total := len(tasks)
	if total == 0 {
		return errors.New("mint has no product tasks")
	}

	mintedCount := 0
	fatalCount := 0
	retryableCount := 0

	for _, task := range tasks {
		switch task.Status {
		case mintdom.MintProductTaskStatusMinted:
			mintedCount++
		case mintdom.MintProductTaskStatusFailedFatal:
			fatalCount++
		case mintdom.MintProductTaskStatusPending,
			mintdom.MintProductTaskStatusFailedRetryable:
			retryableCount++
		}
	}

	if mintedCount == total {
		if err := mintEnt.MarkMinted(time.Now().UTC(), latestSignature); err != nil {
			return err
		}

		if _, err := u.mintRepo.Update(ctx, *mintEnt); err != nil {
			return fmt.Errorf("mark parent minted: %w", err)
		}

		if u.tbMintMarker != nil && reqTBID != "" {
			_, _ = u.tbMintMarker.MarkTokenBlueprintMinted(ctx, reqTBID, actorID)
		}

		return nil
	}

	if fatalCount > 0 && mintedCount+fatalCount == total {
		if err := mintEnt.MarkFailedFatal(); err != nil {
			return err
		}

		if _, err := u.mintRepo.Update(ctx, *mintEnt); err != nil {
			return fmt.Errorf("mark parent failed fatal: %w", err)
		}

		return nil
	}

	if mintedCount > 0 {
		if err := mintEnt.MarkPartiallyMinted(); err != nil {
			return err
		}
	} else {
		if err := mintEnt.MarkMinting(); err != nil {
			return err
		}
	}

	if _, err := u.mintRepo.Update(ctx, *mintEnt); err != nil {
		return fmt.Errorf("update parent mint progress: %w", err)
	}

	if retryableCount > 0 && u.mintTaskEnqueuer != nil {
		if err := u.mintTaskEnqueuer.EnqueueMintTask(ctx, mintEnt.ID); err != nil {
			return fmt.Errorf("enqueue next mint task: %w", err)
		}
	}

	return nil
}

func (u *MintUsecase) finalizeMintIfAllTasksCompleted(
	ctx context.Context,
	mintEnt *mintdom.Mint,
	reqTBID string,
	actorID string,
) (*tokendom.MintResult, error) {
	if u == nil {
		return nil, errors.New("mint usecase is nil")
	}
	if mintEnt == nil {
		return nil, errors.New("mint entity is nil")
	}

	tasks, err := u.mintTaskRepo.ListByMintID(ctx, mintEnt.ID)
	if err != nil {
		return nil, fmt.Errorf("list mint product tasks: %w", err)
	}
	if len(tasks) == 0 {
		return nil, mintdom.ErrMintProductTaskNotFound
	}

	latestSignature := ""
	allMinted := true

	for _, task := range tasks {
		if task.Status != mintdom.MintProductTaskStatusMinted {
			allMinted = false
			break
		}
		if task.Signature != "" {
			latestSignature = task.Signature
		}
	}

	if !allMinted {
		return nil, mintdom.ErrMintProductTaskNotFound
	}

	if err := mintEnt.MarkMinted(time.Now().UTC(), latestSignature); err != nil {
		return nil, err
	}

	if _, err := u.mintRepo.Update(ctx, *mintEnt); err != nil {
		return nil, fmt.Errorf("mark parent minted: %w", err)
	}

	if u.tbMintMarker != nil && reqTBID != "" {
		_, _ = u.tbMintMarker.MarkTokenBlueprintMinted(ctx, reqTBID, actorID)
	}

	return u.mintResultMapper.FromMint(*mintEnt), nil
}

func isRetryableMintError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	retryablePatterns := []string{
		"429",
		"too many requests",
		"rate limit",
		"rate limits",
		"connection rate limits exceeded",
		"timeout",
		"deadline exceeded",
		"temporarily unavailable",
		"temporary failure",
		"connection reset",
		"connection refused",
		"i/o timeout",
		"internal error",
		"status=500",
		"status=502",
		"status=503",
		"status=504",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}

	return false
}
