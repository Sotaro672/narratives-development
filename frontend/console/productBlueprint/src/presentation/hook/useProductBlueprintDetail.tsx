// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import type { ProductIDTagType } from "../../../../shell/src/shared/types/productBlueprint";
import {
  brandLabelFromId,
  formatProductBlueprintDate,
  type SizeRow,
  type ModelNumberRow,
} from "../../infrastructure/api/productBlueprintApi";

import { getProductBlueprintDetail } from "../../application/productBlueprintDetailService";

import type { Fit, ItemType, WashTagOption } from "../../domain/entity/catalog";

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
  itemType: ItemType | "";
  fit: Fit;
  materials: string;
  weight: number;
  washTags: string[];
  productIdTag: ProductIDTagType | "";

  colors: string[];
  colorInput: string;
  sizes: SizeRow[];
  modelNumbers: ModelNumberRow[];

  getCode: (sizeLabel: string, color: string) => string;

  assignee: string;
  creator: string;
  createdAt: string;

  onBack: () => void;
  onSave: () => void;

  onChangeProductName: (v: string) => void;
  onChangeItemType: (v: ItemType) => void;
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

  const [pageTitle, setPageTitle] = React.useState<string>("");

  const [productName, setProductName] = React.useState<string>("");
  const [brand, setBrand] = React.useState<string>("");

  const [itemType, setItemType] = React.useState<ItemType | "">("");
  const [fit, setFit] = React.useState<Fit>("" as Fit);

  const [materials, setMaterials] = React.useState<string>("");
  const [weight, setWeight] = React.useState<number>(0);
  const [washTags, setWashTags] = React.useState<string[]>([]);

  const [productIdTagType, setProductIdTagType] =
    React.useState<ProductIDTagType | "">("");

  const [colorInput, setColorInput] = React.useState("");
  const [colors, setColors] = React.useState<string[]>([]);
  const [sizes, setSizes] = React.useState<SizeRow[]>([]);
  const [modelNumbers, setModelNumbers] = React.useState<ModelNumberRow[]>([]);

  const [assignee, setAssignee] = React.useState("担当者未設定");
  const [creator, setCreator] = React.useState("作成者未設定");
  const [createdAt, setCreatedAt] = React.useState("");

  // ---------------------------------
  // service → 詳細データを反映
  // ---------------------------------
  React.useEffect(() => {
    if (!blueprintId) return;

    (async () => {
      try {
        const detail = await getProductBlueprintDetail(blueprintId);

        console.log("[useProductBlueprintDetail] mapped detail:", detail);

        // service から来る拡張フィールド
        const brandNameFromService = (detail as any).brandName as
          | string
          | undefined;
        const assigneeNameFromService = (detail as any).assigneeName as
          | string
          | undefined;
        const createdByNameFromService = (detail as any).createdByName as
          | string
          | undefined;

        setPageTitle(detail.productName ?? blueprintId);
        setProductName(detail.productName ?? "");

        // brand: service の brandName を優先、なければ従来の brandLabelFromId
        setBrand(
          brandNameFromService ?? brandLabelFromId(detail.brandId),
        );

        setItemType((detail.itemType as ItemType) ?? "");
        setFit((detail.fit as Fit) ?? ("" as Fit));

        setMaterials(detail.material ?? "");
        setWeight(detail.weight ?? 0);
        setWashTags(detail.qualityAssurance ?? []);

        const tagType =
          (detail.productIdTag?.type as ProductIDTagType | undefined) ?? "";
        setProductIdTagType(tagType);

        // variations から colors/sizes/modelNumbers を今後生成予定
        setColors([]);
        setSizes([]);
        setModelNumbers([]);

        // assignee: service の assigneeName を優先
        setAssignee(
          assigneeNameFromService ??
            detail.assigneeId ??
            "担当者未設定",
        );

        // creator: service の createdByName を優先
        setCreator(
          createdByNameFromService ??
            detail.createdBy ??
            "作成者未設定",
        );

        setCreatedAt(formatProductBlueprintDate(detail.createdAt) || "");
      } catch (e) {
        console.error("[useProductBlueprintDetail] fetch failed:", e);
      }
    })();
  }, [blueprintId]);

  // ---------------------------------
  // ModelNumberCard 用
  // ---------------------------------
  const getCode = React.useCallback(
    (sizeLabel: string, color: string): string => {
      const row = modelNumbers.find(
        (m) => m.size === sizeLabel && m.color === color,
      );
      return row?.code ?? "";
    },
    [modelNumbers],
  );

  // ---------------------------------
  // Handlers
  // ---------------------------------
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
    itemType,
    fit,
    materials,
    weight,
    washTags,
    productIdTag: productIdTagType || "",

    colors,
    colorInput,
    sizes,
    modelNumbers,

    getCode,

    assignee,
    creator,
    createdAt,

    onBack,
    onSave,

    onChangeProductName: setProductName,
    onChangeItemType: (v: ItemType) => setItemType(v),
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
