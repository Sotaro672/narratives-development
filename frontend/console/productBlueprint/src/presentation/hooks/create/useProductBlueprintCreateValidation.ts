// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintCreateValidation.ts

import * as React from "react";

import type {
  AlcoholModelNumber,
  ModelNumber,
  VolumeRow,
} from "../../../../../model/src/application/modelCreateService";
import type { ApparelSizeRow } from "../../../domain/entity/apparel";
import type { ProductBlueprintCategorySnapshot } from "../../../domain/entity/productBlueprintCategory";

export type UseProductBlueprintCreateValidationParams = {
  companyId: string;
  productName: string;
  brandId: string;
  productBlueprintCategoryId: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot | null;
  weight: number;

  isApparelCategory: boolean;
  isAlcoholCategory: boolean;

  colors: string[];
  sizes: ApparelSizeRow[];
  modelNumbers: ModelNumber[];

  /**
   * alcohol model variation 用。
   * volume は productBlueprint.categoryFields ではなく model domain 側で扱う。
   */
  volumes: VolumeRow[];
  alcoholModelNumbers: AlcoholModelNumber[];
};

function normalizeString(value: unknown): string {
  return typeof value === "string" ? value.trim() : String(value ?? "").trim();
}

function toVolumeLabel(volume: Pick<VolumeRow, "volumeValue" | "volumeUnit">): string {
  const value =
    typeof volume.volumeValue === "number" && Number.isFinite(volume.volumeValue)
      ? volume.volumeValue
      : 0;

  const unit = normalizeString(volume.volumeUnit) || "ml";

  if (value <= 0) {
    return "";
  }

  return `${value}${unit}`;
}

function hasEmptyModelNumberValue(modelNumber: ModelNumber): boolean {
  return Object.values(modelNumber).some((value) => {
    if (value == null) {
      return true;
    }

    if (typeof value === "string" && value.trim() === "") {
      return true;
    }

    return false;
  });
}

function hasEmptyAlcoholModelNumberValue(
  modelNumber: AlcoholModelNumber,
): boolean {
  if (!modelNumber.code.trim()) {
    return true;
  }

  if (!modelNumber.volumeLabel.trim()) {
    return true;
  }

  if (
    typeof modelNumber.volume.value !== "number" ||
    !Number.isFinite(modelNumber.volume.value) ||
    modelNumber.volume.value <= 0
  ) {
    return true;
  }

  if (!modelNumber.volume.unit.trim()) {
    return true;
  }

  return false;
}

export function useProductBlueprintCreateValidation(
  params: UseProductBlueprintCreateValidationParams,
): () => string[] {
  return React.useCallback((): string[] => {
    const errors: string[] = [];

    if (!params.companyId) {
      errors.push("companyId が取得できません。ログインし直してください。");
    }

    if (!params.productName.trim()) {
      errors.push("商品名は必須です。");
    }

    if (!params.brandId) {
      errors.push("ブランドを選択してください。");
    }

    if (!params.productBlueprintCategoryId || !params.productBlueprintCategory) {
      errors.push("商品カテゴリを選択してください。");
    }

    if (params.weight < 0) {
      errors.push("重さは 0 以上の値を入力してください。");
    }

    if (params.isApparelCategory) {
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
          errors.push("モデルナンバー欄に空欄があります。すべて入力してください。");
        }
      }

      if (params.modelNumbers.length > 0) {
        const seenCodes = new Set<string>();
        const duplicateCodes = new Set<string>();

        params.modelNumbers.forEach((modelNumber) => {
          const code = modelNumber.code?.trim();

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
        const hasEmpty = params.alcoholModelNumbers.some(
          hasEmptyAlcoholModelNumberValue,
        );

        if (hasEmpty) {
          errors.push("容量ごとのモデルナンバー欄に空欄があります。すべて入力してください。");
        }
      }

      if (params.volumes.length > 0) {
        const seenVolumes = new Set<string>();
        const duplicateVolumes = new Set<string>();

        params.volumes.forEach((volume) => {
          const label = toVolumeLabel(volume);

          if (!label) {
            errors.push("容量は 0 より大きい値を入力してください。");
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
          const code = modelNumber.code.trim();

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

      if (params.volumes.length > 0 && params.alcoholModelNumbers.length > 0) {
        const validVolumeLabels = new Set(
          params.volumes.map(toVolumeLabel).filter(Boolean),
        );

        const missingModelNumberVolumes = params.volumes
          .map(toVolumeLabel)
          .filter(Boolean)
          .filter(
            (label) =>
              !params.alcoholModelNumbers.some(
                (modelNumber) => modelNumber.volumeLabel === label,
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
          .map((modelNumber) => modelNumber.volumeLabel)
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