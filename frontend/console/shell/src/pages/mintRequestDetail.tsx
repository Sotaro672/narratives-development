// frontend/console/shell/src/pages/mintRequestDetail.tsx

import { CheckCircle2, Coins } from "lucide-react";

import PageStyle from "../layout/PageStyle/PageStyle";

import { Button } from "../shared/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../shared/ui/card";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../shared/ui/popover";

import ProductBlueprintCard from "../features/productBlueprint/presentation/cards/productBlueprintForm";
import InspectionResultCard from "../features/mint/presentation/components/inspectionResultCard";
import { useMintRequestDetail } from "../features/mint/presentation/hook/useMintRequestDetail";
import TokenBlueprintCard, {
  type TokenBlueprintCardHandlers,
} from "../features/tokenBlueprint/presentation/components/tokenBlueprintCard";

import "../styles/mintRequest.css";

function MintingEffectOverlay() {
  return (
    <div
      className="minting-overlay"
      role="status"
      aria-live="polite"
      aria-busy="true"
    >
      <div className="minting-overlay__content">
        <div className="minting-overlay__coins">
          <Coins
            className="minting-overlay__coin minting-overlay__coin--left"
            size={28}
          />

          <Coins
            className="minting-overlay__coin minting-overlay__coin--center"
            size={40}
          />

          <Coins
            className="minting-overlay__coin minting-overlay__coin--right"
            size={28}
          />
        </div>

        <div className="minting-overlay__spinner" />

        <div className="minting-overlay__title">
          ミント中...
        </div>

        <div className="minting-overlay__description">
          ブロックチェーン上でミント処理を実行しています。
        </div>
      </div>
    </div>
  );
}

const READ_ONLY_TOKEN_BLUEPRINT_CARD_HANDLERS:
  TokenBlueprintCardHandlers = {};

