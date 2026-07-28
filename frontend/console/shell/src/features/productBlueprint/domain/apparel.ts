// frontend/console/shell/src/features/productBlueprint/domain/apparel.ts

// ============================
// Apparel category codes
// ============================
//
// 旧 tops / bottoms は productBlueprintCategory.code の
// apparel.tops / apparel.bottoms として扱う。

export const APPAREL_CATEGORY_OPTIONS = [
  {
    value: "apparel.tops",
    label: "トップス",
  },
  {
    value: "apparel.bottoms",
    label: "ボトムス",
  },
  {
    value: "apparel.outerwear",
    label: "アウター",
  },
  {
    value: "apparel.dress",
    label: "ワンピース",
  },
  {
    value: "apparel.shoes",
    label: "靴",
  },
  {
    value: "apparel.bag",
    label: "バッグ",
  },
  {
    value: "apparel.accessory",
    label: "アクセサリー",
  },
] as const;

export type ApparelCategoryCode =
  (typeof APPAREL_CATEGORY_OPTIONS)[number]["value"];

export type ApparelCategoryOption = {
  value: ApparelCategoryCode;
  label: string;
};

const APPAREL_CATEGORY_CODE_SET: ReadonlySet<string> =
  new Set(
    APPAREL_CATEGORY_OPTIONS.map(
      (option) => option.value,
    ),
  );

export function isApparelCategoryCode(
  value: string,
): value is ApparelCategoryCode {
  return APPAREL_CATEGORY_CODE_SET.has(value);
}

// ============================
// フィット種別
// ============================

export const FIT_OPTIONS = [
  {
    value: "レギュラーフィット",
    label: "レギュラーフィット",
  },
  {
    value: "スリムフィット",
    label: "スリムフィット",
  },
  {
    value: "リラックスフィット",
    label: "リラックスフィット",
  },
  {
    value: "オーバーサイズ",
    label: "オーバーサイズ",
  },
] as const;

export type Fit =
  (typeof FIT_OPTIONS)[number]["value"];

// ============================
// Apparel productBlueprint category fields
// ============================
//
// ProductBlueprint の共通 field:
// - brandId
// - productName
// - productIdTagType
// - description
//
// 上記は categoryFields には入れない。
// 以下は productBlueprint.categoryFields に入る apparel 専用 field。

export type ApparelCategoryFieldKey =
  | "weight"
  | "fit"
  | "material"
  | "washTags";

export type ApparelCategoryFields = {
  weight?: number | null;
  fit?: Fit | null;
  material?: string | null;
  washTags?: string[];
};

export const APPAREL_CATEGORY_FIELD_KEYS: Record<
  ApparelCategoryCode,
  ApparelCategoryFieldKey[]
> = {
  "apparel.tops": [
    "weight",
    "fit",
    "material",
    "washTags",
  ],
  "apparel.bottoms": [
    "weight",
    "fit",
    "material",
    "washTags",
  ],
  "apparel.outerwear": [
    "material",
    "washTags",
  ],
  "apparel.dress": [
    "weight",
    "fit",
    "material",
    "washTags",
  ],
  "apparel.shoes": [
    "material",
    "washTags",
  ],
  "apparel.bag": [
    "material",
    "washTags",
  ],
  "apparel.accessory": [
    "material",
    "washTags",
  ],
};

export function getApparelCategoryFieldKeys(
  categoryCode: string,
): ApparelCategoryFieldKey[] {
  if (!isApparelCategoryCode(categoryCode)) {
    return [];
  }

  return (
    APPAREL_CATEGORY_FIELD_KEYS[
      categoryCode
    ] ?? []
  );
}

// ============================
// Apparel measurement definitions
// ============================
//
// model/src/domain/entity/catalog.ts の MeasurementKey / MeasurementOption を正とする。
// そのため、productBlueprint 側でも option は key ではなく value を持つ。
//
// 入力表では measurements は以下のみ:
// - apparel.tops
// - apparel.bottoms
// - apparel.dress
//
// outerwear / shoes は color / size のみ。
// bag / accessory は model variation を作らない。

export type ApparelMeasurementKey =
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

export type MeasurementOption = {
  value: ApparelMeasurementKey;
  label: string;
};

