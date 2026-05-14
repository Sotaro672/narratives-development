// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";

import type { Brand } from "../../../../brand/src/domain/entity/brand";
import type { ModelNumber } from "../../../../model/src/application/modelCreateService";

import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

import {
  APPAREL_CATEGORY_MEASUREMENT_OPTIONS,
  FIT_OPTIONS,
  WASH_TAG_OPTIONS,
  type ApparelMeasurementOption,
  type ApparelSizeRow as SizeRow,
  type Fit,
} from "../../domain/entity/apparel";

import type {
  CategoryFieldValue,
  CategoryFieldValues,
  ProductBlueprintCategorySnapshot,
} from "../../domain/entity/productBlueprintCategory";

import { createProductBlueprint } from "../../application/productBlueprintCreateService";

import { useProductBlueprintCreateBrand } from "./useProductBlueprintCreateBrand";
import { useProductBlueprintCreateCategory } from "./useProductBlueprintCreateCategory";
import { useProductBlueprintCreateCategoryFields } from "./useProductBlueprintCreateCategoryFields";
import { useProductBlueprintCreateVariations } from "./useProductBlueprintCreateVariations";
import { useProductBlueprintCreateValidation } from "./useProductBlueprintCreateValidation";

export {
  APPAREL_CATEGORY_MEASUREMENT_OPTIONS,
  FIT_OPTIONS,
  WASH_TAG_OPTIONS,
} from "../../domain/entity/apparel";

export interface UseProductBlueprintCreateResult {
  title: string;

  brandId: string;
  brandName: string;
  brandOptions: Brand[];
  brandLoading: boolean;
  brandError: Error | null;
  onChangeBrandId: (id: string) => void;

  productName: string;

  productBlueprintCategoryId: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot | null;
  productBlueprintCategoryLabel: string;
  productBlueprintCategoryOptions: ProductBlueprintCategorySnapshot[];
  productBlueprintCategoryLoading: boolean;
  productBlueprintCategoryError: Error | null;
  isApparelCategory: boolean;

  fit: Fit;
  material: string;
  weight: number;
  qualityAssurance: string[];
  categoryFields: CategoryFieldValues;

  measurementOptions: ApparelMeasurementOption[];

  colors: string[];
  colorInput: string;
  colorRgbMap: Record<string, string>;
  sizes: SizeRow[];
  modelNumbers: ModelNumber[];

  assigneeId: string;
  assigneeName: string;
  createdBy: string;
  createdAt: string;

  onCreate: () => Promise<void>;
  onBack: () => void;

  onChangeProductName: (value: string) => void;
  onChangeProductBlueprintCategory: (
    category: ProductBlueprintCategorySnapshot | null,
  ) => void;

  onChangeFit: (value: Fit) => void;
  onChangeMaterial: (value: string) => void;
  onChangeWeight: (value: number) => void;
  onChangeQualityAssurance: (value: string[]) => void;
  onChangeCategoryField: (key: string, value: CategoryFieldValue) => void;

  onChangeColorInput: (value: string) => void;
  onAddColor: () => void;
  onRemoveColor: (name: string) => void;
  onChangeColorRgb: (name: string, rgbHex: string) => void;

  onAddSize: () => void;
  onRemoveSize: (id: string) => void;
  onChangeSize: (id: string, patch: Partial<Omit<SizeRow, "id">>) => void;

  onChangeModelNumber: (
    sizeLabel: string,
    color: string,
    nextCode: string,
  ) => void;

  onSelectAssignee: (id: string) => void;
  onEditAssignee: () => void;
  onClickAssignee: () => void;
}

