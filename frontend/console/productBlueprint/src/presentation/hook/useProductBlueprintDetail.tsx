// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import type { ProductIDTagType } from "../../../../shell/src/shared/types/productBlueprint";
import {
  brandLabelFromId,
  fetchProductBlueprintById,
  fetchProductBlueprintSizeRows,
  fetchProductBlueprintModelNumberRows,
  formatProductBlueprintDate,
  type SizeRow,
  type ModelNumberRow,
} from "../../infrastructure/api/productBlueprintApi";

// ▼ 選択肢などのカタログ情報をドメイン層から使用
import {
  FIT_OPTIONS,
  PRODUCT_ID_TAG_OPTIONS,
  WASH_TAG_OPTIONS,
} from "../../domain/entity/catalog";
import type { Fit, WashTagOption } from "../../domain/entity/catalog";

export {
  FIT_OPTIONS,
  PRODUCT_ID_TAG_OPTIONS,
  WASH_TAG_OPTIONS,
} from "../../domain/entity/catalog";
export type { Fit, WashTagOption } from "../../domain/entity/catalog";

export interface UseProductBlueprintDetailResult {
  pageTitle: string;

  productName: string;
  brand: string;
  fit: Fit;
  materials: string;
  weight: number;
  washTags: string[];
  productIdTag: ProductIDTagType | "";

  colors: string[];
  colorInput: string;
  sizes: SizeRow[];
  modelNumbers: ModelNumberRow[];

  // ★ 追加：ModelNumberCard 用
  getCode: (sizeLabel: string, color: string) => string;

  assignee: string;
  creator: string;
  createdAt: string;

  onBack: () => void;
  onSave: () => void;

  onChangeProductName: (v: string) => void;
  onChangeFit: (v: Fit) => void;
  onChangeMaterials: (v: string) => void;
  onChangeWeight: (v: number) => void;
  onChangeWashTags: (v: string[]) => void;
  onChangeProductIdTag: (v: string) => void;

  onChangeColorInput: (v: string) => void;
  onAddColor: () => void;
  onRemoveColor: (name: string) => void;

  onRemoveSize: (id: string) => void;

  onEditAssignee: () => void;
  onClickAssignee: () => void;
  onClickCreatedBy: () => void;
}

export function useProductBlueprintDetail(): UseProductBlueprintDetailResult {
  const navigate = useNavigate();
  const { blueprintId } = useParams<{ blueprintId: string }>();

  const blueprint = React.useMemo(
    () => fetchProductBlueprintById(blueprintId),
    [blueprintId],
  );

  const pageTitle =
    blueprint?.productName ?? blueprintId ?? "不明ID";

  const [productName, setProductName] = React.useState(
    () => blueprint?.productName ?? "シルクブラウス プレミアムライン",
  );

  const [brand] = React.useState(
    () => (blueprint ? brandLabelFromId(blueprint.brandId) : ""),
  );

  const [fit, setFit] = React.useState<Fit>("" as Fit);

  const [materials, setMaterials] = React.useState(
    () => blueprint?.material ?? "シルク100%、裏地:ポリエステル100%",
  );

  const [weight, setWeight] = React.useState<number>(
    () => blueprint?.weight ?? 180,
  );

  const [washTags, setWashTags] = React.useState<string[]>(() =>
    blueprint?.qualityAssurance ?? ["手洗い", "ドライクリーニング", "陰干し"],
  );

  const [productIdTagType, setProductIdTagType] =
    React.useState<ProductIDTagType | "">(
      () => (blueprint?.productIdTag as ProductIDTagType | undefined) ?? "",
    );

  const [colorInput, setColorInput] = React.useState("");
  const [colors, setColors] = React.useState<string[]>([
    "ホワイト",
    "ブラック",
    "ネイビー",
  ]);

  const [sizes, setSizes] = React.useState<SizeRow[]>(() =>
    fetchProductBlueprintSizeRows(),
  );

  const [modelNumbers] = React.useState<ModelNumberRow[]>(() =>
    fetchProductBlueprintModelNumberRows(),
  );

  const [assignee, setAssignee] = React.useState(
    () => blueprint?.assigneeId ?? "担当者未設定",
  );
  const [creator] = React.useState(
    () => blueprint?.createdBy ?? "作成者未設定",
  );
  const [createdAt] = React.useState(
    () => formatProductBlueprintDate(blueprint?.createdAt) || "2024/1/15",
  );

  // -----------------------------
  // ★ getCode を追加
  // -----------------------------
  const getCode = React.useCallback(
    (sizeLabel: string, color: string): string => {
      const row = modelNumbers.find(
        (m) => m.size === sizeLabel && m.color === color,
      );
      return row?.code ?? "";
    },
    [modelNumbers],
  );

  // -----------------------------
  // Handlers
  // -----------------------------
  const onSave = React.useCallback(() => {
    alert("保存しました（ダミー）");
  }, []);

  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  const onAddColor = React.useCallback(() => {
    const v = colorInput.trim();
    if (!v || colors.includes(v)) return;
    setColors((prev) => [...prev, v]);
    setColorInput("");
  }, [colorInput, colors]);

  const onRemoveColor = React.useCallback((name: string) => {
    setColors((prev) => prev.filter((c) => c !== name));
  }, []);

  const onRemoveSize = React.useCallback((id: string) => {
    setSizes((prev) => prev.filter((s) => s.id !== id));
  }, []);

  const onEditAssignee = React.useCallback(() => {
    setAssignee("新担当者");
  }, []);

  const onClickAssignee = React.useCallback(() => {
    console.log("assignee clicked:", assignee);
  }, [assignee]);

  const onClickCreatedBy = React.useCallback(() => {
    console.log("createdBy clicked:", creator);
  }, [creator]);

  return {
    pageTitle,

    productName,
    brand,
    fit,
    materials,
    weight,
    washTags,
    productIdTag: productIdTagType || "",

    colors,
    colorInput,
    sizes,
    modelNumbers,

    getCode, // ★ ModelNumberCard に渡す

    assignee,
    creator,
    createdAt,

    onBack,
    onSave,

    onChangeProductName: setProductName,
    onChangeFit: setFit,
    onChangeMaterials: setMaterials,
    onChangeWeight: setWeight,
    onChangeWashTags: setWashTags,
    onChangeProductIdTag: (v: string) =>
      setProductIdTagType(v as ProductIDTagType),

    onChangeColorInput: setColorInput,
    onAddColor,
    onRemoveColor,

    onRemoveSize,

    onEditAssignee,
    onClickAssignee,
    onClickCreatedBy,
  };
}
