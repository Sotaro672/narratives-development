// frontend/console/shell/src/features/model/application/modelCreateService.tsx

import type * as React from "react";
import type {
  ApparelSizeInput,
  MeasurementOption,
} from "../../../shared/types/apparel";

/**
 * Model variation作成画面で使用する型定義と表示用ユーティリティをまとめる。
 *
 * ProductBlueprintは容量を保持しない。
 * Alcohol商品の容量はModel variationのVolumeだけを正とする。
 * 配送用の梱包情報はModel variationごとのShippingPackageを正とする。
 */

/* =========================================================
 * Common
 * =======================================================*/

export type Volume = {
  value: number;
  unit: string;
};

/**
 * 配送用の梱包後情報。
 * weightGramsは重量(g)、widthMmは横(mm)、lengthMmは縦(mm)、heightMmは高さ(mm)を表す。
 */
export type ShippingPackage = {
  weightGrams: number;
  widthMm: number;
  lengthMm: number;
  heightMm: number;
};

export type ModelVariationMode = "edit" | "view";

/* =========================================================
 * SizeVariationCard / apparel variation
 * =======================================================*/

/**
 * Model variation画面で使用するサイズ・採寸入力行。
 * 共通の入力項目はApparelSizeInputを使用し、UI上の行識別に必要なidだけを追加する。
 */
export type SizeRow = ApparelSizeInput & {
  id: string;
};

/* =========================================================
 * VolumeVariationCard / alcohol variation
 * =======================================================*/

/**
 * Alcohol商品の容量入力行。
 * 容量はProductBlueprintやcategoryFieldsには保存せず、Model variationのvolumeとして保存する。
 */
export type VolumeRow = {
  id: string;
  volumeValue: number;
  volumeUnit: string;
};

/* =========================================================
 * ModelNumber
 * =======================================================*/

/**
 * Apparel用model number。
 * sizeとcolorの組み合わせごとにmodel numberと配送用梱包情報を持つ。
 */
export type ApparelModelNumber = {
  kind?: "apparel";
  size: string;
  color: string;
  code: string;
  rgb?: string | number;
  shippingPackage: ShippingPackage;
};

/**
 * Alcohol用model number。
 * 容量はvolumeだけを正とし、model numberごとに配送用梱包情報を持つ。
 */
export type AlcoholModelNumber = {
  kind: "alcohol";
  volume: Volume;
  code: string;
  shippingPackage: ShippingPackage;
};

export type SizeLike = {
  id: string;
  sizeLabel: string;
};

/* =========================================================
 * UseModelCard - apparel
 * =======================================================*/

export type UseModelCardParams = {
  kind?: "apparel";
  sizes: SizeLike[];
  colors: string[];
  modelNumbers: ApparelModelNumber[];
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
  flatModelNumbers: ApparelModelNumber[];
};

/* =========================================================
 * SizeVariationCard
 * =======================================================*/

export type SizePatch = Partial<Omit<SizeRow, "id">>;

export type UseSizeVariationCardParams = {
  sizes: SizeRow[];
  mode?: ModelVariationMode;
  measurementOptions?: MeasurementOption[];
  onChangeSize?: (
    id: string,
    patch: SizePatch,
  ) => void;
};

export type UseSizeVariationCardResult = {
  isEdit: boolean;
  readonlyInputProps: {
    variant?: "readonly";
    readOnly?: boolean;
  };
  measurementHeaders: string[];
  handleChange: (
    id: string,
    key: keyof Omit<SizeRow, "id">,
  ) => (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => void;
};

/* =========================================================
 * Volume helpers
 * =======================================================*/

/**
 * 入力行をModel variation保存用のVolumeへ変換する。
 */
export function volumeRowToVolume(row: VolumeRow): Volume {
  return {
    value: row.volumeValue,
    unit: row.volumeUnit || "ml",
  };
}