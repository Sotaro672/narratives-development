// frontend/console/mintRequest/src/presentation/pages/mintRequestDetail.tsx
import * as React from "react";

import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";
import { Button } from "../../../../shell/src/shared/ui/button";
import { Coins } from "lucide-react";

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

export default function MintRequestDetail() {
  const {
    title,
    loading,
    error,
    inspectionCardData,

    totalMintQuantity,
    onBack,
    handleMint,
    productBlueprintCardView,
    pbPatchLoading,
    pbPatchError,

    // ブランド選択カード用
    brandOptions,
    selectedBrandId,
    selectedBrandName,
    handleSelectBrand,

    // トークン設計一覧カード用
    tokenBlueprintOptions,
    selectedTokenBlueprintId,
    handleSelectTokenBlueprint,

    // ★ mint 情報
    hasMint,

    // ✅ 表示制御（hook 側で算出）
    // minted=true のときだけ非表示、それ以外は表示
    isMintRequested,
    showMintButton,
    showBrandSelectorCard,
    showTokenSelectorCard,

    // ★ 焼却予定日（ScheduledBurnDate）
    scheduledBurnDate,
    setScheduledBurnDate,

    // ★ page から移譲された VM / handlers / labels
    tokenBlueprintCardVm,
    tokenBlueprintCardHandlers,
    mintCreatedAtLabel,
    mintCreatedByLabel,
    mintScheduledBurnDateLabel,
    mintMintedAtLabel,
    mintedLabel,
    onChainTxSignature,
  } = useMintRequestDetail();

  const handleSave = () => {};

  return (
    <PageStyle
      layout="grid-2"
      title={title}
      onBack={onBack}
      onSave={handleSave}
    >
      {/* 左カラム */}
      <div className="space-y-4 mt-4">
        {/* ① Product Blueprint */}
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
              プロダクト基本情報が見つかりません。
            </CardContent>
          </Card>
        )}

        {/* ② Inspection Card */}
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
          <InspectionResultCard data={inspectionCardData} />
        )}

        {/* ③ Token Blueprint Card（選択されている時だけ） */}
        {tokenBlueprintCardVm && (
          <TokenBlueprintCard
            vm={tokenBlueprintCardVm as any}
            handlers={tokenBlueprintCardHandlers as any}
          />
        )}

        {/* ④ ミント申請カード（✅ minted=true のときのみ非表示） */}
        {showMintButton && (
          <Card className="mint-request-card">
            <CardContent className="mint-request-card__body">
              <div className="space-y-3">
                {/* 焼却予定日入力欄（ScheduledBurnDate） */}
                <div className="mint-request-card__burn-date space-y-1">
                  <label className="block text-sm font-medium text-gray-700">
                    焼却予定日（Scheduled Burn Date）
                  </label>
                  <input
                    type="date"
                    className="mint-request-card__burn-date-input"
                    value={scheduledBurnDate}
                    onChange={(e) => setScheduledBurnDate(e.target.value)}
                  />
                  <p className="text-xs text-gray-500">
                    ※ 任意。未入力の場合は焼却予定日なしでミント申請します。
                  </p>
                </div>

                <div className="mint-request-card__actions">
                  <Button
                    onClick={handleMint}
                    className="mint-request-card__button flex items-center gap-2"
                  >
                    <Coins size={16} />
                    ミント申請を実行
                  </Button>
                  <span className="mint-request-card__total">
                    ミント数: <strong>{totalMintQuantity}</strong>
                  </span>
                </div>

                {/* 参考表示（必要なら）：mint が存在するが minted=false の状態 */}
                {hasMint && !isMintRequested && (
                  <p className="text-xs text-gray-500">
                    ※ 既に申請情報はありますが、minted が完了していないため申請カードを表示しています。
                  </p>
                )}
              </div>
            </CardContent>
          </Card>
        )}
      </div>

      {/* 右カラム */}
      <div className="space-y-4 mt-4">
        {/* ✅ mint 情報は「mintが存在するなら常に表示」 */}
        {hasMint && (
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

                <div>minted: {mintedLabel}</div>
                <div>mintedAt: {mintMintedAtLabel}</div>

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

        {/* ✅ minted=true のときだけ、下の選択UIを消す */}
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
                        (selectedTokenBlueprintId === tb.id ? " is-active" : "")
                      }
                      onClick={() => handleSelectTokenBlueprint(tb.id)}
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
  );
}
