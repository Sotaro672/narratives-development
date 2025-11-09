// frontend/production/src/pages/productionDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../admin/src/pages/AdminCard";

// 閲覧モードで呼び出す対象
import ProductBlueprintCard from "../../../productBlueprint/src/pages/productBlueprintCard";
import ColorVariationCard from "../../../model/src/pages/ColorVariationCard";
import SizeVariationCard, {
  type SizeRow,
} from "../../../model/src/pages/SizeVariationCard";
import ModelNumberCard, {
  type ModelNumber,
} from "../../../model/src/pages/ModelNumberCard";

// 生産数カード（編集モードで使用）
import ProductionQuantityCard, {
  type QuantityCell,
} from "./productionQuantityCard";

// 印刷ボタン用
import { Card, CardContent } from "../../../shared/ui/card";
import { Button } from "../../../shared/ui/button";
import { Printer } from "lucide-react";

type Fit =
  | "レギュラーフィット"
  | "スリムフィット"
  | "リラックスフィット"
  | "オーバーサイズ";

export default function ProductionDetail() {
  const navigate = useNavigate();
  const { productionId } = useParams<{ productionId: string }>();

  // ─────────────────────────────────────────
  // モックデータ
  // ─────────────────────────────────────────
  const [productName] = React.useState("シルクブラウス プレミアムライン");
  const [brand] = React.useState("LUMINA Fashion");
  const [fit] = React.useState<Fit>("レギュラーフィット");
  const [materials] = React.useState("シルク100%、裏地:ポリエステル100%");
  const [weight] = React.useState<number>(180);
  const [washTags] = React.useState<string[]>([
    "手洗い",
    "ドライクリーニング",
    "陰干し",
  ]);
  const [productIdTag] = React.useState("QRコード");

  // カラー
  const [colors] = React.useState<string[]>([
    "ホワイト",
    "ブラック",
    "ネイビー",
  ]);
  const [colorInput] = React.useState("");

  // サイズ
  const [sizes] = React.useState<SizeRow[]>([
    { id: "1", sizeLabel: "S", chest: 48, waist: 58, length: 60, shoulder: 38 },
    { id: "2", sizeLabel: "M", chest: 50, waist: 60, length: 62, shoulder: 40 },
    { id: "3", sizeLabel: "L", chest: 52, waist: 62, length: 64, shoulder: 42 },
  ]);

  // モデルナンバー
  const [modelNumbers] = React.useState<ModelNumber[]>([
    { size: "S", color: "ホワイト", code: "LM-SB-S-WHT" },
    { size: "S", color: "ブラック", code: "MN-001" },
    { size: "S", color: "ネイビー", code: "MN-001" },
    { size: "M", color: "ホワイト", code: "LM-SB-M-WHT" },
    { size: "M", color: "ブラック", code: "LM-SB-M-BLK" },
    { size: "M", color: "ネイビー", code: "LM-SB-M-NVY" },
    { size: "L", color: "ホワイト", code: "LM-SB-L-WHT" },
    { size: "L", color: "ブラック", code: "LM-SB-L-BLK" },
    { size: "L", color: "ネイビー", code: "LM-SB-L-NVY" },
  ]);

  // 管理情報
  const [assignee, setAssignee] = React.useState("佐藤 美咲");
  const [creator] = React.useState("佐藤 美咲");
  const [createdAt] = React.useState("2024/1/15");

  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  // 閲覧モード用 no-op
  const noop = () => {};
  const noopStr = (_v: string) => {};
  const noopRemove = (_id: string) => {};

  // ─────────────────────────────────────────
  // 生産数（編集可能）
  // ─────────────────────────────────────────
  const sizeLabels = React.useMemo(
    () => sizes.map((s) => s.sizeLabel),
    [sizes],
  );

  const [quantities, setQuantities] = React.useState<QuantityCell[]>([
    { size: "S", color: "ホワイト", qty: 2 },
    { size: "S", color: "ブラック", qty: 0 },
    { size: "S", color: "ネイビー", qty: 0 },
    { size: "M", color: "ホワイト", qty: 1 },
    { size: "M", color: "ブラック", qty: 1 },
    { size: "M", color: "ネイビー", qty: 1 },
    { size: "L", color: "ホワイト", qty: 1 },
    { size: "L", color: "ブラック", qty: 1 },
    { size: "L", color: "ネイビー", qty: 1 },
  ]);

  const handleChangeQty = React.useCallback(
    (size: string, color: string, nextQty: number) => {
      setQuantities((prev) => {
        const idx = prev.findIndex(
          (q) => q.size === size && q.color === color,
        );
        if (idx === -1) {
          return [...prev, { size, color, qty: nextQty }];
        }
        const next = prev.slice();
        next[idx] = { ...next[idx], qty: nextQty };
        return next;
      });
    },
    [],
  );

  // 総数
  const grandTotal = React.useMemo(
    () => quantities.reduce((sum, q) => sum + (q.qty ?? 0), 0),
    [quantities],
  );

  // 保存（PageHeader の action）
  const handleSave = React.useCallback(() => {
    console.log("保存:", {
      productionId,
      productName,
      quantities,
      grandTotal,
    });
    alert("生産計画を保存しました（ダミー）");
  }, [productionId, productName, quantities, grandTotal]);

  // 印刷
  const handlePrint = React.useCallback(() => {
    console.log("商品IDを印刷する:", {
      productionId,
      productName,
      quantities,
    });
  }, [productionId, productName, quantities]);

  return (
    <PageStyle
      layout="grid-2"
      title={productionId ?? "不明ID"}
      onBack={onBack}
      onSave={handleSave}
    >
      {/* --- 左ペイン --- */}
      <div>
        {/* 基本情報カード：閲覧モード */}
        <ProductBlueprintCard
          mode="view"
          productName={productName}
          brand={brand}
          fit={fit}
          materials={materials}
          weight={weight}
          washTags={washTags}
          productIdTag={productIdTag}
          onChangeProductName={noopStr}
          onChangeFit={noop as (v: Fit) => void}
          onChangeMaterials={noopStr}
          onChangeWeight={noop as (v: number) => void}
          onChangeWashTags={noop as (next: string[]) => void}
          onChangeProductIdTag={noopStr}
        />

        {/* カラーバリエーション：閲覧モード */}
        <ColorVariationCard
          mode="view"
          colors={colors}
          colorInput={colorInput}
          onChangeColorInput={noopStr}
          onAddColor={noop}
          onRemoveColor={noopStr}
        />

        {/* サイズバリエーション：閲覧モード */}
        <SizeVariationCard
          mode="view"
          sizes={sizes}
          onRemove={noopRemove}
        />

        {/* モデルナンバー：閲覧モード */}
        <ModelNumberCard
          mode="view"
          sizes={sizes}
          colors={colors}
          modelNumbers={modelNumbers}
        />

        {/* 生産数：編集モード */}
        <ProductionQuantityCard
          mode="edit"
          sizes={sizeLabels}
          colors={colors}
          quantities={quantities}
          onChangeQty={handleChangeQty}
        />

        {/* 印刷ボタンカード */}
        <Card className="mt-2">
          <CardContent className="py-6">
            <Button
              onClick={handlePrint}
              className="w-full h-12 text-base flex items-center justify-center gap-2"
            >
              <Printer size={18} />
              商品IDを印刷する
            </Button>
          </CardContent>
        </Card>
      </div>

      {/* --- 右ペイン（管理情報） --- */}
      <AdminCard
        title="管理情報"
        assigneeName={assignee}
        createdByName={creator}
        createdAt={createdAt}
        onEditAssignee={() => setAssignee("新担当者")}
        onClickAssignee={() => console.log("assignee clicked:", assignee)}
        onClickCreatedBy={() => console.log("createdBy clicked:", creator)}
      />
    </PageStyle>
  );
}
