// frontend/console/shell/src/features/productBlueprint/presentation/hooks/detail/useProductBlueprintDetail.tsx

import * as React from "react";
import {
  useNavigate,
  useParams,
} from "react-router-dom";

import {
  safeDateTimeLabelJa,
} from "../../../../../shared/util/dateJa";

import {
  getProductBlueprintDetail,
  listModelVariationsByProductBlueprintId,
  updateProductBlueprint,
} from "../../../application/productBlueprintDetailService";

import {
  isApparelCategoryCode,
  type ApparelModelNumberRow as ModelNumberRow,
  type ApparelSizeInput,
} from "../../../../../shared/types/apparel";

import {
  isAlcoholCategoryCode,
} from "../../../domain/alcohol";

import type {
  AlcoholModelNumber,
  VolumeRow,
} from "../../../../model/application/modelCreateService";

import type {
  CategoryFieldValue,
  CategoryFieldValues,
  ProductBlueprintCategorySnapshot,
} from "../../../domain/productBlueprintCategory";

import {
  mapVariationsToUiState,
} from "../../util/variationMapper";

import {
  useBrandOptions,
  type BrandOption,
} from "../shared/useBrandOptions";

import {
  useProductBlueprintVariations,
} from "../shared/useProductBlueprintVariations";

type SizeRow =
  ApparelSizeInput & {
    id: string;
  };

type ModelRefLike = {
  modelId?: string;
  displayOrder?: number;
};

function orderVariationsByModelRefs(
  variations: any[],
  modelRefs:
    | ModelRefLike[]
    | undefined,
): any[] {
  if (
    !Array.isArray(variations) ||
    variations.length === 0
  ) {
    return [];
  }

  if (
    !Array.isArray(modelRefs) ||
    modelRefs.length === 0
  ) {
    return variations;
  }

  const byId =
    new Map<string, any>();

  for (
    const variation
    of variations
  ) {
    const id =
      typeof variation?.id === "string"
        ? variation.id
        : "";

    if (!id) {
      continue;
    }

    if (!byId.has(id)) {
      byId.set(
        id,
        variation,
      );
    }
  }

  const sortedRefs =
    [...modelRefs]
      .filter(
        (
          ref,
        ): ref is {
          modelId: string;
          displayOrder: number;
        } =>
          typeof ref.modelId ===
            "string" &&
          ref.modelId !== "" &&
          typeof ref.displayOrder ===
            "number" &&
          Number.isFinite(
            ref.displayOrder,
          ),
      )
      .sort(
        (a, b) =>
          a.displayOrder -
          b.displayOrder,
      );

  const used =
    new Set<string>();

  const ordered:
    any[] = [];

  for (
    const ref
    of sortedRefs
  ) {
    const variation =
      byId.get(
        ref.modelId,
      );

    if (
      !variation ||
      used.has(
        ref.modelId,
      )
    ) {
      continue;
    }

    used.add(
      ref.modelId,
    );

    ordered.push(
      variation,
    );
  }

  for (
    const variation
    of variations
  ) {
    const id =
      typeof variation?.id === "string"
        ? variation.id
        : "";

    if (
      !id ||
      used.has(id)
    ) {
      continue;
    }

    used.add(id);

    ordered.push(
      variation,
    );
  }

  return ordered;
}

function formatDateTimeYYYYMMDDHHmm(
  value:
    | string
    | null
    | undefined,
): string {
  const label =
    safeDateTimeLabelJa(
      value,
      "",
    );

  if (!label) {
    return "";
  }

  const matched =
    label.match(
      /^(\d{4}\/\d{2}\/\d{2} \d{2}:\d{2})(?::\d{2})?$/,
    );

  return matched?.[1] ??
    label;
}

export interface UseProductBlueprintDetailResult {
  pageTitle: string;

  productName: string;
  brand: string;

  productBlueprintCategoryId:
    string;

  productBlueprintCategory:
    | ProductBlueprintCategorySnapshot
    | null;

  productBlueprintCategoryLabel:
    string;

  isApparelCategory:
    boolean;

  isAlcoholCategory:
    boolean;

  categoryFields:
    CategoryFieldValues;

  onChangeCategoryField: (
    key: string,
    value:
      CategoryFieldValue,
  ) => void;

  brandId: string;
  brandOptions: BrandOption[];
  brandLoading: boolean;
  brandError: Error | null;

  onChangeBrandId: (
    id: string,
  ) => void;

  colors: string[];
  colorInput: string;
  sizes: SizeRow[];

