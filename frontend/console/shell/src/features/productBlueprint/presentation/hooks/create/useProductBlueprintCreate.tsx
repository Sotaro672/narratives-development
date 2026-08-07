// frontend/console/shell/src/features/productBlueprint/presentation/hooks/create/useProductBlueprintCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";
import type { AlcoholModelNumber, ModelNumber, VolumeRow } from "../../../../model/application/modelCreateService";
import { useAuthContext } from "../../../../../auth/application/AuthContext";
import type { ApparelSizeInput, Fit, MeasurementOption } from "../../../../../shared/types/apparel";
import type {
  CategoryFieldValue,
  CategoryFieldValues,
  ProductBlueprintCategorySnapshot,
} from "../../../domain/productBlueprintCategory";
import { createProductBlueprint } from "../../../application/productBlueprintCreateService";
import { useProductBlueprintCreateBrand, type BrandOption } from "./useProductBlueprintCreateBrand";
import { useProductBlueprintCreateCategory } from "./useProductBlueprintCreateCategory";
import { useProductBlueprintCreateCategoryFields } from "./useProductBlueprintCreateCategoryFields";
import { useProductBlueprintValidation } from "../shared/useProductBlueprintValidation";
import { useProductBlueprintVariations } from "../shared/useProductBlueprintVariations";
type SizeRow = ApparelSizeInput & {
  id: string;
};
type FitInputValue = Fit | "";
export {
  APPAREL_CATEGORY_MEASUREMENT_OPTIONS,
  FIT_OPTIONS,
} from "../../../../../shared/types/apparel";
export interface UseProductBlueprintCreateResult {
  title: string;
  brandId: string;
  brandName: string;
  brandOptions: BrandOption[];
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
  isAlcoholCategory: boolean;
  fit: FitInputValue;
  material: string;
  weight: number;
  qualityAssurance: string[];
  categoryFields: CategoryFieldValues;
  measurementOptions: MeasurementOption[];
  colors: string[];
  colorInput: string;
  colorRgbMap: Record<string, string>;
  sizes: SizeRow[];
  modelNumbers: ModelNumber[];
  volumes: VolumeRow[];
  alcoholModelNumbers: AlcoholModelNumber[];
  assigneeId: string;
  assigneeName: string;
  createdBy: string;
  createdAt: string;
  onCreate: () => Promise<void>;
  onBack: () => void;
  onChangeProductName: (value: string) => void;
  onChangeProductBlueprintCategory: (category: ProductBlueprintCategorySnapshot | null) => void;
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
  getCode: (sizeLabel: string, color: string) => string;
  onChangeModelNumber: (sizeLabel: string, color: string, nextCode: string) => void;
  onAddVolume: () => void;
  onRemoveVolume: (id: string) => void;
  onChangeVolume: (id: string, patch: Partial<Omit<VolumeRow, "id">>) => void;
  onChangeAlcoholModelNumber: (volumeLabel: string, nextCode: string) => void;
  onSelectAssignee: (id: string) => void;
  onEditAssignee: () => void;
  onClickAssignee: () => void;
}
function removeModelOwnedCategoryFields(
  category: ProductBlueprintCategorySnapshot | null,
  fields: CategoryFieldValues,
): CategoryFieldValues {
  const next: CategoryFieldValues = {
    ...fields,
  };
  if (category?.kind === "alcohol") {
    delete next.volume;
  }
  return next;
}
function getStringCategoryField(fields: CategoryFieldValues, key: string): string {
  const value = fields[key];
  return typeof value === "string" ? value : "";
}
function getNumberCategoryField(fields: CategoryFieldValues, key: string): number {
  const value = fields[key];
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}
function getStringArrayCategoryField(fields: CategoryFieldValues, key: string): string[] {
  const value = fields[key];
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((item): item is string => typeof item === "string" && item.trim() !== "");
}
function getMemberDisplayLabel(member: {
  id: string;
  email?: string | null;
  displayName?: string | null;
}): string {
  const displayName = String(member.displayName ?? "").trim();
  if (displayName) {
    return displayName;
  }
  const email = String(member.email ?? "").trim();
  if (email) {
    return email;
  }
  return member.id;
}
export function useProductBlueprintCreate(): UseProductBlueprintCreateResult {
  const navigate = useNavigate();
  const { currentMember, user } = useAuthContext();
  const effectiveCompanyId = React.useMemo(
    () => (currentMember?.companyId ?? user?.companyId ?? "").trim(),
    [currentMember?.companyId, user?.companyId],
  );
  const [productName, setProductName] = React.useState("");
  const brand = useProductBlueprintCreateBrand(effectiveCompanyId);
  const category = useProductBlueprintCreateCategory();
  const categoryFields = useProductBlueprintCreateCategoryFields();
  const variations = useProductBlueprintVariations({
    productBlueprintCategory: category.productBlueprintCategory,
  });
  const [assigneeId, setAssigneeId] = React.useState("");
  const [assigneeName, setAssigneeName] = React.useState("");
  const [createdBy] = React.useState("");
  const [createdAt] = React.useState("");
  React.useEffect(() => {
    if (!currentMember || assigneeId) {
      return;
    }
    setAssigneeId(currentMember.id);
    setAssigneeName(getMemberDisplayLabel(currentMember));
  }, [currentMember, assigneeId]);
  const sanitizedCategoryFields = React.useMemo(
    () =>
      removeModelOwnedCategoryFields(
        category.productBlueprintCategory,
        categoryFields.categoryFields,
      ),
    [category.productBlueprintCategory, categoryFields.categoryFields],
  );
  const fit = getStringCategoryField(sanitizedCategoryFields, "fit") as FitInputValue;
  const material = getStringCategoryField(sanitizedCategoryFields, "material");
  const weight = getNumberCategoryField(sanitizedCategoryFields, "weight");
  const qualityAssurance = getStringArrayCategoryField(sanitizedCategoryFields, "washTags");
  const onChangeFit = React.useCallback(
    (value: Fit) => {
      categoryFields.onChangeCategoryField("fit", value);
    },
    [categoryFields.onChangeCategoryField],
  );
  const onChangeMaterial = React.useCallback(
    (value: string) => {
      categoryFields.onChangeCategoryField(
        "material",
        value.trim() === "" ? null : value,
      );
    },
    [categoryFields.onChangeCategoryField],
  );
  const onChangeWeight = React.useCallback(
    (value: number) => {
      categoryFields.onChangeCategoryField(
        "weight",
        Number.isFinite(value) ? Math.max(0, value) : 0,
      );
    },
    [categoryFields.onChangeCategoryField],
  );
  const onChangeQualityAssurance = React.useCallback(
    (value: string[]) => {
      categoryFields.onChangeCategoryField(
        "washTags",
        value.filter((item) => item.trim() !== ""),
      );
    },
    [categoryFields.onChangeCategoryField],
  );
  const validate = useProductBlueprintValidation({
    companyId: effectiveCompanyId,
    productName,
    brandId: brand.brandId,
    productBlueprintCategoryId: category.productBlueprintCategoryId,
    productBlueprintCategory: category.productBlueprintCategory,
    categoryFields: sanitizedCategoryFields,
    isApparelCategory: variations.isApparelCategory,
    isAlcoholCategory: variations.isAlcoholCategory,
    colors: variations.colors,
    sizes: variations.sizes,
    modelNumbers: variations.modelNumbers,
    volumes: variations.volumes,
    alcoholModelNumbers: variations.alcoholModelNumbers,
  });
  const onChangeProductBlueprintCategory = React.useCallback(
    (nextCategory: ProductBlueprintCategorySnapshot | null) => {
      category.onChangeProductBlueprintCategory(nextCategory);
      categoryFields.resetCategoryFields();
      variations.resetVariations();
    },
    [
      category.onChangeProductBlueprintCategory,
      categoryFields.resetCategoryFields,
      variations.resetVariations,
    ],
  );
  const onCreate = React.useCallback(
    async () => {
      const errors = validate();
      if (errors.length > 0) {
        alert(
          `入力内容に不備があります。\n\n- ${errors.join(
            "\n- ",
          )}`,
        );
        return;
      }
      if (!category.productBlueprintCategory) {
        return;
      }
      const apiParams = {
        productName,
        brandId: brand.brandId,
        productBlueprintCategoryId: category.productBlueprintCategory.id,
        productBlueprintCategory: category.productBlueprintCategory,
        fit,
        material,
        weight,
        qualityAssurance,
        productIdTag: {
          type: "qr" as const,
        },
        companyId: effectiveCompanyId,
        colors: variations.isApparelCategory ? variations.colors : [],
        colorRgbMap: variations.isApparelCategory ? variations.colorRgbMap : {},
        sizes: variations.isApparelCategory ? variations.sizes : [],
        modelNumbers: variations.isApparelCategory ? variations.modelNumbers : [],
        volumes: variations.isAlcoholCategory ? variations.volumes : [],
        alcoholModelNumbers: variations.isAlcoholCategory ? variations.alcoholModelNumbers : [],
        assigneeId,
        createdBy: currentMember?.id ?? "",
        categoryFields: sanitizedCategoryFields,
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
    },
    [
      validate,
      effectiveCompanyId,
      category.productBlueprintCategory,
      productName,
      brand.brandId,
      fit,
      material,
      weight,
      qualityAssurance,
      sanitizedCategoryFields,
      variations.isApparelCategory,
      variations.isAlcoholCategory,
      variations.colors,
      variations.colorRgbMap,
      variations.sizes,
      variations.modelNumbers,
      variations.volumes,
      variations.alcoholModelNumbers,
      assigneeId,
      currentMember?.id,
      navigate,
    ],
  );
  const onBack = React.useCallback(() => {
    navigate("/productBlueprint");
  }, [navigate]);
  const onSelectAssignee = React.useCallback(
    (id: string) => {
      const nextId = id.trim();
      if (!nextId) {
        return;
      }
      setAssigneeId(nextId);
      setAssigneeName(
        currentMember?.id === nextId
          ? getMemberDisplayLabel(currentMember)
          : nextId,
      );
    },
    [currentMember],
  );
  const onEditAssignee = React.useCallback(() => {}, []);
  const onClickAssignee = React.useCallback(() => {}, []);
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
    isAlcoholCategory: variations.isAlcoholCategory,
    fit,
    material,
    weight,
    qualityAssurance,
    categoryFields: sanitizedCategoryFields,
    measurementOptions: variations.measurementOptions,
    colors: variations.colors,
    colorInput: variations.colorInput,
    colorRgbMap: variations.colorRgbMap,
    sizes: variations.sizes,
    modelNumbers: variations.modelNumbers,
    volumes: variations.volumes,
    alcoholModelNumbers: variations.alcoholModelNumbers,
    assigneeId,
    assigneeName,
    createdBy,
    createdAt,
    onCreate,
    onBack,
    onChangeProductName: setProductName,
    onChangeProductBlueprintCategory,
    onChangeFit,
    onChangeMaterial,
    onChangeWeight,
    onChangeQualityAssurance,
    onChangeCategoryField: categoryFields.onChangeCategoryField,
    onChangeColorInput: variations.onChangeColorInput,
    onAddColor: variations.onAddColor,
    onRemoveColor: variations.onRemoveColor,
    onChangeColorRgb: variations.onChangeColorRgb,
    onAddSize: variations.onAddSize,
    onRemoveSize: variations.onRemoveSize,
    onChangeSize: variations.onChangeSize,
    getCode: variations.getCode,
    onChangeModelNumber: variations.onChangeModelNumber,
    onAddVolume: variations.onAddVolume,
    onRemoveVolume: variations.onRemoveVolume,
    onChangeVolume: variations.onChangeVolume,
    onChangeAlcoholModelNumber: variations.onChangeAlcoholModelNumber,
    onSelectAssignee,
    onEditAssignee,
    onClickAssignee,
  };
}