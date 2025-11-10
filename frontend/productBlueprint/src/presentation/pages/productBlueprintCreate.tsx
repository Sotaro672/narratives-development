// frontend/productBlueprint/src/presentation/pages/productBlueprintCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";
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
import {
  type ItemType,
  type ProductIDTagType,
} from "../../domain/entity/productBlueprint";

type Fit =
  | "レギュラーフィット"
  | "スリムフィット"
  | "リラックスフィット"
  | "オーバーサイズ";

export default function ProductBlueprintCreate() {
  const navigate = useNavigate();

  const [productName, setProductName] = React.useState("");
  const [brandId, setBrandId] = React.useState("");
  const [itemType, setItemType] = React.useState<ItemType>("tops");

  const [fit, setFit] = React.useState<Fit>("レギュラーフィット");
  const [material, setMaterial] = React.useState("");
  const [weight, setWeight] = React.useState<number>(0);

  const [qualityAssurance, setQualityAssurance] = React.useState<string[]>([]);
  const [productIdTagType, setProductIdTagType] =
    React.useState<ProductIDTagType>("qr");

  const [colorInput, setColorInput] = React.useState("");
  const [colors, setColors] = React.useState<string[]>([]);
  const [sizes, setSizes] = React.useState<SizeRow[]>([]);
  const [modelNumbers] = React.useState<ModelNumber[]>([]);

  const [assigneeId, setAssigneeId] = React.useState("");
  const [createdBy] = React.useState("");
  const [createdAt] = React.useState("");

  // brandId -> brandName は後で brandService で解決予定
  const brandName = React.useMemo(() => {
    return brandId || "";
  }, [brandId]);

  const onCreate = () => {
    // TODO: variations生成＋API呼び出し
    alert("商品設計を作成しました（ダミー）");
    navigate(-1);
  };

  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

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
      title="商品設計を作成"
      onBack={onBack}
      onSave={onCreate}
    >
      <div>
        <ProductBlueprintCard
          mode="edit"
          productName={productName}
          brand={brandName}
          fit={fit}
          materials={material}
          weight={weight}
          washTags={qualityAssurance}
          productIdTag={productIdTagType}
          onChangeProductName={setProductName}
          // brandId 選択UI実装時に onChangeBrandId を追加して brandId を更新予定
          onChangeFit={setFit}
          onChangeMaterials={setMaterial}
          onChangeWeight={setWeight}
          onChangeWashTags={setQualityAssurance}
          onChangeProductIdTag={(v: string) =>
            setProductIdTagType(v as ProductIDTagType)
          }
        />

        <ColorVariationCard
          colors={colors}
          colorInput={colorInput}
          onChangeColorInput={setColorInput}
          onAddColor={addColor}
          onRemoveColor={removeColor}
        />

        <SizeVariationCard
          sizes={sizes}
          onRemove={(id: string) =>
            setSizes((prev) => prev.filter((s) => s.id !== id))
          }
        />

        <ModelNumberCard
          sizes={sizes}
          colors={colors}
          modelNumbers={modelNumbers}
        />
      </div>

      <AdminCard
        title="管理情報"
        assigneeName={assigneeId || "未設定"}
        createdByName={createdBy || "未設定"}
        createdAt={createdAt || "未設定"}
        onEditAssignee={() => setAssigneeId("担当者A")}
        onClickAssignee={() =>
          console.log("assigneeId clicked:", assigneeId)
        }
        onClickCreatedBy={() =>
          console.log("createdBy clicked:", createdBy)
        }
      />
    </PageStyle>
  );
}
