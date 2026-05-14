// frontend/console/productBlueprint/src/domain/entity/categoryCardVisibility.ts

export type CategoryCardVisibility = {
  showVintage: boolean;
  showRegion: boolean;
  showWeight: boolean;
  showFit: boolean;
  showMaterial: boolean;
  showAlcoholContent: boolean;
  showVolume: boolean;
  showWashTags: boolean;
};

export function getCategoryCardVisibility(
  categoryCode: string,
): CategoryCardVisibility {
  const code = categoryCode.trim();

  const isAlcohol =
    code === "alcohol.beer" ||
    code === "alcohol.sake" ||
    code === "alcohol.shochu" ||
    code === "alcohol.spirits" ||
    code === "alcohol.whisky" ||
    code === "alcohol.wine";

  const isApparelMaterialOnly =
    code === "apparel.accessory" ||
    code === "apparel.bag" ||
    code === "apparel.outerwear" ||
    code === "apparel.shoes";

  const isApparelWithFitAndWeight =
    code === "apparel.bottoms" ||
    code === "apparel.dress" ||
    code === "apparel.tops";

  const isCosmetics =
    code === "cosmetics.bodycare" ||
    code === "cosmetics.fragrance" ||
    code === "cosmetics.haircare" ||
    code === "cosmetics.makeup" ||
    code === "cosmetics.skincare";

  return {
    showVintage: isAlcohol,
    showRegion: isAlcohol,
    showWeight: isApparelWithFitAndWeight,
    showFit: isApparelWithFitAndWeight,
    showMaterial:
      isAlcohol ||
      isApparelMaterialOnly ||
      isApparelWithFitAndWeight ||
      isCosmetics,
    showAlcoholContent: isAlcohol,
    showVolume: isAlcohol || isCosmetics,
    showWashTags: isApparelMaterialOnly || isApparelWithFitAndWeight,
  };
}

export function isNumberCategoryField(key: string): boolean {
  return (
    key === "weight" ||
    key === "vintage" ||
    key === "alcoholContent" ||
    key === "volume"
  );
}

export function toCategoryNumberOrNull(value: string): number | null {
  if (value.trim() === "") {
    return null;
  }

  const n = Number(value);
  if (Number.isNaN(n)) {
    return null;
  }

  return n < 0 ? 0 : n;
}

export function toCategoryInputValue(value: unknown): string | number {
  if (typeof value === "number") return value;
  if (typeof value === "string") return value;
  if (typeof value === "boolean") return value ? "true" : "false";
  return "";
}