  modelNumbers:
    ModelNumberRow[];

  colorRgbMap:
    Record<string, string>;

  volumes: VolumeRow[];

  alcoholModelNumbers:
    AlcoholModelNumber[];

  getCode: (
    sizeLabel: string,
    color: string,
  ) => string;

  assignee: string;

  creator: string;
  createdAt: string;
  updater: string;
  updatedAt: string;

  printed: boolean;

  onBack: () => void;
  onSave: () => void;
  onDelete: () => void;

  onChangeProductName: (
    value: string,
  ) => void;

  onChangeProductBlueprintCategory: (
    category:
      | ProductBlueprintCategorySnapshot
      | null,
  ) => void;

  onChangeColorInput: (
    value: string,
  ) => void;

  onAddColor: () => void;

  onRemoveColor: (
    name: string,
  ) => void;

  onChangeColorRgb: (
    name: string,
    hex: string,
  ) => void;

  onRemoveSize: (
    id: string,
  ) => void;

  onAddSize: () => void;

  onChangeSize: (
    id: string,
    patch:
      Partial<
        Omit<
          SizeRow,
          "id"
        >
      >,
  ) => void;

  onChangeModelNumber: (
    sizeLabel: string,
    color: string,
    nextCode: string,
  ) => void;

  onAddVolume: () => void;

  onRemoveVolume: (
    id: string,
  ) => void;

  onChangeVolume: (
    id: string,
    patch:
      Partial<
        Omit<
          VolumeRow,
          "id"
        >
      >,
  ) => void;

  onChangeAlcoholModelNumber: (
    volumeLabel: string,
    nextCode: string,
  ) => void;

  onClickAssignee:
    () => void;
}

