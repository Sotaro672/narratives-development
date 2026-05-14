// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import { safeDateTimeLabelJa } from "../../../../shell/src/shared/util/dateJa";

import {
  getProductBlueprintDetail,
  listModelVariationsByProductBlueprintId,
  updateProductBlueprint,
} from "../../application/productBlueprintDetailService";

import {
  FIT_OPTIONS,
  WASH_TAG_OPTIONS,
  isApparelCategoryCode,
  type ApparelModelNumberRow as ModelNumberRow,
  type ApparelSizeRow as SizeRow,
  type Fit,
} from "../../domain/entity/apparel";

import type { ProductBlueprintCategorySnapshot } from "../../domain/entity/productBlueprintCategory";

import { mapVariationsToUiState } from "../util/variationMapper";
import { useBrandOptions, type BrandOption } from "./useBrandOptions";
import {
  useVariationsEditor,
  type VariationsUiState,
} from "./useVariationsEditor";

export {
  FIT_OPTIONS,
  WASH_TAG_OPTIONS,
} from "../../domain/entity/apparel";

export type { Fit, WashTagOption } from "../../domain/entity/apparel";

// ------------------------------
// displayOrder で variations を並べ替えるユーティリティ
// ------------------------------

type ModelRefLike = { modelId?: string; displayOrder?: number };

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

function getCategoryLabel(
  category: ProductBlueprintCategorySnapshot | null,
): string {
  if (!category) return "";

  return (
    category.nameJa ||
    category.nameEn ||
    category.code ||
    category.id ||
    ""
  );
}

export interface UseProductBlueprintDetailResult {
  pageTitle: string;

  productName: string;
  brand: string;

  productBlueprintCategoryId: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot | null;
  productBlueprintCategoryLabel: string;
  isApparelCategory: boolean;

  fit: Fit;
  materials: string;
  weight: number;
  washTags: string[];

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
  onChangeProductBlueprintCategory: (
    category: ProductBlueprintCategorySnapshot | null,
  ) => void;
  onChangeFit: (v: Fit) => void;
  onChangeMaterials: (v: string) => void;
  onChangeWeight: (v: number) => void;
  onChangeWashTags: (v: string[]) => void;

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

  const [productBlueprintCategory, setProductBlueprintCategory] =
    React.useState<ProductBlueprintCategorySnapshot | null>(null);

  const productBlueprintCategoryId = React.useMemo(
    () => productBlueprintCategory?.id ?? "",
    [productBlueprintCategory],
  );

  const productBlueprintCategoryLabel = React.useMemo(
    () => getCategoryLabel(productBlueprintCategory),
    [productBlueprintCategory],
  );

  const isApparelCategory = React.useMemo(() => {
    const code = String(productBlueprintCategory?.code ?? "").trim();
    return isApparelCategoryCode(code);
  }, [productBlueprintCategory]);

  const [fit, setFit] = React.useState<Fit>("" as Fit);

  const [materials, setMaterials] = React.useState<string>("");
  const [weight, setWeight] = React.useState<number>(0);
  const [washTags, setWashTags] = React.useState<string[]>([]);

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
        const categoryFromDetail = detail.productBlueprintCategory ?? null;

        setPageTitle(detail.productName ?? productBlueprintIdResolved);
        setProductName(detail.productName ?? "");

        setPrinted(Boolean((detail as any).printed));

        setBrandId(detail.brandId ?? "");
        setAssigneeId(detail.assigneeId ?? "");

        setCompanyId(detail.companyId ?? "");
        setBrandNameFromService(brandNameSvc);

        setProductBlueprintCategory(categoryFromDetail);
        setFit((detail.fit as Fit) ?? ("" as Fit));

        setMaterials(detail.material ?? "");
        setWeight(detail.weight ?? 0);
        setWashTags(detail.qualityAssurance ?? []);

        const modelRefs = (detail as any).modelRefs as
          | ModelRefLike[]
          | undefined;

        const categoryCode = String(categoryFromDetail?.code ?? "").trim();

        if (isApparelCategoryCode(categoryCode)) {
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
              categoryCode,
            } as any);

            setFromUiState(uiState as VariationsUiState);
          } catch {
            setFromUiState({
              colors: [],
              sizes: [],
              modelNumbers: [],
              colorRgbMap: {},
            });
          }
        } else {
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

    if (!productBlueprintCategoryId || !productBlueprintCategory) {
      alert("商品カテゴリを選択してください");
      return;
    }

    if (isApparelCategory) {
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
    }

    (async () => {
      try {
        await updateProductBlueprint({
          id: blueprintId,
          productName,
          productBlueprintCategoryId,
          productBlueprintCategory,
          fit,
          material: materials,
          weight,
          qualityAssurance: washTags,
          productIdTagType: "qr",
          sizes: isApparelCategory ? sizes : [],
          modelNumbers: isApparelCategory ? modelNumbers : [],
          colorRgbMap: isApparelCategory ? colorRgbMap : {},
          colors: isApparelCategory ? colors : [],
          brandId,
          assigneeId,
          companyId,
          categoryFields: null,
        });

        alert("保存しました");
      } catch {
        alert("保存に失敗しました");
      }
    })();
  }, [
    blueprintId,
    productName,
    productBlueprintCategoryId,
    productBlueprintCategory,
    fit,
    materials,
    weight,
    washTags,
    sizes,
    modelNumbers,
    colorRgbMap,
    colors,
    brandId,
    assigneeId,
    companyId,
    isApparelCategory,
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

  const onChangeProductBlueprintCategory = React.useCallback(
    (category: ProductBlueprintCategorySnapshot | null) => {
      setProductBlueprintCategory(category);

      const code = String(category?.code ?? "").trim();
      const nextIsApparel = isApparelCategoryCode(code);

      if (!nextIsApparel) {
        setFromUiState({
          colors: [],
          sizes: [],
          modelNumbers: [],
          colorRgbMap: {},
        });
      }
    },
    [setFromUiState],
  );

  return {
    pageTitle,

    productName,
    brand,
    productBlueprintCategoryId,
    productBlueprintCategory,
    productBlueprintCategoryLabel,
    isApparelCategory,
    fit,
    materials,
    weight,
    washTags,

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
    onChangeProductBlueprintCategory,
    onChangeFit: setFit,
    onChangeMaterials: setMaterials,
    onChangeWeight: setWeight,
    onChangeWashTags: setWashTags,

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