// frontend/console/productBlueprint/src/presentation/hooks/detail/useProductBlueprintDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import { safeDateTimeLabelJa } from "../../../../../shell/src/shared/util/dateJa";

import {
  getProductBlueprintDetail,
  listModelVariationsByProductBlueprintId,
  updateProductBlueprint,
} from "../../../application/productBlueprintDetailService";

import {
  isApparelCategoryCode,
  type ApparelModelNumberRow as ModelNumberRow,
  type ApparelSizeRow as SizeRow,
  type Fit,
} from "../../../domain/entity/apparel";

import { isAlcoholCategoryCode } from "../../../domain/entity/alcohol";

import type {
  AlcoholModelNumber,
  VolumeRow,
} from "../../../../../model/src/application/modelCreateService";

import type {
  CategoryFieldValue,
  CategoryFieldValues,
  ProductBlueprintCategorySnapshot,
} from "../../../domain/entity/productBlueprintCategory";

import { mapVariationsToUiState } from "../../util/variationMapper";
import { useBrandOptions, type BrandOption } from "../shared/useBrandOptions";
import {
  useVariationsEditor,
  type VariationsUiState,
} from "./useVariationsEditor";

export {
  FIT_OPTIONS,
  WASH_TAG_OPTIONS,
} from "../../../domain/entity/apparel";

export type { Fit, WashTagOption } from "../../../domain/entity/apparel";

type ModelRefLike = { modelId?: string; displayOrder?: number };

function orderVariationsByModelRefs(
  variations: any[],
  modelRefs: ModelRefLike[] | undefined,
): any[] {
  if (!Array.isArray(variations) || variations.length === 0) {
    return [];
  }

  if (!Array.isArray(modelRefs) || modelRefs.length === 0) {
    return variations;
  }

  const byId = new Map<string, any>();

  for (const variation of variations) {
    const id = typeof variation?.id === "string" ? variation.id.trim() : "";

    if (!id) {
      continue;
    }

    if (!byId.has(id)) {
      byId.set(id, variation);
    }
  }

  const sortedRefs = [...modelRefs]
    .map((ref) => ({
      id: typeof ref?.modelId === "string" ? ref.modelId.trim() : "",
      order:
        typeof ref?.displayOrder === "number"
          ? ref.displayOrder
          : Number.NaN,
    }))
    .filter((ref) => ref.id && Number.isFinite(ref.order))
    .sort((a, b) => a.order - b.order);

  const used = new Set<string>();
  const ordered: any[] = [];

  for (const ref of sortedRefs) {
    const variation = byId.get(ref.id);

    if (!variation) {
      continue;
    }

    if (used.has(ref.id)) {
      continue;
    }

    used.add(ref.id);
    ordered.push(variation);
  }

  for (const variation of variations) {
    const id = typeof variation?.id === "string" ? variation.id.trim() : "";

    if (!id) {
      continue;
    }

    if (used.has(id)) {
      continue;
    }

    used.add(id);
    ordered.push(variation);
  }

  return ordered;
}

function formatDateTimeYYYYMMDDHHmm(v: string | null | undefined): string {
  const label = safeDateTimeLabelJa(v, "");

  if (!label) {
    return "";
  }

  const m = label.match(/^(\d{4}\/\d{2}\/\d{2} \d{2}:\d{2})(?::\d{2})?$/);

  if (m) {
    return m[1];
  }

  return label;
}

function getCategoryLabel(
  category: ProductBlueprintCategorySnapshot | null,
): string {
  if (!category) {
    return "";
  }

  return (
    category.nameJa ||
    category.nameEn ||
    category.code ||
    category.id ||
    ""
  );
}

function normalizeCategoryFieldValue(value: unknown): CategoryFieldValue {
  if (
    typeof value === "string" ||
    typeof value === "number" ||
    typeof value === "boolean" ||
    value === null
  ) {
    return value;
  }

  if (Array.isArray(value)) {
    return value
      .map((item) => {
        if (
          typeof item === "string" ||
          typeof item === "number" ||
          typeof item === "boolean" ||
          item === null
        ) {
          return item;
        }

        return null;
      })
      .filter((item) => item !== null);
  }

  if (typeof value === "object" && value !== null) {
    const out: Record<string, string | number | boolean | null> = {};

    for (const [key, item] of Object.entries(value)) {
      if (
        typeof item === "string" ||
        typeof item === "number" ||
        typeof item === "boolean" ||
        item === null
      ) {
        out[key] = item;
      }
    }

    return out;
  }

  return null;
}

