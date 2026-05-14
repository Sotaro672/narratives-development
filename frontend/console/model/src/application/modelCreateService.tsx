// frontend/console/model/src/application/modelCreateService.tsx

import type * as React from "react";

import type { MeasurementOption } from "../domain/entity/catalog";

/**
 * モデル作成（CreateModelVariation など）のための
 * アプリケーション層の型定義とユーティリティをまとめるファイル。
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
 * SizeVariationCard / apparel Variation 関連の正規型
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
 * VolumeVariationCard / alcohol Variation 関連の正規型
 * =======================================================*/

/**
 * alcohol model variation は容量のみを model 側の正とする。
 *
 * 例:
 * - 720 ml
 * - 1000 ml
 * - 1800 ml
 */
export type VolumeRow = {
  id: string;
  volumeValue: number;
  volumeUnit: string;
};

/**
 * 既存の SizeLike 相当として、画面表示用に容量を label 化した型。
 * ModelNumberCard を流用する場合は sizeLabel に "720ml" のような値を入れる。
 */
export type VolumeLike = {
  id: string;
  volumeLabel: string;
  volume: Volume;
};

/* =========================================================
 * ModelNumber 関連
 * =======================================================*/

/**
 * apparel 用 modelNumber。
 *
 * size + color ごとに modelNumber を持つ。
 */
export type ApparelModelNumber = {
  kind?: "apparel";
  size: string;
  color: string;
  code: string;
  rgb?: string | number;
};

/**
 * alcohol 用 modelNumber。
 *
 * volume ごとに modelNumber を持つ。
 */
export type AlcoholModelNumber = {
  kind: "alcohol";
  volume: Volume;
  volumeLabel: string;
  code: string;
};

/**
 * 既存互換:
 * 既存コードで ModelNumber を apparel 前提として参照している箇所を壊さないため、
 * ModelNumber は従来通り apparel 用の alias とする。
 */
export type ModelNumber = ApparelModelNumber;

export type AnyModelNumber = ApparelModelNumber | AlcoholModelNumber;

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
 * UseAlcoholModelCard - alcohol
 * =======================================================*/

export type UseAlcoholModelCardParams = {
  kind: "alcohol";
  volumes: VolumeLike[];
  modelNumbers: AlcoholModelNumber[];
  onChangeModelNumber?: (volumeLabel: string, nextCode: string) => void;
};

export type UseAlcoholModelCardResult = {
  getCode: (volumeLabel: string) => string;
  onChangeModelNumber: (volumeLabel: string, nextCode: string) => void;
  flatModelNumbers: AlcoholModelNumber[];
};

/* =========================================================
 * SizeVariationCard 関連
 * =======================================================*/

export type SizePatch = Partial<Omit<SizeRow, "id">>;

export type UseSizeVariationCardParams = {
  sizes: SizeRow[];
  mode?: ModelVariationMode;
  measurementOptions?: MeasurementOption[];
  onChangeSize?: (id: string, patch: SizePatch) => void;
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
  ) => (e: React.ChangeEvent<HTMLInputElement>) => void;
};

/* =========================================================
 * VolumeVariationCard 関連
 * =======================================================*/

export type VolumePatch = Partial<Omit<VolumeRow, "id">>;

export type UseVolumeVariationCardParams = {
  volumes: VolumeRow[];
  mode?: ModelVariationMode;
  onChangeVolume?: (id: string, patch: VolumePatch) => void;
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
  ) => (e: React.ChangeEvent<HTMLInputElement>) => void;
};

/* =========================================================
 * Helpers
 * =======================================================*/

export function toVolumeLabel(volume: Volume): string {
  const value = Number.isFinite(volume.value) ? volume.value : 0;
  const unit = volume.unit.trim() || "ml";

  return `${value}${unit}`;
}

export function volumeRowToVolume(row: VolumeRow): Volume {
  return {
    value: row.volumeValue,
    unit: row.volumeUnit,
  };
}

export function volumeRowToVolumeLike(row: VolumeRow): VolumeLike {
  const volume = volumeRowToVolume(row);

  return {
    id: row.id,
    volume,
    volumeLabel: toVolumeLabel(volume),
  };
}

export function volumeRowsToVolumeLikes(rows: VolumeRow[]): VolumeLike[] {
  return rows.map(volumeRowToVolumeLike);
}

export function parseVolumeLabel(label: string): Volume {
  const trimmed = label.trim();

  const match = trimmed.match(/^(\d+(?:\.\d+)?)\s*([a-zA-Z]+)$/);
  if (!match) {
    return {
      value: 0,
      unit: "ml",
    };
  }

  const value = Number(match[1]);
  const unit = match[2] || "ml";

  return {
    value: Number.isFinite(value) ? value : 0,
    unit,
  };
}