// frontend/console/productBlueprint/src/domain/entity/apparel.ts

// ============================
// Apparel category codes
// ============================
//
// itemType は廃止。
// 旧 tops / bottoms は productBlueprintCategory.code の
// apparel.tops / apparel.bottoms として扱う。

export type ApparelCategoryCode =
  | "apparel.tops"
  | "apparel.bottoms"
  | "apparel.outerwear"
  | "apparel.dress"
  | "apparel.shoes"
  | "apparel.bag"
  | "apparel.accessory";

export type ApparelCategoryOption = {
  value: ApparelCategoryCode;
  label: string;
};

export const APPAREL_CATEGORY_OPTIONS: ApparelCategoryOption[] = [
  { value: "apparel.tops", label: "トップス" },
  { value: "apparel.bottoms", label: "ボトムス" },
  { value: "apparel.outerwear", label: "アウター" },
  { value: "apparel.dress", label: "ワンピース" },
  { value: "apparel.shoes", label: "靴" },
  { value: "apparel.bag", label: "バッグ" },
  { value: "apparel.accessory", label: "アクセサリー" },
];

export function isApparelCategoryCode(
  value: string,
): value is ApparelCategoryCode {
  return (
    value === "apparel.tops" ||
    value === "apparel.bottoms" ||
    value === "apparel.outerwear" ||
    value === "apparel.dress" ||
    value === "apparel.shoes" ||
    value === "apparel.bag" ||
    value === "apparel.accessory"
  );
}

// ============================
// Apparel measurement definitions
// ============================

export type ApparelMeasurementKey =
  // tops / outerwear / dress
  | "shoulderWidth"
  | "bodyWidth"
  | "bodyLength"
  | "sleeveLength"
  | "neckWidth"

  // bottoms
  | "waist"
  | "hip"
  | "rise"
  | "inseam"
  | "thighWidth"
  | "hemWidth"
  | "totalLength"

  // shoes
  | "heelHeight"

  // generic
  | "width"
  | "height"
  | "depth";

export type ApparelMeasurementOption = {
  key: ApparelMeasurementKey;
  label: string;
  unit: "cm";
};

export const APPAREL_MEASUREMENT_OPTIONS: ApparelMeasurementOption[] = [
  // tops / outerwear / dress
  { key: "shoulderWidth", label: "肩幅", unit: "cm" },
  { key: "bodyWidth", label: "身幅", unit: "cm" },
  { key: "bodyLength", label: "着丈", unit: "cm" },
  { key: "sleeveLength", label: "袖丈", unit: "cm" },
  { key: "neckWidth", label: "首回り", unit: "cm" },

  // bottoms
  { key: "waist", label: "ウエスト", unit: "cm" },
  { key: "hip", label: "ヒップ", unit: "cm" },
  { key: "rise", label: "股上", unit: "cm" },
  { key: "inseam", label: "股下", unit: "cm" },
  { key: "thighWidth", label: "わたり幅", unit: "cm" },
  { key: "hemWidth", label: "裾幅", unit: "cm" },
  { key: "totalLength", label: "総丈", unit: "cm" },

  // shoes
  { key: "heelHeight", label: "ヒール高", unit: "cm" },

  // generic
  { key: "width", label: "幅", unit: "cm" },
  { key: "height", label: "高さ", unit: "cm" },
  { key: "depth", label: "奥行き", unit: "cm" },
];

export const APPAREL_CATEGORY_MEASUREMENT_KEYS: Record<
  ApparelCategoryCode,
  ApparelMeasurementKey[]
> = {
  "apparel.tops": [
    "shoulderWidth",
    "bodyWidth",
    "bodyLength",
    "sleeveLength",
    "neckWidth",
  ],
  "apparel.bottoms": [
    "waist",
    "hip",
    "rise",
    "inseam",
    "thighWidth",
    "hemWidth",
    "totalLength",
  ],
  "apparel.outerwear": [
    "shoulderWidth",
    "bodyWidth",
    "bodyLength",
    "sleeveLength",
  ],
  "apparel.dress": [
    "shoulderWidth",
    "bodyWidth",
    "bodyLength",
    "sleeveLength",
    "waist",
    "hip",
    "totalLength",
  ],
  "apparel.shoes": ["heelHeight"],
  "apparel.bag": ["width", "height", "depth"],
  "apparel.accessory": ["width", "height"],
};

export const APPAREL_CATEGORY_MEASUREMENT_OPTIONS: Record<
  ApparelCategoryCode,
  ApparelMeasurementOption[]
> = Object.fromEntries(
  Object.entries(APPAREL_CATEGORY_MEASUREMENT_KEYS).map(
    ([categoryCode, keys]) => [
      categoryCode,
      APPAREL_MEASUREMENT_OPTIONS.filter((option) =>
        keys.includes(option.key),
      ),
    ],
  ),
) as Record<ApparelCategoryCode, ApparelMeasurementOption[]>;

export type ApparelMeasurements = Partial<
  Record<ApparelMeasurementKey, number | null>
>;

// ============================
// Apparel model variation input
// ============================

export type ApparelModelVariationPayload = {
  sizeLabel: string;
  color: string;
  modelNumber: string;
  createdBy?: string;
  rgb?: number;
  measurements: ApparelMeasurements;
};

export type ApparelModelNumberRow = {
  size: string;
  color: string;
  code: string;
};

/**
 * Service / API 入力用。
 * 更新処理では row id を使わないため、id は持たせない。
 */