export const APPAREL_MEASUREMENT_OPTIONS: MeasurementOption[] =
  [
    // トップス
    {
      value: "着丈",
      label: "着丈",
    },
    {
      value: "身幅",
      label: "身幅",
    },
    {
      value: "胸囲",
      label: "胸囲",
    },
    {
      value: "肩幅",
      label: "肩幅",
    },
    {
      value: "袖丈",
      label: "袖丈",
    },

    // ボトムス
    {
      value: "ウエスト",
      label: "ウエスト",
    },
    {
      value: "ヒップ",
      label: "ヒップ",
    },
    {
      value: "股上",
      label: "股上",
    },
    {
      value: "股下",
      label: "股下",
    },
    {
      value: "わたり幅",
      label: "わたり幅",
    },
    {
      value: "裾幅",
      label: "裾幅",
    },
  ];

export const APPAREL_CATEGORY_MEASUREMENT_KEYS: Record<
  ApparelCategoryCode,
  ApparelMeasurementKey[]
> = {
  "apparel.tops": [
    "着丈",
    "身幅",
    "胸囲",
    "肩幅",
    "袖丈",
  ],
  "apparel.bottoms": [
    "ウエスト",
    "ヒップ",
    "股上",
    "股下",
    "わたり幅",
    "裾幅",
  ],
  "apparel.outerwear": [],
  "apparel.dress": [
    "着丈",
    "身幅",
    "胸囲",
    "肩幅",
    "袖丈",
    "ウエスト",
    "ヒップ",
  ],
  "apparel.shoes": [],
  "apparel.bag": [],
  "apparel.accessory": [],
};

export const APPAREL_CATEGORY_MEASUREMENT_OPTIONS: Record<
  ApparelCategoryCode,
  MeasurementOption[]
> = Object.fromEntries(
  Object.entries(
    APPAREL_CATEGORY_MEASUREMENT_KEYS,
  ).map(([categoryCode, values]) => [
    categoryCode,
    APPAREL_MEASUREMENT_OPTIONS.filter(
      (option) =>
        values.includes(option.value),
    ),
  ]),
) as Record<
  ApparelCategoryCode,
  MeasurementOption[]
>;

export type ApparelMeasurements = Partial<
  Record<
    ApparelMeasurementKey,
    number | null
  >
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
 *
 * model/src/domain/entity/catalog.ts の SizeRow に合わせた field 名にする。
 */
export type ApparelSizeInput = {
  sizeLabel: string;

  // トップス
  length?: number;
  width?: number;
  chest?: number;
  shoulder?: number;
  sleeveLength?: number;

  // ボトムス
  waist?: number;
  hip?: number;
  rise?: number;
  inseam?: number;
  thigh?: number;
  hemWidth?: number;
};

/**
 * UI 行用。
 * 画面 state やリスト描画では id を使えるようにする。
 */
export type ApparelSizeRow =
  ApparelSizeInput & {
    id: string;
  };

// ============================
// 商品IDタグ選択肢
// ============================
//
// NOTE:
// Product ID tag は apparel 専用ではない。
// 現時点では既存 import 影響を抑えるためここに残す。
// 後続で common/productIdTag.ts のような共通ファイルへ移動するのが望ましい。

