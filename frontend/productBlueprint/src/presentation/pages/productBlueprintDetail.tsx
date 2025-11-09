// frontend/productBlueprint/src/pages/productBlueprintDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import ProductBlueprintCard from "../components/productBlueprintCard";
import ColorVariationCard from "../../../../model/src/presentation/components/ColorVariationCard";
import SizeVariationCard, {
  type SizeRow,
} from "../../../../model/src/presentation/components/SizeVariationCard";
import ModelNumberCard, {
  type ModelNumber,
} from "../../../../model/src/presentation/components/ModelNumberCard";

import { PRODUCT_BLUEPRINTS } from "../../infrastructure/mockdata/mockdata";
import type {
  ProductBlueprint,
  ProductIDTagType,
} from "../../../../shell/src/shared/types/productBlueprint";

type Fit =
  | "レギュラーフィット"
  | "スリムフィット"
  | "リラックスフィット"
  | "オーバーサイズ";

// BrandID → 表示名（モック用）
const brandLabelFromId = (brandId: string): string => {
  switch (brandId) {
    case "brand_lumina":
      return "LUMINA Fashion";
    case "brand_nexus":
      return "NEXUS Street";
    default:
      return brandId || "不明ブランド";
  }
};

// ISO8601 → "YYYY/M/D" 表示
const formatDate = (iso?: string | null): string => {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const y = d.getFullYear();
  const m = d.getMonth() + 1;
  const day = d.getDate();
  return `${y}/${m}/${day}`;
};

// ProductIDTagType → 表示用ラベル
const productIdTagLabel = (type?: ProductIDTagType): string => {
  if (type === "qr") return "QRコード";
  if (type === "nfc") return "NFCタグ";
  return "未設定";
};

export default function ProductBlueprintDetail() {
  const navigate = useNavigate();
  const { blueprintId } = useParams<{ blueprintId: string }>();

  // 対象 Blueprint をモックから取得
  const blueprint: ProductBlueprint | undefined = React.useMemo(
    () => PRODUCT_BLUEPRINTS.find((pb) => pb.id === blueprintId),
    [blueprintId],
  );

  // ─────────────────────────────────────────
  // 初期値（存在しない場合はダミー）
  // ─────────────────────────────────────────
  const [productName, setProductName] = React.useState(
    () => blueprint?.productName ?? "シルクブラウス プレミアムライン",
  );
  const [brand] = React.useState(
    () => (blueprint ? brandLabelFromId(blueprint.brandId) : "LUMINA Fashion"),
  );
  const [fit, setFit] = React.useState<Fit>("レギュラーフィット");
  const [materials, setMaterials] = React.useState(
    () => blueprint?.material ?? "シルク100%、裏地:ポリエステル100%",
  );
  const [weight, setWeight] = React.useState<number>(
    () => blueprint?.weight ?? 180,
  );
  const [washTags, setWashTags] = React.useState<string[]>(
    () => blueprint?.qualityAssurance ?? ["手洗い", "ドライクリーニング", "陰干し"],
  );
  const [productIdTag, setProductIdTag] = React.useState<string>(
    () => productIdTagLabel(blueprint?.productIdTag?.type),
  );

  // カラー（本来は blueprint.variations から導出する想定・ここでは従来モックを維持）
  const [colorInput, setColorInput] = React.useState("");
  const [colors, setColors] = React.useState<string[]>([
    "ホワイト",
    "ブラック",
    "ネイビー",
  ]);

  // サイズ
  const [sizes, setSizes] = React.useState<SizeRow[]>([
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
  const [assignee, setAssignee] = React.useState(
    () => blueprint?.assigneeId ?? "担当者未設定",
  );
  const [creator] = React.useState(
    () => blueprint?.createdBy ?? "作成者未設定",
  );
  const [createdAt] = React.useState(
    () => formatDate(blueprint?.createdAt) || "2024/1/15",
  );

  const onSave = () => {
    // TODO: 後で usecase 経由で API 呼び出しに差し替え
    alert("保存しました（ダミー）");
  };

  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

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
    <PageStyle
      layout="grid-2"
      title={blueprint?.productName ?? blueprintId ?? "不明ID"}
      onBack={onBack}
      onSave={onSave}
    >
      {/* --- 左ペイン --- */}
      <div>
        {/* 商品設計カード：編集モード（Shared Types に準拠した値を渡す） */}
        <ProductBlueprintCard
          mode="edit"
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

        {/* カラー：編集モード */}
        <ColorVariationCard
          mode="edit"
          colors={colors}
          colorInput={colorInput}
          onChangeColorInput={setColorInput}
          onAddColor={addColor}
          onRemoveColor={removeColor}
        />

        {/* サイズ：編集モード */}
        <SizeVariationCard
          mode="edit"
          sizes={sizes}
          onRemove={(id: string) =>
            setSizes((prev) => prev.filter((s) => s.id !== id))
          }
        />

        {/* モデルナンバー：編集モード */}
        <ModelNumberCard
          mode="edit"
          sizes={sizes}
          colors={colors}
          modelNumbers={modelNumbers}
        />
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
