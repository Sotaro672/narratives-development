// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import type { ProductIDTagType } from "../../../../shell/src/shared/types/productBlueprint";
import { safeDateTimeLabelJa } from "../../../../shell/src/shared/util/dateJa";

import type {
  ProductBlueprintSizeRow as SizeRow,
  ModelNumberRow,
} from "../../infrastructure/api/productBlueprintApi";

import {
  getProductBlueprintDetail,
  listModelVariationsByProductBlueprintId,
  updateProductBlueprint,
} from "../../application/productBlueprintDetailService";

import type { Fit, ItemType } from "../../domain/entity/catalog";

import { mapVariationsToUiState } from "../util/variationMapper";
import { useBrandOptions, type BrandOption } from "./useBrandOptions";
import {
  useVariationsEditor,
  type VariationsUiState,
} from "./useVariationsEditor";

export {
  FIT_OPTIONS,
  PRODUCT_ID_TAG_OPTIONS,
  WASH_TAG_OPTIONS,
} from "../../domain/entity/catalog";
export type { Fit, WashTagOption } from "../../domain/entity/catalog";

// ------------------------------
// displayOrder で variations を並べ替えるユーティリティ
// ------------------------------
type ModelRefLike = { modelId?: string; displayOrder?: number };

/**
 * modelRefs(displayOrder) に従い variations を並べ替える。
 * - refs に存在する id は displayOrder 昇順で先頭に並べる
 * - refs に無い variations は “元の順” のまま末尾に回す
 */
function orderVariationsByModelRefs(
  variations: any[],
  modelRefs: ModelRefLike[] | undefined,
): any[] {
  if (!Array.isArray(variations) || variations.length === 0) return [];
  if (!Array.isArray(modelRefs) || modelRefs.length === 0) return variations;

  const byId = new Map<string, any>();
  for (const v of variations) {
    const id = typeof v?.id === "string" ? v.id.trim() : "";
    if (!id) continue;
    if (!byId.has(id)) byId.set(id, v);
  }

  const sortedRefs = [...modelRefs]
    .map((r) => ({
      id: typeof r?.modelId === "string" ? r.modelId.trim() : "",
      order: typeof r?.displayOrder === "number" ? r.displayOrder : Number.NaN,
    }))
    .filter((x) => x.id && Number.isFinite(x.order))
    .sort((a, b) => a.order - b.order);

  const used = new Set<string>();
  const ordered: any[] = [];

  for (const ref of sortedRefs) {
    const v = byId.get(ref.id);
    if (!v) continue;
    if (used.has(ref.id)) continue;
    used.add(ref.id);
    ordered.push(v);
  }

  for (const v of variations) {
    const id = typeof v?.id === "string" ? v.id.trim() : "";
    if (!id) continue;
    if (used.has(id)) continue;
    used.add(id);
    ordered.push(v);
  }

  return ordered;
}

