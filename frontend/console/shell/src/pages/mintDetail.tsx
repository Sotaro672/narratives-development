// frontend/console/shell/src/pages/mintDetail.tsx

import {
  CheckCircle2,
  Coins,
} from "lucide-react";

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
import {
  useMintRequestDetail,
} from "../features/mint/presentation/hook/useMintRequestDetail";

import TokenBlueprintCard from "../features/tokenBlueprint/presentation/components/tokenBlueprintCard";

import "../styles/mintRequest.css";

/**
 * Mint処理中に右カラムへ表示するステータスカード。
 */
function MintingStatusCard() {
  return (
    <Card
      className="pb-select"
      role="status"
      aria-live="polite"
      aria-busy="true"
    >
      <CardHeader>
        <CardTitle>
          ミント処理中
        </CardTitle>
      </CardHeader>

      <CardContent>
        <div className="flex flex-col items-center gap-4 py-4 text-center">
          <div
            className="flex items-end justify-center gap-2 text-blue-600"
            aria-hidden="true"
          >
            <Coins
              size={28}
              className="animate-pulse"
            />

            <Coins
              size={40}
              className="animate-bounce"
            />

            <Coins
              size={28}
              className="animate-pulse"
            />
          </div>

          <div
            className="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-blue-600"
            aria-hidden="true"
          />

          <div className="space-y-1">
            <div className="text-sm font-semibold text-gray-900">
              ミント中...
            </div>

            <p className="text-xs text-gray-500">
              ブロックチェーン上でミント処理を実行しています。
            </p>
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

    totalMintQuantity,
    onBack,
    handleMint,
    isMinting,

    hasMint,

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

    showCompleteInspectionButton,
    isCompletingInspection,
    handleCompleteInspection,

    scheduledBurnDate,
    setScheduledBurnDate,

    tokenBlueprintCardVm,

    mintMintedAtLabel,
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
                  : mintStatus ||
                    "（未設定）";

  return (
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
          />
        )}

        {showMintButton && (
          <Card className="mint-request-card">
            <CardContent className="mint-request-card__body">
              <div className="space-y-3">
                <div className="mint-request-card__burn-date space-y-1">
                  <label className="block text-sm font-medium text-gray-700">
                    焼却予定日（Scheduled Burn Date）
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
        {hasMint &&
          mintRequestRow && (
          <Card className="pb-select">
            <CardHeader>
              <CardTitle>
                ミント情報
              </CardTitle>
            </CardHeader>

            <CardContent>
              <div className="space-y-2 text-sm">
                <div>
                  生産数:{" "}
                  <strong>
                    {
                      mintRequestRow
                        .productionQuantity ??
                      0
                    }
                  </strong>
                </div>

                <div>
                  ミント数:{" "}
                  <strong>
                    {
                      mintRequestRow
                        .mintQuantity ??
                      0
                    }
                  </strong>
                </div>

                <div>
                  検品状態:{" "}
                  <strong>
                    {
                      mintRequestRow
                        .inspectionStatus ||
                      "（未設定）"
                    }
                  </strong>
                </div>

                <div>
                  ミント状態:{" "}
                  <strong>
                    {mintStatusLabel}
                  </strong>
                </div>

                <div>
                  作成者:{" "}
                  {
                    mintRequestRow
                      .createdByName ||
                    mintRequestRow
                      .createdBy ||
                    "（不明）"
                  }
                </div>

                <div>
                  リクエスト者:{" "}
                  {
                    mintRequestRow
                      .requestedByName ||
                    mintRequestRow
                      .requestedBy ||
                    "（不明）"
                  }
                </div>

                <div>
                  ミント日時:{" "}
                  {mintMintedAtLabel}
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        {isMinting && (
          <MintingStatusCard />
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
  );
}