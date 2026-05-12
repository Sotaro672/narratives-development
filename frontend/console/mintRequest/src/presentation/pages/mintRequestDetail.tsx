// frontend/console/mintRequest/src/presentation/pages/mintRequestDetail.tsx
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";
import { Button } from "../../../../shell/src/shared/ui/button";
import { CheckCircle2, Coins } from "lucide-react";

import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../../shell/src/shared/ui/popover";

import ProductBlueprintCard from "../../../../productBlueprint/src/presentation/components/productBlueprintCard";
import InspectionResultCard from "../components/inspectionResultCard";
import TokenBlueprintCard from "../../../../tokenBlueprint/src/presentation/components/tokenBlueprintCard";
import { useMintRequestDetail } from "../hook/useMintRequestDetail";

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
        <div className="minting-overlay__title">ミント中...</div>
        <div className="minting-overlay__description">
          ブロックチェーンへミント申請を送信しています。
        </div>
      </div>
    </div>
  );
}

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

    isMintRequested,
    showMintButton,
    showBrandSelectorCard,
    showTokenSelectorCard,

    showCompleteInspectionButton,
    isCompletingInspection,
    handleCompleteInspection,

    scheduledBurnDate,
    setScheduledBurnDate,

    tokenBlueprintCardVm,
    tokenBlueprintCardHandlers,
    mintCreatedAtLabel,
    mintCreatedByLabel,
    mintScheduledBurnDateLabel,
    mintMintedAtLabel,
    onChainTxSignature,

    requestedByName,
  } = useMintRequestDetail();

  const handleSave = () => {};

  return (
    <>
      {isMinting && <MintingEffectOverlay />}

      <PageStyle
        layout="grid-2"
        title={title}
        onBack={onBack}
        onSave={isMintRequested ? undefined : handleSave}
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
              productName={productBlueprintCardView.productName}
              brand={productBlueprintCardView.brand}
              itemType={productBlueprintCardView.itemType as any}
              fit={productBlueprintCardView.fit as any}
              materials={productBlueprintCardView.materials}
              weight={productBlueprintCardView.weight}
              washTags={productBlueprintCardView.washTags}
              productIdTag={productBlueprintCardView.productIdTag as any}
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
              <InspectionResultCard data={inspectionCardData} />

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
                        onClick={handleCompleteInspection}
                        disabled={isCompletingInspection || isMinting}
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
              vm={tokenBlueprintCardVm as any}
              handlers={tokenBlueprintCardHandlers as any}
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
                      onChange={(e) => setScheduledBurnDate(e.target.value)}
                      disabled={isMinting}
                    />
                    <p className="text-xs text-gray-500">
                      ※ 任意。未入力の場合は焼却予定日なしでミント申請します。
                    </p>
                  </div>

                  <div className="mint-request-card__actions">
                    <Button
                      onClick={handleMint}
                      disabled={isMinting}
                      className="mint-request-card__button flex items-center gap-2"
                    >
                      <Coins size={16} />
                      {isMinting ? "ミント中..." : "ミント申請を実行"}
                    </Button>
                    <span className="mint-request-card__total">
                      ミント数: <strong>{totalMintQuantity}</strong>
                    </span>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}
        </div>

        {/* 右カラム */}
        <div className="space-y-4 mt-4">
          {isMintRequested && (
            <Card className="pb-select">
              <CardHeader>
                <CardTitle>ミント情報</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-2 text-sm">
                  <div>
                    ミント数: <strong>{totalMintQuantity}</strong>
                  </div>
                  <div>作成者: {mintCreatedByLabel}</div>
                  <div>作成日時: {mintCreatedAtLabel}</div>
                  <div>焼却予定日: {mintScheduledBurnDateLabel}</div>
                  <div>リクエスト者: {requestedByName || "（不明）"}</div>
                  <div>ミント日時: {mintMintedAtLabel}</div>

                  {onChainTxSignature && (
                    <div className="break-all">
                      txSignature:{" "}
                      <span className="font-mono text-xs">
                        {onChainTxSignature}
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
                      {brandOptions.map((b) => (
                        <button
                          key={b.id}
                          type="button"
                          className={
                            "pb-select__row" +
                            (selectedBrandId === b.id ? " is-active" : "")
                          }
                          onClick={() => handleSelectBrand(b.id)}
                          disabled={isMinting}
                        >
                          {b.name}
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
                    {tokenBlueprintOptions.map((tb) => (
                      <button
                        key={tb.id}
                        type="button"
                        className={
                          "pb-select__row" +
                          (selectedTokenBlueprintId === tb.id
                            ? " is-active"
                            : "")
                        }
                        onClick={() => handleSelectTokenBlueprint(tb.id)}
                        disabled={isMinting}
                      >
                        {tb.name}
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
    </>
  );
}