// frontend/console/productBlueprint/src/domain/entity/cosmetics.ts

export type CosmeticsCategoryCode =
  | "cosmetics.bodycare"
  | "cosmetics.fragrance"
  | "cosmetics.haircare"
  | "cosmetics.makeup"
  | "cosmetics.skincare";

export type CosmeticsCategoryOption = {
  value: CosmeticsCategoryCode;
  label: string;
};

export const COSMETICS_CATEGORY_OPTIONS: CosmeticsCategoryOption[] = [
  { value: "cosmetics.bodycare", label: "ボディケア" },
  { value: "cosmetics.fragrance", label: "香水" },
  { value: "cosmetics.haircare", label: "ヘアケア" },
  { value: "cosmetics.makeup", label: "メイクアップ" },
  { value: "cosmetics.skincare", label: "スキンケア" },
];

export type CosmeticsCategoryFieldKey = "material" | "volume";

export type CosmeticsCategoryFields = Partial<
  Record<CosmeticsCategoryFieldKey, string | number | null>
>;

export const COSMETICS_CATEGORY_FIELD_KEYS: Record<
  CosmeticsCategoryCode,
  CosmeticsCategoryFieldKey[]
> = {
  "cosmetics.bodycare": ["material", "volume"],
  "cosmetics.fragrance": ["material", "volume"],
  "cosmetics.haircare": ["material", "volume"],
  "cosmetics.makeup": ["material", "volume"],
  "cosmetics.skincare": ["material", "volume"],
};

export function isCosmeticsCategoryCode(
  value: string,
): value is CosmeticsCategoryCode {
  return (
    value === "cosmetics.bodycare" ||
    value === "cosmetics.fragrance" ||
    value === "cosmetics.haircare" ||
    value === "cosmetics.makeup" ||
    value === "cosmetics.skincare"
  );
}

export function getCosmeticsCategoryFieldKeys(
  categoryCode: string,
): CosmeticsCategoryFieldKey[] {
  if (!isCosmeticsCategoryCode(categoryCode)) {
    return [];
  }

  return COSMETICS_CATEGORY_FIELD_KEYS[categoryCode] ?? [];
}