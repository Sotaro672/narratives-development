// frontend/console/shell/src/features/productBlueprint/presentation/hooks/detail/useProductBlueprintDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import { safeDateTimeLabelJa } from "../../../../../shared/util/dateJa";
import {
  deleteProductBlueprint,
  getProductBlueprintDetail,
  updateProductBlueprint,
} from "../../../application/productBlueprintDetailService";
import type {
  ApparelModelNumberRow as ModelNumberRow,
  ApparelSizeInput,
} from "../../../../../shared/types/apparel";
import type {
  AlcoholModelNumber,
  VolumeRow,
} from "../../../../model/application/modelCreateService";
import type {
  CategoryFieldValue,
  CategoryFieldValues,
  ProductBlueprintCategorySnapshot,
} from "../../../domain/productBlueprintCategory";
import { useProductBlueprintValidation } from "../shared/useProductBlueprintValidation";
import { useProductBlueprintVariations } from "../shared/useProductBlueprintVariations";

type SizeRow = ApparelSizeInput & { id: string };

function formatDateTimeYYYYMMDDHHmm(value: string | null | undefined): string {
  const label = safeDateTimeLabelJa(value, "");
  if (!label) return "";

  const matched = label.match(/^(\d{4}\/\d{2}\/\d{2} \d{2}:\d{2})(?::\d{2})?$/);
  return matched?.[1] ?? label;
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
  categoryFields: CategoryFieldValues;
  onChangeCategoryField: (key: string, value: CategoryFieldValue) => void;
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
  onChangeProductName: (value: string) => void;
  onChangeColorInput: (value: string) => void;
  onAddColor: () => void;
  onRemoveColor: (name: string) => void;
  onChangeColorRgb: (name: string, hex: string) => void;
  onRemoveSize: (id: string) => void;
  onAddSize: () => void;
  onChangeSize: (id: string, patch: Partial<Omit<SizeRow, "id">>) => void;
  onChangeModelNumber: (sizeLabel: string, color: string, nextCode: string) => void;
  onAddVolume: () => void;
  onRemoveVolume: (id: string) => void;
  onChangeVolume: (id: string, patch: Partial<Omit<VolumeRow, "id">>) => void;
  onChangeAlcoholModelNumber: (volumeLabel: string, nextCode: string) => void;
  onClickAssignee: () => void;
}

