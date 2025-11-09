// frontend/productBlueprint/src/pages/productBlueprintCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../admin/src/presentation/components/AdminCard";
import ProductBlueprintCard from "./productBlueprintCard";
import ColorVariationCard from "../../../model/src/pages/ColorVariationCard";
import SizeVariationCard, {
  type SizeRow,
} from "../../../model/src/pages/SizeVariationCard";
import ModelNumberCard, {
  type ModelNumber,
} from "../../../model/src/pages/ModelNumberCard";

type Fit =
  | "レギュラーフィット"
  | "スリムフィット"
  | "リラックスフィット"
  | "オーバーサイズ";

export default function ProductBlueprintCreate() {
  const navigate = useNavigate();

  // ─────────────────────────────────────────
  // 初期値はすべて空（プリフィルなし）
  // ─────────────────────────────────────────
  const [productName, setProductName] = React.useState("");
  const [brand] = React.useState("");
  const [fit, setFit] = React.useState<Fit>("レギュラーフィット");
  const [materials, setMaterials] = React.useState("");
  const [weight, setWeight] = React.useState<number>(0);
  const [washTags, setWashTags] = React.useState<string[]>([]);
  const [productIdTag, setProductIdTag] = React.useState("");

  // カラー
  const [colorInput, setColorInput] = React.useState("");
  const [colors, setColors] = React.useState<string[]>([]);

  // サイズ
  const [sizes, setSizes] = React.useState<SizeRow[]>([]);

  // モデルナンバー
  const [modelNumbers] = React.useState<ModelNumber[]>([]);

  // 管理情報（新規作成では空）
  const [assignee, setAssignee] = React.useState("");
  const [creator] = React.useState("");
  const [createdAt] = React.useState("");

  // 作成ボタン押下時
  const onCreate = () => {
    alert("商品設計を作成しました（ダミー）");
    navigate(-1);
  };

  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  // カラー追加/削除ハンドラ
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
      onSave={onCreate} // 保存ボタン → 「作成ボタン」扱い
    >
      {/* --- 左ペイン --- */}
      <div>
        {/* 商品設計カード（編集モード・プリフィルなし） */}
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

        {/* カラー variation */}
        <ColorVariationCard
          colors={colors}
          colorInput={colorInput}
          onChangeColorInput={setColorInput}
          onAddColor={addColor}
          onRemoveColor={removeColor}
        />

        {/* サイズ variation */}
        <SizeVariationCard
          sizes={sizes}
          onRemove={(id: string) =>
            setSizes((prev) => prev.filter((s) => s.id !== id))
          }
        />

        {/* モデルナンバー */}
        <ModelNumberCard
          sizes={sizes}
          colors={colors}
          modelNumbers={modelNumbers}
        />
      </div>

      {/* --- 右ペイン（管理情報） --- */}
      <AdminCard
        title="管理情報"
        assigneeName={assignee || "未設定"}
        createdByName={creator || "未設定"}
        createdAt={createdAt || "未設定"}
        onEditAssignee={() => setAssignee("担当者A")}
        onClickAssignee={() => console.log("assignee clicked:", assignee)}
        onClickCreatedBy={() => console.log("createdBy clicked:", creator)}
      />
    </PageStyle>
  );
}