export default function MintRequestDetail() {
  const {
    title,
    loading,
    error,
    inspectionCardData,

    totalMintQuantity,
    onBack,
    handleMint,
    isMinting,

    hasMint,
    isMintCompleted,

    productBlueprintCardView,
    pbPatchLoading,
    pbPatchError,

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

    mintProgress,
    showMintProgress,

    showCompleteInspectionButton,
    isCompletingInspection,
    handleCompleteInspection,

    scheduledBurnDate,
    setScheduledBurnDate,

    tokenBlueprintCardVm,

    mintCreatedAtLabel,
    mintCreatedByLabel,
    mintScheduledBurnDateLabel,
    mintMintedAtLabel,
    onChainTxSignature,

    requestedByName,
  } = useMintRequestDetail();

  const progressPercentage =
    mintProgress?.percentage ?? 0;

  const failedMintCount =
    (mintProgress?.failedRetryable ?? 0) +
    (mintProgress?.failedFatal ?? 0);

  const mintStatusLabel =
    isMintCompleted
      ? "ミント完了"
      : "ミント中";

  return (
    <>
      {isMinting && (
        <MintingEffectOverlay />
      )}

      <PageStyle
        layout="grid-2"
        title={title}
        onBack={onBack}
      >
        {/* 左カラム */}
        <div className="space-y-4 mt-4">
          {pbPatchLoading ? (
            <Card className="mint-request-card">
              <CardContent className="mint-request-card__body">
                プロダクト基本情報を読み込み中です…
              </CardContent>
            </Card>
          ) : pbPatchError ? (
            <Card className="mint-request-card">
              <CardContent className="mint-request-card__body text-red-600">
                {pbPatchError}
              </CardContent>
            </Card>
          ) : productBlueprintCardView ? (
            <ProductBlueprintCard
              mode="view"
              productName={
                productBlueprintCardView.productName
              }
              brandName={
                productBlueprintCardView.brandName
              }
              productBlueprintCategory={
                productBlueprintCardView
                  .productBlueprintCategory ??
                null
              }
            />
          ) : (
            <Card className="mint-request-card">
              <CardContent className="mint-request-card__body">
                プロダクト基本情報を読み込み中です…
              </CardContent>
            </Card>
          )}

          {loading ? (
            <Card className="mint-request-card">
              <CardContent className="mint-request-card__body">
                検査結果を読み込み中です…
              </CardContent>
            </Card>
          ) : error ? (
            <Card className="mint-request-card">
              <CardContent className="mint-request-card__body text-red-600">
                {error}
              </CardContent>
            </Card>
          ) : (
            <>
              <InspectionResultCard
                data={inspectionCardData}
              />

              {showCompleteInspectionButton && (
                <Card className="mint-request-card">
                  <CardContent className="mint-request-card__body">
                    <div className="space-y-3">
                      <div>
                        <div className="text-sm font-medium text-gray-900">
                          検品完了
                        </div>

                        <p className="text-xs text-gray-500 mt-1">
                          除外対象がない場合でも、ここで検品完了を確定できます。
                          完了後、未入力の検品結果は合格として扱われます。
                        </p>
                      </div>

                      <Button
                        type="button"
                        onClick={
                          handleCompleteInspection
                        }
                        disabled={
                          isCompletingInspection ||
                          isMinting
                        }
                        className="mint-request-card__button flex items-center gap-2"
                      >
                        <CheckCircle2 size={16} />

                        {isCompletingInspection
                          ? "検品完了中..."
                          : "検品を完了する"}
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              )}
            </>
          )}

          {tokenBlueprintCardVm && (
            <TokenBlueprintCard
              vm={tokenBlueprintCardVm}
              handlers={
                READ_ONLY_TOKEN_BLUEPRINT_CARD_HANDLERS
              }
            />
          )}

          {showMintButton && (
            <Card className="mint-request-card">
              <CardContent className="mint-request-card__body">
                <div className="space-y-3">
                  <div className="mint-request-card__burn-date space-y-1">
                    <label className="block text-sm font-medium text-gray-700">
                      焼却予定日（Scheduled Burn
                      Date）
                    </label>

                    <input
                      type="date"
                      className="mint-request-card__burn-date-input"
                      value={scheduledBurnDate}
                      onChange={(event) =>
                        setScheduledBurnDate(
                          event.target.value,
                        )
                      }
                      disabled={isMinting}
                    />

                    <p className="text-xs text-gray-500">
                      ※
                      任意。未入力の場合は焼却予定日なしでミント申請します。
                    </p>
                  </div>

                  <div className="mint-request-card__actions">
                    <Button
                      type="button"
                      onClick={handleMint}
                      disabled={isMinting}
                      className="mint-request-card__button flex items-center gap-2"
                    >
                      <Coins size={16} />
                      ミント申請を実行
                    </Button>

                    <span className="mint-request-card__total">
                      ミント数:{" "}
                      <strong>
                        {totalMintQuantity}
                      </strong>
                    </span>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}
        </div>

        {/* 右カラム */}
        <div className="space-y-4 mt-4">
          {hasMint && (
            <Card className="pb-select">
              <CardHeader>
                <CardTitle>
                  ミント情報
                </CardTitle>
              </CardHeader>

              <CardContent>
                <div className="space-y-2 text-sm">
                  <div>
                    状態:{" "}
                    <strong>
                      {mintStatusLabel}
                    </strong>
                  </div>

                  <div>
                    ミント数:{" "}
                    <strong>
                      {totalMintQuantity}
                    </strong>
                  </div>

                  {showMintProgress &&
                    mintProgress && (
                      <div className="mint-request-progress">
                        <div className="mint-request-progress__head">
                          <span>
                            ミント進捗
                          </span>

                          <strong>
                            {
                              mintProgress.minted
                            }{" "}
                            /{" "}
                            {
                              mintProgress.total
                            }
                          </strong>
                        </div>

                        <div
                          className="mint-request-progress__bar"
                          role="progressbar"
                          aria-label="ミント進捗"
                          aria-valuemin={0}
                          aria-valuemax={
                            mintProgress.total
                          }
                          aria-valuenow={
                            mintProgress.minted
                          }
                        >
                          <div
                            className="mint-request-progress__fill"
                            style={{
                              width: `${progressPercentage}%`,
                            }}
                          />
                        </div>

                        <div className="mint-request-progress__meta">
                          <span>
                            {progressPercentage}%
                          </span>

                          <span>
                            処理中:{" "}
                            {
                              mintProgress.minting
                            }{" "}
                            / 待機中:{" "}
                            {
                              mintProgress.pending
                            }
                          </span>
                        </div>

                        {failedMintCount >
                          0 && (
                          <div className="mint-request-progress__error">
                            失敗:{" "}
                            {failedMintCount}
                          </div>
                        )}
                      </div>
                    )}

                  <div>
                    作成者:{" "}
                    {mintCreatedByLabel}
                  </div>

                  <div>
                    作成日時:{" "}
                    {mintCreatedAtLabel}
                  </div>

                  <div>
                    焼却予定日:{" "}
                    {
                      mintScheduledBurnDateLabel
                    }
                  </div>

                  <div>
                    リクエスト者:{" "}
                    {requestedByName ||
                      "（不明）"}
                  </div>

                  <div>
                    ミント日時:{" "}
                    {mintMintedAtLabel}
                  </div>

                  {onChainTxSignature && (
                    <div className="break-all">
                      txSignature:{" "}
                      <span className="font-mono text-xs">
                        {
                          onChainTxSignature
                        }
                      </span>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          )}

          {showBrandSelectorCard && (
            <Card className="pb-select">
              <CardHeader>
                <CardTitle>
                  ブランド選択
                </CardTitle>
              </CardHeader>

              <CardContent>
                <Popover>
                  <PopoverTrigger>
                    <div className="pb-select__trigger">
                      {selectedBrandName ||
                        "ブランドを選択"}
                    </div>
                  </PopoverTrigger>

                  <PopoverContent>
                    <div className="pb-select__list">
                      {brandOptions.map(
                        (brand) => (
                          <button
                            key={brand.id}
                            type="button"
                            className={
                              "pb-select__row" +
                              (
                                selectedBrandId ===
                                brand.id
                                  ? " is-active"
                                  : ""
                              )
                            }
                            onClick={() =>
                              handleSelectBrand(
                                brand.id,
                              )
                            }
                            disabled={
                              isMinting
                            }
                          >
                            {brand.name}
                          </button>
                        ),
                      )}

                      {brandOptions.length ===
                        0 && (
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
                <CardTitle>
                  トークン設計一覧
                </CardTitle>
              </CardHeader>

              <CardContent>
                {!selectedBrandId && (
                  <div className="pb-select__empty">
                    先にブランドを選択してください。
                  </div>
                )}

                {selectedBrandId &&
                  tokenBlueprintOptions.length >
                    0 && (
                    <div className="pb-select__list">
                      {tokenBlueprintOptions.map(
                        (
                          tokenBlueprint,
                        ) => (
                          <button
                            key={
                              tokenBlueprint.id
                            }
                            type="button"
                            className={
                              "pb-select__row" +
                              (
                                selectedTokenBlueprintId ===
                                tokenBlueprint.id
                                  ? " is-active"
                                  : ""
                              )
                            }
                            onClick={() =>
                              handleSelectTokenBlueprint(
                                tokenBlueprint.id,
                              )
                            }
                            disabled={
                              isMinting
                            }
                          >
                            {
                              tokenBlueprint.tokenName
                            }
                          </button>
                        ),
                      )}
                    </div>
                  )}

                {selectedBrandId &&
                  tokenBlueprintOptions.length ===
                    0 && (
                    <div className="pb-select__empty">
                      選択中のブランドに紐づくトークン設計がありません。
                    </div>
                  )}
              </CardContent>
            </Card>
          )}
        </div>
      </PageStyle>
    </>
  );
}