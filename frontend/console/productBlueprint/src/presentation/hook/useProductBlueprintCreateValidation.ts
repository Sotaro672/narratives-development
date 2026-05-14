// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintCreateValidation.ts

import * as React from "react";

import type { ModelNumber } from "../../../../model/src/application/modelCreateService";
import type { ApparelSizeRow } from "../../domain/entity/apparel";
import type { ProductBlueprintCategorySnapshot } from "../../domain/entity/productBlueprintCategory";

export type UseProductBlueprintCreateValidationParams = {
  companyId: string;
  productName: string;
  brandId: string;
  productBlueprintCategoryId: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot | null;
  weight: number;
  isApparelCategory: boolean;
  colors: string[];
  sizes: ApparelSizeRow[];
  modelNumbers: ModelNumber[];
};

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
        const hasEmpty = params.modelNumbers.some((modelNumber) =>
          Object.values(modelNumber).some((value) => {
            if (value == null) {
              return true;
            }

            if (typeof value === "string" && value.trim() === "") {
              return true;
            }

            return false;
          }),
        );

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
          const labelRaw = size.sizeLabel;
          const label =
            typeof labelRaw === "string"
              ? labelRaw.trim()
              : String(labelRaw ?? "").trim();

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

    return errors;
  }, [params]);
}