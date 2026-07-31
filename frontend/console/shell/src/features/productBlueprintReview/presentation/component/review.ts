// frontend\console\shell\src\features\productBlueprintReview\presentation\component\review.ts

import type { ReviewStatus } from "../../../../shared/types/productBlueprintReview";

export const ReviewStatusLabelJa: Record<ReviewStatus, string> = {
  PUBLISHED: "公開",
  HIDDEN: "非公開",
  REMOVED: "削除",
};

export function statusLabelJa(
  status: ReviewStatus | string | null | undefined,
): string {
  return (
    ReviewStatusLabelJa[status as ReviewStatus] ??
    String(status ?? "-")
  );
}

export function ratingToStars(
  rating: number,
  max = 5,
): string {
  const normalizedMax = Math.max(
    0,
    Math.floor(Number(max) || 0),
  );

  const normalizedRating = Math.max(
    0,
    Math.min(
      normalizedMax,
      Math.round(Number(rating) || 0),
    ),
  );

  return (
    "★".repeat(normalizedRating) +
    "☆".repeat(normalizedMax - normalizedRating)
  );
}