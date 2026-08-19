// frontend/console/shell/src/features/productBlueprint/presentation/hooks/detail/useProductBlueprintDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import { safeDateTimeLabelJa } from "../../../../../shared/util/dateJa";
import { useAssigneeSelection } from "../../../../admin/presentation/hook/useAssigneeSelection";
import {
  deleteProductBlueprint,
  getProductBlueprintDetail,
  updateProductBlueprint,
} from "../../../application/productBlueprintDetailService";
import {
  APPAREL_CATEGORY_OPTIONS,
  type ApparelModelNumberRow as ModelNumberRow,
  type ApparelSizeInput,
} from "../../../../../shared/types/apparel";
import type {
  AlcoholModelNumber,
  VolumeRow,
} from "../../../../model/application/modelCreateService";
import {
  ALCOHOL_CATEGORY_OPTIONS,
} from "../../../domain/alcohol";
import {
  COSMETICS_CATEGORY_OPTIONS,
} from "../../../domain/cosmetics";
import {
  HEALTHCARE_CATEGORY_OPTIONS,
} from "../../../domain/healthcare";
import {
  OTHER_CATEGORY_OPTIONS,
} from "../../../domain/other";
import {
  toProductBlueprintCategoryPathKey,
  type CategoryFieldValue,
  type CategoryFieldValues,
  type ProductBlueprintCategoryPath,
} from "../../../domain/productBlueprintCategory";
import { useProductBlueprintValidation } from "../shared/useProductBlueprintValidation";
import { useProductBlueprintVariations } from "../shared/useProductBlueprintVariations";

type SizeRow = ApparelSizeInput & { id: string };

const CATEGORY_LABEL_BY_PATH_KEY: Readonly<
  Record<string, string>
> = Object.fromEntries(
  [
    ...APPAREL_CATEGORY_OPTIONS,
    ...ALCOHOL_CATEGORY_OPTIONS,
    ...COSMETICS_CATEGORY_OPTIONS,
    ...HEALTHCARE_CATEGORY_OPTIONS,
    ...OTHER_CATEGORY_OPTIONS,
  ].map(
    (option) => [
      option.value,
      option.label,
    ],
  ),
);

function getProductBlueprintCategoryLabel(
  productBlueprintCategoryPath:
    ProductBlueprintCategoryPath | null,
): string {
  if (
    !productBlueprintCategoryPath ||
    productBlueprintCategoryPath.length === 0
  ) {
    return "";
  }

  const pathKey =
    toProductBlueprintCategoryPathKey(
      productBlueprintCategoryPath,
    );

  return (
    CATEGORY_LABEL_BY_PATH_KEY[pathKey] ??
    productBlueprintCategoryPath[
      productBlueprintCategoryPath.length - 1
    ] ??
    ""
  );
}

export interface UseProductBlueprintDetailResult {
  pageTitle: string;
  productName: string;
  brand: string;
  productBlueprintCategoryPath: ProductBlueprintCategoryPath | null;
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
  assigneeId: string;
  assignee: string;
  assigneeCandidates: {
    id: string;
    name: string;
  }[];
  loadingMembers: boolean;
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
  onSelectAssignee: (id: string) => void;
  onClickAssignee: () => void;
}

