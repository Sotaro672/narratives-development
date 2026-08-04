//frontend\amol\src\features\market\types\marketReview.ts
import type {
  ProductBlueprintReviewPage,
} from "./review";

export type MarketProductBlueprintReviewsResponse =
  | ProductBlueprintReviewPage
  | {
      data?: unknown;
      items?: unknown;
      reviews?: unknown;
      page?: unknown;
      perPage?: unknown;
      total?: unknown;
      totalCount?: unknown;
      hasNext?: unknown;
    };