export function useProductBlueprintDetail(): UseProductBlueprintDetailResult {
  const navigate = useNavigate();
  const { blueprintId } = useParams<{ blueprintId: string }>();

  const [productName, setProductName] = React.useState("");
  const [brand, setBrand] = React.useState("");
  const [productBlueprintCategory, setProductBlueprintCategory] =
    React.useState<ProductBlueprintCategorySnapshot | null>(null);
  const [categoryFields, setCategoryFields] = React.useState<CategoryFieldValues>({});
  const [assignee, setAssignee] = React.useState("担当者未設定");
  const [creator, setCreator] = React.useState("作成者未設定");
  const [createdAt, setCreatedAt] = React.useState("");
  const [updater, setUpdater] = React.useState("");
  const [updatedAt, setUpdatedAt] = React.useState("");
  const [printed, setPrinted] = React.useState(false);
  const [brandId, setBrandId] = React.useState("");
  const [assigneeId, setAssigneeId] = React.useState("");
  const [companyId, setCompanyId] = React.useState("");

  const productBlueprintCategoryId = productBlueprintCategory?.id ?? "";
  const productBlueprintCategoryLabel =
    productBlueprintCategory?.nameJa ||
    productBlueprintCategory?.nameEn ||
    productBlueprintCategory?.code ||
    productBlueprintCategory?.id ||
    "";
  const pageTitle = productName || blueprintId || "";

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
  } = useProductBlueprintVariations({ productBlueprintCategory });

  const validate = useProductBlueprintValidation({
    companyId,
    productName,
    brandId,
    productBlueprintCategoryId,
    productBlueprintCategory,
    categoryFields,
    isApparelCategory,
    isAlcoholCategory,
    colors,
    sizes,
    modelNumbers,
    volumes,
    alcoholModelNumbers,
  });

  React.useEffect(() => {
    if (!blueprintId) return;

    void (async () => {
      try {
        const detail = await getProductBlueprintDetail(blueprintId);

        setProductName(detail.productName);
        setBrandId(detail.brandId);
        setBrand(detail.brandName);
        setPrinted(detail.printed);
        setAssigneeId(detail.assigneeId);
        setCompanyId(detail.companyId);
        setProductBlueprintCategory(detail.productBlueprintCategory);
        setCategoryFields(detail.categoryFields ?? {});
        setFromUiState(detail.modelState);

        setAssignee(detail.assigneeName);
        setCreator(detail.createdByName);
        setCreatedAt(formatDateTimeYYYYMMDDHHmm(detail.createdAt));

        const nextUpdater = detail.updatedByName;
        const nextUpdatedAt = formatDateTimeYYYYMMDDHHmm(detail.updatedAt);
        if (nextUpdater && nextUpdatedAt) {
          setUpdater(nextUpdater);
          setUpdatedAt(nextUpdatedAt);
        } else {
          setUpdater("");
          setUpdatedAt("");
        }
      } catch {
        resetVariations();
      }
    })();
  }, [blueprintId, resetVariations, setFromUiState]);

  const onChangeCategoryField = React.useCallback(
    (key: string, value: CategoryFieldValue) => {
      setCategoryFields((previous) => ({ ...previous, [key]: value }));
    },
    [],
  );

  const onSave = React.useCallback(() => {
    if (!blueprintId) {
      alert("商品設計IDが不明です。");
      return;
    }

    const errors = validate();
    if (errors.length > 0) {
      alert(`入力内容に不備があります。\n\n- ${errors.join("\n- ")}`);
      return;
    }

    if (!productBlueprintCategory) return;

    void updateProductBlueprint({
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
      categoryFields,
    })
      .then((detail) => {
        setProductName(detail.productName);
        setBrandId(detail.brandId);
        setBrand(detail.brandName);
        setPrinted(detail.printed);
        setAssigneeId(detail.assigneeId);
        setCompanyId(detail.companyId);
        setProductBlueprintCategory(detail.productBlueprintCategory);
        setCategoryFields(detail.categoryFields ?? {});
        setFromUiState(detail.modelState);

        setAssignee(detail.assigneeName);
        setCreator(detail.createdByName);
        setCreatedAt(formatDateTimeYYYYMMDDHHmm(detail.createdAt));

        const nextUpdater = detail.updatedByName;
        const nextUpdatedAt = formatDateTimeYYYYMMDDHHmm(detail.updatedAt);
        if (nextUpdater && nextUpdatedAt) {
          setUpdater(nextUpdater);
          setUpdatedAt(nextUpdatedAt);
        } else {
          setUpdater("");
          setUpdatedAt("");
        }

        alert("保存しました。");
      })
      .catch((error: unknown) => {
        alert(error instanceof Error ? error.message : "保存に失敗しました。");
      });
  }, [
    blueprintId,
    validate,
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
    setFromUiState,
  ]);

  const onDelete = React.useCallback(() => {
    if (!blueprintId) {
      alert("商品設計IDが不明です。");
      return;
    }

    if (printed) {
      alert("印刷済みの商品設計は削除できません。");
      return;
    }

    const confirmed = window.confirm(
      "この商品設計を完全に削除します。\n" +
        "関連するモデルも削除されます。\n" +
        "この操作は取り消せません。\n\n" +
        "削除しますか？",
    );
    if (!confirmed) return;

    void deleteProductBlueprint(blueprintId)
      .then(() => {
        navigate("/productBlueprint");
      })
      .catch((error: unknown) => {
        alert(error instanceof Error ? error.message : "削除に失敗しました。");
      });
  }, [blueprintId, printed, navigate]);

  const onBack = React.useCallback(() => {
    navigate("/productBlueprint");
  }, [navigate]);

  const onClickAssignee = React.useCallback(() => {}, []);

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