// frontend/console/shell/src/features/productBlueprint/domain/categoryCardVisibility.ts

import {
  toProductBlueprintCategoryPathKey,
  type ProductBlueprintCategoryPath,
} from "./productBlueprintCategory";

export type CategoryCardVisibility = Readonly<{
  showVintage: boolean;
  showRegion: boolean;
  showWeight: boolean;
  showFit: boolean;
  showMaterial: boolean;
  showAlcoholContent: boolean;
  showVolume: boolean;
  showWashTags: boolean;
}>;

const EMPTY_VISIBILITY: CategoryCardVisibility = {
  showVintage: false,
  showRegion: false,
  showWeight: false,
  showFit: false,
  showMaterial: false,
  showAlcoholContent: false,
  showVolume: false,
  showWashTags: false,
};

const ALCOHOL_VISIBILITY: CategoryCardVisibility = {
  showVintage: true,
  showRegion: true,
  showWeight: false,
  showFit: false,
  showMaterial: true,
  showAlcoholContent: true,
  showVolume: true,
  showWashTags: false,
};

const APPAREL_MATERIAL_ONLY_VISIBILITY: CategoryCardVisibility = {
  showVintage: false,
  showRegion: false,
  showWeight: false,
  showFit: false,
  showMaterial: true,
  showAlcoholContent: false,
  showVolume: false,
  showWashTags: true,
};

const APPAREL_WITH_FIT_AND_WEIGHT_VISIBILITY: CategoryCardVisibility = {
  showVintage: false,
  showRegion: false,
  showWeight: true,
  showFit: true,
  showMaterial: true,
  showAlcoholContent: false,
  showVolume: false,
  showWashTags: true,
};

const COSMETICS_VISIBILITY: CategoryCardVisibility = {
  showVintage: false,
  showRegion: false,
  showWeight: false,
  showFit: false,
  showMaterial: true,
  showAlcoholContent: false,
  showVolume: true,
  showWashTags: false,
};

const CATEGORY_CARD_VISIBILITY_BY_PATH_KEY: Readonly<
  Record<string, CategoryCardVisibility>
> = {
  "alcohol.beer": ALCOHOL_VISIBILITY,
  "alcohol.sake": ALCOHOL_VISIBILITY,
  "alcohol.shochu": ALCOHOL_VISIBILITY,
  "alcohol.spirits": ALCOHOL_VISIBILITY,
  "alcohol.whisky": ALCOHOL_VISIBILITY,
  "alcohol.wine": ALCOHOL_VISIBILITY,

  "apparel.accessory": APPAREL_MATERIAL_ONLY_VISIBILITY,
  "apparel.bag": APPAREL_MATERIAL_ONLY_VISIBILITY,
  "apparel.outerwear": APPAREL_MATERIAL_ONLY_VISIBILITY,
  "apparel.shoes": APPAREL_MATERIAL_ONLY_VISIBILITY,

  "apparel.bottoms": APPAREL_WITH_FIT_AND_WEIGHT_VISIBILITY,
  "apparel.dress": APPAREL_WITH_FIT_AND_WEIGHT_VISIBILITY,
  "apparel.tops": APPAREL_WITH_FIT_AND_WEIGHT_VISIBILITY,

  "cosmetics.bodycare": COSMETICS_VISIBILITY,
  "cosmetics.fragrance": COSMETICS_VISIBILITY,
  "cosmetics.haircare": COSMETICS_VISIBILITY,
  "cosmetics.makeup": COSMETICS_VISIBILITY,
  "cosmetics.skincare": COSMETICS_VISIBILITY,
};

const NUMBER_CATEGORY_FIELDS = new Set([
  "weight",
  "vintage",
  "alcoholContent",
  "volume",
]);

export function getCategoryCardVisibility(
  productBlueprintCategoryPath: ProductBlueprintCategoryPath,
): CategoryCardVisibility {
  const pathKey =
    toProductBlueprintCategoryPathKey(
      productBlueprintCategoryPath,
    );

  return (
    CATEGORY_CARD_VISIBILITY_BY_PATH_KEY[pathKey] ??
    EMPTY_VISIBILITY
  );
}

export function isNumberCategoryField(
  key: string,
): boolean {
  return NUMBER_CATEGORY_FIELDS.has(key);
}

export function toCategoryNumberOrNull(
  value: string,
): number | null {
  if (value === "") {
    return null;
  }

  const parsed = Number(value);

  if (
    !Number.isFinite(parsed) ||
    parsed < 0
  ) {
    return null;
  }

  return parsed;
}

export function toCategoryInputValue(
  value: unknown,
): string | number {
  if (
    typeof value === "number" &&
    Number.isFinite(value)
  ) {
    return value;
  }

  if (typeof value === "string") {
    return value;
  }

  if (typeof value === "boolean") {
    return value ? "true" : "false";
  }

  return "";
}