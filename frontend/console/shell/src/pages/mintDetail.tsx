// frontend/console/shell/src/pages/mintDetail.tsx

import { CheckCircle2, Coins } from "lucide-react";

import PageStyle from "../layout/PageStyle/PageStyle";
import { Button } from "../shared/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../shared/ui/card";
import { Popover, PopoverContent, PopoverTrigger } from "../shared/ui/popover";

import ProductBlueprintCard from "../features/productBlueprint/presentation/cards/productBlueprintForm";
import InspectionResultCard from "../features/mint/presentation/components/inspectionResultCard";
import { useMintRequestDetail } from "../features/mint/presentation/hook/useMintRequestDetail";
import type { MintTaskProgressDTO } from "../features/mint/infrastructure/dto/mintRequestManagementRow";
import TokenBlueprintCard from "../features/tokenBlueprint/presentation/components/tokenBlueprintCard";

import "../styles/mintRequest.css";

function formatSol(value: number): string {
  if (!Number.isFinite(value)) {
    return "0";
  }

  return value.toLocaleString("ja-JP", {
    minimumFractionDigits: 0,
    maximumFractionDigits: 9,
  });
}

/**
 * Mint処理中に右カラムへ表示するステータスカード。
 */
function MintingStatusCard() {
  return (
    <Card className="pb-select" role="status" aria-live="polite" aria-busy="true">
      <CardHeader>
        <CardTitle>ミント処理中</CardTitle>
      </CardHeader>

      <CardContent>
        <div className="flex flex-col items-center gap-4 py-4 text-center">
          <div className="flex items-end justify-center gap-2 text-blue-600" aria-hidden="true">
            <Coins size={28} className="animate-pulse" />
            <Coins size={40} className="animate-bounce" />
            <Coins size={28} className="animate-pulse" />
          </div>

          <div
            className="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-blue-600"
            aria-hidden="true"
          />

          <div className="space-y-1">
            <div className="text-sm font-semibold text-gray-900">ミント中...</div>
            <p className="text-xs text-gray-500">ブロックチェーン上でミント処理を実行しています。</p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

/**
 * mints/{mintId}/products の集計結果を表示する進捗カード。
 */
function MintProgressCard({ progress }: { progress: MintTaskProgressDTO }) {
  const percentage = Math.min(100, Math.max(0, progress.percentage));

  return (
    <Card className="pb-select" role="status" aria-live="polite">
      <CardHeader>
        <CardTitle>ミント進捗</CardTitle>
      </CardHeader>

      <CardContent>
        <div className="space-y-4">
          <div className="space-y-2">
            <div className="flex items-center justify-between gap-4 text-sm">
              <span className="text-gray-600">進捗</span>
              <strong className="text-gray-900">{percentage}%</strong>
            </div>

            <div
              className="h-2 w-full overflow-hidden rounded-full bg-gray-200"
              role="progressbar"
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={percentage}
              aria-label="ミント進捗"
            >
              <div
                className="h-full rounded-full bg-blue-600 transition-all duration-300"
                style={{ width: `${percentage}%` }}
              />
            </div>

            <div className="text-right text-xs text-gray-500">
              <strong className="text-gray-900">{progress.minted}</strong> / {progress.total} 完了
            </div>
          </div>

          <div className="border-t border-gray-200 pt-3">
            <div className="space-y-2 text-sm">
              <div className="flex items-center justify-between gap-4">
                <span className="text-gray-600">待機中</span>
                <strong className="text-gray-900">{progress.pending}</strong>
              </div>

              <div className="flex items-center justify-between gap-4">
                <span className="text-gray-600">ミント中</span>
                <strong className="text-gray-900">{progress.minting}</strong>
              </div>

              <div className="flex items-center justify-between gap-4">
                <span className="text-gray-600">完了</span>
                <strong className="text-gray-900">{progress.minted}</strong>
              </div>

              <div className="flex items-center justify-between gap-4">
                <span className="text-gray-600">再試行待ち</span>
                <strong className={progress.failedRetryable > 0 ? "text-amber-600" : "text-gray-900"}>
                  {progress.failedRetryable}
                </strong>
              </div>

              <div className="flex items-center justify-between gap-4">
                <span className="text-gray-600">失敗</span>
                <strong className={progress.failedFatal > 0 ? "text-red-600" : "text-gray-900"}>
                  {progress.failedFatal}
                </strong>
              </div>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

export default function MintRequestDetail() {
  const {
    title,
    loading,
    error,
    inspectionCardData,
    mintRequestRow,
    mintStatus,
    mintProgress,
    onBack,
    handleMint,
    isMinting,
    hasMint,
    productBlueprintCardView,
    productBlueprintLoading,
    productBlueprintError,
    brandOptions,
    selectedBrandId,
    selectedBrandName,
    handleSelectBrand,
    tokenBlueprintOptions,
    selectedTokenBlueprintId,
    handleSelectTokenBlueprint,
    showMintButton,
    showBrandSelectorCard,
    showTokenSelectorCard,
    showCompleteInspectionButton,
    isCompletingInspection,
    handleCompleteInspection,
    tokenBlueprintCardVm,
    mintMintedAtLabel,
    mintFundingEstimate,
    mintFundingEstimateLoading,
    mintFundingEstimateError,
  } = useMintRequestDetail();

  const mintStatusLabel =
    mintStatus === "MINTED"
      ? "ミント完了"
      : mintStatus === "QUEUED"
        ? "ミント待機中"
        : mintStatus === "MINTING"
          ? "ミント中"
          : mintStatus === "PARTIALLY_MINTED"
            ? "一部ミント完了"
            : mintStatus === "FAILED_RETRYABLE"
              ? "再試行待ち"
              : mintStatus === "FAILED_FATAL"
                ? "ミント失敗"
                : mintStatus === "CREATED"
                  ? "作成済み"
                  : mintStatus || "（未設定）";

  const canSubmitMint =
    !isMinting &&
    !mintFundingEstimateLoading &&
    Boolean(mintFundingEstimate) &&
    mintFundingEstimate?.estimate.sufficient === true;

  return (
    <PageStyle layout="grid-2" title={title} onBack={onBack}>
      {/* 左カラム */}
      <div className="space-y-4 mt-4">
        {productBlueprintLoading ? (
          <Card className="mint-request-card">
            <CardContent className="mint-request-card__body">プロダクト基本情報を読み込み中です…</CardContent>
          </Card>
        ) : productBlueprintError ? (
          <Card className="mint-request-card">
            <CardContent className="mint-request-card__body text-red-600">{productBlueprintError}</CardContent>
          </Card>
        ) : productBlueprintCardView ? (
          <ProductBlueprintCard
            mode="view"
            productName={productBlueprintCardView.productName}
            brandName={productBlueprintCardView.brandName}
            productBlueprintCategory={productBlueprintCardView.productBlueprintCategory ?? null}
          />
        ) : (
          <Card className="mint-request-card">
            <CardContent className="mint-request-card__body">プロダクト基本情報を読み込み中です…</CardContent>
          </Card>
        )}

        {loading ? (
          <Card className="mint-request-card">
            <CardContent className="mint-request-card__body">検査結果を読み込み中です…</CardContent>
          </Card>
        ) : error ? (
          <Card className="mint-request-card">
            <CardContent className="mint-request-card__body text-red-600">{error}</CardContent>
          </Card>
        ) : (
          <>
            <InspectionResultCard data={inspectionCardData} />

            {showCompleteInspectionButton && (
              <Card className="mint-request-card">
                <CardContent className="mint-request-card__body">
                  <div className="space-y-3">
                    <div>
                      <div className="text-sm font-medium text-gray-900">検品完了</div>
                      <p className="text-xs text-gray-500 mt-1">
                        除外対象がない場合でも、ここで検品完了を確定できます。
                        完了後、未入力の検品結果は合格として扱われます。
                      </p>
                    </div>

                    <Button
                      type="button"
                      onClick={handleCompleteInspection}
                      disabled={isCompletingInspection || isMinting}
                      className="mint-request-card__button flex items-center gap-2"
                    >
                      <CheckCircle2 size={16} />
                      {isCompletingInspection ? "検品完了中..." : "検品を完了する"}
                    </Button>
                  </div>
                </CardContent>
              </Card>
            )}
          </>
        )}

        {tokenBlueprintCardVm && <TokenBlueprintCard vm={tokenBlueprintCardVm} />}

        {showMintButton && (
          <Card className="mint-request-card">
            <CardHeader>
              <CardTitle>SOL見積</CardTitle>
            </CardHeader>

            <CardContent className="mint-request-card__body">
              <div className="space-y-4">
                {!selectedTokenBlueprintId ? (
                  <div className="text-sm text-gray-500">
                    トークン設計を選択すると、ミントに必要なSOLを見積もります。
                  </div>
                ) : mintFundingEstimateLoading ? (
                  <div
                    className="flex items-center gap-3 text-sm text-gray-600"
                    role="status"
                    aria-live="polite"
                  >
                    <div
                      className="h-5 w-5 animate-spin rounded-full border-2 border-gray-200 border-t-blue-600"
                      aria-hidden="true"
                    />
                    SOL見積を取得中です…
                  </div>
                ) : mintFundingEstimateError ? (
                  <div className="text-sm text-red-600">{mintFundingEstimateError}</div>
                ) : mintFundingEstimate ? (
                  <div className="space-y-4">
                    <div className="grid gap-2 text-sm">
                      <div className="flex items-center justify-between gap-4">
                        <span className="text-gray-600">Reserve Wallet残高</span>
                        <strong className="text-gray-900">
                          {formatSol(mintFundingEstimate.reserve.balanceSol)} SOL
                        </strong>
                      </div>
                    </div>

                    <div className="border-t border-gray-200 pt-3">
                      <div className="space-y-2 text-sm">
                        <div className="flex items-center justify-between gap-4">
                          <span className="text-gray-600">1件あたりMint手数料</span>
                          <strong className="text-gray-900">
                            {formatSol(mintFundingEstimate.estimate.mintTransactionFeePerItemSol)} SOL
                          </strong>
                        </div>

                        <div className="flex items-center justify-between gap-4">
                          <span className="text-gray-600">Mint手数料合計</span>
                          <strong className="text-gray-900">
                            {formatSol(mintFundingEstimate.estimate.mintTransactionFeeTotalSol)} SOL
                          </strong>
                        </div>

                        <div className="flex items-center justify-between gap-4">
                          <span className="text-gray-600">初回作成費</span>
                          <strong className="text-gray-900">
                            {formatSol(mintFundingEstimate.estimate.initialCreationCostSol)} SOL
                          </strong>
                        </div>
                      </div>
                    </div>

                    <div className="border-t border-gray-200 pt-3">
                      <div className="flex items-center justify-between gap-4 text-sm">
                        <span className="font-semibold text-gray-900">最終必要SOL合計</span>
                        <strong className="text-gray-900">
                          {formatSol(mintFundingEstimate.estimate.totalRequiredSol)} SOL
                        </strong>
                      </div>
                    </div>

                    <div
                      className={
                        "rounded-md px-3 py-2 text-sm font-medium " +
                        (mintFundingEstimate.estimate.sufficient
                          ? "bg-green-50 text-green-700"
                          : "bg-red-50 text-red-700")
                      }
                    >
                      {mintFundingEstimate.estimate.sufficient
                        ? "SOL残高はミント実行に必要な条件を満たしています。"
                        : "Reserve WalletのSOL残高が不足しています。"}
                    </div>
                  </div>
                ) : null}

                <div className="mint-request-card__actions">
                  <Button
                    type="button"
                    onClick={handleMint}
                    disabled={!canSubmitMint}
                    className="mint-request-card__button flex items-center gap-2"
                  >
                    <Coins size={16} />
                    ミント申請を実行
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        )}
      </div>

      {/* 右カラム */}
      <div className="space-y-4 mt-4">
        {hasMint && mintRequestRow && (
          <Card className="pb-select">
            <CardHeader>
              <CardTitle>ミント情報</CardTitle>
            </CardHeader>

            <CardContent>
              <div className="space-y-2 text-sm">
                <div>
                  生産数: <strong>{mintRequestRow.productionQuantity ?? 0}</strong>
                </div>

                <div>
                  ミント数: <strong>{mintRequestRow.mintQuantity ?? 0}</strong>
                </div>

                <div>
                  検品状態: <strong>{mintRequestRow.inspectionStatus || "（未設定）"}</strong>
                </div>

                <div>
                  ミント状態: <strong>{mintStatusLabel}</strong>
                </div>

                <div>
                  作成者: {mintRequestRow.createdByName || mintRequestRow.createdBy || "（不明）"}
                </div>

                <div>
                  リクエスト者: {mintRequestRow.requestedByName || mintRequestRow.requestedBy || "（不明）"}
                </div>

                <div>ミント日時: {mintMintedAtLabel}</div>
              </div>
            </CardContent>
          </Card>
        )}

        {hasMint && mintProgress && <MintProgressCard progress={mintProgress} />}

        {isMinting && <MintingStatusCard />}

        {showBrandSelectorCard && (
          <Card className="pb-select">
            <CardHeader>
              <CardTitle>ブランド選択</CardTitle>
            </CardHeader>

            <CardContent>
              <Popover>
                <PopoverTrigger>
                  <div className="pb-select__trigger">{selectedBrandName || "ブランドを選択"}</div>
                </PopoverTrigger>

                <PopoverContent>
                  <div className="pb-select__list">
                    {brandOptions.map((brand) => (
                      <button
                        key={brand.id}
                        type="button"
                        className={"pb-select__row" + (selectedBrandId === brand.id ? " is-active" : "")}
                        onClick={() => handleSelectBrand(brand.id)}
                        disabled={isMinting}
                      >
                        {brand.name}
                      </button>
                    ))}

                    {brandOptions.length === 0 && (
                      <div className="pb-select__empty">ブランド候補が未設定です</div>
                    )}
                  </div>
                </PopoverContent>
              </Popover>
            </CardContent>
          </Card>
        )}

        {showTokenSelectorCard && (
          <Card className="pb-select">
            <CardHeader>
              <CardTitle>トークン設計一覧</CardTitle>
            </CardHeader>

            <CardContent>
              {!selectedBrandId && <div className="pb-select__empty">先にブランドを選択してください。</div>}

              {selectedBrandId && tokenBlueprintOptions.length > 0 && (
                <div className="pb-select__list">
                  {tokenBlueprintOptions.map((tokenBlueprint) => (
                    <button
                      key={tokenBlueprint.id}
                      type="button"
                      className={
                        "pb-select__row" +
                        (selectedTokenBlueprintId === tokenBlueprint.id ? " is-active" : "")
                      }
                      onClick={() => handleSelectTokenBlueprint(tokenBlueprint.id)}
                      disabled={isMinting}
                    >
                      {tokenBlueprint.tokenName}
                    </button>
                  ))}
                </div>
              )}

              {selectedBrandId && tokenBlueprintOptions.length === 0 && (
                <div className="pb-select__empty">
                  選択中のブランドに紐づくトークン設計がありません。
                </div>
              )}
            </CardContent>
          </Card>
        )}
      </div>
    </PageStyle>
  );
}