export function useProductBlueprintDetail(): UseProductBlueprintDetailResult {
  const navigate = useNavigate();
  const { blueprintId } = useParams<{ blueprintId: string }>();

  const [productName, setProductName] = React.useState("");
  const [brand, setBrand] = React.useState("");
  const [
    productBlueprintCategoryPath,
    setProductBlueprintCategoryPath,
  ] =
    React.useState<ProductBlueprintCategoryPath | null>(null);
  const [categoryFields, setCategoryFields] = React.useState<CategoryFieldValues>({});
  const [creator, setCreator] = React.useState("作成者未設定");
  const [createdAt, setCreatedAt] = React.useState("");
  const [updater, setUpdater] = React.useState("");
  const [updatedAt, setUpdatedAt] = React.useState("");
  const [printed, setPrinted] = React.useState(false);
  const [brandId, setBrandId] = React.useState("");
  const [companyId, setCompanyId] = React.useState("");
  const [initialAssigneeId, setInitialAssigneeId] = React.useState("");
  const [initialAssigneeName, setInitialAssigneeName] = React.useState("");

  const {
    assigneeId,
    assigneeName: assignee,
    assigneeCandidates,
    loadingMembers,
    handleSelectAssignee,
  } = useAssigneeSelection({
    initialAssigneeId,
    initialAssigneeName,
    defaultToCurrentMember: false,
  });

  const productBlueprintCategoryLabel =
    React.useMemo(
      () =>
        getProductBlueprintCategoryLabel(
          productBlueprintCategoryPath,
        ),
      [productBlueprintCategoryPath],
    );

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
  } = useProductBlueprintVariations({
    productBlueprintCategoryPath,
  });

  const validate = useProductBlueprintValidation({
    companyId,
    productName,
    brandId,
    productBlueprintCategoryPath,
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

    setInitialAssigneeId("");
    setInitialAssigneeName("");

    void (async () => {
      try {
        const detail = await getProductBlueprintDetail(blueprintId);

        setProductName(detail.productName);
        setBrandId(detail.brandId);
        setBrand(detail.brandName);
        setPrinted(detail.printed);
        setCompanyId(detail.companyId);
        setProductBlueprintCategoryPath(
          [
            ...detail.productBlueprintCategoryPath,
          ],
        );
        setCategoryFields(detail.categoryFields ?? {});
        setFromUiState(detail.modelState);
        setInitialAssigneeId(detail.assigneeId);
        setInitialAssigneeName(detail.assigneeName);
        setCreator(detail.createdByName);
        setCreatedAt(safeDateTimeLabelJa(detail.createdAt, ""));

        const nextUpdater = detail.updatedByName;
        const nextUpdatedAt = safeDateTimeLabelJa(detail.updatedAt, "");

        if (nextUpdater && nextUpdatedAt) {
          setUpdater(nextUpdater);
          setUpdatedAt(nextUpdatedAt);
        } else {
          setUpdater("");
          setUpdatedAt("");
        }
      } catch {
        setProductBlueprintCategoryPath(null);
        resetVariations();
      }
    })();
  }, [blueprintId, resetVariations, setFromUiState]);

  const onChangeCategoryField = React.useCallback(
    (key: string, value: CategoryFieldValue) => {
      setCategoryFields((previous) => ({
        ...previous,
        [key]: value,
      }));
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

    if (
      !productBlueprintCategoryPath ||
      productBlueprintCategoryPath.length === 0
    ) {
      return;
    }

    if (!assigneeId) {
      alert("担当者を選択してください。");
      return;
    }

    void updateProductBlueprint({
      id: blueprintId,
      productName,
      productBlueprintCategoryPath: [
        ...productBlueprintCategoryPath,
      ],
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
        setCompanyId(detail.companyId);
        setProductBlueprintCategoryPath(
          [
            ...detail.productBlueprintCategoryPath,
          ],
        );
        setCategoryFields(detail.categoryFields ?? {});
        setFromUiState(detail.modelState);
        setInitialAssigneeId(detail.assigneeId);
        setInitialAssigneeName(detail.assigneeName);
        setCreator(detail.createdByName);
        setCreatedAt(safeDateTimeLabelJa(detail.createdAt, ""));

        const nextUpdater = detail.updatedByName;
        const nextUpdatedAt = safeDateTimeLabelJa(detail.updatedAt, "");

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
    productBlueprintCategoryPath,
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

  const onSelectAssignee = React.useCallback(
    (id: string) => {
      handleSelectAssignee(id);
    },
    [handleSelectAssignee],
  );

  const onClickAssignee = React.useCallback(() => {}, []);

  return {
    pageTitle,
    productName,
    brand,
    productBlueprintCategoryPath,
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
    assigneeId,
    assignee,
    assigneeCandidates,
    loadingMembers,
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
    onSelectAssignee,
    onClickAssignee,
  };
}