export type ApparelSizeInput = {
  sizeLabel: string;

  // tops / outerwear / dress
  shoulderWidth?: number;
  bodyWidth?: number;
  bodyLength?: number;
  sleeveLength?: number;
  neckWidth?: number;

  // bottoms / dress
  waist?: number;
  hip?: number;
  rise?: number;
  inseam?: number;
  thighWidth?: number;
  hemWidth?: number;
  totalLength?: number;

  // shoes
  heelHeight?: number;

  // bag / accessory / generic
  width?: number;
  height?: number;
  depth?: number;
};

/**
 * UI 行用。
 * 画面 state やリスト描画では id を使えるようにする。
 */
export type ApparelSizeRow = ApparelSizeInput & {
  id: string;
};

// ============================
// フィット種別
// ============================

export type Fit =
  | "レギュラーフィット"
  | "スリムフィット"
  | "リラックスフィット"
  | "オーバーサイズ";

export const FIT_OPTIONS: { value: Fit; label: string }[] = [
  { value: "レギュラーフィット", label: "レギュラーフィット" },
  { value: "スリムフィット", label: "スリムフィット" },
  { value: "リラックスフィット", label: "リラックスフィット" },
  { value: "オーバーサイズ", label: "オーバーサイズ" },
];

// ============================
// 商品IDタグ選択肢
// ============================
//
// NOTE:
// Product ID tag は apparel 専用ではない。
// 現時点では既存 import 影響を抑えるためここに残す。
// 後続で common/productIdTag.ts のような共通ファイルへ移動するのが望ましい。

export const PRODUCT_ID_TAG_OPTIONS: { value: string; label: string }[] = [
  { value: "QRコード", label: "QRコード" },
  { value: "NFC", label: "NFC" },
];

// ============================
// 品質保証（洗濯タグ）
// 6つのカテゴリー階層付き
// ============================

export type WashTagCategory =
  | "洗濯"
  | "漂白"
  | "乾燥"
  | "アイロン"
  | "ドライクリーニング"
  | "ウェットクリーニング";

export type WashTagOption = {
  value: string;
  label: string;
  category: WashTagCategory;
};

export const WASH_TAG_OPTIONS: WashTagOption[] = [
  { category: "洗濯", value: "手洗い", label: "手洗い" },
  { category: "洗濯", value: "洗濯機可", label: "洗濯機可" },
  { category: "洗濯", value: "弱い洗濯", label: "弱い洗濯" },
  { category: "洗濯", value: "液温30℃限度", label: "液温30℃限度" },
  { category: "洗濯", value: "液温40℃限度", label: "液温40℃限度" },
  { category: "洗濯", value: "水洗い不可", label: "水洗い不可" },

  { category: "漂白", value: "酸素系漂白可", label: "酸素系漂白可" },
  { category: "漂白", value: "塩素系漂白可", label: "塩素系漂白可" },
  { category: "漂白", value: "漂白不可", label: "漂白不可" },

  { category: "乾燥", value: "タンブル乾燥可 低温", label: "タンブル乾燥可（低温）" },
  { category: "乾燥", value: "タンブル乾燥可 中温", label: "タンブル乾燥可（中温）" },
  { category: "乾燥", value: "タンブル乾燥不可", label: "タンブル乾燥不可" },
  { category: "乾燥", value: "つり干し", label: "つり干し" },
  { category: "乾燥", value: "日陰つり干し", label: "日陰つり干し" },
  { category: "乾燥", value: "平干し", label: "平干し" },
  { category: "乾燥", value: "日陰平干し", label: "日陰平干し" },

  { category: "アイロン", value: "アイロン低温", label: "アイロン低温（110℃まで）" },
  { category: "アイロン", value: "アイロン中温", label: "アイロン中温（150℃まで）" },
  { category: "アイロン", value: "アイロン高温", label: "アイロン高温（200℃まで）" },
  { category: "アイロン", value: "アイロン不可", label: "アイロン不可" },

  {
    category: "ドライクリーニング",
    value: "ドライクリーニング可",
    label: "ドライクリーニング可",
  },
  {
    category: "ドライクリーニング",
    value: "石油系ドライ可",
    label: "石油系ドライクリーニング可",
  },
  {
    category: "ドライクリーニング",
    value: "ドライクリーニング不可",
    label: "ドライクリーニング不可",
  },

  {
    category: "ウェットクリーニング",
    value: "ウェットクリーニング可",
    label: "ウェットクリーニング可",
  },
  {
    category: "ウェットクリーニング",
    value: "ウェットクリーニング弱",
    label: "ウェットクリーニング（弱）",
  },
  {
    category: "ウェットクリーニング",
    value: "ウェットクリーニング非常に弱",
    label: "ウェットクリーニング（非常に弱）",
  },
  {
    category: "ウェットクリーニング",
    value: "ウェットクリーニング不可",
    label: "ウェットクリーニング不可",
  },
];

// ============================
// helpers
// ============================

export function getApparelMeasurementOptions(
  categoryCode: string,
): ApparelMeasurementOption[] {
  if (!isApparelCategoryCode(categoryCode)) {
    return [];
  }

  return APPAREL_CATEGORY_MEASUREMENT_OPTIONS[categoryCode] ?? [];
}

export function isApparelMeasurementRequiredCategory(
  categoryCode: string,
): boolean {
  return (
    categoryCode === "apparel.tops" ||
    categoryCode === "apparel.bottoms" ||
    categoryCode === "apparel.outerwear" ||
    categoryCode === "apparel.dress"
  );
}

export function normalizeApparelMeasurements(
  measurements: ApparelMeasurements | undefined | null,
): Record<string, number> {
  const out: Record<string, number> = {};

  for (const [key, value] of Object.entries(measurements ?? {})) {
    if (value == null) continue;
    if (Number.isNaN(value)) continue;
    out[key] = value;
  }

  return out;
}