// frontend/productBlueprint/src/presentation/pages/productBlueprintDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import ProductBlueprintCard from "../components/productBlueprintCard";
import ColorVariationCard from "../../../../model/src/presentation/components/ColorVariationCard";
import SizeVariationCard from "../../../../model/src/presentation/components/SizeVariationCard";
import ModelNumberCard from "../../../../model/src/presentation/components/ModelNumberCard";

import { PRODUCT_BLUEPRINTS } from "../../infrastructure/mockdata/productBlueprint_mockdata";
import {
  MODEL_NUMBERS,
  SIZE_VARIATIONS,
} from "../../../../model/src/infrastructure/mockdata/mockdata";
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

// ProductIDTagType → 表示用ラベル（必要な場面で使用）
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
  // backend/internal/domain/productBlueprint/entity.go に合わせたフィールドを使用
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
    () =>
      blueprint?.qualityAssurance ?? [
        "手洗い",
        "ドライクリーニング",
        "陰干し",
      ],
  );

  // Tag は entity.go / shared types 準拠で productIdTagType のみを扱う
  const [productIdTagType, setProductIdTagType] =
    React.useState<ProductIDTagType | "">(
      () => blueprint?.productIdTagType ?? "",
    );

  // カラー（本来は blueprint.variations から復元するが、現状モック固定）
  const [colorInput, setColorInput] = React.useState("");
  const [colors, setColors] = React.useState<string[]>([
    "ホワイト",
    "ブラック",
    "ネイビー",
  ]);

  // サイズ（mockdata.ts からインポート）
  const [sizes, setSizes] = React.useState(() =>
    SIZE_VARIATIONS.map((v, i) => ({
      id: String(i + 1),
      sizeLabel: v.size,
      chest: v.measurements["身幅"] ?? 0,
      waist: v.measurements["ウエスト"] ?? 0,
      length: v.measurements["着丈"] ?? 0,
      shoulder: v.measurements["肩幅"] ?? 0,
    })),
  );

  // モデルナンバー（mockdata.ts からインポート）
  const [modelNumbers] = React.useState(() =>
    MODEL_NUMBERS.map((m) => ({
      size: m.size,
      color: m.color,
      code: m.modelNumber,
    })),
  );

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
        <ProductBlueprintCard
          mode="edit"
          productName={productName}
          brand={brand}
          fit={fit}
          materials={materials}
          weight={weight}
          washTags={washTags}
          // カードにはコード値("qr" | "nfc")を渡す
          productIdTag={productIdTagType || ""}
          onChangeProductName={setProductName}
          onChangeFit={setFit}
          onChangeMaterials={setMaterials}
          onChangeWeight={setWeight}
          onChangeWashTags={setWashTags}
          onChangeProductIdTag={(v: string) =>
            setProductIdTagType(v as ProductIDTagType)
          }
        />

        <ColorVariationCard
          mode="edit"
          colors={colors}
          colorInput={colorInput}
          onChangeColorInput={setColorInput}
          onAddColor={addColor}
          onRemoveColor={removeColor}
        />

        <SizeVariationCard
          mode="edit"
          sizes={sizes}
          onRemove={(id: string) =>
            setSizes((prev) => prev.filter((s) => s.id !== id))
          }
        />

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