export function useProductBlueprintDetail():
  UseProductBlueprintDetailResult {
  const navigate =
    useNavigate();

  const {
    blueprintId,
  } = useParams<{
    blueprintId: string;
  }>();

  const [
    productName,
    setProductName,
  ] =
    React.useState("");

  const [
    productBlueprintCategory,
    setProductBlueprintCategory,
  ] =
    React.useState<
      | ProductBlueprintCategorySnapshot
      | null
    >(null);

  const [
    categoryFields,
    setCategoryFields,
  ] =
    React.useState<
      CategoryFieldValues
    >({});

  const [
    assignee,
    setAssignee,
  ] =
    React.useState(
      "担当者未設定",
    );

  const [
    creator,
    setCreator,
  ] =
    React.useState(
      "作成者未設定",
    );

  const [
    createdAt,
    setCreatedAt,
  ] =
    React.useState("");

  const [
    updater,
    setUpdater,
  ] =
    React.useState("");

  const [
    updatedAt,
    setUpdatedAt,
  ] =
    React.useState("");

  const [
    printed,
    setPrinted,
  ] =
    React.useState(false);

  const [
    brandId,
    setBrandId,
  ] =
    React.useState("");

  const [
    assigneeId,
    setAssigneeId,
  ] =
    React.useState("");

  const [
    companyId,
    setCompanyId,
  ] =
    React.useState("");

  const productBlueprintCategoryId =
    productBlueprintCategory?.id ??
    "";

  const productBlueprintCategoryLabel =
    productBlueprintCategory?.nameJa ||
    productBlueprintCategory?.nameEn ||
    productBlueprintCategory?.code ||
    productBlueprintCategory?.id ||
    "";

  const pageTitle =
    productName ||
    blueprintId ||
    "";

  const {
    isApparelCategory,
    isAlcoholCategory,
    colors,
    colorInput,
    sizes,
    modelNumbers,
    colorRgbMap,
    volumes,
    alcoholModelNumbers,
    getCode,
    setFromUiState,
    resetVariations,
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
  } =
    useProductBlueprintVariations({
      productBlueprintCategory,
    });

  const {
    brandOptions,
    brandLoading,
    brandError,
    resolvedBrandName,
  } =
    useBrandOptions({
      companyId,
      brandId,
    });

  const brand =
    resolvedBrandName;

  React.useEffect(() => {
    if (!blueprintId) {
      return;
    }

    void (async () => {
      try {
        const detail =
          await getProductBlueprintDetail(
            blueprintId,
          );

        const categoryFromDetail =
          detail
            .productBlueprintCategory;

        const nextCategoryFields =
          detail.categoryFields ??
          {};

        setProductName(
          detail.productName,
        );

        setPrinted(
          detail.printed === true,
        );

        setBrandId(
          detail.brandId,
        );

        setAssigneeId(
          detail.assigneeId ??
            "",
        );

        setCompanyId(
          detail.companyId ??
            "",
        );

        setProductBlueprintCategory(
          categoryFromDetail,
        );

        setCategoryFields(
          nextCategoryFields,
        );

        const nextCategoryCode =
          categoryFromDetail.code;

        if (
          isApparelCategoryCode(
            nextCategoryCode,
          ) ||
          isAlcoholCategoryCode(
            nextCategoryCode,
          )
        ) {
          try {
            const variations =
              await listModelVariationsByProductBlueprintId(
                detail.id,
              );

            const orderedVariations =
              orderVariationsByModelRefs(
                variations as any[],
                detail.modelRefs,
              );

            setFromUiState(
              mapVariationsToUiState({
                varsAny:
                  orderedVariations,

                categoryCode:
                  nextCategoryCode,
              }),
            );
          } catch {
            resetVariations();
          }
        } else {
          resetVariations();
        }

        setAssignee(
          detail.assigneeName ||
            detail.assigneeId ||
            "担当者未設定",
        );

        setCreator(
          detail.createdByName ||
            detail.createdBy ||
            "作成者未設定",
        );

        setCreatedAt(
          formatDateTimeYYYYMMDDHHmm(
            detail.createdAt,
          ),
        );

        const nextUpdater =
          (
            detail.updatedByName ||
            detail.updatedBy ||
            ""
          ).trim();

        const nextUpdatedAt =
          formatDateTimeYYYYMMDDHHmm(
            detail.updatedAt,
          );

        if (
          nextUpdater &&
          nextUpdatedAt
        ) {
          setUpdater(
            nextUpdater,
          );

          setUpdatedAt(
            nextUpdatedAt,
          );
        } else {
          setUpdater("");
          setUpdatedAt("");
        }
      } catch {
        resetVariations();
      }
    })();
  }, [
    blueprintId,
    resetVariations,
    setFromUiState,
  ]);

  const onChangeCategoryField =
    React.useCallback(
      (
        key: string,
        value:
          CategoryFieldValue,
      ) => {
        setCategoryFields(
          (previous) => ({
            ...previous,
            [key]: value,
          }),
        );
      },
      [],
    );

  const onSave =
    React.useCallback(() => {
      if (!blueprintId) {
        alert(
          "商品設計IDが不明です。",
        );

        return;
      }

      if (
        !productBlueprintCategory
      ) {
        alert(
          "商品カテゴリを選択してください。",
        );

        return;
      }

      void updateProductBlueprint({
        id:
          blueprintId,

        productName,

        productBlueprintCategoryId,

        productBlueprintCategory,

        productIdTagType:
          "qr",

        sizes:
          isApparelCategory
            ? sizes
            : [],

        modelNumbers:
          isApparelCategory
            ? modelNumbers
            : [],

        colorRgbMap:
          isApparelCategory
            ? colorRgbMap
            : {},

        colors:
          isApparelCategory
            ? colors
            : [],

        volumes:
          isAlcoholCategory
            ? volumes
            : [],

        alcoholModelNumbers:
          isAlcoholCategory
            ? alcoholModelNumbers
            : [],

        brandId,
        assigneeId,
        companyId,
        categoryFields,
      })
        .then(() => {
          alert(
            "保存しました。",
          );
        })
        .catch(
          (
            error: unknown,
          ) => {
            alert(
              error instanceof Error
                ? error.message
                : "保存に失敗しました。",
            );
          },
        );
    }, [
      blueprintId,
      productName,
      productBlueprintCategoryId,
      productBlueprintCategory,
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
    ]);

  const onDelete =
    React.useCallback(() => {
      alert(
        "削除機能は現在無効です。",
      );
    }, []);

  const onBack =
    React.useCallback(() => {
      navigate(
        "/productBlueprint",
      );
    }, [
      navigate,
    ]);

  const onClickAssignee =
    React.useCallback(
      () => {},
      [],
    );

  const onChangeBrandId =
    React.useCallback(
      (
        id: string,
      ) => {
        setBrandId(
          id,
        );
      },
      [],
    );

  const onChangeProductBlueprintCategory =
    React.useCallback(
      (
        category:
          | ProductBlueprintCategorySnapshot
          | null,
      ) => {
        setProductBlueprintCategory(
          category,
        );

        setCategoryFields(
          {},
        );

        resetVariations();
      },
      [
        resetVariations,
      ],
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

    categoryFields,
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

    onChangeProductName:
      setProductName,

    onChangeProductBlueprintCategory,

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