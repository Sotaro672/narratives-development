//frontend\console\shell\src\shared\format\review.ts
import type { ReviewStatus } from "../../../../productBlueprintReview/src/domain/entity";

export const ReviewStatusLabelJa: Record<ReviewStatus, string> = {
  PUBLISHED: "公開",
  HIDDEN: "非公開",
  REMOVED: "削除",
};

export function statusLabelJa(s: ReviewStatus | string | null | undefined): string {
  return (ReviewStatusLabelJa as any)[s ?? ""] ?? String(s ?? "-");
}

export function ratingToStars(rating: number, max = 5): string {
  const r = Math.max(0, Math.min(max, Math.round(Number(rating || 0))));
  return "★".repeat(r) + "☆".repeat(max - r);
}