// frontend/console/shell/src/shared/types/apparel.ts

// ============================
// Apparel category definitions
// ============================

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

const APPAREL_CATEGORY_CODE_SET:
  ReadonlySet<string> = new Set(
  APPAREL_CATEGORY_OPTIONS.map(
    (option) => option.value,
  ),
);

export function isApparelCategoryCode(
  value: string,
): value is ApparelCategoryCode {
  return APPAREL_CATEGORY_CODE_SET.has(
    value,
  );
}

// ============================
// Apparel fit definitions
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
// Apparel category fields
// ============================

/**
 * ProductBlueprint.categoryFieldsに保存する
 * apparel固有のfield key。
 *
 * color / size / measurementsはModelVariation側で扱うため、
 * categoryFieldsには含めない。
 */
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

/**
 * apparelカテゴリごとに利用可能な
 * ProductBlueprint.categoryFieldsのkey一覧。
 */
export const APPAREL_CATEGORY_FIELD_KEYS:
  Record<
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

/**
 * apparelカテゴリコードに対応する
 * ProductBlueprint.categoryFieldsのkey一覧を返す。
 */
export function getApparelCategoryFieldKeys(
  categoryCode: string,
): ApparelCategoryFieldKey[] {
  if (
    !isApparelCategoryCode(
      categoryCode,
    )
  ) {
    return [];
  }

  return (
    APPAREL_CATEGORY_FIELD_KEYS[
      categoryCode
    ] ?? []
  );
}

// ============================
// Measurement definitions
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

/**
 * 既存コードから段階的に移行するためのalias。
 *
 * 新規コードではMeasurementKeyを使用する。
 */
export type ApparelMeasurementKey =
  MeasurementKey;

export type MeasurementOption = {
  value: MeasurementKey;
  label: string;
};

export const MEASUREMENT_OPTIONS:
  MeasurementOption[] = [
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

/**
 * 既存コードから段階的に移行するためのalias。
 *
 * 配列を複製せず、MEASUREMENT_OPTIONSと
 * 同じ定義を参照する。
 */
export const APPAREL_MEASUREMENT_OPTIONS =
  MEASUREMENT_OPTIONS;

// ============================
// Category measurement mappings
// ============================

export const APPAREL_CATEGORY_MEASUREMENT_KEYS:
  Record<
    ApparelCategoryCode,
    MeasurementKey[]
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

export const APPAREL_CATEGORY_MEASUREMENT_OPTIONS:
  Record<
    ApparelCategoryCode,
    MeasurementOption[]
  > = Object.fromEntries(
  APPAREL_CATEGORY_OPTIONS.map(
    (category) => {
      const measurementKeys =
        APPAREL_CATEGORY_MEASUREMENT_KEYS[
          category.value
        ];

      const options =
        MEASUREMENT_OPTIONS.filter(
          (option) =>
            measurementKeys.includes(
              option.value,
            ),
        );

      return [
        category.value,
        options,
      ];
    },
  ),
) as Record<
  ApparelCategoryCode,
  MeasurementOption[]
>;

// ============================
// Measurements
// ============================

export type ApparelMeasurements = Partial<
  Record<
    MeasurementKey,
    number | null
  >
>;

export type ApparelMeasurementInput =
  | Record<
      string,
      number | null | undefined
    >
  | undefined
  | null;

// ============================
// Apparel size input
// ============================

/**
 * ModelとProductBlueprintで共通利用する、
 * UI固有IDを持たないサイズ・採寸入力型。
 *
 * Model側:
 *
 * type SizeRow = ApparelSizeInput & {
 *   modelId: string;
 * };
 *
 * ProductBlueprint側:
 *
 * type ApparelSizeRow = ApparelSizeInput & {
 *   id: string;
 * };
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

// ============================
// Apparel model number
// ============================

/**
 * サイズと色の組み合わせに対応する型番。
 *
 * ModelVariationの作成・更新処理で共通利用する。
 */
export type ApparelModelNumberRow = {
  size: string;
  color: string;
  code: string;
};

// ============================
// Size field mapping
// ============================

type ApparelMeasurementSizeField =
  Exclude<
    keyof ApparelSizeInput,
    "sizeLabel"
  >;

const SIZE_FIELD_TO_MEASUREMENT_KEY:
  Readonly<
    Record<
      ApparelMeasurementSizeField,
      MeasurementKey
    >
  > = {
  length: "着丈",
  width: "身幅",
  chest: "胸囲",
  shoulder: "肩幅",
  sleeveLength: "袖丈",

  waist: "ウエスト",
  hip: "ヒップ",
  rise: "股上",
  inseam: "股下",
  thigh: "わたり幅",
  hemWidth: "裾幅",
};

function toMeasurementKeyFromSizeField(
  key: string,
): MeasurementKey | null {
  if (
    !Object.prototype.hasOwnProperty.call(
      SIZE_FIELD_TO_MEASUREMENT_KEY,
      key,
    )
  ) {
    return null;
  }

  return SIZE_FIELD_TO_MEASUREMENT_KEY[
    key as ApparelMeasurementSizeField
  ];
}

// ============================
// Normalization
// ============================

/**
 * 採寸値をAPI・Repositoryで扱える形式へ正規化する。
 *
 * - null / undefinedを除外
 * - NaN / Infinity / -Infinityを除外
 * - SizeInputの英語field名を日本語の採寸キーへ変換
 * - 日本語の採寸キーはそのまま維持
 * - 有効値がない場合は空objectを返す
 */
export function normalizeApparelMeasurements(
  measurements: ApparelMeasurementInput,
): Record<string, number> {
  const normalized:
    Record<string, number> = {};

  for (
    const [key, value]
    of Object.entries(
      measurements ?? {},
    )
  ) {
    if (
      typeof value !== "number" ||
      !Number.isFinite(value)
    ) {
      continue;
    }

    const measurementKey =
      toMeasurementKeyFromSizeField(
        key,
      ) ?? key;

    normalized[measurementKey] =
      value;
  }

  return normalized;
}

/**
 * ModelVariationのrequest用に採寸値を正規化する。
 *
 * 有効な採寸値がない場合はundefinedを返し、
 * request bodyからmeasurementsを省略可能にする。
 */
export function normalizeApparelMeasurementsForRequest(
  measurements: ApparelMeasurementInput,
): Record<string, number> | undefined {
  const normalized =
    normalizeApparelMeasurements(
      measurements,
    );

  return Object.keys(
    normalized,
  ).length > 0
    ? normalized
    : undefined;
}