function removeModelOwnedCategoryFields(
  fields: CategoryFieldValues,
): CategoryFieldValues {
  const next: CategoryFieldValues = { ...fields };

  /**
   * alcohol volume は model domain 管轄。
   * ProductBlueprint.categoryFields には保持しない。
   */
  delete next.volume;

  return next;
}

function normalizeCategoryFields(value: unknown): CategoryFieldValues {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return {};
  }

  const out: CategoryFieldValues = {};

  for (const [key, fieldValue] of Object.entries(value)) {
    if (key === "volume") {
      continue;
    }

    out[key] = normalizeCategoryFieldValue(fieldValue);
  }

  return out;
}

function getStringFromFields(
  fields: CategoryFieldValues,
  key: string,
): string {
  const value = fields[key];
  return typeof value === "string" ? value : "";
}

function getNumberFromFields(
  fields: CategoryFieldValues,
  key: string,
): number | null {
  const value = fields[key];

  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  return null;
}

function getStringArrayFromFields(
  fields: CategoryFieldValues,
  key: string,
): string[] {
  const value = fields[key];

  if (!Array.isArray(value)) {
    return [];
  }

  return value.filter(
    (item): item is string => typeof item === "string" && item.trim() !== "",
  );
}

function buildApparelCategoryFieldsForSave(args: {
  base: CategoryFieldValues;
  fit: Fit;
  material: string;
  weight: number;
  washTags: string[];
}): CategoryFieldValues {
  return removeModelOwnedCategoryFields({
    ...args.base,
    fit: args.fit || null,
    material: args.material.trim() === "" ? null : args.material,
    weight: Number.isFinite(args.weight) ? args.weight : 0,
    washTags: args.washTags,
  });
}

function emptyVariationsUiState(): VariationsUiState {
  return {
    colors: [],
    sizes: [],
    modelNumbers: [],
    colorRgbMap: {},
    volumes: [],
    alcoholModelNumbers: [],
  };
}

export interface UseProductBlueprintDetailResult {
  pageTitle: string;

  productName: string;
  brand: string;

  productBlueprintCategoryId: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot | null;
  productBlueprintCategoryLabel: string;
  isApparelCategory: boolean;
  isAlcoholCategory: boolean;

  fit: Fit;
  materials: string;
  weight: number;
  washTags: string[];

  categoryFields: CategoryFieldValues;
  onChangeCategoryField: (key: string, value: CategoryFieldValue) => void;

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

  volumes: VolumeRow[];
  alcoholModelNumbers: AlcoholModelNumber[];

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

