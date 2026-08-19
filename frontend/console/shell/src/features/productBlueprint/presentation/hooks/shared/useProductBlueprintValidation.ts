// frontend/console/shell/src/features/productBlueprint/presentation/hooks/shared/useProductBlueprintValidation.ts

import * as React from "react";
import type {
  AlcoholModelNumber,
  ApparelModelNumber,
  ShippingPackage,
  VolumeRow,
} from "../../../../model/application/modelCreateService";
import type { ApparelSizeInput } from "../../../../../shared/types/apparel";
import {
  isValidWashTags,
  type CategoryFieldValues,
  type ProductBlueprintCategoryPath,
} from "../../../domain/productBlueprintCategory";

type ApparelSizeRow = ApparelSizeInput & {
  id: string;
};

export type UseProductBlueprintValidationParams = {
  companyId: string;
  productName: string;
  brandId: string;
  productBlueprintCategoryPath: ProductBlueprintCategoryPath | null;
  categoryFields: CategoryFieldValues;
  isApparelCategory: boolean;
  isAlcoholCategory: boolean;
  colors: string[];
  sizes: ApparelSizeRow[];
  modelNumbers: ApparelModelNumber[];
  volumes: VolumeRow[];
  alcoholModelNumbers: AlcoholModelNumber[];
};

function normalizeString(value: unknown): string {
  return typeof value === "string" ? value.trim() : String(value ?? "").trim();
}

function isValidProductBlueprintCategoryPath(
  productBlueprintCategoryPath: ProductBlueprintCategoryPath | null,
): boolean {
  return (
    Array.isArray(productBlueprintCategoryPath) &&
    productBlueprintCategoryPath.length > 0 &&
    productBlueprintCategoryPath.every((segment) => segment !== "")
  );
}

function isPositiveInteger(value: unknown): boolean {
  return (
    typeof value === "number" &&
    Number.isFinite(value) &&
    Number.isInteger(value) &&
    value > 0
  );
}

function isValidShippingPackage(
  shippingPackage: ShippingPackage | null | undefined,
): boolean {
  if (!shippingPackage) {
    return false;
  }

  return (
    isPositiveInteger(shippingPackage.weightGrams) &&
    isPositiveInteger(shippingPackage.widthMm) &&
    isPositiveInteger(shippingPackage.lengthMm) &&
    isPositiveInteger(shippingPackage.heightMm)
  );
}

function toVolumeLabel(
  volume: Pick<VolumeRow, "volumeValue" | "volumeUnit">,
): string {
  const value =
    typeof volume.volumeValue === "number" && Number.isFinite(volume.volumeValue)
      ? volume.volumeValue
      : 0;

  const unit = normalizeString(volume.volumeUnit);

  if (value <= 0 || !unit) {
    return "";
  }

  return `${value}${unit}`;
}

function toAlcoholModelNumberVolumeLabel(
  modelNumber: AlcoholModelNumber,
): string {
  return toVolumeLabel({
    volumeValue: modelNumber.volume.value,
    volumeUnit: modelNumber.volume.unit,
  });
}

function hasEmptyModelNumberValue(
  modelNumber: ApparelModelNumber,
): boolean {
  return (
    !normalizeString(modelNumber.size) ||
    !normalizeString(modelNumber.color) ||
    !normalizeString(modelNumber.code)
  );
}

function hasEmptyAlcoholModelNumberValue(
  modelNumber: AlcoholModelNumber,
): boolean {
  return (
    !normalizeString(modelNumber.code) ||
    !toAlcoholModelNumberVolumeLabel(modelNumber)
  );
}

function getInvalidApparelShippingPackageModelNumbers(
  modelNumbers: ApparelModelNumber[],
): string[] {
  return modelNumbers
    .filter((modelNumber) => !isValidShippingPackage(modelNumber.shippingPackage))
    .map((modelNumber) => {
      const code = normalizeString(modelNumber.code);
      const size = normalizeString(modelNumber.size);
      const color = normalizeString(modelNumber.color);

      if (code) {
        return code;
      }

      return [size, color].filter(Boolean).join(" / ");
    })
    .filter(Boolean);
}

function getInvalidAlcoholShippingPackageModelNumbers(
  modelNumbers: AlcoholModelNumber[],
): string[] {
  return modelNumbers
    .filter((modelNumber) => !isValidShippingPackage(modelNumber.shippingPackage))
    .map((modelNumber) => {
      const code = normalizeString(modelNumber.code);

      if (code) {
        return code;
      }

      return toAlcoholModelNumberVolumeLabel(modelNumber);
    })
    .filter(Boolean);
}

