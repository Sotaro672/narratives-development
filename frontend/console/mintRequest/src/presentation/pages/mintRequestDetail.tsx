// frontend/console/mintRequest/src/presentation/pages/mintRequestDetail.tsx
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
import { useMintRequestDetail } from "../hook/useMintRequestDetail";

import "../styles/mintRequest.css";

export default function MintRequestDetail() {
  const {
    title,
    loading,
    error,
    inspectionCardData,

    // ★ blueprint → tokenBlueprint に改名
    tokenBlueprint,

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
  } = useMintRequestDetail();

  // PageHeader 用の保存ボタンハンドラ（現状はダミー）
  const handleSave = () => {
    // TODO: 必要になったら保存処理を実装
  };

  const currentFilterLabel =
    selectedBrandName || selectedBrandId || "未選択";

  return (
    <PageStyle
      layout="grid-2"
      title={title}
      onBack={onBack}
      onSave={handleSave} // PageHeader に保存ボタン表示
    >
      {/* 左カラム */}
      <div className="space-y-4 mt-4">
        {/* ① プロダクト基本情報（閲覧モード） */}
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

        {/* ② 検査結果カード */}
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

        {/* ③ TokenBlueprintCard（将来用。現状は非表示） */}
        {false && tokenBlueprint && (
          <div className="mt-4">
            {/* TokenBlueprintCard を実装する際にここで tokenBlueprint を利用 */}
          </div>
        )}

        {/* ④ ミント申請ボタン */}
        <Card className="mint-request-card">
          <CardContent className="mint-request-card__body">
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
          </CardContent>
        </Card>
      </div>

      {/* 右カラム */}
      <div className="space-y-4 mt-4">
        {/* ブランド選択カード（ProductionCreate と同系 UI） */}
        <Card className="pb-select">
          <CardHeader>
            <CardTitle>ブランド選択</CardTitle>
          </CardHeader>
          <CardContent>
            <Popover>
              <PopoverTrigger>
                <div className="pb-select__trigger">
                  {/* ★ デフォルトは何も選択されていない状態 */}
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
      </div>
    </PageStyle>
  );
}
