// frontend/amol/src/features/brand/presentation/utils/buildBrandInitial.ts

export function buildBrandInitial(
  brandName: string,
): string {
  const normalizedBrandName =
    brandName.trim();

  if (!normalizedBrandName) {
    return "B";
  }

  return normalizedBrandName
    .slice(0, 1)
    .toUpperCase();
}