function formatDateTimeYYYYMMDDHHmm(v: string | null | undefined): string {
  const label = safeDateTimeLabelJa(v, "");
  if (!label) return "";

  const m = label.match(/^(\d{4}\/\d{2}\/\d{2} \d{2}:\d{2})(?::\d{2})?$/);
  if (m) return m[1];

  return label;
}

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

  brandId: string;
  brandOptions: BrandOption[];
  brandLoading: boolean;
  brandError: Error | null;
  onChangeBrandId: (id: string) => void;

  colors: string[];
  colorInput: string;
  sizes: SizeRow[];
  modelNumbers: ModelNumberRow[];

  colorRgbMap: Record<string, string>;

  getCode: (sizeLabel: string, color: string) => string;

  assignee: string;

  creator: string;
  createdAt: string;
  updater: string;
  updatedAt: string;

  printed: boolean;

  onBack: () => void;
  onSave: () => void;
  onDelete: () => void;

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
  onChangeColorRgb: (name: string, hex: string) => void;

  onRemoveSize: (id: string) => void;
  onAddSize: () => void;
  onChangeSize: (id: string, patch: Partial<Omit<SizeRow, "id">>) => void;

  onChangeModelNumber: (
    sizeLabel: string,
    color: string,
    nextCode: string,
  ) => void;

  onClickAssignee: () => void;
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

  const [assignee, setAssignee] = React.useState("担当者未設定");

  const [creator, setCreator] = React.useState("作成者未設定");
  const [createdAt, setCreatedAt] = React.useState("");
  const [updater, setUpdater] = React.useState("");
  const [updatedAt, setUpdatedAt] = React.useState("");

  const [printed, setPrinted] = React.useState<boolean>(false);

  const [brandId, setBrandId] = React.useState<string>("");
  const [assigneeId, setAssigneeId] = React.useState<string>("");

  const [companyId, setCompanyId] = React.useState<string>("");
  const [brandNameFromService, setBrandNameFromService] =
    React.useState<string>("");

  const {
    colors,
    colorInput,
    sizes,
    modelNumbers,
    colorRgbMap,
    getCode,
    setFromUiState,
    onChangeColorInput,
    onAddColor,
    onRemoveColor,
    onChangeColorRgb,
    onRemoveSize,
    onAddSize,
    onChangeSize,
    onChangeModelNumber,
  } = useVariationsEditor();

  const {
    brandOptions,
    brandLoading,
    brandError,
    resolvedBrandName,
    getBrandNameById,
  } = useBrandOptions({
    companyId,
    brandId,
    brandNameFromService,
  });

  React.useEffect(() => {
    if (!blueprintId) return;

    (async () => {
      try {
        const detail = await getProductBlueprintDetail(blueprintId);

        const brandNameSvc = String((detail as any).brandName ?? "").trim();
        const assigneeNameFromService = (detail as any).assigneeName as
          | string
          | undefined;

        const createdByNameFromService = (detail as any).createdByName as
          | string
          | undefined;

        const updatedByNameFromService = (detail as any).updatedByName as
          | string
          | undefined;

        const productBlueprintIdResolved = detail.id ?? blueprintId;
        const itemTypeFromDetail = detail.itemType as ItemType;

        setPageTitle(detail.productName ?? productBlueprintIdResolved);
        setProductName(detail.productName ?? "");

        setPrinted(Boolean((detail as any).printed));

        setBrandId(detail.brandId ?? "");
        setAssigneeId(detail.assigneeId ?? "");

        setCompanyId(detail.companyId ?? "");
        setBrandNameFromService(brandNameSvc);

        setItemType(itemTypeFromDetail ?? "");
        setFit((detail.fit as Fit) ?? ("" as Fit));

        setMaterials(detail.material ?? "");
        setWeight(detail.weight ?? 0);
        setWashTags(detail.qualityAssurance ?? []);

        const tagType =
          (detail.productIdTag?.type as ProductIDTagType | undefined) ?? "";
        setProductIdTagType(tagType);

        const modelRefs = (detail as any).modelRefs as
          | ModelRefLike[]
          | undefined;

        try {
          const variations = await listModelVariationsByProductBlueprintId(
            productBlueprintIdResolved,
          );

          const ordered = orderVariationsByModelRefs(
            variations as any[],
            modelRefs,
          );

          const uiState = mapVariationsToUiState({
            varsAny: ordered as any[],
            itemType: itemTypeFromDetail,
          });

          setFromUiState(uiState as VariationsUiState);
        } catch {
          setFromUiState({
            colors: [],
            sizes: [],
            modelNumbers: [],
            colorRgbMap: {},
          });
        }

        setAssignee(
          assigneeNameFromService ?? detail.assigneeId ?? "担当者未設定",
        );

        setCreator(
          createdByNameFromService ?? detail.createdBy ?? "作成者未設定",
        );

        setCreatedAt(
          formatDateTimeYYYYMMDDHHmm((detail as any).createdAt) || "",
        );

        const updatedByRaw =
          (updatedByNameFromService ?? (detail as any).updatedBy ?? "") as any;
        const updaterName = String(updatedByRaw ?? "").trim();
        const updatedAtDisp =
          formatDateTimeYYYYMMDDHHmm((detail as any).updatedAt) || "";

        if (!updaterName || !updatedAtDisp) {
          setUpdater("");
          setUpdatedAt("");
        } else {
          setUpdater(updaterName);
          setUpdatedAt(updatedAtDisp);
        }
      } catch {
        //
      }
    })();
  }, [blueprintId, setFromUiState]);

  React.useEffect(() => {
    setBrand(resolvedBrandName ?? "");
  }, [resolvedBrandName]);

  const onSave = React.useCallback(() => {
    if (!blueprintId) {
      alert("商品設計ID が不明です");
      return;
    }

    if (!itemType) {
      alert("アイテム種別を選択してください");
      return;
    }

    const hasEmptyModelNumber = sizes.some((s) => {
      const sizeLabel = (s.sizeLabel ?? "").trim();
      if (!sizeLabel) return false;

      return colors.some((c) => {
        const color = (c ?? "").trim();
        if (!color) return false;

        const code = getCode(sizeLabel, color);
        return !code || !code.trim();
      });
    });

    if (hasEmptyModelNumber) {
      alert("モデルナンバーが空欄です");
      return;
    }

    (async () => {
      try {
        await updateProductBlueprint({
          id: blueprintId,
          productName,
          itemType: itemType as ItemType,
          fit,
          material: materials,
          weight,
          qualityAssurance: washTags,
          productIdTagType: productIdTagType || null,
          sizes,
          modelNumbers,
          colorRgbMap,
          brandId,
          assigneeId,
        } as any);

        alert("保存しました");
      } catch {
        alert("保存に失敗しました");
      }
    })();
  }, [
    blueprintId,
    productName,
    itemType,
    fit,
    materials,
    weight,
    washTags,
    productIdTagType,
    sizes,
    modelNumbers,
    colorRgbMap,
    brandId,
    assigneeId,
    colors,
    getCode,
  ]);

  const onDelete = React.useCallback(() => {
    alert("削除機能は現在無効です");
  }, []);

  const onBack = React.useCallback(() => {
    navigate("/productBlueprint");
  }, [navigate]);

  const onClickAssignee = React.useCallback(() => {}, []);

  const onChangeBrandId = React.useCallback(
    (id: string) => {
      const nextId = String(id ?? "").trim();
      setBrandId(nextId);

      const nextName = getBrandNameById(nextId);
      if (nextName) {
        setBrand(nextName);
      } else {
        setBrand(brandNameFromService || "");
      }
    },
    [getBrandNameById, brandNameFromService],
  );

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

    brandId,
    brandOptions,
    brandLoading,
    brandError,
    onChangeBrandId,

    colors,
    colorInput,
    sizes,
    modelNumbers,
    colorRgbMap,

    getCode,

    assignee,

    creator,
    createdAt,
    updater,
    updatedAt,

    printed,

    onBack,
    onSave,
    onDelete,

    onChangeProductName: setProductName,
    onChangeItemType: (v: ItemType) => setItemType(v),
    onChangeFit: setFit,
    onChangeMaterials: setMaterials,
    onChangeWeight: setWeight,
    onChangeWashTags: setWashTags,
    onChangeProductIdTag: (v: string) =>
      setProductIdTagType(v as ProductIDTagType),

    onChangeColorInput,
    onAddColor,
    onRemoveColor,
    onChangeColorRgb,

    onRemoveSize,
    onAddSize,
    onChangeSize,

    onChangeModelNumber,

    onClickAssignee,
  };
}