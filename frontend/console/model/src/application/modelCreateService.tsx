// frontend/console/model/src/application/modelCreateService.tsx

// React のイベント型だけ型として利用
import type * as React from "react";

// 採寸系の型は model ドメインの catalog から
import type {
  MeasurementOption,
  SizeRow,
  MeasurementKey,
} from "../domain/entity/catalog";

// ★ HTTP リポジトリ（CreateModelVariation 用）
import {
  createModelVariations,
  type CreateModelVariationRequest,
} from "../infrastructure/repository/modelRepositoryHTTP";

/**
 * モデル作成（CreateModelVariation など）のための
 * アプリケーション層の型定義とユーティリティをまとめるファイル。
 */

/* =========================================================
 * ModelNumber 関連
 * =======================================================*/

export type ModelNumber = {
  size: string;
  color: string;
  code: string;
  rgb?: string | number;
};

export type SizeLike = { id: string; sizeLabel: string };

export type UseModelCardParams = {
  sizes: SizeLike[];
  colors: string[];
  modelNumbers: ModelNumber[];
  colorRgbMap?: Record<string, string>;
};

export type UseModelCardResult = {
  getCode: (sizeLabel: string, color: string) => string;
  onChangeModelNumber: (
    sizeLabel: string,
    color: string,
    nextCode: string,
  ) => void;
  flatModelNumbers: ModelNumber[];
};

/* =========================================================
 * SizeVariationCard 関連
 * =======================================================*/

export type SizePatch = Partial<Omit<SizeRow, "id">>;

export type UseSizeVariationCardParams = {
  sizes: SizeRow[];
  mode?: "edit" | "view";
  measurementOptions?: MeasurementOption[];
  onChangeSize?: (id: string, patch: SizePatch) => void;
};

export type UseSizeVariationCardResult = {
  isEdit: boolean;
  readonlyInputProps: { variant?: "readonly"; readOnly?: boolean };
  measurementHeaders: string[];
  handleChange: (
    id: string,
    key: keyof Omit<SizeRow, "id">,
  ) => (e: React.ChangeEvent<HTMLInputElement>) => void;
};

/* =========================================================
 * ProductBlueprint Create 後に受け取る JSON 用の型
 * =======================================================*/

export type NewModelVariationMeasurements = Partial<
  Record<MeasurementKey, number | null>
>;

export type NewModelVariationPayload = {
  sizeLabel: string;
  color: string;
  rgb?: number | string;
  modelNumber: string;
  createdBy: string;
  measurements: NewModelVariationMeasurements;
};

export type ModelVariationsFromProductBlueprint = {
  productBlueprintId: string;
  variations: NewModelVariationPayload[];
};

/**
 * NewModelVariationMeasurements を backend API 用の Record<string, number> に正規化
 */
function normalizeMeasurements(
  raw: NewModelVariationMeasurements | undefined,
): Record<string, number> {
  const result: Record<string, number> = {};
  if (!raw) return result;

  for (const [key, value] of Object.entries(raw)) {
    if (value == null) continue;
    const n = Number(value);
    if (Number.isNaN(n)) continue;
    result[key] = n;
  }

  return result;
}

/**
 * PB 作成後の variations JSON → CreateModelVariationRequest に変換して
 * backend API を叩く
 */
export async function createModelVariationsFromProductBlueprint(
  payload: ModelVariationsFromProductBlueprint,
): Promise<void> {
  const requests: CreateModelVariationRequest[] = payload.variations.map(
    (v) => {
      const measurements = normalizeMeasurements(v.measurements);

      let rgbInt: number | undefined = undefined;
      if (typeof v.rgb === "string") {
        const hex = v.rgb.replace("#", "");
        if (/^[0-9a-fA-F]{6}$/.test(hex)) {
          rgbInt = parseInt(hex, 16);
        }
      } else if (typeof v.rgb === "number") {
        rgbInt = v.rgb;
      }

      return {
        productBlueprintId: payload.productBlueprintId,
        modelNumber: v.modelNumber,
        size: v.sizeLabel,
        color: v.color,
        rgb: rgbInt,
        measurements,
        version: 1,
      };
    },
  );

  await createModelVariations(payload.productBlueprintId, requests);
}
