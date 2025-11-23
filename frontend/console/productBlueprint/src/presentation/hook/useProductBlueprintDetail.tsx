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

// 他のプレゼン層からも使えるように再エクスポートしておく
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

/**
 * 商品設計詳細画面のロジック・状態管理用カスタムフック
 * ページコンポーネント側にはスタイル/構造のみを残す。
 */
export function useProductBlueprintDetail(): UseProductBlueprintDetailResult {
  const navigate = useNavigate();
  const { blueprintId } = useParams<{ blueprintId: string }>();

  // 対象 Blueprint をモック API から取得
  const blueprint = React.useMemo(
    () => fetchProductBlueprintById(blueprintId),
    [blueprintId],
  );

  const pageTitle =
    blueprint?.productName ?? blueprintId ?? "不明ID";

  // ─────────────────────────────────────────
  // 初期値（存在しない場合はダミー）
  // backend/internal/domain/productBlueprint/entity.go に合わせたフィールドを使用
  // ─────────────────────────────────────────
  const [productName, setProductName] = React.useState(
    () => blueprint?.productName ?? "シルクブラウス プレミアムライン",
  );

  const [brand] = React.useState(
    () => (blueprint ? brandLabelFromId(blueprint.brandId) : ""),
  );

  // ★ フィットは自動で「レギュラーフィット」にせず、空からスタート
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

  // サイズ（API から取得）
  const [sizes, setSizes] = React.useState<SizeRow[]>(() =>
    fetchProductBlueprintSizeRows(),
  );

  // モデルナンバー（API から取得）
  const [modelNumbers] = React.useState<ModelNumberRow[]>(() =>
    fetchProductBlueprintModelNumberRows(),
  );

  // 管理情報
  const [assignee, setAssignee] = React.useState(
    () => blueprint?.assigneeId ?? "担当者未設定",
  );
  const [creator] = React.useState(
    () => blueprint?.createdBy ?? "作成者未設定",
  );
  const [createdAt] = React.useState(
    () => formatProductBlueprintDate(blueprint?.createdAt) || "2024/1/15",
  );

  const onSave = React.useCallback(() => {
    alert("保存しました（ダミー）");
  }, []);

  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  // VariationCard handlers
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
