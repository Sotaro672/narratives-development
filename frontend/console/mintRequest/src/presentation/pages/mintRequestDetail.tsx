// frontend/console/mintRequest/src/presentation/pages/mintRequestDetail.tsx
import { useState } from "react";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import { Card, CardContent } from "../../../../shell/src/shared/ui/card";
import { Button } from "../../../../shell/src/shared/ui/button";
import { Coins } from "lucide-react";

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
  } = useMintRequestDetail();

  // PageHeader 用の保存ボタンハンドラ（現状はダミー）
  const handleSave = () => {
    // TODO: 必要になったら保存処理を実装
  };

  // ── 右カラム: ブランド選択用 state（将来的に tokenBlueprint 絞り込みに利用） ──
  const [selectedBrandId, setSelectedBrandId] = useState<string>("");

  // TODO: 後で useMintRequestDetail などに渡して tokenBlueprint の絞り込みに利用する
  const handleBrandChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setSelectedBrandId(e.target.value);
  };

  // 現状はダミーの options（後で API から取得したブランド一覧に差し替える想定）
  const brandOptions: { id: string; name: string }[] = [
    // 例:
    // { id: "all", name: "すべてのブランド" },
    // { id: "brand-1", name: "Brand A" },
    // { id: "brand-2", name: "Brand B" },
  ];

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

      {/* 右カラム：ブランド選択カード（tokenBlueprint 絞り込み用） */}
      <div className="space-y-4 mt-4">
        <Card className="mint-request-card">
          <CardContent className="mint-request-card__body space-y-3">
            <div className="flex flex-col gap-1">
              <span className="font-semibold text-sm">ブランドで絞り込み</span>
              <span className="text-xs text-gray-500">
                選択したブランドに紐づく TokenBlueprint を表示します。
              </span>
            </div>

            <div className="flex flex-col gap-2">
              <label className="text-xs text-gray-600" htmlFor="brand-filter">
                ブランド
              </label>
              <select
                id="brand-filter"
                className="mint-request-select w-full"
                value={selectedBrandId}
                onChange={handleBrandChange}
              >
                <option value="">すべてのブランド</option>
                {brandOptions.map((b) => (
                  <option key={b.id} value={b.id}>
                    {b.name}
                  </option>
                ))}
              </select>
            </div>

            {/* 将来的に、選択中ブランドや該当 TokenBlueprint 件数などを表示しても良い */}
            {selectedBrandId && (
              <div className="text-xs text-gray-600">
                選択中のブランド ID: <code>{selectedBrandId}</code>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </PageStyle>
  );
}
