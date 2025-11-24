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
// 採寸項目（本ファイルへ移譲）
// ============================

// 採寸キー（トップス・ボトムス共通で使える union）
export type MeasurementKey =
  // トップス系
  | "着丈"
  | "身幅"
  | "肩幅"
  | "袖丈"
  // ボトムス系
  | "ウエスト"
  | "ヒップ"
  | "股上"
  | "股下"
  | "わたり幅"
  | "裾幅";

// UI 用採寸選択肢
export type MeasurementOption = { value: MeasurementKey; label: string };

// 全採寸項目のマスタ
export const MEASUREMENT_OPTIONS: MeasurementOption[] = [
  // トップス
  { value: "着丈", label: "着丈" },
  { value: "身幅", label: "身幅" },
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

// アイテム種別ごとの採寸項目紐づけ
export const ITEM_TYPE_MEASUREMENT_KEYS: Record<ItemType, MeasurementKey[]> = {
  トップス: ["着丈", "身幅", "肩幅", "袖丈"],
  ボトムス: ["ウエスト", "ヒップ", "股上", "股下", "わたり幅", "裾幅"],
};

// UI でそのまま使えるように Option まで紐づけたマップ
export const ITEM_TYPE_MEASUREMENT_OPTIONS: Record<
  ItemType,
  MeasurementOption[]
> = {
  トップス: MEASUREMENT_OPTIONS.filter((m) =>
    ITEM_TYPE_MEASUREMENT_KEYS["トップス"].includes(m.value),
  ),
  ボトムス: MEASUREMENT_OPTIONS.filter((m) =>
    ITEM_TYPE_MEASUREMENT_KEYS["ボトムス"].includes(m.value),
  ),
};

// ============================
// サイズ行（SizeVariation / useModelCard 共通）
// ============================

export type SizeRow = {
  id: string;
  sizeLabel: string;

  // ───────── トップス系（正規フィールド）─────────
  // 着丈
  length?: number;
  // 身幅
  chest?: number;
  // 肩幅
  shoulder?: number;
  // 袖丈
  sleeveLength?: number;

  // ───────── ボトムス系（正規フィールド）─────────
  // ウエスト
  waist?: number;
  // ヒップ
  hip?: number;
  // 股上
  rise?: number;
  // 股下
  inseam?: number;
  // わたり幅
  thigh?: number;
  // 裾幅
  hemWidth?: number;

  // ───────── 旧／別名フィールド（型エラー回避用の alias）─────────
  // lengthTop → length の alias 的フィールド
  lengthTop?: number;
  // bodyWidth → chest の alias
  bodyWidth?: number;
  // shoulderWidth → shoulder の alias
  shoulderWidth?: number;
  // thighWidth → thigh の alias
  thighWidth?: number;
};
