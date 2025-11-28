// frontend/console/model/src/application/modelCreateService.tsx

// React のイベント型だけ型として利用
import type * as React from "react";

// 採寸系の型は model ドメインの catalog から
import type {
  MeasurementOption,
  SizeRow,
} from "../domain/entity/catalog";

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