export function useProductBlueprintValidation(
  params: UseProductBlueprintValidationParams,
): () => string[] {
  return React.useCallback((): string[] => {
    const errors: string[] = [];

    if (!normalizeString(params.companyId)) {
      errors.push("companyId が取得できません。ログインし直してください。");
    }

    if (!normalizeString(params.productName)) {
      errors.push("商品名は必須です。");
    }

    if (!normalizeString(params.brandId)) {
      errors.push("ブランドを選択してください。");
    }

    if (
      !isValidProductBlueprintCategoryPath(
        params.productBlueprintCategoryPath,
      )
    ) {
      errors.push("商品カテゴリを選択してください。");
    }

    const weight = params.categoryFields.weight;

    if (
      weight !== undefined &&
      weight !== null &&
      (typeof weight !== "number" || !Number.isFinite(weight) || weight < 0)
    ) {
      errors.push("重さは0以上の有限値を入力してください。");
    }

    if (params.isApparelCategory) {
      if (!isValidWashTags(params.categoryFields.washTags)) {
        errors.push("洗濯表示を1つ以上選択してください。");
      }

      if (params.colors.length === 0) {
        errors.push("カラーバリエーションを1つ以上登録してください。");
      }

      if (params.sizes.length === 0) {
        errors.push("サイズバリエーションを1つ以上登録してください。");
      }

      if (params.modelNumbers.length === 0) {
        errors.push("モデルナンバーを1つ以上登録してください。");
      } else {
        const hasEmpty = params.modelNumbers.some(hasEmptyModelNumberValue);

        if (hasEmpty) {
          errors.push(
            "モデルナンバー欄に空欄があります。すべて入力してください。",
          );
        }

        const invalidShippingPackageModelNumbers =
          getInvalidApparelShippingPackageModelNumbers(params.modelNumbers);

        if (invalidShippingPackageModelNumbers.length > 0) {
          errors.push(
            `配送用梱包情報の重量・横・縦・高さはすべて1以上の整数で入力してください。（対象モデル: ${invalidShippingPackageModelNumbers.join("、")}）`,
          );
        }
      }

      if (params.modelNumbers.length > 0) {
        const seenCodes = new Set<string>();
        const duplicateCodes = new Set<string>();

        params.modelNumbers.forEach((modelNumber) => {
          const code = normalizeString(modelNumber.code);

          if (!code) {
            return;
          }

          if (seenCodes.has(code)) {
            duplicateCodes.add(code);
          } else {
            seenCodes.add(code);
          }
        });

        if (duplicateCodes.size > 0) {
          errors.push(
            `モデルナンバーが重複しています。（重複コード: ${Array.from(
              duplicateCodes,
            ).join("、")}）`,
          );
        }
      }

      if (params.sizes.length > 0) {
        const seenSizes = new Set<string>();
        const duplicateSizes = new Set<string>();

        params.sizes.forEach((size) => {
          const label = normalizeString(size.sizeLabel);

          if (!label) {
            return;
          }

          if (seenSizes.has(label)) {
            duplicateSizes.add(label);
          } else {
            seenSizes.add(label);
          }
        });

        if (duplicateSizes.size > 0) {
          errors.push(
            `サイズ名が重複しています。（重複サイズ: ${Array.from(
              duplicateSizes,
            ).join("、")}）`,
          );
        }
      }
    }

    if (params.isAlcoholCategory) {
      if (params.volumes.length === 0) {
        errors.push("容量バリエーションを1つ以上登録してください。");
      }

      if (params.alcoholModelNumbers.length === 0) {
        errors.push("容量ごとのモデルナンバーを1つ以上登録してください。");
      } else {
        const hasEmpty =
          params.alcoholModelNumbers.some(hasEmptyAlcoholModelNumberValue);

        if (hasEmpty) {
          errors.push(
            "容量ごとのモデルナンバー欄に空欄があります。すべて入力してください。",
          );
        }

        const invalidShippingPackageModelNumbers =
          getInvalidAlcoholShippingPackageModelNumbers(
            params.alcoholModelNumbers,
          );

        if (invalidShippingPackageModelNumbers.length > 0) {
          errors.push(
            `配送用梱包情報の重量・横・縦・高さはすべて1以上の整数で入力してください。（対象モデル: ${invalidShippingPackageModelNumbers.join("、")}）`,
          );
        }
      }

      if (params.volumes.length > 0) {
        const seenVolumes = new Set<string>();
        const duplicateVolumes = new Set<string>();

        params.volumes.forEach((volume) => {
          const label = toVolumeLabel(volume);

          if (!label) {
            errors.push("容量は0より大きい値と単位を入力してください。");
            return;
          }

          if (seenVolumes.has(label)) {
            duplicateVolumes.add(label);
          } else {
            seenVolumes.add(label);
          }
        });

        if (duplicateVolumes.size > 0) {
          errors.push(
            `容量が重複しています。（重複容量: ${Array.from(
              duplicateVolumes,
            ).join("、")}）`,
          );
        }
      }

      if (params.alcoholModelNumbers.length > 0) {
        const seenCodes = new Set<string>();
        const duplicateCodes = new Set<string>();

        params.alcoholModelNumbers.forEach((modelNumber) => {
          const code = normalizeString(modelNumber.code);

          if (!code) {
            return;
          }

          if (seenCodes.has(code)) {
            duplicateCodes.add(code);
          } else {
            seenCodes.add(code);
          }
        });

        if (duplicateCodes.size > 0) {
          errors.push(
            `モデルナンバーが重複しています。（重複コード: ${Array.from(
              duplicateCodes,
            ).join("、")}）`,
          );
        }
      }

      if (
        params.volumes.length > 0 &&
        params.alcoholModelNumbers.length > 0
      ) {
        const validVolumeLabels = new Set(
          params.volumes.map(toVolumeLabel).filter(Boolean),
        );

        const missingModelNumberVolumes = params.volumes
          .map(toVolumeLabel)
          .filter(Boolean)
          .filter(
            (label) =>
              !params.alcoholModelNumbers.some(
                (modelNumber) =>
                  toAlcoholModelNumberVolumeLabel(modelNumber) === label,
              ),
          );

        if (missingModelNumberVolumes.length > 0) {
          errors.push(
            `モデルナンバー未入力の容量があります。（対象: ${missingModelNumberVolumes.join(
              "、",
            )}）`,
          );
        }

        const invalidModelNumberVolumes = params.alcoholModelNumbers
          .map(toAlcoholModelNumberVolumeLabel)
          .filter(Boolean)
          .filter((label) => !validVolumeLabels.has(label));

        if (invalidModelNumberVolumes.length > 0) {
          errors.push(
            `存在しない容量に紐づくモデルナンバーがあります。（対象: ${invalidModelNumberVolumes.join(
              "、",
            )}）`,
          );
        }
      }
    }

    return errors;
  }, [params]);
}