  onAddVolume: () => void;
  onRemoveVolume: (id: string) => void;
  onChangeVolume: (id: string, patch: Partial<Omit<VolumeRow, "id">>) => void;
  onChangeAlcoholModelNumber: (
    volumeLabel: string,
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

  const isAlcoholCategory = React.useMemo(() => {
    const code = String(productBlueprintCategory?.code ?? "").trim();
    return isAlcoholCategoryCode(code);
  }, [productBlueprintCategory]);

  const [fit, setFit] = React.useState<Fit>("" as Fit);
  const [materials, setMaterials] = React.useState<string>("");
  const [weight, setWeight] = React.useState<number>(0);
  const [washTags, setWashTags] = React.useState<string[]>([]);
  const [categoryFields, setCategoryFields] =
    React.useState<CategoryFieldValues>({});

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
    volumes,
    alcoholModelNumbers,
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
    onAddVolume,
    onRemoveVolume,
    onChangeVolume,
    onChangeAlcoholModelNumber,
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
    if (!blueprintId) {
      return;
    }

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
        const categoryFieldsFromDetail = removeModelOwnedCategoryFields(
          normalizeCategoryFields((detail as any).categoryFields),
        );

        const fitFromCategoryFields = getStringFromFields(
          categoryFieldsFromDetail,
          "fit",
        );

        const materialFromCategoryFields = getStringFromFields(
          categoryFieldsFromDetail,
          "material",
        );

        const weightFromCategoryFields = getNumberFromFields(
          categoryFieldsFromDetail,
          "weight",
        );

        const washTagsFromCategoryFields = getStringArrayFromFields(
          categoryFieldsFromDetail,
          "washTags",
        );

        setPageTitle(detail.productName ?? productBlueprintIdResolved);
        setProductName(detail.productName ?? "");

        setPrinted(Boolean((detail as any).printed));

        setBrandId(detail.brandId ?? "");
        setAssigneeId(detail.assigneeId ?? "");

        setCompanyId(detail.companyId ?? "");
        setBrandNameFromService(brandNameSvc);

        setProductBlueprintCategory(categoryFromDetail);
        setCategoryFields(categoryFieldsFromDetail);

        setFit(
          (fitFromCategoryFields ||
            ((detail as any).fit as string | undefined) ||
            "") as Fit,
        );

        setMaterials(
          materialFromCategoryFields ||
            ((detail as any).material as string | undefined) ||
            "",
        );

        setWeight(
          weightFromCategoryFields ??
            (typeof (detail as any).weight === "number"
              ? ((detail as any).weight as number)
              : 0),
        );

        setWashTags(
          washTagsFromCategoryFields.length > 0
            ? washTagsFromCategoryFields
            : Array.isArray((detail as any).qualityAssurance)
              ? ((detail as any).qualityAssurance as unknown[]).filter(
                  (tag): tag is string =>
                    typeof tag === "string" && tag.trim() !== "",
                )
              : [],
        );

        const modelRefs = (detail as any).modelRefs as
          | ModelRefLike[]
          | undefined;

        const categoryCode = String(categoryFromDetail?.code ?? "").trim();

        /**
         * 色・サイズ・採寸・モデルナンバー、および酒類の容量・モデルナンバーは
         * productBlueprint detail 本体ではなく、
         * /models/by-blueprint/:id/variations のレスポンスを正とする。
         */
        if (
          isApparelCategoryCode(categoryCode) ||
          isAlcoholCategoryCode(categoryCode)
        ) {
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

            setFromUiState({
              ...emptyVariationsUiState(),
              ...(uiState as VariationsUiState),
            });
          } catch {
            setFromUiState(emptyVariationsUiState());
          }
        } else {
          setFromUiState(emptyVariationsUiState());
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

  const onChangeCategoryField = React.useCallback(
    (key: string, value: CategoryFieldValue) => {
      if (key === "volume") {
        setCategoryFields((prev) => {
          const next = { ...prev };
          delete next.volume;
          return next;
        });
        return;
      }

      setCategoryFields((prev) => ({
        ...prev,
        [key]: value,
      }));

      if (key === "fit" && typeof value === "string") {
        setFit(value as Fit);
        return;
      }

      if (key === "material") {
        setMaterials(typeof value === "string" ? value : "");
        return;
      }

      if (key === "weight") {
        setWeight(typeof value === "number" ? value : 0);
        return;
      }

      if (key === "washTags" || key === "qualityAssurance") {
        setWashTags(getStringArrayFromFields({ [key]: value }, key));
      }
    },
    [],
  );

  const onChangeFit = React.useCallback((value: Fit) => {
    setFit(value);

    setCategoryFields((prev) =>
      removeModelOwnedCategoryFields({
        ...prev,
        fit: value || null,
      }),
    );
  }, []);

  const onChangeMaterials = React.useCallback((value: string) => {
    setMaterials(value);

    setCategoryFields((prev) =>
      removeModelOwnedCategoryFields({
        ...prev,
        material: value.trim() === "" ? null : value,
      }),
    );
  }, []);

  const onChangeWeight = React.useCallback((value: number) => {
    const next = Number.isFinite(value) ? value : 0;

    setWeight(next);

    setCategoryFields((prev) =>
      removeModelOwnedCategoryFields({
        ...prev,
        weight: next,
      }),
    );
  }, []);

  const onChangeWashTags = React.useCallback((value: string[]) => {
    const next = Array.isArray(value)
      ? value.filter((tag) => typeof tag === "string" && tag.trim() !== "")
      : [];

    setWashTags(next);

    setCategoryFields((prev) =>
      removeModelOwnedCategoryFields({
        ...prev,
        washTags: next,
      }),
    );
  }, []);

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
      const hasEmptyModelNumber = sizes.some((size) => {
        const sizeLabel = (size.sizeLabel ?? "").trim();

        if (!sizeLabel) {
          return false;
        }

        return colors.some((colorValue) => {
          const color = (colorValue ?? "").trim();

          if (!color) {
            return false;
          }

          const code = getCode(sizeLabel, color);
          return !code || !code.trim();
        });
      });

      if (hasEmptyModelNumber) {
        alert("モデルナンバーが空欄です");
        return;
      }
    }

