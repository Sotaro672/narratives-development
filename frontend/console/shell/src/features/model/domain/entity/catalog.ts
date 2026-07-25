// frontend/console/model/src/domain/entity/catalog.ts

// ============================
// アイテム種別（モデル／商品設計共通）
// ============================

export type ItemType = "トップス" | "ボトムス";

export const ITEM_TYPE_OPTIONS: { value: ItemType; label: string }[] = [
  { value: "トップス", label: "トップス" },
  { value: "ボトムス", label: "ボトムス" },
];

// ============================
// 採寸項目
// ============================

export type MeasurementKey =
  // トップス
  | "着丈"
  | "身幅"
  | "胸囲"
  | "肩幅"
  | "袖丈"
  // ボトムス
  | "ウエスト"
  | "ヒップ"
  | "股上"
  | "股下"
  | "わたり幅"
  | "裾幅";

export type MeasurementOption = { value: MeasurementKey; label: string };

export const MEASUREMENT_OPTIONS: MeasurementOption[] = [
  // トップス
  { value: "着丈", label: "着丈" },
  { value: "身幅", label: "身幅" },
  { value: "胸囲", label: "胸囲" },
  { value: "肩幅", label: "肩幅" },
  { value: "袖丈", label: "袖丈" },

  // ボトムス
  { value: "ウエスト", label: "ウエスト" },
  { value: "ヒップ", label: "ヒップ" },
  { value: "股上", label: "股上" },
  { value: "股下", label: "股下" },
  { value: "わたり幅", label: "わたり幅" },
  { value: "裾幅", label: "裾幅" },
];

export const ITEM_TYPE_MEASUREMENT_KEYS: Record<ItemType, MeasurementKey[]> = {
  トップス: ["着丈", "身幅", "胸囲", "肩幅", "袖丈"],
  ボトムス: ["ウエスト", "ヒップ", "股上", "股下", "わたり幅", "裾幅"],
};

export const ITEM_TYPE_MEASUREMENT_OPTIONS: Record<
  ItemType,
  MeasurementOption[]
> = {
  トップス: MEASUREMENT_OPTIONS.filter((m) =>
    ITEM_TYPE_MEASUREMENT_KEYS["トップス"].includes(m.value)
  ),
  ボトムス: MEASUREMENT_OPTIONS.filter((m) =>
    ITEM_TYPE_MEASUREMENT_KEYS["ボトムス"].includes(m.value)
  ),
};

// ============================
// サイズ行
// ============================

export type SizeRow = {
  modelId: string;
  sizeLabel: string;

  // トップス
  length?: number;       // 着丈
  width?: number;        // 身幅
  chest?: number;         // 胸囲（新規）
  shoulder?: number;     // 肩幅
  sleeveLength?: number; // 袖丈

  // ボトムス
  waist?: number;   // ウエスト
  hip?: number;     // ヒップ
  rise?: number;    // 股上
  inseam?: number;  // 股下
  thigh?: number;   // わたり幅
  hemWidth?: number; // 裾幅
};