export function useProductBlueprintCreate(): UseProductBlueprintCreateResult {
  const navigate = useNavigate();
  const { currentMember, user } = useAuth();

  const effectiveCompanyId = React.useMemo(
    () => (currentMember?.companyId ?? user?.companyId ?? "").trim(),
    [currentMember?.companyId, user?.companyId],
  );

  const [productName, setProductName] = React.useState("");

  const brand = useProductBlueprintCreateBrand(effectiveCompanyId);
  const category = useProductBlueprintCreateCategory();

  const categoryFields = useProductBlueprintCreateCategoryFields(
    category.productBlueprintCategory,
  );

  const variations = useProductBlueprintCreateVariations(
    category.productBlueprintCategory,
  );

  const [assigneeId, setAssigneeId] = React.useState("");
  const [assigneeName, setAssigneeName] = React.useState("");
  const [createdBy] = React.useState("");
  const [createdAt] = React.useState("");

  React.useEffect(() => {
    if (!currentMember) {
      return;
    }

    if (assigneeId) {
      return;
    }

    const memberId = currentMember.id;
    const label =
      currentMember.fullName || currentMember.email || currentMember.id;

    setAssigneeId(memberId);
    setAssigneeName(label);
  }, [currentMember, assigneeId]);

  const validate = useProductBlueprintCreateValidation({
    companyId: effectiveCompanyId,
    productName,
    brandId: brand.brandId,
    productBlueprintCategoryId: category.productBlueprintCategoryId,
    productBlueprintCategory: category.productBlueprintCategory,
    weight: categoryFields.weight,
    isApparelCategory: variations.isApparelCategory,
    colors: variations.colors,
    sizes: variations.sizes,
    modelNumbers: variations.modelNumbers,
  });

  const onChangeProductBlueprintCategory = React.useCallback(
    (nextCategory: ProductBlueprintCategorySnapshot | null) => {
      category.onChangeProductBlueprintCategory(nextCategory);
      categoryFields.resetCategoryFields();

      if (!nextCategory || nextCategory.kind !== "apparel") {
        variations.resetVariations();
      }
    },
    [category, categoryFields, variations],
  );

  const onCreate = React.useCallback(async () => {
    const errors = validate();

    if (errors.length > 0) {
      alert(`入力内容に不備があります。\n\n- ${errors.join("\n- ")}`);
      return;
    }

    if (!effectiveCompanyId) {
      alert("companyId が取得できません。ログインし直してください。");
      return;
    }

    if (!category.productBlueprintCategory) {
      alert("商品カテゴリを選択してください。");
      return;
    }

    const apiParams = {
      productName,
      brandId: brand.brandId,
      productBlueprintCategoryId: category.productBlueprintCategory.id,
      productBlueprintCategory: category.productBlueprintCategory,

      fit: categoryFields.fit,
      material: categoryFields.material,
      weight: categoryFields.weight,
      qualityAssurance: categoryFields.qualityAssurance,

      productIdTag: { type: "qr" as const },
      companyId: effectiveCompanyId,

      colors: variations.isApparelCategory ? variations.colors : [],
      colorRgbMap: variations.isApparelCategory ? variations.colorRgbMap : {},
      sizes: variations.isApparelCategory ? variations.sizes : [],
      modelNumbers: variations.isApparelCategory ? variations.modelNumbers : [],

      assigneeId,
      createdBy: currentMember?.id ?? "",
      categoryFields: categoryFields.categoryFields,
    };

    try {
      const created = await createProductBlueprint(apiParams);
      const createdId = String((created as any)?.id ?? "");

      alert("商品設計の作成が完了しました。");

      if (createdId) {
        navigate(`/productBlueprint/detail/${createdId}`);
        return;
      }

      navigate("/productBlueprint");
    } catch (error: unknown) {
      alert(
        error instanceof Error
          ? error.message
          : "商品設計の作成に失敗しました。時間をおいて再度お試しください。",
      );

      throw error;
    }
  }, [
    validate,
    effectiveCompanyId,
    category.productBlueprintCategory,
    productName,
    brand.brandId,
    categoryFields.fit,
    categoryFields.material,
    categoryFields.weight,
    categoryFields.qualityAssurance,
    categoryFields.categoryFields,
    variations.isApparelCategory,
    variations.colors,
    variations.colorRgbMap,
    variations.sizes,
    variations.modelNumbers,
    assigneeId,
    currentMember?.id,
    navigate,
  ]);

  const onBack = React.useCallback(() => {
    navigate("/productBlueprint");
  }, [navigate]);

  const onSelectAssignee = React.useCallback(
    (id: string) => {
      const nextId = String(id ?? "").trim();

      if (!nextId) {
        return;
      }

      let nextName = "";

      if (currentMember?.id === nextId) {
        nextName =
          currentMember.fullName || currentMember.email || currentMember.id;
      } else {
        nextName = nextId;
      }

      setAssigneeId(nextId);
      setAssigneeName(nextName);
    },
    [currentMember],
  );

  const onEditAssignee = React.useCallback(() => {
    // 担当者選択UIの編集イベント用
  }, []);

  const onClickAssignee = React.useCallback(() => {
    // 担当者選択UIのクリックイベント用
  }, []);

  return {
    title: "商品設計を作成",

    brandId: brand.brandId,
    brandName: brand.brandName,
    brandOptions: brand.brandOptions,
    brandLoading: brand.brandLoading,
    brandError: brand.brandError,
    onChangeBrandId: brand.onChangeBrandId,

    productName,

    productBlueprintCategoryId: category.productBlueprintCategoryId,
    productBlueprintCategory: category.productBlueprintCategory,
    productBlueprintCategoryLabel: category.productBlueprintCategoryLabel,
    productBlueprintCategoryOptions: category.productBlueprintCategoryOptions,
    productBlueprintCategoryLoading: category.productBlueprintCategoryLoading,
    productBlueprintCategoryError: category.productBlueprintCategoryError,
    isApparelCategory: variations.isApparelCategory,

    fit: categoryFields.fit,
    material: categoryFields.material,
    weight: categoryFields.weight,
    qualityAssurance: categoryFields.qualityAssurance,
    categoryFields: categoryFields.categoryFields,

    measurementOptions: variations.measurementOptions,

    colors: variations.colors,
    colorInput: variations.colorInput,
    colorRgbMap: variations.colorRgbMap,
    sizes: variations.sizes,
    modelNumbers: variations.modelNumbers,

    assigneeId,
    assigneeName,
    createdBy,
    createdAt,

    onCreate,
    onBack,

    onChangeProductName: setProductName,
    onChangeProductBlueprintCategory,

    onChangeFit: categoryFields.onChangeFit,
    onChangeMaterial: categoryFields.onChangeMaterial,
    onChangeWeight: categoryFields.onChangeWeight,
    onChangeQualityAssurance: categoryFields.onChangeQualityAssurance,
    onChangeCategoryField: categoryFields.onChangeCategoryField,

    onChangeColorInput: variations.onChangeColorInput,
    onAddColor: variations.onAddColor,
    onRemoveColor: variations.onRemoveColor,
    onChangeColorRgb: variations.onChangeColorRgb,

    onAddSize: variations.onAddSize,
    onRemoveSize: variations.onRemoveSize,
    onChangeSize: variations.onChangeSize,
    onChangeModelNumber: variations.onChangeModelNumber,

    onSelectAssignee,
    onEditAssignee,
    onClickAssignee,
  };
}