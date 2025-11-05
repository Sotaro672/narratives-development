// frontend/productBlueprint/src/pages/productBlueprintDetail.tsx
import * as React from "react";
import PageHeader from "./PageHeader";
import AdminCard from "../../../admin/src/pages/AdminCard";
import ProductBlueprintCard from "./productBlueprintCard";
import VariationCard from "../../../model/src/pages/ColorVariationCard";
import SizeVariationCard, { type SizeRow } from "../../../model/src/pages/SizeVariationCard";
import ModelNumberCard, { type ModelNumber } from "../../../model/src/pages/ModelNumberCard";
import "./productBlueprintDetail.css";

type Fit =
  | "レギュラーフィット"
  | "スリムフィット"
  | "リラックスフィット"
  | "オーバーサイズ";

export default function ProductBlueprintDetail() {
  // 基本情報
  const [productName, setProductName] = React.useState("シルクブラウス プレミアムライン");
  const [brand] = React.useState("LUMINA Fashion");
  const [fit, setFit] = React.useState<Fit>("レギュラーフィット");
  const [materials, setMaterials] = React.useState("シルク100%、裏地:ポリエステル100%");
  const [weight, setWeight] = React.useState<number>(180);
  const [washTags, setWashTags] = React.useState<string[]>(["手洗い", "ドライクリーニング", "陰干し"]);
  const [productIdTag, setProductIdTag] = React.useState("QRコード");

  // カラー
  const [colorInput, setColorInput] = React.useState("");
  const [colors, setColors] = React.useState<string[]>(["ホワイト", "ブラック", "ネイビー"]);

  // サイズ
  const [sizes, setSizes] = React.useState<SizeRow[]>([
    { id: "1", sizeLabel: "S", chest: 48, waist: 58, length: 60, shoulder: 38 },
    { id: "2", sizeLabel: "M", chest: 50, waist: 60, length: 62, shoulder: 40 },
    { id: "3", sizeLabel: "L", chest: 52, waist: 62, length: 64, shoulder: 42 },
  ]);

  // モデルナンバー（サイズ×カラーのマトリクス）
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

  const onSave = () => alert("保存しました（ダミー）");

  // VariationCard handlers
  const addColor = () => {
    const v = colorInput.trim();
    if (!v || colors.includes(v)) return;
    setColors((prev) => [...prev, v]);
    setColorInput("");
  };
  const removeColor = (name: string) =>
    setColors((prev) => prev.filter((c) => c !== name));

  return (
    <div className="pbp">
      <PageHeader title="商品設計詳細" onSave={onSave} />

      <div className="grid-2">
        {/* 左ペイン */}
        <div>
          <ProductBlueprintCard
            productName={productName}
            brand={brand}
            fit={fit}
            materials={materials}
            weight={weight}
            washTags={washTags}
            productIdTag={productIdTag}
            onChangeProductName={setProductName}
            onChangeFit={setFit}
            onChangeMaterials={setMaterials}
            onChangeWeight={setWeight}
            onChangeWashTags={setWashTags}
            onChangeProductIdTag={setProductIdTag}
          />

          <VariationCard
            colors={colors}
            colorInput={colorInput}
            onChangeColorInput={setColorInput}
            onAddColor={addColor}
            onRemoveColor={removeColor}
          />

          <SizeVariationCard
            sizes={sizes}
            onRemove={(id) => setSizes((prev) => prev.filter((s) => s.id !== id))}
          />

          <ModelNumberCard sizes={sizes} colors={colors} modelNumbers={modelNumbers} />
        </div>

        {/* 右ペイン */}
        <AdminCard
          assignee={assignee}
          creator={creator}
          createdAt={createdAt}
          onEditAssignee={(next) => setAssignee(next)}
        />
      </div>
    </div>
  );
}
