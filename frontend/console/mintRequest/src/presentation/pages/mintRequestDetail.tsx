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

    // ★ useMintRequestDetail 側で追加した「選択中 TokenBlueprintOption」
    selectedTokenBlueprint,

    // ★ 申請済みモード制御用
    isMintRequested,
    requestedBy,
    requestedAt,
  } = useMintRequestDetail();

  // ★ 焼却予定日（ScheduledBurnDate）入力用ローカル状態
  //   - まずは UI 側に入力欄だけ追加（実際の保存は後続の実装で紐付け）
  const [scheduledBurnDate, setScheduledBurnDate] = React.useState<string>("");

  const handleSave = () => {};

  const currentFilterLabel =
    selectedBrandName || selectedBrandId || "未選択";

  // ★ 左カラム TokenBlueprintCard に渡す ViewModel
  const tokenBlueprintCardVm = selectedTokenBlueprint
    ? {
        id: selectedTokenBlueprint.id,
        name: selectedTokenBlueprint.name,
        symbol: selectedTokenBlueprint.symbol,
        brandId: selectedBrandId,
        brandName: selectedBrandName,
        description: "", // description は取得していないので空
        iconUrl: selectedTokenBlueprint.iconUrl,
        isEditMode: false,
        brandOptions: brandOptions.map((b) => ({ id: b.id, name: b.name })),
      }
    : null;

  const tokenBlueprintCardHandlers = {
    onPreview: () => {},
  };

  // ★ requestedAt の表示用フォーマット
  const requestedAtLabel = requestedAt
    ? new Date(requestedAt).toLocaleString("ja-JP")
    : "（未登録）";

  const requestedByLabel = requestedBy || "（不明）";

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

        {/* ③ Token Blueprint Card
            - 未申請: 選択中トークンがあれば表示
            - 申請済み: requestedBy/At/tokenBlueprintId があれば表示（hook 側で vm 解決済み） */}
        {tokenBlueprintCardVm && (
          <TokenBlueprintCard
            vm={tokenBlueprintCardVm}
            handlers={tokenBlueprintCardHandlers}
          />
        )}

        {/* ④ ミント申請カード（未申請モードのみ表示） */}
        {!isMintRequested && (
          <Card className="mint-request-card">
            <CardContent className="mint-request-card__body">
              <div className="space-y-3">
                {/* ★ 焼却予定日入力欄（ScheduledBurnDate） */}
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
              </div>
            </CardContent>
          </Card>
        )}
      </div>

      {/* 右カラム */}
      <div className="space-y-4 mt-4">
        {isMintRequested ? (
          // ★ 申請済みモード: ミント数 + requestedBy + requestedAt を表示
          <Card className="pb-select">
            <CardHeader>
              <CardTitle>ミント申請情報</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-2 text-sm">
                <div>
                  ミント数: <strong>{totalMintQuantity}</strong>
                </div>
                <div>申請者: {requestedByLabel}</div>
                <div>申請日時: {requestedAtLabel}</div>
              </div>
            </CardContent>
          </Card>
        ) : (
          <>
            {/* ブランド選択カード（未申請モードのみ） */}
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

                <div className="mt-2 text-xs text-gray-500">
                  現在のフィルタ: {currentFilterLabel}
                </div>
              </CardContent>
            </Card>

            {/* トークン一覧カード（未申請モードのみ） */}
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
          </>
        )}
      </div>
    </PageStyle>
  );
}
