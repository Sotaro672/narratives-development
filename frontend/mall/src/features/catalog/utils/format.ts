// frontend/amol/src/features/catalog/utils/format.ts

export function formatPrice(price: number): string {
  if (!Number.isFinite(price)) {
    return "価格未設定";
  }

  return `${price.toLocaleString("ja-JP")}円`;
}

export function renderRatingStars(rating: number): string {
  const safeRating = Math.max(0, Math.min(5, Math.round(rating)));

  return "★".repeat(safeRating) + "☆".repeat(5 - safeRating);
}