    if (isAlcoholCategory) {
      if (volumes.length === 0) {
        alert("容量バリエーションを1つ以上登録してください。");
        return;
      }

      const hasInvalidVolume = volumes.some((volume) => {
        const value = volume.volumeValue;
        const unit = String(volume.volumeUnit ?? "").trim();

        return (
          typeof value !== "number" ||
          !Number.isFinite(value) ||
          value <= 0 ||
          !unit
        );
      });

      if (hasInvalidVolume) {
        alert("容量は 0 より大きい値と単位を入力してください。");
        return;
      }

      const hasEmptyAlcoholModelNumber = volumes.some((volume) => {
        const label = `${volume.volumeValue}${String(
          volume.volumeUnit ?? "",
        ).trim()}`;

        if (!label.trim()) {
          return false;
        }

        return !alcoholModelNumbers.some(
          (modelNumber) =>
            modelNumber.volumeLabel === label && modelNumber.code.trim(),
        );
      });

      if (hasEmptyAlcoholModelNumber) {
        alert("容量ごとのモデルナンバーをすべて入力してください。");
        return;
      }
    }

    (async () => {
      try {
        const nextCategoryFields = isApparelCategory
          ? buildApparelCategoryFieldsForSave({
              base: categoryFields,
              fit,
              material: materials,
              weight,
              washTags,
            })
          : removeModelOwnedCategoryFields(categoryFields);

        await updateProductBlueprint({
          id: blueprintId,
          productName,
          productBlueprintCategoryId,
          productBlueprintCategory,
          productIdTagType: "qr",
          sizes: isApparelCategory ? sizes : [],
          modelNumbers: isApparelCategory ? modelNumbers : [],
          colorRgbMap: isApparelCategory ? colorRgbMap : {},
          colors: isApparelCategory ? colors : [],
          volumes: isAlcoholCategory ? volumes : [],
          alcoholModelNumbers: isAlcoholCategory ? alcoholModelNumbers : [],
          brandId,
          assigneeId,
          companyId,
          categoryFields: nextCategoryFields,
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
    categoryFields,
    sizes,
    modelNumbers,
    colorRgbMap,
    colors,
    volumes,
    alcoholModelNumbers,
    brandId,
    assigneeId,
    companyId,
    isApparelCategory,
    isAlcoholCategory,
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
      setCategoryFields({});

      const code = String(category?.code ?? "").trim();
      const nextIsApparel = isApparelCategoryCode(code);
      const nextIsAlcohol = isAlcoholCategoryCode(code);

      if (!nextIsApparel && !nextIsAlcohol) {
        setFromUiState(emptyVariationsUiState());
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
    isAlcoholCategory,

    fit,
    materials,
    weight,
    washTags,

    categoryFields: removeModelOwnedCategoryFields(categoryFields),
    onChangeCategoryField,

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

    volumes,
    alcoholModelNumbers,

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

    onChangeFit,
    onChangeMaterials,
    onChangeWeight,
    onChangeWashTags,

    onChangeColorInput,
    onAddColor,
    onRemoveColor,
    onChangeColorRgb,

    onRemoveSize,
    onAddSize,
    onChangeSize,

    onChangeModelNumber,

    onAddVolume,
    onRemoveVolume,
    onChangeVolume,
    onChangeAlcoholModelNumber,

    onClickAssignee,
  };
}