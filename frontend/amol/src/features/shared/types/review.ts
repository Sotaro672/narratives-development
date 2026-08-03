// frontend/amol/src/features/shared/types/review.ts

export type ProductBlueprintReview = {
  id: string;
  productBlueprintId: string;
  avatarId: string;
  avatarName: string;
  avatarIcon: string;
  rating: number;
  title: string;
  body: string;
  helpfulVotes: number;
  totalVotes: number;
  reviewedAt: string;
  status: string;
};

export type ProductBlueprintReviewPage = {
  items: ProductBlueprintReview[];
  page: number;
  perPage: number;
  total: number;
  hasNext: boolean;
};