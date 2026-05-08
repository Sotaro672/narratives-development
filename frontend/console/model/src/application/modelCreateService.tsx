// frontend/console/model/src/application/modelCreateService.tsx

import type * as React from "react";

import type { MeasurementOption } from "../domain/entity/catalog";

/**
 * モデル作成（CreateModelVariation など）のための
 * アプリケーション層の型定義とユーティリティをまとめるファイル。
 */

/* =========================================================
 * SizeVariationCard / Variation 関連の正規型
 * =======================================================*/

export type SizeRow = {
  id: string;
  sizeLabel: string;
  length?: number;
  width?: number;
  chest?: number;
  shoulder?: number;
  sleeveLength?: number;
  waist?: number;
  hip?: number;
  rise?: number;
  inseam?: number;
  thigh?: number;
  hemWidth?: number;
};

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
  onChangeModelNumber?: (
    sizeLabel: string,
    color: string,
    nextCode: string,
  ) => void;
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