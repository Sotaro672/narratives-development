// frontend/console/shell/src/features/model/application/modelCreateService.tsx

import type * as React from "react";

import type { MeasurementOption } from "../domain/entity/catalog";

/**
 * Model variation作成画面で使用する型定義と
 * 表示用ユーティリティをまとめる。
 *
 * ProductBlueprintは容量を保持しない。
 * Alcohol商品の容量はModel variationのVolumeだけを正とする。
 */

/* =========================================================
 * Common
 * =======================================================*/

export type ModelVariationKind = "apparel" | "alcohol";

export type Volume = {
  value: number;
  unit: string;
};

export type ModelVariationMode = "edit" | "view";

/* =========================================================
 * SizeVariationCard / apparel variation
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
 * VolumeVariationCard / alcohol variation
 * =======================================================*/

/**
 * Alcohol商品の容量入力行。
 *
 * 容量はProductBlueprintやcategoryFieldsには保存せず、
 * Model variationのvolumeとして保存する。
 */
export type VolumeRow = {
  id: string;
  volumeValue: number;
  volumeUnit: string;
};

/**
 * Alcohol variationの画面表示用データ。
 *
 * volumeLabelは保持しない。
 * 表示文字列はtoVolumeLabel(volume)で都度生成する。
 */
export type VolumeLike = {
  id: string;
  volume: Volume;
};

/* =========================================================
 * ModelNumber
 * =======================================================*/

/**
 * Apparel用model number。
 *
 * sizeとcolorの組み合わせごとにmodel numberを持つ。
 */
export type ApparelModelNumber = {
  kind?: "apparel";
  size: string;
  color: string;
  code: string;
  rgb?: string | number;
};

/**
 * Alcohol用model number。
 *
 * 容量はvolumeだけを正とする。
 * 表示用のvolumeLabelは保持しない。
 */
export type AlcoholModelNumber = {
  kind: "alcohol";
  volume: Volume;
  code: string;
};

/**
 * 既存のApparel向け利用箇所で使用する型。
 */
export type ModelNumber = ApparelModelNumber;

export type AnyModelNumber =
  | ApparelModelNumber
  | AlcoholModelNumber;

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
  getCode: (
    sizeLabel: string,
    color: string,
  ) => string;
  onChangeModelNumber: (
    sizeLabel: string,
    color: string,
    nextCode: string,
  ) => void;
  flatModelNumbers: ApparelModelNumber[];
};

/* =========================================================
 * UseAlcoholModelCard - alcohol
 * =======================================================*/

export type UseAlcoholModelCardParams = {
  kind: "alcohol";
  volumes: VolumeLike[];
  modelNumbers: AlcoholModelNumber[];
  onChangeModelNumber?: (
    volume: Volume,
    nextCode: string,
  ) => void;
};

export type UseAlcoholModelCardResult = {
  getCode: (volume: Volume) => string;
  onChangeModelNumber: (
    volume: Volume,
    nextCode: string,
  ) => void;
  flatModelNumbers: AlcoholModelNumber[];
};

/* =========================================================
 * SizeVariationCard
 * =======================================================*/

export type SizePatch = Partial<
  Omit<SizeRow, "id">
>;

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
 * VolumeVariationCard
 * =======================================================*/

export type VolumePatch = Partial<
  Omit<VolumeRow, "id">
>;

export type UseVolumeVariationCardParams = {
  volumes: VolumeRow[];
  mode?: ModelVariationMode;
  onChangeVolume?: (
    id: string,
    patch: VolumePatch,
  ) => void;
};

export type UseVolumeVariationCardResult = {
  isEdit: boolean;
  readonlyInputProps: {
    variant?: "readonly";
    readOnly?: boolean;
  };
  handleChange: (
    id: string,
    key: keyof Omit<VolumeRow, "id">,
  ) => (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => void;
};

/* =========================================================
 * Volume helpers
 * =======================================================*/

/**
 * Model variationのVolumeから表示文字列を生成する。
 *
 * 表示文字列は派生値であり、保存対象にはしない。
 */
export function toVolumeLabel(
  volume: Volume,
): string {
  const value = Number.isFinite(volume.value)
    ? volume.value
    : 0;

  const unit = volume.unit || "ml";

  return `${value}${unit}`;
}

/**
 * 入力行をModel variation保存用のVolumeへ変換する。
 */
export function volumeRowToVolume(
  row: VolumeRow,
): Volume {
  return {
    value: row.volumeValue,
    unit: row.volumeUnit || "ml",
  };
}
