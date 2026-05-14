// frontend/console/productBlueprint/src/domain/entity/alcohol.ts

export type AlcoholCategoryCode =
  | "alcohol.beer"
  | "alcohol.sake"
  | "alcohol.shochu"
  | "alcohol.spirits"
  | "alcohol.whisky"
  | "alcohol.wine";

export type AlcoholCategoryOption = {
  value: AlcoholCategoryCode;
  label: string;
};

export const ALCOHOL_CATEGORY_OPTIONS: AlcoholCategoryOption[] = [
  { value: "alcohol.beer", label: "ビール" },
  { value: "alcohol.sake", label: "日本酒" },
  { value: "alcohol.shochu", label: "焼酎" },
  { value: "alcohol.spirits", label: "スピリッツ" },
  { value: "alcohol.whisky", label: "ウイスキー" },
  { value: "alcohol.wine", label: "ワイン" },
];

export type AlcoholCategoryFieldKey =
  | "vintage"
  | "region"
  | "material"
  | "alcoholContent";

export type AlcoholCategoryFields = Partial<
  Record<AlcoholCategoryFieldKey, string | number | null>
>;

export const ALCOHOL_CATEGORY_FIELD_KEYS: Record<
  AlcoholCategoryCode,
  AlcoholCategoryFieldKey[]
> = {
  "alcohol.beer": ["vintage", "region", "material", "alcoholContent"],
  "alcohol.sake": ["vintage", "region", "material", "alcoholContent"],
  "alcohol.shochu": ["vintage", "region", "material", "alcoholContent"],
  "alcohol.spirits": ["vintage", "region", "material", "alcoholContent"],
  "alcohol.whisky": ["vintage", "region", "material", "alcoholContent"],
  "alcohol.wine": ["vintage", "region", "material", "alcoholContent"],
};

export function isAlcoholCategoryCode(
  value: string,
): value is AlcoholCategoryCode {
  return (
    value === "alcohol.beer" ||
    value === "alcohol.sake" ||
    value === "alcohol.shochu" ||
    value === "alcohol.spirits" ||
    value === "alcohol.whisky" ||
    value === "alcohol.wine"
  );
}

export function getAlcoholCategoryFieldKeys(
  categoryCode: string,
): AlcoholCategoryFieldKey[] {
  if (!isAlcoholCategoryCode(categoryCode)) {
    return [];
  }

  return ALCOHOL_CATEGORY_FIELD_KEYS[categoryCode] ?? [];
}