export const PRODUCT_ID_TAG_OPTIONS: {
  value: string;
  label: string;
}[] = [
  {
    value: "QRコード",
    label: "QRコード",
  },
  {
    value: "NFC",
    label: "NFC",
  },
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

export const WASH_TAG_OPTIONS: WashTagOption[] =
  [
    {
      category: "洗濯",
      value: "手洗い",
      label: "手洗い",
    },
    {
      category: "洗濯",
      value: "洗濯機可",
      label: "洗濯機可",
    },
    {
      category: "洗濯",
      value: "弱い洗濯",
      label: "弱い洗濯",
    },
    {
      category: "洗濯",
      value: "液温30℃限度",
      label: "液温30℃限度",
    },
    {
      category: "洗濯",
      value: "液温40℃限度",
      label: "液温40℃限度",
    },
    {
      category: "洗濯",
      value: "水洗い不可",
      label: "水洗い不可",
    },

    {
      category: "漂白",
      value: "酸素系漂白可",
      label: "酸素系漂白可",
    },
    {
      category: "漂白",
      value: "塩素系漂白可",
      label: "塩素系漂白可",
    },
    {
      category: "漂白",
      value: "漂白不可",
      label: "漂白不可",
    },

    {
      category: "乾燥",
      value: "タンブル乾燥可 低温",
      label:
        "タンブル乾燥可（低温）",
    },
    {
      category: "乾燥",
      value: "タンブル乾燥可 中温",
      label:
        "タンブル乾燥可（中温）",
    },
    {
      category: "乾燥",
      value: "タンブル乾燥不可",
      label: "タンブル乾燥不可",
    },
    {
      category: "乾燥",
      value: "つり干し",
      label: "つり干し",
    },
    {
      category: "乾燥",
      value: "日陰つり干し",
      label: "日陰つり干し",
    },
    {
      category: "乾燥",
      value: "平干し",
      label: "平干し",
    },
    {
      category: "乾燥",
      value: "日陰平干し",
      label: "日陰平干し",
    },

    {
      category: "アイロン",
      value: "アイロン低温",
      label:
        "アイロン低温（110℃まで）",
    },
    {
      category: "アイロン",
      value: "アイロン中温",
      label:
        "アイロン中温（150℃まで）",
    },
    {
      category: "アイロン",
      value: "アイロン高温",
      label:
        "アイロン高温（200℃まで）",
    },
    {
      category: "アイロン",
      value: "アイロン不可",
      label: "アイロン不可",
    },

    {
      category:
        "ドライクリーニング",
      value:
        "ドライクリーニング可",
      label:
        "ドライクリーニング可",
    },
    {
      category:
        "ドライクリーニング",
      value: "石油系ドライ可",
      label:
        "石油系ドライクリーニング可",
    },
    {
      category:
        "ドライクリーニング",
      value:
        "ドライクリーニング不可",
      label:
        "ドライクリーニング不可",
    },

    {
      category:
        "ウェットクリーニング",
      value:
        "ウェットクリーニング可",
      label:
        "ウェットクリーニング可",
    },
    {
      category:
        "ウェットクリーニング",
      value:
        "ウェットクリーニング弱",
      label:
        "ウェットクリーニング（弱）",
    },
    {
      category:
        "ウェットクリーニング",
      value:
        "ウェットクリーニング非常に弱",
      label:
        "ウェットクリーニング（非常に弱）",
    },
    {
      category:
        "ウェットクリーニング",
      value:
        "ウェットクリーニング不可",
      label:
        "ウェットクリーニング不可",
    },
  ];

// ============================
// helpers
// ============================

function toMeasurementKeyFromSizeField(
  key: string,
): ApparelMeasurementKey | null {
  switch (key) {
    case "length":
      return "着丈";

    case "width":
      return "身幅";

    case "chest":
      return "胸囲";

    case "shoulder":
      return "肩幅";

    case "sleeveLength":
      return "袖丈";

    case "waist":
      return "ウエスト";

    case "hip":
      return "ヒップ";

    case "rise":
      return "股上";

    case "inseam":
      return "股下";

    case "thigh":
      return "わたり幅";

    case "hemWidth":
      return "裾幅";

    default:
      return null;
  }
}

export function normalizeApparelMeasurements(
  measurements:
    | Record<
        string,
        number | null | undefined
      >
    | undefined
    | null,
): Record<string, number> {
  const out: Record<string, number> = {};

  for (const [key, value] of Object.entries(
    measurements ?? {},
  )) {
    if (
      typeof value !== "number" ||
      !Number.isFinite(value)
    ) {
      continue;
    }

    const measurementKey =
      toMeasurementKeyFromSizeField(key) ??
      key;

    out[measurementKey] = value;
  }

  return out;
}

export function normalizeApparelMeasurementsForRequest(
  measurements:
    | Record<
        string,
        number | null | undefined
      >
    | undefined
    | null,
): Record<string, number> | undefined {
  const normalized =
    normalizeApparelMeasurements(
      measurements,
    );

  return Object.keys(normalized).length > 0
    ? normalized
    : undefined;
}