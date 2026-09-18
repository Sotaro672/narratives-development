// frontend/console/shell/src/pages/mintDetail.tsx

import { CheckCircle2, Coins } from "lucide-react";

import PageStyle from "../layout/PageStyle/PageStyle";
import { Button } from "../shared/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../shared/ui/card";
import { Popover, PopoverContent, PopoverTrigger } from "../shared/ui/popover";
import Text from "../shared/ui/text";

import ProductBlueprintCard from "../features/productBlueprint/presentation/components/productBlueprintForm";
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
        <div className="mint-progress">
          <div className="mint-progress__progress">
            <div className="mint-progress__row">
              <Text tone="muted">進捗</Text>
              <Text weight="semibold">{percentage}%</Text>
            </div>

            <progress
              className="mint-progress__bar"
              value={percentage}
              max={100}
              aria-label="ミント進捗"
            />

            <Text as="div" size="xs" tone="muted" className="mint-progress__summary">
              <Text size="xs" weight="semibold">
                {progress.minted}
              </Text>{" "}
              / {progress.total} 完了
            </Text>
          </div>

          <div className="mint-progress__details">
            <div className="mint-progress__rows">
              <div className="mint-progress__row">
                <Text tone="muted">待機中</Text>
                <Text weight="semibold">{progress.pending}</Text>
              </div>

              <div className="mint-progress__row">
                <Text tone="muted">ミント中</Text>
                <Text weight="semibold">{progress.minting}</Text>
              </div>

              <div className="mint-progress__row">
                <Text tone="muted">完了</Text>
                <Text weight="semibold">{progress.minted}</Text>
              </div>

              <div className="mint-progress__row">
                <Text tone="muted">再試行待ち</Text>
                <Text
                  weight="semibold"
                  className={
                    progress.failedRetryable > 0
                      ? "mint-progress__value--warning"
                      : undefined
                  }
                >
                  {progress.failedRetryable}
                </Text>
              </div>

              <div className="mint-progress__row">
                <Text tone="muted">失敗</Text>
                <Text
                  weight="semibold"
                  className={
                    progress.failedFatal > 0
                      ? "mint-progress__value--danger"
                      : undefined
                  }
                >
                  {progress.failedFatal}
                </Text>
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
      <div className="mint-detail__column">
        {productBlueprintLoading ? (
          <Card className="mint-request-card">
            <CardContent className="mint-request-card__body">
              <Text tone="muted">プロダクト基本情報を読み込み中です…</Text>
            </CardContent>
          </Card>
        ) : productBlueprintError ? (
          <Card className="mint-request-card">
            <CardContent className="mint-request-card__body">
              <Text tone="destructive" role="alert">
                {productBlueprintError}
              </Text>
            </CardContent>
          </Card>
        ) : productBlueprintCardView ? (
          <ProductBlueprintCard
            mode="view"
            productName={productBlueprintCardView.productName}
            brandName={productBlueprintCardView.brandName}
            productBlueprintCategoryPath={
              productBlueprintCardView.productBlueprintCategoryPath ?? null
            }
          />
        ) : (
          <Card className="mint-request-card">
            <CardContent className="mint-request-card__body">
              <Text tone="muted">プロダクト基本情報を読み込み中です…</Text>
            </CardContent>
          </Card>
        )}

        {loading ? (
          <Card className="mint-request-card">
            <CardContent className="mint-request-card__body">
              <Text tone="muted">検査結果を読み込み中です…</Text>
            </CardContent>
          </Card>
        ) : error ? (
          <Card className="mint-request-card">
            <CardContent className="mint-request-card__body">
              <Text tone="destructive" role="alert">
                {error}
              </Text>
            </CardContent>
          </Card>
        ) : (
          <>
            <InspectionResultCard data={inspectionCardData} />

            {showCompleteInspectionButton && (
              <Card className="mint-request-card">
                <CardContent className="mint-request-card__body">
                  <div className="mint-inspection-complete">
                    <div>
                      <Text as="div" weight="medium">
                        検品完了
                      </Text>
                      <Text as="p" size="xs" tone="muted" className="mint-inspection-complete__description">
                        除外対象がない場合でも、ここで検品完了を確定できます。
                        完了後、未入力の検品結果は合格として扱われます。
                      </Text>
                    </div>

                    <Button
                      type="button"
                      onClick={handleCompleteInspection}
                      disabled={isCompletingInspection || isMinting}
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
              <div className="mint-funding">
                {!selectedTokenBlueprintId ? (
                  <Text as="div" tone="muted">
                    トークン設計を選択すると、ミントに必要なSOLを見積もります。
                  </Text>
                ) : mintFundingEstimateLoading ? (
                  <div className="mint-funding__loading" role="status" aria-live="polite">
                    <div className="mint-funding__spinner" aria-hidden="true" />
                    <Text tone="muted">SOL見積を取得中です…</Text>
                  </div>
                ) : mintFundingEstimateError ? (
                  <Text as="div" tone="destructive" role="alert">
                    {mintFundingEstimateError}
                  </Text>
                ) : mintFundingEstimate ? (
                  <div className="mint-funding__estimate">
                    <div className="mint-funding__rows">
                      <div className="mint-funding__row">
                        <Text tone="muted">Reserve Wallet残高</Text>
                        <Text weight="semibold">
                          {formatSol(mintFundingEstimate.reserve.balanceSol)} SOL
                        </Text>
                      </div>
                    </div>

                    <div className="mint-funding__section">
                      <div className="mint-funding__rows">
                        <div className="mint-funding__row">
                          <Text tone="muted">1件あたりMint手数料</Text>
                          <Text weight="semibold">
                            {formatSol(
                              mintFundingEstimate.estimate.mintTransactionFeePerItemSol,
                            )}{" "}
                            SOL
                          </Text>
                        </div>

                        <div className="mint-funding__row">
                          <Text tone="muted">Mint手数料合計</Text>
                          <Text weight="semibold">
                            {formatSol(
                              mintFundingEstimate.estimate.mintTransactionFeeTotalSol,
                            )}{" "}
                            SOL
                          </Text>
                        </div>

                        <div className="mint-funding__row">
                          <Text tone="muted">初回作成費</Text>
                          <Text weight="semibold">
                            {formatSol(
                              mintFundingEstimate.estimate.initialCreationCostSol,
                            )}{" "}
                            SOL
                          </Text>
                        </div>
                      </div>
                    </div>

                    <div className="mint-funding__section">
                      <div className="mint-funding__row">
                        <Text weight="semibold">最終必要SOL合計</Text>
                        <Text weight="semibold">
                          {formatSol(
                            mintFundingEstimate.estimate.totalRequiredSol,
                          )}{" "}
                          SOL
                        </Text>
                      </div>
                    </div>

                    <Text
                      as="div"
                      weight="medium"
                      className={
                        mintFundingEstimate.estimate.sufficient
                          ? "mint-funding__status mint-funding__status--sufficient"
                          : "mint-funding__status mint-funding__status--insufficient"
                      }
                    >
                      {mintFundingEstimate.estimate.sufficient
                        ? "SOL残高はミント実行に必要な条件を満たしています。"
                        : "Reserve WalletのSOL残高が不足しています。"}
                    </Text>
                  </div>
                ) : null}

                <div className="mint-request-card__actions">
                  <Button
                    type="button"
                    onClick={handleMint}
                    disabled={!canSubmitMint}
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
      <div className="mint-detail__column">
        {hasMint && mintRequestRow && (
          <Card className="pb-select">
            <CardHeader>
              <CardTitle>ミント情報</CardTitle>
            </CardHeader>

            <CardContent>
              <div className="mint-info">
                <Text as="div">
                  生産数:{" "}
                  <Text weight="semibold">
                    {mintRequestRow.productionQuantity ?? 0}
                  </Text>
                </Text>

                <Text as="div">
                  ミント数:{" "}
                  <Text weight="semibold">
                    {mintRequestRow.mintQuantity ?? 0}
                  </Text>
                </Text>

                <Text as="div">
                  ミント状態:{" "}
                  <Text weight="semibold">
                    {mintStatusLabel}
                  </Text>
                </Text>

                <Text as="div">
                  リクエスト者:{" "}
                  {mintRequestRow.requestedByName ||
                    mintRequestRow.requestedBy ||
                    "（不明）"}
                </Text>

                <Text as="div">
                  ミント日時: {mintMintedAtLabel}
                </Text>
              </div>
            </CardContent>
          </Card>
        )}

        {hasMint && mintStatus !== "MINTED" && mintProgress && (
          <MintProgressCard progress={mintProgress} />
        )}

        {showBrandSelectorCard && (
          <Card className="pb-select">
            <CardHeader>
              <CardTitle>ブランド選択</CardTitle>
            </CardHeader>

            <CardContent>
              <Popover>
                <PopoverTrigger>
                  <div className="pb-select__trigger">
                    {selectedBrandName || "ブランドを選択"}
                  </div>
                </PopoverTrigger>

                <PopoverContent>
                  <div className="pb-select__list">
                    {brandOptions.map((brand) => (
                      <button
                        key={brand.id}
                        type="button"
                        className={
                          "pb-select__row" +
                          (selectedBrandId === brand.id ? " is-active" : "")
                        }
                        onClick={() => handleSelectBrand(brand.id)}
                        disabled={isMinting}
                      >
                        {brand.name}
                      </button>
                    ))}

                    {brandOptions.length === 0 && (
                      <div className="pb-select__empty">
                        ブランド候補が未設定です
                      </div>
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
              {!selectedBrandId && (
                <div className="pb-select__empty">
                  先にブランドを選択してください。
                </div>
              )}

              {selectedBrandId && tokenBlueprintOptions.length > 0 && (
                <div className="pb-select__list">
                  {tokenBlueprintOptions.map((tokenBlueprint) => (
                    <button
                      key={tokenBlueprint.id}
                      type="button"
                      className={
                        "pb-select__row" +
                        (selectedTokenBlueprintId === tokenBlueprint.id
                          ? " is-active"
                          : "")
                      }
                      onClick={() =>
                        handleSelectTokenBlueprint(tokenBlueprint.id)
                      }
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