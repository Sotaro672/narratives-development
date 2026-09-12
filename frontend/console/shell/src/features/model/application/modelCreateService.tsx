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
 *
 * 配送用梱包寸法について:
 * - Presentation層ではcm単位で入力・表示する。
 * - Application層以降ではmm単位の整数値を正規値として扱う。
 * - ShippingPackageのwidthMm / lengthMm / heightMmは常にmm単位とする。
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
 *
 * Application層以降で使用する正規形式。
 *
 * - weightGrams: 重量(g)
 * - widthMm: 横(mm)
 * - lengthMm: 縦(mm)
 * - heightMm: 高さ(mm)
 *
 * Console上では寸法をcm単位で入力・表示するが、
 * ShippingPackageへ格納する時点ではmm単位の整数へ変換する。
 *
 * 例:
 * - 32.5cm → widthMm: 325
 * - 24cm   → lengthMm: 240
 * - 8cm    → heightMm: 80
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
 *
 * ApparelSizeInputの採寸値はmm単位で扱う。
 */
export type SizeRow = ApparelSizeInput & {
  id: string;
};

/* =========================================================
 * VolumeVariationCard / alcohol variation
 * =======================================================*/

/**
 * Alcohol商品の容量入力行。
 * 容量はProductBlueprintやcategoryFieldsには保存せず、
 * Model variationのvolumeとして保存する。
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
 *
 * shippingPackageの寸法値はmm単位。
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
 *
 * shippingPackageの寸法値はmm単位。
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