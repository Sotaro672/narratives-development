// frontend/console/productBlueprint/src/domain/entity/other.ts

export type OtherCategoryCode = "other.general";

export type OtherCategoryOption = {
  value: OtherCategoryCode;
  label: string;
};

export const OTHER_CATEGORY_OPTIONS: OtherCategoryOption[] = [
  { value: "other.general", label: "その他一般" },
];

export type OtherCategoryFieldKey = never;

export type OtherCategoryFields = Record<string, never>;

export const OTHER_CATEGORY_FIELD_KEYS: Record<
  OtherCategoryCode,
  OtherCategoryFieldKey[]
> = {
  "other.general": [],
};

export function isOtherCategoryCode(value: string): value is OtherCategoryCode {
  return value === "other.general";
}

export function getOtherCategoryFieldKeys(
  categoryCode: string,
): OtherCategoryFieldKey[] {
  if (!isOtherCategoryCode(categoryCode)) {
    return [];
  }

  return OTHER_CATEGORY_FIELD_KEYS[categoryCode] ?? [];
}